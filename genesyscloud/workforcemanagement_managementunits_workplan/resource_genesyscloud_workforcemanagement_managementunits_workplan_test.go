package workforcemanagement_managementunits_workplan

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
	activityCode "github.com/mypurecloud/terraform-provider-genesyscloud/genesyscloud/workforcemanagement_businessunits_activitycode"
	managementUnits "github.com/mypurecloud/terraform-provider-genesyscloud/genesyscloud/workforcemanagement_managementunits"
)

/*
The resource_genesyscloud_workforcemanagement_managementunits_workplan_test.go contains all of the test cases for running the resource
tests for workforcemanagement_managementunits_workplan.
*/

func TestAccResourceWorkPlan(t *testing.T) {
	var (
		buResourceLabel   = "bu_1"
		buName            = "TestBU" + uuid.NewString()
		muResourceLabel   = "mu_1"
		muName            = "Test MU " + uuid.NewString()
		codeResourceLabel = "code_1"
		codeName          = "Test Break " + uuid.NewString()
		resourceLabel     = "work_plan_1"
		name1             = "Test Work Plan " + uuid.NewString()
		name2             = "Test Work Plan Updated " + uuid.NewString()
		fullResource      = ResourceType + "." + resourceLabel
	)

	baseConfig := businessUnits.GenerateWorkforcemanagementBusinessUnitResource(
		buResourceLabel,
		buName,
		businessUnits.GenerateWorkforcemanagementBusinessUnitSettings("Sunday", "Etc/UTC", "", ""),
	) +
		managementUnits.GenerateManagementUnitResource(muResourceLabel, muName, businessUnits.ResourceType+"."+buResourceLabel+".id") +
		activityCode.GenerateBusinessUnitActivityCodeResource(codeResourceLabel, businessUnits.ResourceType+"."+buResourceLabel+".id", codeName, "Break")
	managementUnitId := managementUnits.ResourceType + "." + muResourceLabel + ".id"
	codeId := activityCode.ResourceType + "." + codeResourceLabel + ".id"

	dayShift := fmt.Sprintf(`shifts {
		name                                   = "Day"
		days                                   = ["Monday", "Tuesday", "Wednesday", "Thursday", "Friday"]
		flexible_start_time                    = false
		exact_start_time_minutes_from_midnight = 480
		flexible_paid_time                     = false
		exact_paid_time_minutes                = 480
		activities {
			activity_code_id                        = %s
			length_minutes                          = 15
			start_time_is_relative_to_shift_start   = true
			flexible_start_time                     = false
			exact_start_time_minutes                = 240
			counts_as_paid_time                     = true
		}
	}`, codeId)
	nightShift := `shifts {
		name                                   = "Night"
		days                                   = ["Saturday", "Sunday"]
		flexible_start_time                    = false
		exact_start_time_minutes_from_midnight = 1200
		flexible_paid_time                     = false
		exact_paid_time_minutes                = 480
	}`

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { util.TestAccPreCheck(t) },
		ProviderFactories: provider.GetProviderFactories(providerResources, providerDataSources),
		Steps: []resource.TestStep{
			{
				// A disabled work plan may be invalid, so the test does not depend on a complete set of constraints
				Config: baseConfig + GenerateWorkPlanResource(
					resourceLabel,
					managementUnitId,
					name1,
					`enabled = false`,
					`maximum_days = 5`,
					`constrain_weekly_paid_time = true`,
					`flexible_weekly_paid_time = false`,
					`weekly_exact_paid_minutes = 2400`,
					`optional_days = ["Saturday"]`,
					dayShift,
				),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrPair(fullResource, "management_unit_id", managementUnits.ResourceType+"."+muResourceLabel, "id"),
					resource.TestCheckResourceAttr(fullResource, "name", name1),
					resource.TestCheckResourceAttr(fullResource, "enabled", util.FalseValue),
					resource.TestCheckResourceAttr(fullResource, "maximum_days", "5"),
					resource.TestCheckResourceAttr(fullResource, "weekly_exact_paid_minutes", "2400"),
					resource.TestCheckResourceAttr(fullResource, "shifts.#", "1"),
					resource.TestCheckResourceAttr(fullResource, "shifts.0.name", "Day"),
					resource.TestCheckResourceAttr(fullResource, "shifts.0.exact_start_time_minutes_from_midnight", "480"),
					resource.TestCheckResourceAttr(fullResource, "shifts.0.activities.#", "1"),
					resource.TestCheckResourceAttr(fullResource, "shifts.0.activities.0.length_minutes", "15"),
					resource.TestCheckResourceAttrPair(fullResource, "shifts.0.activities.0.activity_code_id", activityCode.ResourceType+"."+codeResourceLabel, "id"),
				),
			},
			{
				// Rename, change a constraint and add a second shift
				Config: baseConfig + GenerateWorkPlanResource(
					resourceLabel,
					managementUnitId,
					name2,
					`enabled = false`,
					`maximum_days = 6`,
					`constrain_weekly_paid_time = true`,
					`flexible_weekly_paid_time = false`,
					`weekly_exact_paid_minutes = 2400`,
					`optional_days = ["Saturday"]`,
					dayShift,
					nightShift,
				),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(fullResource, "name", name2),
					resource.TestCheckResourceAttr(fullResource, "maximum_days", "6"),
					resource.TestCheckResourceAttr(fullResource, "shifts.#", "2"),
					resource.TestCheckResourceAttr(fullResource, "shifts.1.name", "Night"),
				),
			},
			{
				// Remove the second shift again, which is deleted by name
				Config: baseConfig + GenerateWorkPlanResource(
					resourceLabel,
					managementUnitId,
					name2,
					`enabled = false`,
					`maximum_days = 6`,
					`constrain_weekly_paid_time = true`,
					`flexible_weekly_paid_time = false`,
					`weekly_exact_paid_minutes = 2400`,
					`optional_days = ["Saturday"]`,
					dayShift,
				),
				Check: resource.TestCheckResourceAttr(fullResource, "shifts.#", "1"),
			},
			{
				// Import/Read
				ResourceName:      fullResource,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
		CheckDestroy: testVerifyWorkPlanDestroyed,
	})
}

func testVerifyWorkPlanDestroyed(state *terraform.State) error {
	workforceManagementApi := platformclientv2.NewWorkforceManagementApi()
	for _, res := range state.RootModule().Resources {
		if res.Type != ResourceType {
			continue
		}

		managementUnitId, workPlanId := SplitWorkPlanTerraformId(res.Primary.ID)
		workPlan, resp, err := workforceManagementApi.GetWorkforcemanagementManagementunitWorkplan(managementUnitId, workPlanId, nil)
		if workPlan != nil {
			return fmt.Errorf("work plan (%s) still exists", res.Primary.ID)
		}
		if util.IsStatus404(resp) {
			continue
		}
		return fmt.Errorf("unexpected error: %s", err)
	}

	return nil
}
