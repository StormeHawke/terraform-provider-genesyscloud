package workforcemanagement_businessunits_timeofflimit

import (
	"context"
	"fmt"
	"log"
	"strings"
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
	NOTE: This resource's Id is in the format <businessUnitId>/<timeOffLimitId> so we can persist the id of the parent business unit.
	The business_unit_id field can not be used for this because the time off limit api does not return the business unit.
*/

/*
The resource_genesyscloud_workforcemanagement_businessunits_timeofflimit.go contains all of the methods that perform the core logic for a resource.
*/

// getAllTimeOffLimits retrieves all of the time off limits via Terraform in the Genesys Cloud and is used for the exporter
func getAllTimeOffLimits(ctx context.Context, clientConfig *platformclientv2.Configuration) (resourceExporter.ResourceIDMetaMap, diag.Diagnostics) {
	proxy := getTimeOffLimitProxy(clientConfig)
	resources := make(resourceExporter.ResourceIDMetaMap)

	businessUnits, resp, err := proxy.getAllBusinessUnits(ctx)
	if err != nil {
		return nil, util.BuildAPIDiagnosticError(ResourceType, fmt.Sprintf("Failed to get business units: %v", err), resp)
	}
	managementUnits, resp, err := proxy.getAllManagementUnits(ctx)
	if err != nil {
		return nil, util.BuildAPIDiagnosticError(ResourceType, fmt.Sprintf("Failed to get management units: %v", err), resp)
	}
	managementUnitNames := make(map[string]string, len(*managementUnits))
	for _, managementUnit := range *managementUnits {
		if managementUnit.Id != nil && managementUnit.Name != nil {
			managementUnitNames[*managementUnit.Id] = *managementUnit.Name
		}
	}

	for _, businessUnit := range *businessUnits {
		if businessUnit.Id == nil || businessUnit.Name == nil {
			continue
		}
		limits, resp, err := proxy.getAllTimeOffLimits(ctx, *businessUnit.Id)
		if err != nil {
			return nil, util.BuildAPIDiagnosticError(ResourceType, fmt.Sprintf("Failed to get time off limits of business unit %s: %v", *businessUnit.Id, err), resp)
		}

		for _, limit := range *limits {
			if limit.Id == nil {
				continue
			}
			resources[*businessUnit.Id+"/"+*limit.Id] = &resourceExporter.ResourceMeta{BlockLabel: *businessUnit.Name + "_" + limitLabel(limit, managementUnitNames)}
		}
	}

	return resources, nil
}

// limitLabel names a limit after the management unit or staffing group it applies to
func limitLabel(limit platformclientv2.Butimeofflimitresponse, managementUnitNames map[string]string) string {
	if limit.StaffingGroup != nil && limit.StaffingGroup.Id != nil {
		return "staffinggroup_" + *limit.StaffingGroup.Id
	}
	if limit.ManagementUnit != nil && limit.ManagementUnit.Id != nil {
		if name, found := managementUnitNames[*limit.ManagementUnit.Id]; found {
			return name
		}
		return *limit.ManagementUnit.Id
	}
	return *limit.Id
}

// createTimeOffLimit is used by the workforcemanagement_businessunits_timeofflimit resource to create a Genesys Cloud time off limit
func createTimeOffLimit(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	sdkConfig := meta.(*provider.ProviderMeta).ClientConfig
	proxy := getTimeOffLimitProxy(sdkConfig)
	businessUnitId := d.Get("business_unit_id").(string)

	request := &platformclientv2.Bucreatetimeofflimitrequest{Granularity: platformclientv2.String(d.Get("granularity").(string))}
	if managementUnitId, ok := d.GetOk("management_unit_id"); ok {
		request.ManagementUnitId = platformclientv2.String(managementUnitId.(string))
	}
	if staffingGroupId, ok := d.GetOk("staffing_group_id"); ok {
		request.StaffingGroupId = platformclientv2.String(staffingGroupId.(string))
	}
	if startTime, ok := d.GetOk("full_day_time_off_start_time"); ok {
		request.FullDayTimeOffStartTime = platformclientv2.String(startTime.(string))
	}

	log.Printf("Creating business unit %s time off limit", businessUnitId)
	created, resp, err := proxy.createTimeOffLimit(ctx, businessUnitId, request)
	if err != nil {
		return util.BuildAPIDiagnosticError(ResourceType, fmt.Sprintf("Failed to create business unit %s time off limit: %s", businessUnitId, err), resp)
	}
	d.SetId(businessUnitId + "/" + *created.Id)

	if ranges := readValueRanges(d.Get("value_range")); len(ranges) > 0 {
		if diagErr := setValues(ctx, proxy, businessUnitId, *created.Id, ranges); diagErr != nil {
			return diagErr
		}
		if diagErr := verifyValuesSettled(ctx, proxy, businessUnitId, *created.Id, ranges); diagErr != nil {
			return diagErr
		}
	}

	log.Printf("Created business unit %s time off limit %s", businessUnitId, *created.Id)
	return readTimeOffLimit(ctx, d, meta)
}

// readTimeOffLimit is used by the workforcemanagement_businessunits_timeofflimit resource to read a time off limit from Genesys Cloud
func readTimeOffLimit(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	sdkConfig := meta.(*provider.ProviderMeta).ClientConfig
	proxy := getTimeOffLimitProxy(sdkConfig)
	cc := consistency_checker.NewConsistencyCheck(ctx, d, meta, ResourceTimeOffLimit(), constants.ConsistencyChecks(), ResourceType)
	businessUnitId, limitId := SplitTimeOffLimitTerraformId(d.Id())

	log.Printf("Reading business unit %s time off limit %s", businessUnitId, limitId)

	return util.WithRetriesForRead(ctx, d, func() *retry.RetryError {
		limit, resp, getErr := proxy.getTimeOffLimitById(ctx, businessUnitId, limitId)
		if getErr != nil {
			if util.IsStatus404(resp) {
				return retry.RetryableError(util.BuildWithRetriesApiDiagnosticError(ResourceType, fmt.Sprintf("Failed to read business unit %s time off limit %s: %s", businessUnitId, limitId, getErr), resp))
			}
			return retry.NonRetryableError(util.BuildWithRetriesApiDiagnosticError(ResourceType, fmt.Sprintf("Failed to read business unit %s time off limit %s: %s", businessUnitId, limitId, getErr), resp))
		}

		_ = d.Set("business_unit_id", businessUnitId)
		if limit.Granularity != nil {
			_ = d.Set("granularity", *limit.Granularity)
		}
		if limit.FullDayTimeOffStartTime != nil {
			_ = d.Set("full_day_time_off_start_time", *limit.FullDayTimeOffStartTime)
		}

		// A limit of a staffing group also reports the management unit of the staffing group, but it applies to the staffing group
		_ = d.Set("management_unit_id", nil)
		_ = d.Set("staffing_group_id", nil)
		if limit.StaffingGroup != nil && limit.StaffingGroup.Id != nil {
			_ = d.Set("staffing_group_id", *limit.StaffingGroup.Id)
		} else if limit.ManagementUnit != nil && limit.ManagementUnit.Id != nil {
			_ = d.Set("management_unit_id", *limit.ManagementUnit.Id)
		}

		// The values can only be read for date ranges, so only the ranges that are managed are read
		managed := readValueRanges(d.Get("value_range"))
		if len(managed) > 0 {
			values, unreadable, diagErr := queryManagedValues(ctx, proxy, businessUnitId, limitId, managed)
			if diagErr != nil {
				return retry.NonRetryableError(fmt.Errorf("%s", diagErr[0].Summary))
			}
			read := make([]valueRange, 0, len(managed))
			for _, configured := range managed {
				if unreadable[configured.start.Format(dateLayout)] {
					// Genesys Cloud cannot read values that are outside its window around today, so the range is kept as it is
					log.Printf("The range of business unit %s time off limit %s that starts on %s is outside the dates Genesys Cloud can read, so it is not checked", businessUnitId, limitId, configured.start.Format(dateLayout))
					read = append(read, configured)
					continue
				}
				read = append(read, readRangeValues(values, configured))
			}
			_ = d.Set("value_range", flattenValueRanges(read))
		}

		log.Printf("Read business unit %s time off limit %s", businessUnitId, limitId)
		return cc.CheckState(d)
	})
}

// updateTimeOffLimit is used by the workforcemanagement_businessunits_timeofflimit resource to update a time off limit in Genesys Cloud.
// Everything but the value ranges forces a new time off limit, so only value ranges can be updated.
func updateTimeOffLimit(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	sdkConfig := meta.(*provider.ProviderMeta).ClientConfig
	proxy := getTimeOffLimitProxy(sdkConfig)
	businessUnitId, limitId := SplitTimeOffLimitTerraformId(d.Id())

	if d.HasChange("value_range") {
		oldRanges, newRanges := d.GetChange("value_range")
		if changed := changedRanges(readValueRanges(oldRanges), readValueRanges(newRanges)); len(changed) > 0 {
			log.Printf("Updating the values of business unit %s time off limit %s", businessUnitId, limitId)
			if diagErr := setValues(ctx, proxy, businessUnitId, limitId, changed); diagErr != nil {
				return diagErr
			}
		}
	}

	return readTimeOffLimit(ctx, d, meta)
}

// queryManagedValues reads the values of the managed ranges. One query reads at most 30 ranges and 366 days in total, so more are read with several queries.
// Genesys Cloud only reads dates in a window around today, and rejects a whole query when one of its ranges is outside it. When that happens the ranges
// are read one by one, and the ranges that are outside the window are returned by start date as unreadable instead of failing the read.
func queryManagedValues(ctx context.Context, proxy *timeOffLimitProxy, businessUnitId string, limitId string, managed []valueRange) (*platformclientv2.Butimeofflimitvaluesforgranularityresponse, map[string]bool, diag.Diagnostics) {
	allValues := make([]platformclientv2.Butimeofflimitvalues, 0, len(managed))
	unreadable := make(map[string]bool)

	for _, chunk := range chunkRanges(managed, maxRangesPerQuery) {
		values, resp, err := proxy.queryLimitValues(ctx, businessUnitId, limitId, buildQueryRanges(chunk))
		if err == nil {
			if values != nil && values.LimitValues != nil {
				allValues = append(allValues, *values.LimitValues...)
			}
			continue
		}
		if !isOutsideWindow(resp) {
			return nil, nil, util.BuildAPIDiagnosticError(ResourceType, fmt.Sprintf("Failed to read the values of business unit %s time off limit %s: %s", businessUnitId, limitId, err), resp)
		}

		for _, single := range chunk {
			values, resp, err := proxy.queryLimitValues(ctx, businessUnitId, limitId, buildQueryRanges([]valueRange{single}))
			switch {
			case err == nil:
				if values != nil && values.LimitValues != nil {
					allValues = append(allValues, *values.LimitValues...)
				}
			case isOutsideWindow(resp):
				unreadable[single.start.Format(dateLayout)] = true
			default:
				return nil, nil, util.BuildAPIDiagnosticError(ResourceType, fmt.Sprintf("Failed to read the values of business unit %s time off limit %s: %s", businessUnitId, limitId, err), resp)
			}
		}
	}
	return &platformclientv2.Butimeofflimitvaluesforgranularityresponse{LimitValues: &allValues}, unreadable, nil
}

// isOutsideWindow reports whether Genesys Cloud rejected a request because its dates are too far from today
func isOutsideWindow(resp *platformclientv2.APIResponse) bool {
	return resp != nil && resp.StatusCode == 400 && strings.Contains(resp.ErrorMessage, "days back")
}

// setValues sets the values of the ranges. One request covers at most 366 days in total, so more are set with several requests.
// Each request carries the version of the time off limit, which also changes when time off requests are allocated against it,
// so a version conflict reads the time off limit again and sends the values again.
func setValues(ctx context.Context, proxy *timeOffLimitProxy, businessUnitId string, limitId string, ranges []valueRange) diag.Diagnostics {
	for _, chunk := range chunkRanges(ranges, len(ranges)) {
		chunk := chunk
		diagErr := util.WithRetries(ctx, 60*time.Second, func() *retry.RetryError {
			current, resp, err := proxy.getTimeOffLimitById(ctx, businessUnitId, limitId)
			if err != nil {
				return retry.NonRetryableError(util.BuildWithRetriesApiDiagnosticError(ResourceType, fmt.Sprintf("Failed to read business unit %s time off limit %s before setting its values: %s", businessUnitId, limitId, err), resp))
			}

			_, resp, err = proxy.setLimitValues(ctx, businessUnitId, limitId, buildSetValuesRequest(chunk, current.Metadata))
			if err == nil {
				return nil
			}
			builtDiagErr := util.BuildWithRetriesApiDiagnosticError(ResourceType, fmt.Sprintf("Failed to set the values of business unit %s time off limit %s: %s", businessUnitId, limitId, err), resp)
			if util.IsStatus409(resp) {
				return retry.RetryableError(builtDiagErr)
			}
			return retry.NonRetryableError(builtDiagErr)
		})
		if diagErr != nil {
			return diagErr
		}
	}
	return nil
}

var (
	// valueVerificationDelay is how long to wait before checking the values of a new time off limit, and valueVerificationAttempts how many times
	valueVerificationDelay    = 10 * time.Second
	valueVerificationAttempts = 3
)

// verifyValuesSettled checks that the values of a new time off limit are still there a moment after they were set, and sets them again if they are not.
// The id of a time off limit is the id of its management unit or staffing group, so a time off limit that replaces a deleted one has the same id.
// Deleting a time off limit queues a purge of its values, which Genesys Cloud does not check against a time off limit created in the meantime, so a
// purge that runs after the new values were set removes them.
func verifyValuesSettled(ctx context.Context, proxy *timeOffLimitProxy, businessUnitId string, limitId string, ranges []valueRange) diag.Diagnostics {
	for attempt := 0; attempt < valueVerificationAttempts; attempt++ {
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(valueVerificationDelay):
		}

		read, unreadable, diagErr := queryManagedValues(ctx, proxy, businessUnitId, limitId, ranges)
		if diagErr != nil {
			return diagErr
		}
		if len(unreadable) > 0 || valuesMatch(read, ranges) {
			return nil
		}

		log.Printf("The values of business unit %s time off limit %s are not there, setting them again", businessUnitId, limitId)
		if diagErr := setValues(ctx, proxy, businessUnitId, limitId, ranges); diagErr != nil {
			return diagErr
		}
	}
	return nil
}

// deleteTimeOffLimit is used by the workforcemanagement_businessunits_timeofflimit resource to delete a time off limit from Genesys Cloud
func deleteTimeOffLimit(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	sdkConfig := meta.(*provider.ProviderMeta).ClientConfig
	proxy := getTimeOffLimitProxy(sdkConfig)
	businessUnitId, limitId := SplitTimeOffLimitTerraformId(d.Id())

	log.Printf("Deleting business unit %s time off limit %s", businessUnitId, limitId)
	resp, err := proxy.deleteTimeOffLimit(ctx, businessUnitId, limitId)
	if err != nil {
		if util.IsStatus404(resp) {
			log.Printf("Business unit %s time off limit %s already deleted", businessUnitId, limitId)
			return nil
		}
		return util.BuildAPIDiagnosticError(ResourceType, fmt.Sprintf("Failed to delete business unit %s time off limit %s: %s", businessUnitId, limitId, err), resp)
	}

	log.Printf("Deleted business unit %s time off limit %s", businessUnitId, limitId)
	return nil
}
