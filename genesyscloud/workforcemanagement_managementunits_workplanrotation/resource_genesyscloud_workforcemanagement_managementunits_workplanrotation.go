package workforcemanagement_managementunits_workplanrotation

import (
	"context"
	"fmt"
	"log"
	"net/http"
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
	NOTE: This resource's Id is in the format <managementUnitId>/<workPlanRotationId> so we can persist the id of the parent management unit.
	The management_unit_id field can not be used for this because the work plan rotation api does not return the management unit.
*/

/*
The resource_genesyscloud_workforcemanagement_managementunits_workplanrotation.go contains all of the methods that perform the core logic for a resource.
*/

// getAllWorkPlanRotations retrieves all of the work plan rotations via Terraform in the Genesys Cloud and is used for the exporter
func getAllWorkPlanRotations(ctx context.Context, clientConfig *platformclientv2.Configuration) (resourceExporter.ResourceIDMetaMap, diag.Diagnostics) {
	proxy := getWorkPlanRotationProxy(clientConfig)
	resources := make(resourceExporter.ResourceIDMetaMap)

	managementUnits, resp, err := proxy.getAllManagementUnits(ctx)
	if err != nil {
		return nil, util.BuildAPIDiagnosticError(ResourceType, fmt.Sprintf("Failed to get management units: %v", err), resp)
	}

	for _, managementUnit := range *managementUnits {
		rotations, resp, err := proxy.getAllWorkPlanRotations(ctx, *managementUnit.Id)
		if err != nil {
			return nil, util.BuildAPIDiagnosticError(ResourceType, fmt.Sprintf("Failed to get work plan rotations of management unit %s: %v", *managementUnit.Id, err), resp)
		}

		for _, rotation := range *rotations {
			resources[*managementUnit.Id+"/"+*rotation.Id] = &resourceExporter.ResourceMeta{BlockLabel: *managementUnit.Name + "_" + *rotation.Name}
		}
	}

	return resources, nil
}

// createWorkPlanRotation is used by the workforcemanagement_managementunits_workplanrotation resource to create a Genesys Cloud work plan rotation
func createWorkPlanRotation(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	sdkConfig := meta.(*provider.ProviderMeta).ClientConfig
	proxy := getWorkPlanRotationProxy(sdkConfig)
	managementUnitId := d.Get("management_unit_id").(string)
	config := d.GetRawConfig()
	rotation := buildAddRequest(config)

	log.Printf("Creating management unit %s work plan rotation %s", managementUnitId, *rotation.Name)
	created, resp, err := proxy.createWorkPlanRotation(ctx, managementUnitId, rotation)
	if err != nil {
		return util.BuildAPIDiagnosticError(ResourceType, fmt.Sprintf("Failed to create management unit %s work plan rotation %s: %s", managementUnitId, *rotation.Name, err), resp)
	}
	d.SetId(managementUnitId + "/" + *created.Id)

	// Genesys Cloud always creates a work plan rotation enabled, so a disabled one needs a second request
	if enabled := boolOf(config, "enabled"); enabled != nil && !*enabled {
		if diagErr := disableWorkPlanRotation(ctx, proxy, managementUnitId, *created.Id); diagErr != nil {
			return diagErr
		}
	}

	log.Printf("Created management unit %s work plan rotation %s %s", managementUnitId, *created.Id, *rotation.Name)
	return readWorkPlanRotation(ctx, d, meta)
}

// disableWorkPlanRotation disables a work plan rotation that was just created
func disableWorkPlanRotation(ctx context.Context, proxy *workPlanRotationProxy, managementUnitId string, rotationId string) diag.Diagnostics {
	return util.WithRetries(ctx, 60*time.Second, func() *retry.RetryError {
		current, resp, err := proxy.getWorkPlanRotationById(ctx, managementUnitId, rotationId)
		if err != nil {
			readDiagErr := util.BuildWithRetriesApiDiagnosticError(ResourceType, fmt.Sprintf("Failed to read management unit %s work plan rotation %s before disabling it: %s", managementUnitId, rotationId, err), resp)
			if isTransientStatus(resp) {
				return retry.RetryableError(readDiagErr)
			}
			return retry.NonRetryableError(readDiagErr)
		}

		request := &platformclientv2.Updateworkplanrotationrequest{Enabled: platformclientv2.Bool(false)}
		if current.Metadata.Version != nil {
			request.Metadata = &platformclientv2.Wfmversionedentitymetadata{Version: current.Metadata.Version}
		}
		_, resp, err = proxy.updateWorkPlanRotation(ctx, managementUnitId, rotationId, request)
		if err == nil {
			return nil
		}
		builtDiagErr := util.BuildWithRetriesApiDiagnosticError(ResourceType, fmt.Sprintf("Failed to disable management unit %s work plan rotation %s: %s", managementUnitId, rotationId, err), resp)
		if util.IsStatus409(resp) || isTransientStatus(resp) {
			return retry.RetryableError(builtDiagErr)
		}
		return retry.NonRetryableError(builtDiagErr)
	})
}

// readWorkPlanRotation is used by the workforcemanagement_managementunits_workplanrotation resource to read a work plan rotation from Genesys Cloud
func readWorkPlanRotation(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	sdkConfig := meta.(*provider.ProviderMeta).ClientConfig
	proxy := getWorkPlanRotationProxy(sdkConfig)
	cc := consistency_checker.NewConsistencyCheck(ctx, d, meta, ResourceWorkPlanRotation(), constants.ConsistencyChecks(), ResourceType)
	managementUnitId, rotationId := SplitWorkPlanRotationTerraformId(d.Id())

	log.Printf("Reading management unit %s work plan rotation %s", managementUnitId, rotationId)

	return util.WithRetriesForRead(ctx, d, func() *retry.RetryError {
		rotation, resp, getErr := proxy.getWorkPlanRotationById(ctx, managementUnitId, rotationId)
		if getErr != nil {
			// A 404 is not retried: the work plan rotation is readable as soon as it is created, so a 404 means it is gone and WithRetriesForRead removes it from state
			return retry.NonRetryableError(util.BuildWithRetriesApiDiagnosticError(ResourceType, fmt.Sprintf("Failed to read management unit %s work plan rotation %s: %s", managementUnitId, rotationId, getErr), resp))
		}

		_ = d.Set("management_unit_id", managementUnitId)
		_ = d.Set("name", *rotation.Name)
		_ = d.Set("enabled", *rotation.Enabled)
		_ = d.Set("start_date", rotation.DateRange.StartBusinessUnitDate.Format(dateLayout))
		_ = d.Set("end_date", nil)
		// Only the end of the date range is nullable
		if rotation.DateRange.EndBusinessUnitDate != nil {
			_ = d.Set("end_date", rotation.DateRange.EndBusinessUnitDate.Format(dateLayout))
		}
		_ = d.Set("work_plan_ids", flattenWorkPlanIds(rotation.Pattern))
		_ = d.Set("agents", flattenAgents(rotation.Agents))

		log.Printf("Read management unit %s work plan rotation %s", managementUnitId, rotationId)
		return cc.CheckState(d)
	})
}

// updateWorkPlanRotation is used by the workforcemanagement_managementunits_workplanrotation resource to update a work plan rotation in Genesys Cloud
func updateWorkPlanRotation(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	sdkConfig := meta.(*provider.ProviderMeta).ClientConfig
	proxy := getWorkPlanRotationProxy(sdkConfig)
	managementUnitId, rotationId := SplitWorkPlanRotationTerraformId(d.Id())

	log.Printf("Updating management unit %s work plan rotation %s", managementUnitId, rotationId)

	// The patch is built against the current agents of the work plan rotation and carries its version for optimistic locking,
	// so a version conflict reads the work plan rotation again and rebuilds the patch.
	diagErr := util.WithRetries(ctx, 60*time.Second, func() *retry.RetryError {
		current, resp, err := proxy.getWorkPlanRotationById(ctx, managementUnitId, rotationId)
		if err != nil {
			readDiagErr := util.BuildWithRetriesApiDiagnosticError(ResourceType, fmt.Sprintf("Failed to read management unit %s work plan rotation %s before update: %s", managementUnitId, rotationId, err), resp)
			if isTransientStatus(resp) {
				return retry.RetryableError(readDiagErr)
			}
			return retry.NonRetryableError(readDiagErr)
		}

		_, resp, err = proxy.updateWorkPlanRotation(ctx, managementUnitId, rotationId, buildUpdateRequest(d.GetRawConfig(), current))
		if err == nil {
			return nil
		}
		builtDiagErr := util.BuildWithRetriesApiDiagnosticError(ResourceType, fmt.Sprintf("Failed to update management unit %s work plan rotation %s: %s", managementUnitId, rotationId, err), resp)
		if util.IsStatus409(resp) || isTransientStatus(resp) {
			return retry.RetryableError(builtDiagErr)
		}
		return retry.NonRetryableError(builtDiagErr)
	})
	if diagErr != nil {
		return diagErr
	}

	log.Printf("Updated management unit %s work plan rotation %s", managementUnitId, rotationId)
	return readWorkPlanRotation(ctx, d, meta)
}

func isTransientStatus(resp *platformclientv2.APIResponse) bool {
	return util.IsStatus429(resp) || (resp != nil && resp.StatusCode >= http.StatusInternalServerError)
}

// deleteWorkPlanRotation is used by the workforcemanagement_managementunits_workplanrotation resource to delete a work plan rotation from Genesys Cloud
func deleteWorkPlanRotation(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	sdkConfig := meta.(*provider.ProviderMeta).ClientConfig
	proxy := getWorkPlanRotationProxy(sdkConfig)
	managementUnitId, rotationId := SplitWorkPlanRotationTerraformId(d.Id())

	log.Printf("Deleting management unit %s work plan rotation %s", managementUnitId, rotationId)
	return deleteWorkPlanRotationAndRemoveAgents(ctx, proxy, managementUnitId, rotationId)
}

// deleteWorkPlanRotationAndRemoveAgents deletes the work plan rotation after removing its agents, because Genesys Cloud refuses to delete a
// work plan rotation that still has agents ("Remove the agents and try again").
func deleteWorkPlanRotationAndRemoveAgents(ctx context.Context, proxy *workPlanRotationProxy, managementUnitId string, rotationId string) diag.Diagnostics {
	rotation, resp, err := proxy.getWorkPlanRotationById(ctx, managementUnitId, rotationId)
	if err != nil {
		if util.IsStatus404(resp) {
			log.Printf("Management unit %s work plan rotation %s already deleted", managementUnitId, rotationId)
			return nil
		}
		return util.BuildAPIDiagnosticError(ResourceType, fmt.Sprintf("Failed to read management unit %s work plan rotation %s before deleting it: %s", managementUnitId, rotationId, err), resp)
	}

	if len(*rotation.Agents) > 0 {
		if diagErr := removeAgents(ctx, proxy, managementUnitId, rotationId); diagErr != nil {
			return diagErr
		}
	}

	resp, err = proxy.deleteWorkPlanRotation(ctx, managementUnitId, rotationId)
	if err != nil {
		if util.IsStatus404(resp) {
			log.Printf("Management unit %s work plan rotation %s already deleted", managementUnitId, rotationId)
			return nil
		}
		return util.BuildAPIDiagnosticError(ResourceType, fmt.Sprintf("Failed to delete management unit %s work plan rotation %s: %s", managementUnitId, rotationId, err), resp)
	}

	log.Printf("Deleted management unit %s work plan rotation %s", managementUnitId, rotationId)
	return nil
}

// removeAgents removes every agent from the work plan rotation. The patch carries the version, so a conflict reads the work plan rotation again.
func removeAgents(ctx context.Context, proxy *workPlanRotationProxy, managementUnitId string, rotationId string) diag.Diagnostics {
	return util.WithRetries(ctx, 60*time.Second, func() *retry.RetryError {
		current, resp, err := proxy.getWorkPlanRotationById(ctx, managementUnitId, rotationId)
		if err != nil {
			readDiagErr := util.BuildWithRetriesApiDiagnosticError(ResourceType, fmt.Sprintf("Failed to read management unit %s work plan rotation %s before removing its agents: %s", managementUnitId, rotationId, err), resp)
			if isTransientStatus(resp) {
				return retry.RetryableError(readDiagErr)
			}
			return retry.NonRetryableError(readDiagErr)
		}

		removals := buildAgentUpdates(*current.Agents, nil)
		if len(removals) == 0 {
			return nil
		}
		request := &platformclientv2.Updateworkplanrotationrequest{Agents: &removals}
		if current.Metadata.Version != nil {
			request.Metadata = &platformclientv2.Wfmversionedentitymetadata{Version: current.Metadata.Version}
		}

		_, resp, err = proxy.updateWorkPlanRotation(ctx, managementUnitId, rotationId, request)
		if err == nil {
			return nil
		}
		builtDiagErr := util.BuildWithRetriesApiDiagnosticError(ResourceType, fmt.Sprintf("Failed to remove the agents of management unit %s work plan rotation %s before deleting it: %s", managementUnitId, rotationId, err), resp)
		if util.IsStatus409(resp) || isTransientStatus(resp) {
			return retry.RetryableError(builtDiagErr)
		}
		return retry.NonRetryableError(builtDiagErr)
	})
}
