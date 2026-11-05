package workforcemanagement_managementunits

import (
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/mypurecloud/platform-client-sdk-go/v200/platformclientv2"
	"github.com/mypurecloud/terraform-provider-genesyscloud/genesyscloud/provider"
	"github.com/mypurecloud/terraform-provider-genesyscloud/genesyscloud/util"
	businessUnits "github.com/mypurecloud/terraform-provider-genesyscloud/genesyscloud/workforcemanagement_businessunits"
)

/*
The resource_genesyscloud_workforcemanagement_managementunits_test.go contains all of the test cases for running the resource
tests for workforcemanagement_managementunits.
*/

func TestAccResourceManagementUnit(t *testing.T) {
	var (
		bu1ResourceLabel = "bu_1"
		bu1Name          = "TestBU" + uuid.NewString()
		bu2ResourceLabel = "bu_2"
		bu2Name          = "TestBU" + uuid.NewString()
		resourceLabel    = "management_unit_1"
		name1            = "Test MU " + uuid.NewString()
		name2            = "Test MU Updated " + uuid.NewString()
		fullResource     = ResourceType + "." + resourceLabel
	)

	businessUnitConfigs := generateTestBusinessUnit(bu1ResourceLabel, bu1Name) + generateTestBusinessUnit(bu2ResourceLabel, bu2Name)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { util.TestAccPreCheck(t) },
		ProviderFactories: provider.GetProviderFactories(providerResources, providerDataSources),
		Steps: []resource.TestStep{
			{
				// Create with settings
				Config: businessUnitConfigs + GenerateManagementUnitResource(
					resourceLabel,
					name1,
					businessUnitIdRef(bu1ResourceLabel),
					`settings {
						adherence {
							adherence_target_percent       = 85
							severe_alert_threshold_minutes = 10
							track_on_queue_activity        = true
							ignored_activity_categories    = ["Break"]
						}
						time_off {
							submission_range_enforced         = true
							submission_range_type             = "Relative"
							submission_earliest_days_from_now = 1
							submission_latest_days_from_now   = 90
						}
						scheduling {
							max_occupancy_percent_for_deferred_work = 80
							default_shrinkage_percent               = 10
							start_day_of_weekend                    = "Saturday"
							schedule_visibility {
								enabled      = true
								future_weeks = 2
							}
						}
						shift_trading {
							enabled             = true
							allow_direct_trades = true
							min_hours_in_future = 24
							unequal_paid        = "AdminReview"
						}
					}`,
				),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(fullResource, "name", name1),
					resource.TestCheckResourceAttrPair(fullResource, "business_unit_id", businessUnits.ResourceType+"."+bu1ResourceLabel, "id"),
					resource.TestCheckResourceAttr(fullResource, "settings.0.adherence.0.adherence_target_percent", "85"),
					resource.TestCheckResourceAttr(fullResource, "settings.0.adherence.0.track_on_queue_activity", util.TrueValue),
					resource.TestCheckResourceAttr(fullResource, "settings.0.adherence.0.ignored_activity_categories.#", "1"),
					resource.TestCheckResourceAttr(fullResource, "settings.0.time_off.0.submission_latest_days_from_now", "90"),
					resource.TestCheckResourceAttr(fullResource, "settings.0.scheduling.0.max_occupancy_percent_for_deferred_work", "80"),
					resource.TestCheckResourceAttr(fullResource, "settings.0.scheduling.0.schedule_visibility.0.future_weeks", "2"),
					resource.TestCheckResourceAttr(fullResource, "settings.0.shift_trading.0.enabled", util.TrueValue),
					resource.TestCheckResourceAttr(fullResource, "settings.0.shift_trading.0.unequal_paid", "AdminReview"),
				),
			},
			{
				// Update the name and some settings
				Config: businessUnitConfigs + GenerateManagementUnitResource(
					resourceLabel,
					name2,
					businessUnitIdRef(bu1ResourceLabel),
					`settings {
						adherence {
							adherence_target_percent       = 90
							severe_alert_threshold_minutes = 10
							track_on_queue_activity        = true
							ignored_activity_categories    = ["Break", "Meal"]
						}
						shift_trading {
							enabled             = true
							allow_direct_trades = false
							min_hours_in_future = 48
						}
					}`,
				),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(fullResource, "name", name2),
					resource.TestCheckResourceAttr(fullResource, "settings.0.adherence.0.adherence_target_percent", "90"),
					resource.TestCheckResourceAttr(fullResource, "settings.0.adherence.0.ignored_activity_categories.#", "2"),
					resource.TestCheckResourceAttr(fullResource, "settings.0.shift_trading.0.allow_direct_trades", util.FalseValue),
					resource.TestCheckResourceAttr(fullResource, "settings.0.shift_trading.0.min_hours_in_future", "48"),
				),
			},
			{
				// Move to another business unit
				Config: businessUnitConfigs + GenerateManagementUnitResource(
					resourceLabel,
					name2,
					businessUnitIdRef(bu2ResourceLabel),
				),
				Check: resource.TestCheckResourceAttrPair(fullResource, "business_unit_id", businessUnits.ResourceType+"."+bu2ResourceLabel, "id"),
			},
			{
				// Import/Read
				ResourceName:      fullResource,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
		CheckDestroy: testVerifyManagementUnitDestroyed,
	})
}

func generateTestBusinessUnit(resourceLabel string, name string) string {
	return businessUnits.GenerateWorkforcemanagementBusinessUnitResource(
		resourceLabel,
		name,
		businessUnits.GenerateWorkforcemanagementBusinessUnitSettings("Sunday", "Etc/UTC", "", ""),
	)
}

func businessUnitIdRef(buResourceLabel string) string {
	return businessUnits.ResourceType + "." + buResourceLabel + ".id"
}

func testVerifyManagementUnitDestroyed(state *terraform.State) error {
	workforceManagementApi := platformclientv2.NewWorkforceManagementApi()
	for _, res := range state.RootModule().Resources {
		if res.Type != ResourceType {
			continue
		}

		managementUnit, resp, err := workforceManagementApi.GetWorkforcemanagementManagementunit(res.Primary.ID, nil)
		if managementUnit != nil {
			return fmt.Errorf("management unit (%s) still exists", res.Primary.ID)
		}
		if util.IsStatus404(resp) {
			continue
		}
		return fmt.Errorf("unexpected error: %s", err)
	}

	return nil
}
