package workforcemanagement_businessunits_timeofflimit

import (
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/mypurecloud/platform-client-sdk-go/v200/platformclientv2"
	"github.com/mypurecloud/terraform-provider-genesyscloud/genesyscloud/provider"
	"github.com/mypurecloud/terraform-provider-genesyscloud/genesyscloud/util"
	businessUnits "github.com/mypurecloud/terraform-provider-genesyscloud/genesyscloud/workforcemanagement_businessunits"
	managementUnits "github.com/mypurecloud/terraform-provider-genesyscloud/genesyscloud/workforcemanagement_managementunits"
)

/*
The resource_genesyscloud_workforcemanagement_businessunits_timeofflimit_test.go contains all of the test cases for running the resource
tests for workforcemanagement_businessunits_timeofflimit.
*/

func generateLimitDependencies(buName string, muName string) string {
	return businessUnits.GenerateWorkforcemanagementBusinessUnitResource(
		"bu_1",
		buName,
		businessUnits.GenerateWorkforcemanagementBusinessUnitSettings("Sunday", "Etc/UTC", "", ""),
	) + managementUnits.GenerateManagementUnitResource("mu_1", muName, businessUnits.ResourceType+".bu_1.id")
}

func TestAccResourceTimeOffLimit(t *testing.T) {
	var (
		resourceLabel = "limit_1"
		fullResource  = ResourceType + "." + resourceLabel
	)

	// Genesys Cloud only accepts limit dates in a window around today, so the dates are relative to it
	firstDate := time.Now().AddDate(0, 0, 30).Format(dateLayout)
	secondDate := time.Now().AddDate(0, 0, 90).Format(dateLayout)
	dependencies := generateLimitDependencies("TestBU"+uuid.NewString(), "Test MU "+uuid.NewString())
	businessUnitId := businessUnits.ResourceType + ".bu_1.id"
	managementUnitId := managementUnits.ResourceType + ".mu_1.id"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { util.TestAccPreCheck(t) },
		ProviderFactories: provider.GetProviderFactories(providerResources, providerDataSources),
		Steps: []resource.TestStep{
			{
				// A limit without managed values
				Config: dependencies + GenerateTimeOffLimitResource(resourceLabel, businessUnitId, managementUnitId, granularityDaily),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrPair(fullResource, "business_unit_id", businessUnits.ResourceType+".bu_1", "id"),
					resource.TestCheckResourceAttrPair(fullResource, "management_unit_id", managementUnits.ResourceType+".mu_1", "id"),
					resource.TestCheckResourceAttr(fullResource, "granularity", granularityDaily),
					resource.TestCheckResourceAttr(fullResource, "value_range.#", "0"),
				),
			},
			{
				// Manage the values of two date ranges
				Config: dependencies + GenerateTimeOffLimitResource(resourceLabel, businessUnitId, managementUnitId, granularityDaily,
					fmt.Sprintf(`value_range {
						start_date            = "%[1]s"
						limit_minutes_per_day = [480, 480, 0, 960]
					}`, firstDate, secondDate),
					fmt.Sprintf(`value_range {
						start_date            = "%[2]s"
						limit_minutes_per_day = [240]
					}`, firstDate, secondDate)),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(fullResource, "value_range.#", "2"),
					resource.TestCheckResourceAttr(fullResource, "value_range.0.start_date", firstDate),
					resource.TestCheckResourceAttr(fullResource, "value_range.0.limit_minutes_per_day.3", "960"),
					resource.TestCheckResourceAttr(fullResource, "value_range.1.limit_minutes_per_day.0", "240"),
				),
			},
			{
				// Change one of the ranges, which only sends that range
				Config: dependencies + GenerateTimeOffLimitResource(resourceLabel, businessUnitId, managementUnitId, granularityDaily,
					fmt.Sprintf(`value_range {
						start_date            = "%[1]s"
						limit_minutes_per_day = [480, 480, 480, 960]
					}`, firstDate, secondDate),
					fmt.Sprintf(`value_range {
						start_date            = "%[2]s"
						limit_minutes_per_day = [240]
					}`, firstDate, secondDate)),
				Check: resource.TestCheckResourceAttr(fullResource, "value_range.0.limit_minutes_per_day.2", "480"),
			},
			{
				// Import/Read: the values of the ranges cannot be known on import
				ResourceName:            fullResource,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"value_range"},
			},
		},
		CheckDestroy: testVerifyTimeOffLimitDestroyed,
	})
}

func testVerifyTimeOffLimitDestroyed(state *terraform.State) error {
	workforceManagementApi := platformclientv2.NewWorkforceManagementApi()
	for _, res := range state.RootModule().Resources {
		if res.Type != ResourceType {
			continue
		}

		businessUnitId, limitId := SplitTimeOffLimitTerraformId(res.Primary.ID)
		limit, resp, err := workforceManagementApi.GetWorkforcemanagementBusinessunitTimeofflimit(businessUnitId, limitId)
		if limit != nil {
			return fmt.Errorf("time off limit (%s) still exists", res.Primary.ID)
		}
		if util.IsStatus404(resp) {
			continue
		}
		return fmt.Errorf("unexpected error: %s", err)
	}

	return nil
}
