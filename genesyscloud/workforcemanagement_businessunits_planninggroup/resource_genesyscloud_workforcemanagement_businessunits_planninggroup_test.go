package workforcemanagement_businessunits_planninggroup

import (
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/mypurecloud/platform-client-sdk-go/v200/platformclientv2"
	"github.com/mypurecloud/terraform-provider-genesyscloud/genesyscloud/provider"
	routingQueue "github.com/mypurecloud/terraform-provider-genesyscloud/genesyscloud/routing_queue"
	"github.com/mypurecloud/terraform-provider-genesyscloud/genesyscloud/util"
	businessUnits "github.com/mypurecloud/terraform-provider-genesyscloud/genesyscloud/workforcemanagement_businessunits"
	serviceGoalTemplate "github.com/mypurecloud/terraform-provider-genesyscloud/genesyscloud/workforcemanagement_businessunits_servicegoaltemplate"
)

/*
The resource_genesyscloud_workforcemanagement_businessunits_planninggroup_test.go contains all of the test cases for running the resource
tests for workforcemanagement_businessunits_planninggroup.
*/

func TestUnitSplitPlanningGroupTerraformId(t *testing.T) {
	businessUnitId, planningGroupId := SplitPlanningGroupTerraformId("bu-1/group-2")
	if businessUnitId != "bu-1" || planningGroupId != "group-2" {
		t.Errorf("expected bu-1 and group-2, got %s and %s", businessUnitId, planningGroupId)
	}
}

func TestUnitModifyServiceGoalTemplateIdStateValue(t *testing.T) {
	if got := ModifyServiceGoalTemplateIdStateValue("bu-1/template-2"); got != "template-2" {
		t.Errorf("expected template-2, got %s", got)
	}
	if got := ModifyServiceGoalTemplateIdStateValue("template-2"); got != "template-2" {
		t.Errorf("expected template-2, got %s", got)
	}
}

func TestAccResourcePlanningGroup(t *testing.T) {
	var (
		buResourceLabel    = "bu_1"
		buName             = "TestBU" + uuid.NewString()
		sgtResourceLabel   = "service_goal_template_1"
		sgtName            = "Test Service Goal Template " + uuid.NewString()
		queueResourceLabel = "queue_1"
		queueName          = "Test Queue " + uuid.NewString()
		resourceLabel      = "planning_group_1"
		name1              = "Test Planning Group " + uuid.NewString()
		name2              = "Test Planning Group Updated " + uuid.NewString()
	)

	baseConfig := generateTestBusinessUnit(buResourceLabel, buName) +
		serviceGoalTemplate.GenerateServiceGoalTemplateResource(
			sgtResourceLabel,
			businessUnitIdRef(buResourceLabel),
			sgtName,
			`abandon_rate {
				include = true
				percent = 5
			}`,
		) +
		routingQueue.GenerateRoutingQueueResourceBasic(queueResourceLabel, queueName)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { util.TestAccPreCheck(t) },
		ProviderFactories: provider.GetProviderFactories(providerResources, providerDataSources),
		Steps: []resource.TestStep{
			{
				// Create without a service goal template or route paths
				Config: baseConfig + GeneratePlanningGroupResource(resourceLabel, businessUnitIdRef(buResourceLabel), name1),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrPair(ResourceType+"."+resourceLabel, "business_unit_id", businessUnits.ResourceType+"."+buResourceLabel, "id"),
					resource.TestCheckResourceAttr(ResourceType+"."+resourceLabel, "name", name1),
					resource.TestCheckResourceAttr(ResourceType+"."+resourceLabel, "route_paths.#", "0"),
				),
			},
			{
				// Update with a service goal template and a route path
				Config: baseConfig + GeneratePlanningGroupResource(
					resourceLabel,
					businessUnitIdRef(buResourceLabel),
					name2,
					fmt.Sprintf("service_goal_template_id = %s.%s.id", serviceGoalTemplate.ResourceType, sgtResourceLabel),
					fmt.Sprintf(`route_paths {
						queue_id   = %s.%s.id
						media_type = "Voice"
					}`, routingQueue.ResourceType, queueResourceLabel),
				),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(ResourceType+"."+resourceLabel, "name", name2),
					resource.TestCheckResourceAttr(ResourceType+"."+resourceLabel, "route_paths.#", "1"),
					resource.TestCheckResourceAttrSet(ResourceType+"."+resourceLabel, "service_goal_template_id"),
				),
			},
			{
				// Import/Read
				ResourceName:      ResourceType + "." + resourceLabel,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
		CheckDestroy: testVerifyPlanningGroupDestroyed,
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

func testVerifyPlanningGroupDestroyed(state *terraform.State) error {
	workforceManagementApi := platformclientv2.NewWorkforceManagementApi()
	for _, res := range state.RootModule().Resources {
		if res.Type != ResourceType {
			continue
		}

		businessUnitId, planningGroupId := SplitPlanningGroupTerraformId(res.Primary.ID)
		planningGroup, resp, err := workforceManagementApi.GetWorkforcemanagementBusinessunitPlanninggroup(businessUnitId, planningGroupId)
		if planningGroup != nil {
			return fmt.Errorf("planning group (%s) still exists", res.Primary.ID)
		}
		if util.IsStatus404(resp) {
			continue
		}
		return fmt.Errorf("unexpected error: %s", err)
	}

	return nil
}
