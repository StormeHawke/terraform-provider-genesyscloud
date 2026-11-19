package workforcemanagement_businessunits_timeoffplan

import (
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/mypurecloud/terraform-provider-genesyscloud/genesyscloud/provider"
	"github.com/mypurecloud/terraform-provider-genesyscloud/genesyscloud/util"
	businessUnits "github.com/mypurecloud/terraform-provider-genesyscloud/genesyscloud/workforcemanagement_businessunits"
)

func TestAccDataSourceTimeOffPlan(t *testing.T) {
	var (
		resourceLabel   = "plan_1"
		dataSourceLabel = "plan_data"
		name            = "Test Time Off Plan " + uuid.NewString()
	)

	businessUnitId := businessUnits.ResourceType + ".bu_1.id"
	config := generatePlanDependencies("TestBU"+uuid.NewString(), "Test MU "+uuid.NewString(), "Test Time Off "+uuid.NewString()) +
		GenerateTimeOffPlanResource(resourceLabel, businessUnitId, name) +
		generateTimeOffPlanDataSource(dataSourceLabel, businessUnitId, name, ResourceType+"."+resourceLabel)

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

func generateTimeOffPlanDataSource(dataSourceLabel string, businessUnitId string, name string, dependsOn string) string {
	return fmt.Sprintf(`data "%s" "%s" {
	business_unit_id = %s
	name             = "%s"
	depends_on       = [%s]
}
`, ResourceType, dataSourceLabel, businessUnitId, name, dependsOn)
}
