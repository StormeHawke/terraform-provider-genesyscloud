package workforcemanagement_businessunits_planninggroup

import (
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/mypurecloud/terraform-provider-genesyscloud/genesyscloud/provider"
	"github.com/mypurecloud/terraform-provider-genesyscloud/genesyscloud/util"
)

func TestAccDataSourcePlanningGroup(t *testing.T) {
	var (
		buResourceLabel = "bu_1"
		buName          = "TestBU" + uuid.NewString()
		resourceLabel   = "planning_group_1"
		dataSourceLabel = "planning_group_data"
		name            = "Test Planning Group " + uuid.NewString()
	)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { util.TestAccPreCheck(t) },
		ProviderFactories: provider.GetProviderFactories(providerResources, providerDataSources),
		Steps: []resource.TestStep{
			{
				Config: generateTestBusinessUnit(buResourceLabel, buName) +
					GeneratePlanningGroupResource(resourceLabel, businessUnitIdRef(buResourceLabel), name) +
					generatePlanningGroupDataSource(dataSourceLabel, businessUnitIdRef(buResourceLabel), name, ResourceType+"."+resourceLabel),
				Check: resource.TestCheckResourceAttrPair(
					"data."+ResourceType+"."+dataSourceLabel, "id",
					ResourceType+"."+resourceLabel, "id",
				),
			},
		},
	})
}

func generatePlanningGroupDataSource(dataSourceLabel string, businessUnitId string, name string, dependsOn string) string {
	return fmt.Sprintf(`data "%s" "%s" {
	business_unit_id = %s
	name             = "%s"
	depends_on       = [%s]
}
`, ResourceType, dataSourceLabel, businessUnitId, name, dependsOn)
}
