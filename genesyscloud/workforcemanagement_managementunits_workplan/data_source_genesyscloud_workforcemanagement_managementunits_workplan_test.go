package workforcemanagement_managementunits_workplan

import (
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/mypurecloud/terraform-provider-genesyscloud/genesyscloud/provider"
	"github.com/mypurecloud/terraform-provider-genesyscloud/genesyscloud/util"
	businessUnits "github.com/mypurecloud/terraform-provider-genesyscloud/genesyscloud/workforcemanagement_businessunits"
	managementUnits "github.com/mypurecloud/terraform-provider-genesyscloud/genesyscloud/workforcemanagement_managementunits"
)

func TestAccDataSourceWorkPlan(t *testing.T) {
	var (
		buResourceLabel = "bu_1"
		buName          = "TestBU" + uuid.NewString()
		muResourceLabel = "mu_1"
		muName          = "Test MU " + uuid.NewString()
		resourceLabel   = "work_plan_1"
		dataSourceLabel = "work_plan_data"
		name            = "Test Work Plan " + uuid.NewString()
	)

	managementUnitId := managementUnits.ResourceType + "." + muResourceLabel + ".id"
	config := businessUnits.GenerateWorkforcemanagementBusinessUnitResource(
		buResourceLabel,
		buName,
		businessUnits.GenerateWorkforcemanagementBusinessUnitSettings("Sunday", "Etc/UTC", "", ""),
	) +
		managementUnits.GenerateManagementUnitResource(muResourceLabel, muName, businessUnits.ResourceType+"."+buResourceLabel+".id") +
		GenerateWorkPlanResource(resourceLabel, managementUnitId, name, `enabled = false`) +
		generateWorkPlanDataSource(dataSourceLabel, managementUnitId, name, ResourceType+"."+resourceLabel)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { util.TestAccPreCheck(t) },
		ProviderFactories: provider.GetProviderFactories(providerResources, providerDataSources),
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.TestCheckResourceAttrPair(
					"data."+ResourceType+"."+dataSourceLabel, "id",
					ResourceType+"."+resourceLabel, "id",
				),
			},
		},
	})
}

func generateWorkPlanDataSource(dataSourceLabel string, managementUnitId string, name string, dependsOn string) string {
	return fmt.Sprintf(`data "%s" "%s" {
	management_unit_id = %s
	name               = "%s"
	depends_on         = [%s]
}
`, ResourceType, dataSourceLabel, managementUnitId, name, dependsOn)
}
