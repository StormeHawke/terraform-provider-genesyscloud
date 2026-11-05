package workforcemanagement_businessunits_servicegoaltemplate

import (
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/mypurecloud/terraform-provider-genesyscloud/genesyscloud/provider"
	"github.com/mypurecloud/terraform-provider-genesyscloud/genesyscloud/util"
)

func TestAccDataSourceServiceGoalTemplate(t *testing.T) {
	var (
		buResourceLabel = "bu_1"
		buName          = "TestBU" + uuid.NewString()
		resourceLabel   = "service_goal_template_1"
		dataSourceLabel = "service_goal_template_data"
		name            = "Test Service Goal Template " + uuid.NewString()
	)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { util.TestAccPreCheck(t) },
		ProviderFactories: provider.GetProviderFactories(providerResources, providerDataSources),
		Steps: []resource.TestStep{
			{
				Config: generateTestBusinessUnit(buResourceLabel, buName) +
					GenerateServiceGoalTemplateResource(
						resourceLabel,
						businessUnitIdRef(buResourceLabel),
						name,
						`abandon_rate {
							include = true
							percent = 5
						}`,
					) +
					generateServiceGoalTemplateDataSource(dataSourceLabel, businessUnitIdRef(buResourceLabel), name, ResourceType+"."+resourceLabel),
				Check: resource.TestCheckResourceAttrPair(
					"data."+ResourceType+"."+dataSourceLabel, "id",
					ResourceType+"."+resourceLabel, "id",
				),
			},
		},
	})
}

func generateServiceGoalTemplateDataSource(dataSourceLabel string, businessUnitId string, name string, dependsOn string) string {
	return fmt.Sprintf(`data "%s" "%s" {
	business_unit_id = %s
	name             = "%s"
	depends_on       = [%s]
}
`, ResourceType, dataSourceLabel, businessUnitId, name, dependsOn)
}
