package workforcemanagement_managementunits

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/mypurecloud/terraform-provider-genesyscloud/genesyscloud/provider"
	"github.com/mypurecloud/terraform-provider-genesyscloud/genesyscloud/util"
)

/*
   The data_source_genesyscloud_workforcemanagement_managementunits.go contains the data source implementation
   for the resource.
*/

// dataSourceManagementUnitRead retrieves the id of the management unit with the given name in the given business unit
func dataSourceManagementUnitRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	sdkConfig := meta.(*provider.ProviderMeta).ClientConfig
	proxy := getManagementUnitProxy(sdkConfig)

	businessUnitId := d.Get("business_unit_id").(string)
	name := d.Get("name").(string)

	managementUnitId, resp, err := proxy.getManagementUnitIdByName(ctx, businessUnitId, name)
	if err != nil {
		return util.BuildAPIDiagnosticError(ResourceType, fmt.Sprintf("Error searching management unit %s | error: %s", name, err), resp)
	}

	d.SetId(managementUnitId)
	return nil
}
