package workforcemanagement_managementunits_workplanrotation

import (
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/mypurecloud/platform-client-sdk-go/v200/platformclientv2"
	"github.com/mypurecloud/terraform-provider-genesyscloud/genesyscloud/provider"
	"github.com/mypurecloud/terraform-provider-genesyscloud/genesyscloud/user"
	"github.com/mypurecloud/terraform-provider-genesyscloud/genesyscloud/util"
	businessUnits "github.com/mypurecloud/terraform-provider-genesyscloud/genesyscloud/workforcemanagement_businessunits"
	managementUnits "github.com/mypurecloud/terraform-provider-genesyscloud/genesyscloud/workforcemanagement_managementunits"
	managementUnitAgents "github.com/mypurecloud/terraform-provider-genesyscloud/genesyscloud/workforcemanagement_managementunits_agents"
	workPlan "github.com/mypurecloud/terraform-provider-genesyscloud/genesyscloud/workforcemanagement_managementunits_workplan"
)

/*
The resource_genesyscloud_workforcemanagement_managementunits_workplanrotation_test.go contains all of the test cases for running the resource
tests for workforcemanagement_managementunits_workplanrotation.
*/

// generateRotationDependencies returns a business unit, a management unit and two enabled work plans, which a rotation needs for its pattern
func generateRotationDependencies(buName string, muName string) string {
	shift := `shifts {
		name                                   = "Day"
		days                                   = ["Monday", "Tuesday", "Wednesday", "Thursday", "Friday"]
		flexible_start_time                    = false
		exact_start_time_minutes_from_midnight = 480
		flexible_paid_time                     = false
		exact_paid_time_minutes                = 480
	}`
	managementUnitId := managementUnits.ResourceType + ".mu_1.id"
	return businessUnits.GenerateWorkforcemanagementBusinessUnitResource(
		"bu_1",
		buName,
		businessUnits.GenerateWorkforcemanagementBusinessUnitSettings("Sunday", "Etc/UTC", "", ""),
	) +
		managementUnits.GenerateManagementUnitResource("mu_1", muName, businessUnits.ResourceType+".bu_1.id") +
		workPlan.GenerateWorkPlanResource("plan_a", managementUnitId, "Plan A "+uuid.NewString(), `enabled = true`, shift) +
		workPlan.GenerateWorkPlanResource("plan_b", managementUnitId, "Plan B "+uuid.NewString(), `enabled = true`, shift)
}

func TestAccResourceWorkPlanRotation(t *testing.T) {
	var (
		resourceLabel = "rotation_1"
		fullResource  = ResourceType + "." + resourceLabel
		name1         = "Test Rotation " + uuid.NewString()
		name2         = "Test Rotation Updated " + uuid.NewString()
		userLabel     = "user_1"
	)

	dependencies := generateRotationDependencies("TestBU"+uuid.NewString(), "Test MU "+uuid.NewString())
	managementUnitId := managementUnits.ResourceType + ".mu_1.id"
	planA := workPlan.ResourceType + ".plan_a.id"
	planB := workPlan.ResourceType + ".plan_b.id"
	userId := user.ResourceType + "." + userLabel + ".id"

	// An agent can only be added to a rotation when it is a member of the management unit
	agentDependencies := user.GenerateBasicUserResource(userLabel, "terraform-"+uuid.NewString()+"@example.com", "Test User") +
		managementUnitAgents.GenerateManagementUnitAgentsResource("agents_1", managementUnitId, fmt.Sprintf("[%s]", userId), "[]")

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { util.TestAccPreCheck(t) },
		ProviderFactories: provider.GetProviderFactories(providerResources, providerDataSources),
		Steps: []resource.TestStep{
			{
				// Create an open ended rotation of two work plans
				Config: dependencies + GenerateWorkPlanRotationResource(resourceLabel, managementUnitId, name1, "2026-01-05", fmt.Sprintf("[%s, %s]", planA, planB)),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrPair(fullResource, "management_unit_id", managementUnits.ResourceType+".mu_1", "id"),
					resource.TestCheckResourceAttr(fullResource, "name", name1),
					resource.TestCheckResourceAttr(fullResource, "enabled", util.TrueValue),
					resource.TestCheckResourceAttr(fullResource, "start_date", "2026-01-05"),
					resource.TestCheckNoResourceAttr(fullResource, "end_date"),
					resource.TestCheckResourceAttr(fullResource, "work_plan_ids.#", "2"),
					resource.TestCheckResourceAttrPair(fullResource, "work_plan_ids.0", workPlan.ResourceType+".plan_a", "id"),
				),
			},
			{
				// Rename, end the rotation, reverse the pattern and disable it
				Config: dependencies + GenerateWorkPlanRotationResource(resourceLabel, managementUnitId, name2, "2026-01-05", fmt.Sprintf("[%s, %s]", planB, planA),
					`end_date = "2026-12-28"`, `enabled = false`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(fullResource, "name", name2),
					resource.TestCheckResourceAttr(fullResource, "end_date", "2026-12-28"),
					resource.TestCheckResourceAttr(fullResource, "enabled", util.FalseValue),
					resource.TestCheckResourceAttrPair(fullResource, "work_plan_ids.0", workPlan.ResourceType+".plan_b", "id"),
				),
			},
			{
				// Add an agent
				Config: dependencies + agentDependencies + GenerateWorkPlanRotationResource(resourceLabel, managementUnitId, name2, "2026-01-05", fmt.Sprintf("[%s, %s]", planB, planA),
					`end_date = "2026-12-28"`,
					`enabled = false`,
					fmt.Sprintf(`agents {
						user_id    = %s
						start_date = "2026-01-05"
						position   = 1
					}`, userId),
					fmt.Sprintf(`depends_on = [%s.agents_1]`, managementUnitAgents.ResourceType),
				),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(fullResource, "agents.#", "1"),
					resource.TestCheckTypeSetElemNestedAttrs(fullResource, "agents.*", map[string]string{"start_date": "2026-01-05", "position": "1"}),
				),
			},
			{
				// Import/Read
				ResourceName:      fullResource,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
		CheckDestroy: testVerifyWorkPlanRotationDestroyed,
	})
}

func testVerifyWorkPlanRotationDestroyed(state *terraform.State) error {
	workforceManagementApi := platformclientv2.NewWorkforceManagementApi()
	for _, res := range state.RootModule().Resources {
		if res.Type != ResourceType {
			continue
		}

		managementUnitId, rotationId := SplitWorkPlanRotationTerraformId(res.Primary.ID)
		rotation, resp, err := workforceManagementApi.GetWorkforcemanagementManagementunitWorkplanrotation(managementUnitId, rotationId)
		if rotation != nil {
			return fmt.Errorf("work plan rotation (%s) still exists", res.Primary.ID)
		}
		if util.IsStatus404(resp) {
			continue
		}
		return fmt.Errorf("unexpected error: %s", err)
	}

	return nil
}
