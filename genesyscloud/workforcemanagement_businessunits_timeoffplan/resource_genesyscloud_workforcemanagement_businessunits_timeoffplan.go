package workforcemanagement_businessunits_timeoffplan

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
	NOTE: This resource's Id is in the format <businessUnitId>/<timeOffPlanId> so we can persist the id of the parent business unit.
	The business_unit_id field can not be used for this because the time off plan api does not return the business unit.
*/

/*
The resource_genesyscloud_workforcemanagement_businessunits_timeoffplan.go contains all of the methods that perform the core logic for a resource.
*/

// getAllTimeOffPlans retrieves all of the time off plans via Terraform in the Genesys Cloud and is used for the exporter
func getAllTimeOffPlans(ctx context.Context, clientConfig *platformclientv2.Configuration) (resourceExporter.ResourceIDMetaMap, diag.Diagnostics) {
	proxy := getTimeOffPlanProxy(clientConfig)
	resources := make(resourceExporter.ResourceIDMetaMap)

	businessUnits, resp, err := proxy.getAllBusinessUnits(ctx)
	if err != nil {
		return nil, util.BuildAPIDiagnosticError(ResourceType, fmt.Sprintf("Failed to get business units: %v", err), resp)
	}

	for _, businessUnit := range *businessUnits {
		plans, resp, err := proxy.getAllTimeOffPlans(ctx, *businessUnit.Id)
		if err != nil {
			return nil, util.BuildAPIDiagnosticError(ResourceType, fmt.Sprintf("Failed to get time off plans of business unit %s: %v", *businessUnit.Id, err), resp)
		}

		for _, plan := range *plans {
			resources[*businessUnit.Id+"/"+*plan.Id] = &resourceExporter.ResourceMeta{BlockLabel: *businessUnit.Name + "_" + *plan.Name}
		}
	}

	return resources, nil
}

// createTimeOffPlan is used by the workforcemanagement_businessunits_timeoffplan resource to create a Genesys Cloud time off plan
func createTimeOffPlan(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	sdkConfig := meta.(*provider.ProviderMeta).ClientConfig
	proxy := getTimeOffPlanProxy(sdkConfig)
	businessUnitId := d.Get("business_unit_id").(string)
	plan := buildCreateRequest(d.GetRawConfig(), d.Get)

	log.Printf("Creating business unit %s time off plan %s", businessUnitId, *plan.Name)
	created, resp, err := proxy.createTimeOffPlan(ctx, businessUnitId, plan)
	if err != nil {
		return util.BuildAPIDiagnosticError(ResourceType, fmt.Sprintf("Failed to create business unit %s time off plan %s: %s", businessUnitId, *plan.Name, err), resp)
	}

	d.SetId(businessUnitId + "/" + *created.Id)
	log.Printf("Created business unit %s time off plan %s %s", businessUnitId, *created.Id, *plan.Name)
	return readTimeOffPlan(ctx, d, meta)
}

// readTimeOffPlan is used by the workforcemanagement_businessunits_timeoffplan resource to read a time off plan from Genesys Cloud
func readTimeOffPlan(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	sdkConfig := meta.(*provider.ProviderMeta).ClientConfig
	proxy := getTimeOffPlanProxy(sdkConfig)
	cc := consistency_checker.NewConsistencyCheck(ctx, d, meta, ResourceTimeOffPlan(), constants.ConsistencyChecks(), ResourceType)
	businessUnitId, planId := SplitTimeOffPlanTerraformId(d.Id())

	log.Printf("Reading business unit %s time off plan %s", businessUnitId, planId)

	return util.WithRetriesForRead(ctx, d, func() *retry.RetryError {
		plan, resp, getErr := proxy.getTimeOffPlanById(ctx, businessUnitId, planId)
		if getErr != nil {
			// A 404 is not retried: the time off plan is readable as soon as it is created, so a 404 means it is gone and WithRetriesForRead removes it from state
			return retry.NonRetryableError(util.BuildWithRetriesApiDiagnosticError(ResourceType, fmt.Sprintf("Failed to read business unit %s time off plan %s: %s", businessUnitId, planId, getErr), resp))
		}

		_ = d.Set("business_unit_id", businessUnitId)
		_ = d.Set("name", *plan.Name)
		_ = d.Set("enabled", *plan.Enabled)
		_ = d.Set("auto_approval_rule", *plan.AutoApprovalRule)
		_ = d.Set("days_before_start_to_expire_from_waitlist", *plan.DaysBeforeStartToExpireFromWaitlist)
		setIfNotNil(d, "auto_publish_approved_time_off_requests", plan.AutoPublishApprovedTimeOffRequests)
		setIfNotNil(d, "count_against_time_off_limits", plan.CountAgainstTimeOffLimits)

		_ = d.Set("activity_code_ids", flattenStringSet(currentIds(derefStrings(plan.ActivityCodeIds))))
		var restricted []string
		if plan.RestrictedActivityCodes != nil {
			restricted = derefStrings(plan.RestrictedActivityCodes.Ids)
		}
		_ = d.Set("restricted_activity_code_ids", flattenStringSet(currentIds(restricted)))
		_ = d.Set("override_dates", flattenOverrideDates(plan.OverrideDates))
		_ = d.Set("hris_time_off_type", flattenHrisType(plan.HrisTimeOffType))
		_ = d.Set("business_unit_association", flattenBusinessUnitAssociation(plan.BusinessUnitAssociation))
		_ = d.Set("management_unit_association", flattenManagementUnitAssociation(plan.ManagementUnitAssociation))
		_ = d.Set("time_off_limit_ids", flattenLimitIds(plan.TimeOffLimits))

		log.Printf("Read business unit %s time off plan %s", businessUnitId, planId)
		return cc.CheckState(d)
	})
}

func isTransientStatus(resp *platformclientv2.APIResponse) bool {
	return util.IsStatus429(resp) || (resp != nil && resp.StatusCode >= http.StatusInternalServerError)
}

func setIfNotNil[T any](d *schema.ResourceData, key string, value *T) {
	if value != nil {
		_ = d.Set(key, *value)
	}
}

// updateTimeOffPlan is used by the workforcemanagement_businessunits_timeoffplan resource to update a time off plan in Genesys Cloud
func updateTimeOffPlan(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	sdkConfig := meta.(*provider.ProviderMeta).ClientConfig
	proxy := getTimeOffPlanProxy(sdkConfig)
	businessUnitId, planId := SplitTimeOffPlanTerraformId(d.Id())

	log.Printf("Updating business unit %s time off plan %s", businessUnitId, planId)

	// The patch only contains what differs from the current time off plan and carries its version for optimistic locking,
	// so a version conflict reads the time off plan again and rebuilds the patch.
	// Rate limiting and server errors are transient, and retrying is safe because every attempt rebuilds the patch from the freshly read time off plan.
	diagErr := util.WithRetries(ctx, 60*time.Second, func() *retry.RetryError {
		current, resp, err := proxy.getTimeOffPlanById(ctx, businessUnitId, planId)
		if err != nil {
			readDiagErr := util.BuildWithRetriesApiDiagnosticError(ResourceType, fmt.Sprintf("Failed to read business unit %s time off plan %s before update: %s", businessUnitId, planId, err), resp)
			if isTransientStatus(resp) {
				return retry.RetryableError(readDiagErr)
			}
			return retry.NonRetryableError(readDiagErr)
		}

		_, resp, err = proxy.updateTimeOffPlan(ctx, businessUnitId, planId, buildUpdateRequest(d.GetRawConfig(), d.Get, current))
		if err == nil {
			return nil
		}
		builtDiagErr := util.BuildWithRetriesApiDiagnosticError(ResourceType, fmt.Sprintf("Failed to update business unit %s time off plan %s: %s", businessUnitId, planId, err), resp)
		if util.IsStatus409(resp) || isTransientStatus(resp) {
			return retry.RetryableError(builtDiagErr)
		}
		return retry.NonRetryableError(builtDiagErr)
	})
	if diagErr != nil {
		return diagErr
	}

	log.Printf("Updated business unit %s time off plan %s", businessUnitId, planId)
	return readTimeOffPlan(ctx, d, meta)
}

// deleteTimeOffPlan is used by the workforcemanagement_businessunits_timeoffplan resource to delete a time off plan from Genesys Cloud
func deleteTimeOffPlan(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	sdkConfig := meta.(*provider.ProviderMeta).ClientConfig
	proxy := getTimeOffPlanProxy(sdkConfig)
	businessUnitId, planId := SplitTimeOffPlanTerraformId(d.Id())

	log.Printf("Deleting business unit %s time off plan %s", businessUnitId, planId)
	resp, err := proxy.deleteTimeOffPlan(ctx, businessUnitId, planId)
	if err != nil {
		if util.IsStatus404(resp) {
			log.Printf("Business unit %s time off plan %s already deleted", businessUnitId, planId)
			return nil
		}
		return util.BuildAPIDiagnosticError(ResourceType, fmt.Sprintf("Failed to delete business unit %s time off plan %s: %s", businessUnitId, planId, err), resp)
	}

	log.Printf("Deleted business unit %s time off plan %s", businessUnitId, planId)
	return nil
}
