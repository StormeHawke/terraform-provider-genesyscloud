package workforcemanagement_businessunits_timeoffplan

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/mypurecloud/terraform-provider-genesyscloud/genesyscloud/provider"
	"github.com/mypurecloud/terraform-provider-genesyscloud/genesyscloud/util"
)

/*
   The data_source_genesyscloud_workforcemanagement_businessunits_timeoffplan.go contains the data source implementation
   for the resource.
*/

// dataSourceTimeOffPlanRead retrieves by name the id in question
func dataSourceTimeOffPlanRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	sdkConfig := meta.(*provider.ProviderMeta).ClientConfig
	proxy := getTimeOffPlanProxy(sdkConfig)

	businessUnitId := d.Get("business_unit_id").(string)
	name := d.Get("name").(string)

	planId, resp, err := proxy.getTimeOffPlanIdByName(ctx, businessUnitId, name)
	if err != nil {
		return util.BuildAPIDiagnosticError(ResourceType, fmt.Sprintf("Error searching business unit %s time off plan %s | error: %s", businessUnitId, name, err), resp)
	}

	d.SetId(businessUnitId + "/" + planId)
	return nil
}
