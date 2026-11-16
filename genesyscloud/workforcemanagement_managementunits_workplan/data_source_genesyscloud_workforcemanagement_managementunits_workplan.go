package workforcemanagement_managementunits_workplan

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/mypurecloud/terraform-provider-genesyscloud/genesyscloud/provider"
	"github.com/mypurecloud/terraform-provider-genesyscloud/genesyscloud/util"
)

/*
   The data_source_genesyscloud_workforcemanagement_managementunits_workplan.go contains the data source implementation
   for the resource.
*/

// dataSourceWorkPlanRead retrieves by name the id in question
func dataSourceWorkPlanRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	sdkConfig := meta.(*provider.ProviderMeta).ClientConfig
	proxy := getWorkPlanProxy(sdkConfig)

	managementUnitId := d.Get("management_unit_id").(string)
	name := d.Get("name").(string)

	workPlanId, resp, err := proxy.getWorkPlanIdByName(ctx, managementUnitId, name)
	if err != nil {
		return util.BuildAPIDiagnosticError(ResourceType, fmt.Sprintf("Error searching management unit %s work plan %s | error: %s", managementUnitId, name, err), resp)
	}

	d.SetId(managementUnitId + "/" + workPlanId)
	return nil
}
