package workforcemanagement_managementunits_workplanrotation

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/mypurecloud/terraform-provider-genesyscloud/genesyscloud/provider"
	"github.com/mypurecloud/terraform-provider-genesyscloud/genesyscloud/util"
)

/*
   The data_source_genesyscloud_workforcemanagement_managementunits_workplanrotation.go contains the data source implementation
   for the resource.
*/

// dataSourceWorkPlanRotationRead retrieves by name the id in question
func dataSourceWorkPlanRotationRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	sdkConfig := meta.(*provider.ProviderMeta).ClientConfig
	proxy := getWorkPlanRotationProxy(sdkConfig)

	managementUnitId := d.Get("management_unit_id").(string)
	name := d.Get("name").(string)

	rotationId, resp, err := proxy.getWorkPlanRotationIdByName(ctx, managementUnitId, name)
	if err != nil {
		return util.BuildAPIDiagnosticError(ResourceType, fmt.Sprintf("Error searching management unit %s work plan rotation %s | error: %s", managementUnitId, name, err), resp)
	}

	d.SetId(managementUnitId + "/" + rotationId)
	return nil
}
