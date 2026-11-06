package workforcemanagement_managementunits_workplan

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/mypurecloud/platform-client-sdk-go/v200/platformclientv2"
	"github.com/mypurecloud/terraform-provider-genesyscloud/genesyscloud/consistency_checker"
	"github.com/mypurecloud/terraform-provider-genesyscloud/genesyscloud/provider"
	resourceExporter "github.com/mypurecloud/terraform-provider-genesyscloud/genesyscloud/resource_exporter"
	"github.com/mypurecloud/terraform-provider-genesyscloud/genesyscloud/util"
	"github.com/mypurecloud/terraform-provider-genesyscloud/genesyscloud/util/constants"
)

/*
	NOTE: This resource's Id is in the format <managementUnitId>/<workPlanId> so we can persist the id of the parent management unit.
	The management_unit_id field can not be used for this because the work plan api does not return the management unit.
*/

/*
The resource_genesyscloud_workforcemanagement_managementunits_workplan.go contains all of the methods that perform the core logic for a resource.
*/

// getAllWorkPlans retrieves all of the work plans via Terraform in the Genesys Cloud and is used for the exporter
func getAllWorkPlans(ctx context.Context, clientConfig *platformclientv2.Configuration) (resourceExporter.ResourceIDMetaMap, diag.Diagnostics) {
	proxy := getWorkPlanProxy(clientConfig)
	resources := make(resourceExporter.ResourceIDMetaMap)

	managementUnits, resp, err := proxy.getAllManagementUnits(ctx)
	if err != nil {
		return nil, util.BuildAPIDiagnosticError(ResourceType, fmt.Sprintf("Failed to get management units: %v", err), resp)
	}

	for _, managementUnit := range *managementUnits {
		if managementUnit.Id == nil || managementUnit.Name == nil {
			continue
		}
		workPlans, resp, err := proxy.getAllWorkPlans(ctx, *managementUnit.Id)
		if err != nil {
			return nil, util.BuildAPIDiagnosticError(ResourceType, fmt.Sprintf("Failed to get work plans of management unit %s: %v", *managementUnit.Id, err), resp)
		}

		for _, workPlan := range *workPlans {
			if workPlan.Id == nil || workPlan.Name == nil {
				continue
			}
			resources[*managementUnit.Id+"/"+*workPlan.Id] = &resourceExporter.ResourceMeta{BlockLabel: *managementUnit.Name + "_" + *workPlan.Name}
		}
	}

	return resources, nil
}

// createWorkPlan is used by the workforcemanagement_managementunits_workplan resource to create a Genesys Cloud work plan
func createWorkPlan(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	sdkConfig := meta.(*provider.ProviderMeta).ClientConfig
	proxy := getWorkPlanProxy(sdkConfig)
	managementUnitId := d.Get("management_unit_id").(string)
	workPlan := buildCreateWorkPlanRequest(d.GetRawConfig())

	log.Printf("Creating management unit %s work plan %s", managementUnitId, *workPlan.Name)
	created, resp, err := proxy.createWorkPlan(ctx, managementUnitId, workPlan)
	if err != nil {
		details := validationDetails(ctx, proxy, managementUnitId, newWorkPlanId, resp, func() *platformclientv2.Workplan {
			request, _ := buildUpdateWorkPlanRequest(d.GetRawConfig(), &platformclientv2.Workplan{})
			return request
		})
		return util.BuildAPIDiagnosticError(ResourceType, fmt.Sprintf("Failed to create management unit %s work plan %s: %s%s", managementUnitId, *workPlan.Name, err, details), resp)
	}

	d.SetId(managementUnitId + "/" + *created.Id)
	log.Printf("Created management unit %s work plan %s %s", managementUnitId, *created.Id, *workPlan.Name)
	return readWorkPlan(ctx, d, meta)
}

// readWorkPlan is used by the workforcemanagement_managementunits_workplan resource to read a work plan from Genesys Cloud
func readWorkPlan(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	sdkConfig := meta.(*provider.ProviderMeta).ClientConfig
	proxy := getWorkPlanProxy(sdkConfig)
	cc := consistency_checker.NewConsistencyCheck(ctx, d, meta, ResourceWorkPlan(), constants.ConsistencyChecks(), ResourceType)
	managementUnitId, workPlanId := SplitWorkPlanTerraformId(d.Id())

	log.Printf("Reading management unit %s work plan %s", managementUnitId, workPlanId)

	return util.WithRetriesForRead(ctx, d, func() *retry.RetryError {
		workPlan, resp, getErr := proxy.getWorkPlanById(ctx, managementUnitId, workPlanId)
		if getErr != nil {
			if util.IsStatus404(resp) {
				return retry.RetryableError(util.BuildWithRetriesApiDiagnosticError(ResourceType, fmt.Sprintf("Failed to read management unit %s work plan %s: %s", managementUnitId, workPlanId, getErr), resp))
			}
			return retry.NonRetryableError(util.BuildWithRetriesApiDiagnosticError(ResourceType, fmt.Sprintf("Failed to read management unit %s work plan %s: %s", managementUnitId, workPlanId, getErr), resp))
		}

		_ = d.Set("management_unit_id", managementUnitId)
		scalars := scalarsToMap(workPlan, workPlanScalarKeys)
		for _, key := range workPlanScalarKeys {
			value, found := scalars[key]
			if found {
				_ = d.Set(key, value)
			} else {
				_ = d.Set(key, nil)
			}
		}
		if workPlan.Valid != nil {
			_ = d.Set("valid", *workPlan.Valid)
		}
		if workPlan.OptionalDays != nil && workPlan.OptionalDays.Values != nil {
			_ = d.Set("optional_days", stringSet(*workPlan.OptionalDays.Values))
		}
		_ = d.Set("shift_start_variances", flattenShiftStartVariances(workPlan.ShiftStartVariances))
		priorShifts, _ := d.Get("shifts").([]interface{})
		_ = d.Set("shifts", flattenShifts(workPlan.Shifts, priorShifts))
		_ = d.Set("agent_ids", stringSet(existingAgentIds(workPlan)))

		log.Printf("Read management unit %s work plan %s", managementUnitId, workPlanId)
		return cc.CheckState(d)
	})
}

// updateWorkPlan is used by the workforcemanagement_managementunits_workplan resource to update a work plan in Genesys Cloud
func updateWorkPlan(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	sdkConfig := meta.(*provider.ProviderMeta).ClientConfig
	proxy := getWorkPlanProxy(sdkConfig)
	managementUnitId, workPlanId := SplitWorkPlanTerraformId(d.Id())

	log.Printf("Updating management unit %s work plan %s", managementUnitId, workPlanId)

	// The patch is built against the current shifts and agents of the work plan, and carries its version for optimistic
	// locking, so a version conflict reads the work plan again and rebuilds the patch.
	diagErr := util.WithRetries(ctx, 60*time.Second, func() *retry.RetryError {
		current, resp, err := proxy.getWorkPlanById(ctx, managementUnitId, workPlanId)
		if err != nil {
			return retry.NonRetryableError(util.BuildWithRetriesApiDiagnosticError(ResourceType, fmt.Sprintf("Failed to read management unit %s work plan %s before update: %s", managementUnitId, workPlanId, err), resp))
		}

		request, err := buildUpdateWorkPlanRequest(d.GetRawConfig(), current)
		if err != nil {
			return retry.NonRetryableError(fmt.Errorf("failed to update management unit %s work plan %s: %w", managementUnitId, workPlanId, err))
		}

		_, resp, err = proxy.updateWorkPlan(ctx, managementUnitId, workPlanId, request)
		if err == nil {
			return nil
		}
		if util.IsStatus409(resp) {
			return retry.RetryableError(util.BuildWithRetriesApiDiagnosticError(ResourceType, fmt.Sprintf("Failed to update management unit %s work plan %s: %s", managementUnitId, workPlanId, err), resp))
		}
		details := validationDetails(ctx, proxy, managementUnitId, workPlanId, resp, func() *platformclientv2.Workplan { return request })
		return retry.NonRetryableError(util.BuildWithRetriesApiDiagnosticError(ResourceType, fmt.Sprintf("Failed to update management unit %s work plan %s: %s%s", managementUnitId, workPlanId, err, details), resp))
	})
	if diagErr != nil {
		return diagErr
	}

	log.Printf("Updated management unit %s work plan %s", managementUnitId, workPlanId)
	return readWorkPlan(ctx, d, meta)
}

// deleteWorkPlan is used by the workforcemanagement_managementunits_workplan resource to delete a work plan from Genesys Cloud
func deleteWorkPlan(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	sdkConfig := meta.(*provider.ProviderMeta).ClientConfig
	proxy := getWorkPlanProxy(sdkConfig)
	managementUnitId, workPlanId := SplitWorkPlanTerraformId(d.Id())

	log.Printf("Deleting management unit %s work plan %s", managementUnitId, workPlanId)
	return deleteWorkPlanAndUnassignAgents(ctx, proxy, managementUnitId, workPlanId)
}

// deleteWorkPlanAndUnassignAgents deletes the work plan after unassigning its agents, because Genesys Cloud refuses to delete a work plan
// that still has agents ("Remove the agents and try again"). A work plan that is part of a work plan rotation or override cannot be
// deleted either, which is reported by the api with the rotations involved.
func deleteWorkPlanAndUnassignAgents(ctx context.Context, proxy *workPlanProxy, managementUnitId string, workPlanId string) diag.Diagnostics {
	workPlan, resp, err := proxy.getWorkPlanById(ctx, managementUnitId, workPlanId)
	if err != nil {
		if util.IsStatus404(resp) {
			log.Printf("Management unit %s work plan %s already deleted", managementUnitId, workPlanId)
			return nil
		}
		return util.BuildAPIDiagnosticError(ResourceType, fmt.Sprintf("Failed to read management unit %s work plan %s before deleting it: %s", managementUnitId, workPlanId, err), resp)
	}

	if agentIds := existingAgentIds(workPlan); len(agentIds) > 0 {
		if diagErr := unassignAgents(ctx, proxy, managementUnitId, workPlanId, agentIds); diagErr != nil {
			return diagErr
		}
	}

	resp, err = proxy.deleteWorkPlan(ctx, managementUnitId, workPlanId)
	if err != nil {
		if util.IsStatus404(resp) {
			log.Printf("Management unit %s work plan %s already deleted", managementUnitId, workPlanId)
			return nil
		}
		return util.BuildAPIDiagnosticError(ResourceType, fmt.Sprintf("Failed to delete management unit %s work plan %s: %s", managementUnitId, workPlanId, err), resp)
	}

	log.Printf("Deleted management unit %s work plan %s", managementUnitId, workPlanId)
	return nil
}

// unassignAgents removes the agents from the work plan. The patch carries the work plan version, so a conflict reads the work plan again.
func unassignAgents(ctx context.Context, proxy *workPlanProxy, managementUnitId string, workPlanId string, agentIds []string) diag.Diagnostics {
	return util.WithRetries(ctx, 60*time.Second, func() *retry.RetryError {
		current, resp, err := proxy.getWorkPlanById(ctx, managementUnitId, workPlanId)
		if err != nil {
			return retry.NonRetryableError(util.BuildWithRetriesApiDiagnosticError(ResourceType, fmt.Sprintf("Failed to read management unit %s work plan %s before unassigning its agents: %s", managementUnitId, workPlanId, err), resp))
		}

		request := &platformclientv2.Workplan{Agents: &[]platformclientv2.Deletableuserreference{}}
		for _, agentId := range existingAgentIds(current) {
			*request.Agents = append(*request.Agents, platformclientv2.Deletableuserreference{Id: platformclientv2.String(agentId), Delete: platformclientv2.Bool(true)})
		}
		if len(*request.Agents) == 0 {
			return nil
		}
		if current.Metadata != nil && current.Metadata.Version != nil {
			request.Metadata = &platformclientv2.Wfmversionedentitymetadata{Version: current.Metadata.Version}
		}

		_, resp, err = proxy.updateWorkPlan(ctx, managementUnitId, workPlanId, request)
		if err == nil {
			return nil
		}
		builtDiagErr := util.BuildWithRetriesApiDiagnosticError(ResourceType, fmt.Sprintf("Failed to unassign the %d agent(s) of management unit %s work plan %s before deleting it: %s", len(agentIds), managementUnitId, workPlanId, err), resp)
		if util.IsStatus409(resp) {
			return retry.RetryableError(builtDiagErr)
		}
		return retry.NonRetryableError(builtDiagErr)
	})
}

// newWorkPlanId is the id the validate route takes for a work plan that does not exist yet
const newWorkPlanId = "new"

// validationDetails explains why a rejected create or update failed validation. The save only reports that the work plan failed
// validation, while the validate route (which saves nothing) says what is wrong, so it is asked after a 400.
// It returns an empty string when the failure was not a validation failure or the messages could not be read.
func validationDetails(ctx context.Context, proxy *workPlanProxy, managementUnitId string, workPlanId string, saveResp *platformclientv2.APIResponse, request func() *platformclientv2.Workplan) string {
	if saveResp == nil || saveResp.StatusCode != 400 {
		return ""
	}
	workPlan := request()
	if workPlan == nil {
		return ""
	}

	validation, _, err := proxy.validateWorkPlan(ctx, managementUnitId, workPlanId, toValidationRequest(workPlan))
	if err != nil {
		return ""
	}
	if messages := formatValidationMessages(validation); messages != "" {
		return "\n\nValidation messages:\n" + messages
	}
	return ""
}
