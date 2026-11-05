package workforcemanagement_businessunits_planninggroup

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/mypurecloud/terraform-provider-genesyscloud/genesyscloud/provider"
	"github.com/mypurecloud/terraform-provider-genesyscloud/genesyscloud/util"
)

/*
   The data_source_genesyscloud_workforcemanagement_businessunits_planninggroup.go contains the data source implementation
   for the resource.
*/

// dataSourcePlanningGroupRead retrieves by name the id in question
func dataSourcePlanningGroupRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	sdkConfig := meta.(*provider.ProviderMeta).ClientConfig
	proxy := getPlanningGroupProxy(sdkConfig)

	businessUnitId := d.Get("business_unit_id").(string)
	name := d.Get("name").(string)

	return util.WithRetries(ctx, 15*time.Second, func() *retry.RetryError {
		planningGroupId, resp, retryable, err := proxy.getPlanningGroupIdByName(ctx, businessUnitId, name)

		if err != nil && !retryable {
			return retry.NonRetryableError(util.BuildWithRetriesApiDiagnosticError(ResourceType, fmt.Sprintf("Error searching business unit %s planning group %s | error: %s", businessUnitId, name, err), resp))
		}

		if retryable {
			return retry.RetryableError(util.BuildWithRetriesApiDiagnosticError(ResourceType, fmt.Sprintf("No business unit %s planning group found with name %s", businessUnitId, name), resp))
		}

		d.SetId(businessUnitId + "/" + planningGroupId)
		return nil
	})
}
