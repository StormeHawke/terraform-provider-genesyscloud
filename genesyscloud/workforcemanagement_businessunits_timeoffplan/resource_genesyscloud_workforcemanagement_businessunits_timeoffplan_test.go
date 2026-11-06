package workforcemanagement_businessunits_timeoffplan

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
	activityCode "github.com/mypurecloud/terraform-provider-genesyscloud/genesyscloud/workforcemanagement_businessunits_activitycode"
	timeOffLimit "github.com/mypurecloud/terraform-provider-genesyscloud/genesyscloud/workforcemanagement_businessunits_timeofflimit"
	managementUnits "github.com/mypurecloud/terraform-provider-genesyscloud/genesyscloud/workforcemanagement_managementunits"
)

/*
The resource_genesyscloud_workforcemanagement_businessunits_timeoffplan_test.go contains all of the test cases for running the resource
tests for workforcemanagement_businessunits_timeoffplan.
*/

// generatePlanDependencies returns a business unit, a management unit, a time off activity code, and an activity code that is not a time off
// one, because a time off plan uses the first and its restricted activity codes are the second.
func generatePlanDependencies(buName string, muName string, codeName string) string {
	return businessUnits.GenerateWorkforcemanagementBusinessUnitResource(
		"bu_1",
		buName,
		businessUnits.GenerateWorkforcemanagementBusinessUnitSettings("Sunday", "Etc/UTC", "", ""),
	) +
		managementUnits.GenerateManagementUnitResource("mu_1", muName, businessUnits.ResourceType+".bu_1.id") +
		activityCode.GenerateBusinessUnitActivityCodeResource("code_1", businessUnits.ResourceType+".bu_1.id", codeName, "TimeOff") +
		activityCode.GenerateBusinessUnitActivityCodeResource("code_2", businessUnits.ResourceType+".bu_1.id", codeName+" restricted", "Meeting")
}

func TestAccResourceTimeOffPlan(t *testing.T) {
	var (
		resourceLabel = "plan_1"
		fullResource  = ResourceType + "." + resourceLabel
		name1         = "Test Time Off Plan " + uuid.NewString()
		name2         = "Test Time Off Plan Updated " + uuid.NewString()
	)

	// Genesys Cloud only accepts override dates in a window around today, so the dates are relative to it
	firstDate := time.Now().AddDate(0, 0, 30).Format(dateLayout)
	secondDate := time.Now().AddDate(0, 0, 60).Format(dateLayout)
	dependencies := generatePlanDependencies("TestBU"+uuid.NewString(), "Test MU "+uuid.NewString(), "Test Time Off "+uuid.NewString())
	businessUnitId := businessUnits.ResourceType + ".bu_1.id"
	managementUnitId := managementUnits.ResourceType + ".mu_1.id"
	codeId := activityCode.ResourceType + ".code_1.id"
	restrictedCodeId := activityCode.ResourceType + ".code_2.id"

	association := fmt.Sprintf(`business_unit_association {
		management_unit_ids = [%s]
	}`, managementUnitId)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { util.TestAccPreCheck(t) },
		ProviderFactories: provider.GetProviderFactories(providerResources, providerDataSources),
		Steps: []resource.TestStep{
			{
				Config: dependencies + GenerateTimeOffPlanResource(resourceLabel, businessUnitId, name1,
					fmt.Sprintf(`activity_code_ids = [%s]`, codeId),
					`auto_approval_rule = "Always"`,
					fmt.Sprintf(`override_dates {
						date = "%s"
						type = "Blocked"
					}`, firstDate),
					association,
				),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrPair(fullResource, "business_unit_id", businessUnits.ResourceType+".bu_1", "id"),
					resource.TestCheckResourceAttr(fullResource, "name", name1),
					resource.TestCheckResourceAttr(fullResource, "auto_approval_rule", "Always"),
					resource.TestCheckResourceAttr(fullResource, "count_against_time_off_limits", util.FalseValue),
					resource.TestCheckResourceAttr(fullResource, "activity_code_ids.#", "1"),
					resource.TestCheckResourceAttr(fullResource, "override_dates.#", "1"),
					resource.TestCheckResourceAttr(fullResource, "business_unit_association.0.management_unit_ids.#", "1"),
				),
			},
			{
				// Rename, check activity codes (which needs restricted activity codes), change the dates, and disable it
				Config: dependencies + GenerateTimeOffPlanResource(resourceLabel, businessUnitId, name2,
					fmt.Sprintf(`activity_code_ids = [%s]`, codeId),
					`auto_approval_rule = "CheckActivityCodes"`,
					fmt.Sprintf(`restricted_activity_code_ids = [%s]`, restrictedCodeId),
					`enabled = false`,
					`days_before_start_to_expire_from_waitlist = 3`,
					fmt.Sprintf(`override_dates {
						date = "%s"
						type = "ManualReview"
					}`, firstDate),
					fmt.Sprintf(`override_dates {
						date = "%s"
						type = "Blocked"
					}`, secondDate),
					association,
				),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(fullResource, "name", name2),
					resource.TestCheckResourceAttr(fullResource, "auto_approval_rule", "CheckActivityCodes"),
					resource.TestCheckResourceAttr(fullResource, "restricted_activity_code_ids.#", "1"),
					resource.TestCheckResourceAttr(fullResource, "enabled", util.FalseValue),
					resource.TestCheckResourceAttr(fullResource, "days_before_start_to_expire_from_waitlist", "3"),
					resource.TestCheckResourceAttr(fullResource, "override_dates.#", "2"),
				),
			},
			{
				// Go back to a rule that does not check activity codes, which clears the restricted activity codes, and remove the dates and codes
				Config: dependencies + GenerateTimeOffPlanResource(resourceLabel, businessUnitId, name2,
					`auto_approval_rule = "Never"`,
					`enabled = false`,
					`days_before_start_to_expire_from_waitlist = 3`,
					association,
				),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(fullResource, "restricted_activity_code_ids.#", "0"),
					resource.TestCheckResourceAttr(fullResource, "override_dates.#", "0"),
					resource.TestCheckResourceAttr(fullResource, "activity_code_ids.#", "0"),
				),
			},
			{
				// Import/Read
				ResourceName:      fullResource,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
		CheckDestroy: testVerifyTimeOffPlanDestroyed,
	})
}

// A time off plan that counts against limits needs the time off limits of what it applies to to exist, so it depends on the limit
func TestAccResourceTimeOffPlanCountingAgainstLimits(t *testing.T) {
	var (
		resourceLabel = "plan_1"
		fullResource  = ResourceType + "." + resourceLabel
	)

	dependencies := generatePlanDependencies("TestBU"+uuid.NewString(), "Test MU "+uuid.NewString(), "Test Time Off "+uuid.NewString()) +
		timeOffLimit.GenerateTimeOffLimitResource("limit_1", businessUnits.ResourceType+".bu_1.id", managementUnits.ResourceType+".mu_1.id", "Daily")

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { util.TestAccPreCheck(t) },
		ProviderFactories: provider.GetProviderFactories(providerResources, providerDataSources),
		Steps: []resource.TestStep{
			{
				Config: dependencies + GenerateTimeOffPlanResource(resourceLabel, businessUnits.ResourceType+".bu_1.id", "Test Time Off Plan "+uuid.NewString(),
					`auto_approval_rule = "CheckLimits"`,
					`count_against_time_off_limits = true`,
					fmt.Sprintf(`business_unit_association {
						management_unit_ids = [%s]
					}`, managementUnits.ResourceType+".mu_1.id"),
					fmt.Sprintf(`depends_on = [%s.limit_1]`, timeOffLimit.ResourceType),
				),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(fullResource, "auto_approval_rule", "CheckLimits"),
					resource.TestCheckResourceAttr(fullResource, "count_against_time_off_limits", util.TrueValue),
					resource.TestCheckResourceAttr(fullResource, "time_off_limit_ids.#", "1"),
				),
			},
		},
		CheckDestroy: testVerifyTimeOffPlanDestroyed,
	})
}

func testVerifyTimeOffPlanDestroyed(state *terraform.State) error {
	workforceManagementApi := platformclientv2.NewWorkforceManagementApi()
	for _, res := range state.RootModule().Resources {
		if res.Type != ResourceType {
			continue
		}

		businessUnitId, planId := SplitTimeOffPlanTerraformId(res.Primary.ID)
		plan, resp, err := workforceManagementApi.GetWorkforcemanagementBusinessunitTimeoffplan(businessUnitId, planId)
		if plan != nil {
			return fmt.Errorf("time off plan (%s) still exists", res.Primary.ID)
		}
		if util.IsStatus404(resp) {
			continue
		}
		return fmt.Errorf("unexpected error: %s", err)
	}

	return nil
}
