package workforcemanagement_managementunits_workplanrotation

import (
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/mypurecloud/terraform-provider-genesyscloud/genesyscloud/provider"
	"github.com/mypurecloud/terraform-provider-genesyscloud/genesyscloud/util"
	managementUnits "github.com/mypurecloud/terraform-provider-genesyscloud/genesyscloud/workforcemanagement_managementunits"
	workPlan "github.com/mypurecloud/terraform-provider-genesyscloud/genesyscloud/workforcemanagement_managementunits_workplan"
)

func TestAccDataSourceWorkPlanRotation(t *testing.T) {
	var (
		resourceLabel   = "rotation_1"
		dataSourceLabel = "rotation_data"
		name            = "Test Rotation " + uuid.NewString()
	)

	managementUnitId := managementUnits.ResourceType + ".mu_1.id"
	config := generateRotationDependencies("TestBU"+uuid.NewString(), "Test MU "+uuid.NewString()) +
		GenerateWorkPlanRotationResource(resourceLabel, managementUnitId, name, "2026-01-05", fmt.Sprintf("[%s]", workPlan.ResourceType+".plan_a.id")) +
		generateWorkPlanRotationDataSource(dataSourceLabel, managementUnitId, name, ResourceType+"."+resourceLabel)

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

func generateWorkPlanRotationDataSource(dataSourceLabel string, managementUnitId string, name string, dependsOn string) string {
	return fmt.Sprintf(`data "%s" "%s" {
	management_unit_id = %s
	name               = "%s"
	depends_on         = [%s]
}
`, ResourceType, dataSourceLabel, managementUnitId, name, dependsOn)
}
