package workforcemanagement_businessunits_planninggroup

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
	NOTE: This resource's Id is in the format <businessUnitId>/<planningGroupId> so we can persist the id of the parent business unit.
	The business_unit_id field can not be used for this because the planning group api does not return the business unit.
*/

/*
The resource_genesyscloud_workforcemanagement_businessunits_planninggroup.go contains all of the methods that perform the core logic for a resource.
*/

// getAllPlanningGroups retrieves all of the business unit planning groups via Terraform in the Genesys Cloud and is used for the exporter.
func getAllPlanningGroups(ctx context.Context, clientConfig *platformclientv2.Configuration) (resourceExporter.ResourceIDMetaMap, diag.Diagnostics) {
	proxy := getPlanningGroupProxy(clientConfig)
	resources := make(resourceExporter.ResourceIDMetaMap)

	businessUnits, resp, err := proxy.getAllBusinessUnits(ctx)
	if err != nil {
		return nil, util.BuildAPIDiagnosticError(ResourceType, fmt.Sprintf("Failed to get business units: %v", err), resp)
	}

	for _, businessUnit := range *businessUnits {
		planningGroups, resp, err := proxy.getAllPlanningGroups(ctx, *businessUnit.Id)
		if err != nil {
			return nil, util.BuildAPIDiagnosticError(ResourceType, fmt.Sprintf("Failed to get planning groups of business unit %s: %v", *businessUnit.Id, err), resp)
		}

		for _, planningGroup := range *planningGroups {
			resources[*businessUnit.Id+"/"+*planningGroup.Id] = &resourceExporter.ResourceMeta{BlockLabel: *businessUnit.Name + "_" + *planningGroup.Name}
		}
	}

	return resources, nil
}

// createPlanningGroup is used by the workforcemanagement_businessunits_planninggroup resource to create a Genesys Cloud business unit planning group
func createPlanningGroup(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	sdkConfig := meta.(*provider.ProviderMeta).ClientConfig
	proxy := getPlanningGroupProxy(sdkConfig)
	businessUnitId := d.Get("business_unit_id").(string)
	planningGroup := buildCreatePlanningGroupRequest(d)

	log.Printf("Creating business unit %s planning group %s", businessUnitId, *planningGroup.Name)
	created, resp, err := proxy.createPlanningGroup(ctx, businessUnitId, planningGroup)
	if err != nil {
		return util.BuildAPIDiagnosticError(ResourceType, fmt.Sprintf("Failed to create business unit %s planning group %s: %s", businessUnitId, *planningGroup.Name, err), resp)
	}

	d.SetId(businessUnitId + "/" + *created.Id)
	log.Printf("Created business unit %s planning group %s %s", businessUnitId, *created.Id, *planningGroup.Name)
	return readPlanningGroup(ctx, d, meta)
}

// readPlanningGroup is used by the workforcemanagement_businessunits_planninggroup resource to read a business unit planning group from Genesys Cloud
func readPlanningGroup(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	sdkConfig := meta.(*provider.ProviderMeta).ClientConfig
	proxy := getPlanningGroupProxy(sdkConfig)
	cc := consistency_checker.NewConsistencyCheck(ctx, d, meta, ResourcePlanningGroup(), constants.ConsistencyChecks(), ResourceType)
	businessUnitId, planningGroupId := SplitPlanningGroupTerraformId(d.Id())

	log.Printf("Reading business unit %s planning group %s", businessUnitId, planningGroupId)

	return util.WithRetriesForRead(ctx, d, func() *retry.RetryError {
		planningGroup, resp, getErr := proxy.getPlanningGroupById(ctx, businessUnitId, planningGroupId)
		if getErr != nil {
			// A 404 is not retried: the planning group is readable as soon as it is created, so a 404 means it is gone and WithRetriesForRead removes it from state
			return retry.NonRetryableError(util.BuildWithRetriesApiDiagnosticError(ResourceType, fmt.Sprintf("Failed to read business unit %s planning group %s: %s", businessUnitId, planningGroupId, getErr), resp))
		}

		_ = d.Set("business_unit_id", businessUnitId)
		_ = d.Set("name", *planningGroup.Name)
		_ = d.Set("service_goal_template_id", flattenServiceGoalTemplateId(planningGroup.ServiceGoalTemplate))
		_ = d.Set("route_paths", flattenRoutePaths(planningGroup.RoutePaths))

		log.Printf("Read business unit %s planning group %s", businessUnitId, planningGroupId)
		return cc.CheckState(d)
	})
}

// updatePlanningGroup is used by the workforcemanagement_businessunits_planninggroup resource to update a business unit planning group in Genesys Cloud
func updatePlanningGroup(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	sdkConfig := meta.(*provider.ProviderMeta).ClientConfig
	proxy := getPlanningGroupProxy(sdkConfig)
	businessUnitId, planningGroupId := SplitPlanningGroupTerraformId(d.Id())

	log.Printf("Updating business unit %s planning group %s", businessUnitId, planningGroupId)

	// The update request carries the current metadata version for optimistic locking, so a version conflict re-reads the planning group and tries again.
	// Rate limiting and server errors are transient, and retrying is safe because every attempt rebuilds the request from the freshly read metadata.
	diagErr := util.WithRetries(ctx, 60*time.Second, func() *retry.RetryError {
		current, resp, err := proxy.getPlanningGroupById(ctx, businessUnitId, planningGroupId)
		if err != nil {
			readDiagErr := util.BuildWithRetriesApiDiagnosticError(ResourceType, fmt.Sprintf("Failed to read business unit %s planning group %s before update: %s", businessUnitId, planningGroupId, err), resp)
			if isTransientStatus(resp) {
				return retry.RetryableError(readDiagErr)
			}
			return retry.NonRetryableError(readDiagErr)
		}

		_, resp, err = proxy.updatePlanningGroup(ctx, businessUnitId, planningGroupId, buildUpdatePlanningGroupRequest(d, current.Metadata))
		if err == nil {
			return nil
		}
		builtDiagErr := util.BuildWithRetriesApiDiagnosticError(ResourceType, fmt.Sprintf("Failed to update business unit %s planning group %s: %s", businessUnitId, planningGroupId, err), resp)
		if util.IsStatus409(resp) || isTransientStatus(resp) {
			return retry.RetryableError(builtDiagErr)
		}
		return retry.NonRetryableError(builtDiagErr)
	})
	if diagErr != nil {
		return diagErr
	}

	log.Printf("Updated business unit %s planning group %s", businessUnitId, planningGroupId)
	return readPlanningGroup(ctx, d, meta)
}

func isTransientStatus(resp *platformclientv2.APIResponse) bool {
	return util.IsStatus429(resp) || (resp != nil && resp.StatusCode >= http.StatusInternalServerError)
}

// deletePlanningGroup is used by the workforcemanagement_businessunits_planninggroup resource to delete a business unit planning group from Genesys Cloud
func deletePlanningGroup(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	sdkConfig := meta.(*provider.ProviderMeta).ClientConfig
	proxy := getPlanningGroupProxy(sdkConfig)
	businessUnitId, planningGroupId := SplitPlanningGroupTerraformId(d.Id())

	log.Printf("Deleting business unit %s planning group %s", businessUnitId, planningGroupId)
	resp, err := proxy.deletePlanningGroup(ctx, businessUnitId, planningGroupId)
	if err != nil {
		if util.IsStatus404(resp) {
			log.Printf("Business unit %s planning group %s already deleted", businessUnitId, planningGroupId)
			return nil
		}
		return util.BuildAPIDiagnosticError(ResourceType, fmt.Sprintf("Failed to delete business unit %s planning group %s: %s", businessUnitId, planningGroupId, err), resp)
	}

	log.Printf("Deleted business unit %s planning group %s", businessUnitId, planningGroupId)
	return nil
}
