package workforcemanagement_managementunits

import (
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/mypurecloud/terraform-provider-genesyscloud/genesyscloud/provider"
	"github.com/mypurecloud/terraform-provider-genesyscloud/genesyscloud/util"
)

func TestAccDataSourceManagementUnit(t *testing.T) {
	var (
		buResourceLabel = "bu_1"
		buName          = "TestBU" + uuid.NewString()
		resourceLabel   = "management_unit_1"
		dataSourceLabel = "management_unit_data"
		name            = "Test MU " + uuid.NewString()
	)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { util.TestAccPreCheck(t) },
		ProviderFactories: provider.GetProviderFactories(providerResources, providerDataSources),
		Steps: []resource.TestStep{
			{
				Config: generateTestBusinessUnit(buResourceLabel, buName) +
					GenerateManagementUnitResource(resourceLabel, name, businessUnitIdRef(buResourceLabel)) +
					generateManagementUnitDataSource(dataSourceLabel, name, ResourceType+"."+resourceLabel),
				Check: resource.TestCheckResourceAttrPair(
					"data."+ResourceType+"."+dataSourceLabel, "id",
					ResourceType+"."+resourceLabel, "id",
				),
			},
		},
	})
}

func generateManagementUnitDataSource(dataSourceLabel string, name string, dependsOn string) string {
	return fmt.Sprintf(`data "%s" "%s" {
	name       = "%s"
	depends_on = [%s]
}
`, ResourceType, dataSourceLabel, name, dependsOn)
}
