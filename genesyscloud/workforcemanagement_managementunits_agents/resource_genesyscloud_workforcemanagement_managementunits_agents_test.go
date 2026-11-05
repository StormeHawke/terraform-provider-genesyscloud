package workforcemanagement_managementunits_agents

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
)

/*
The resource_genesyscloud_workforcemanagement_managementunits_agents_test.go contains all of the test cases for running the resource
tests for workforcemanagement_managementunits_agents.
*/

func TestAccResourceManagementUnitAgents(t *testing.T) {
	var (
		buResourceLabel = "bu_1"
		buName          = "TestBU" + uuid.NewString()
		muResourceLabel = "mu_1"
		muName          = "Test MU " + uuid.NewString()
		resourceLabel   = "agents_1"
		fullResource    = ResourceType + "." + resourceLabel
		userLabels      = []string{"user_1", "user_2", "user_3"}
	)

	baseConfig := businessUnits.GenerateWorkforcemanagementBusinessUnitResource(
		buResourceLabel,
		buName,
		businessUnits.GenerateWorkforcemanagementBusinessUnitSettings("Sunday", "Etc/UTC", "", ""),
	) + managementUnits.GenerateManagementUnitResource(muResourceLabel, muName, businessUnits.ResourceType+"."+buResourceLabel+".id")
	for _, label := range userLabels {
		baseConfig += user.GenerateBasicUserResource(label, "terraform-"+uuid.NewString()+"@example.com", "Test User "+label)
	}
	userId := func(label string) string { return user.ResourceType + "." + label + ".id" }
	managementUnitId := managementUnits.ResourceType + "." + muResourceLabel + ".id"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { util.TestAccPreCheck(t) },
		ProviderFactories: provider.GetProviderFactories(providerResources, providerDataSources),
		Steps: []resource.TestStep{
			{
				// Two schedulable agents and one unschedulable agent
				Config: baseConfig + GenerateManagementUnitAgentsResource(
					resourceLabel,
					managementUnitId,
					fmt.Sprintf("[%s, %s]", userId("user_1"), userId("user_2")),
					fmt.Sprintf("[%s]", userId("user_3")),
				),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrPair(fullResource, "management_unit_id", managementUnits.ResourceType+"."+muResourceLabel, "id"),
					resource.TestCheckResourceAttr(fullResource, "schedulable_user_ids.#", "2"),
					resource.TestCheckResourceAttr(fullResource, "unschedulable_user_ids.#", "1"),
					resource.TestCheckTypeSetElemAttrPair(fullResource, "schedulable_user_ids.*", user.ResourceType+".user_1", "id"),
					resource.TestCheckTypeSetElemAttrPair(fullResource, "unschedulable_user_ids.*", user.ResourceType+".user_3", "id"),
				),
			},
			{
				// Flip user_2 to unschedulable and remove user_1 from the management unit
				Config: baseConfig + GenerateManagementUnitAgentsResource(
					resourceLabel,
					managementUnitId,
					fmt.Sprintf("[%s]", userId("user_3")),
					fmt.Sprintf("[%s]", userId("user_2")),
				),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(fullResource, "schedulable_user_ids.#", "1"),
					resource.TestCheckResourceAttr(fullResource, "unschedulable_user_ids.#", "1"),
					resource.TestCheckTypeSetElemAttrPair(fullResource, "schedulable_user_ids.*", user.ResourceType+".user_3", "id"),
					resource.TestCheckTypeSetElemAttrPair(fullResource, "unschedulable_user_ids.*", user.ResourceType+".user_2", "id"),
				),
			},
			{
				// Import/Read
				ResourceName:      fullResource,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
		CheckDestroy: testVerifyManagementUnitAgentsDestroyed,
	})
}

func testVerifyManagementUnitAgentsDestroyed(state *terraform.State) error {
	workforceManagementApi := platformclientv2.NewWorkforceManagementApi()
	for _, res := range state.RootModule().Resources {
		if res.Type != ResourceType {
			continue
		}

		listing, resp, err := workforceManagementApi.GetWorkforcemanagementManagementunitUsers(res.Primary.ID)
		if err != nil {
			if util.IsStatus404(resp) {
				// The management unit is gone, so its agents are too
				continue
			}
			return fmt.Errorf("unexpected error: %s", err)
		}
		if listing != nil && listing.Entities != nil && len(*listing.Entities) > 0 {
			return fmt.Errorf("management unit (%s) still has %d agent(s)", res.Primary.ID, len(*listing.Entities))
		}
	}

	return nil
}
