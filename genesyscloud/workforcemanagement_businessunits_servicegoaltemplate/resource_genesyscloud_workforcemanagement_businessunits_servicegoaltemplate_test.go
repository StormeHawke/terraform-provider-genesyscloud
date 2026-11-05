package workforcemanagement_businessunits_servicegoaltemplate

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
The resource_genesyscloud_workforcemanagement_businessunits_servicegoaltemplate_test.go contains all of the test cases for running the resource
tests for workforcemanagement_businessunits_servicegoaltemplate.
*/

func TestUnitSplitServiceGoalTemplateTerraformId(t *testing.T) {
	businessUnitId, templateId := SplitServiceGoalTemplateTerraformId("bu-1/template-2")
	if businessUnitId != "bu-1" || templateId != "template-2" {
		t.Errorf("expected bu-1 and template-2, got %s and %s", businessUnitId, templateId)
	}
}

func TestAccResourceServiceGoalTemplate(t *testing.T) {
	var (
		buResourceLabel = "bu_1"
		buName          = "TestBU" + uuid.NewString()
		resourceLabel   = "service_goal_template_1"
		name1           = "Test Service Goal Template " + uuid.NewString()
		name2           = "Test Service Goal Template Updated " + uuid.NewString()
	)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { util.TestAccPreCheck(t) },
		ProviderFactories: provider.GetProviderFactories(providerResources, providerDataSources),
		Steps: []resource.TestStep{
			{
				Config: generateTestBusinessUnit(buResourceLabel, buName) + GenerateServiceGoalTemplateResource(
					resourceLabel,
					businessUnitIdRef(buResourceLabel),
					name1,
					`service_level {
						include = true
						percent = 80
						seconds = 20
					}`,
					`average_speed_of_answer {
						include = true
						seconds = 30
					}`,
					`abandon_rate {
						include = false
					}`,
				),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrPair(ResourceType+"."+resourceLabel, "business_unit_id", businessUnits.ResourceType+"."+buResourceLabel, "id"),
					resource.TestCheckResourceAttr(ResourceType+"."+resourceLabel, "name", name1),
					resource.TestCheckResourceAttr(ResourceType+"."+resourceLabel, "service_level.0.include", util.TrueValue),
					resource.TestCheckResourceAttr(ResourceType+"."+resourceLabel, "service_level.0.percent", "80"),
					resource.TestCheckResourceAttr(ResourceType+"."+resourceLabel, "service_level.0.seconds", "20"),
					resource.TestCheckResourceAttr(ResourceType+"."+resourceLabel, "average_speed_of_answer.0.seconds", "30"),
					resource.TestCheckResourceAttr(ResourceType+"."+resourceLabel, "abandon_rate.0.include", util.FalseValue),
				),
			},
			{
				Config: generateTestBusinessUnit(buResourceLabel, buName) + GenerateServiceGoalTemplateResource(
					resourceLabel,
					businessUnitIdRef(buResourceLabel),
					name2,
					`service_level {
						include = true
						percent = 90
						seconds = 15
					}`,
					`average_speed_of_answer {
						include = false
					}`,
					`abandon_rate {
						include = true
						percent = 5
					}`,
					`impact_override {
						enabled = true
						impact {
							service_level {
								increase_by_percent = 10
								decrease_by_percent = 5
							}
							average_speed_of_answer {
								increase_by_percent = 20
								decrease_by_percent = 10
							}
							abandon_rate {
								increase_by_percent = 15
								decrease_by_percent = 5
							}
						}
					}`,
				),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(ResourceType+"."+resourceLabel, "name", name2),
					resource.TestCheckResourceAttr(ResourceType+"."+resourceLabel, "service_level.0.percent", "90"),
					resource.TestCheckResourceAttr(ResourceType+"."+resourceLabel, "average_speed_of_answer.0.include", util.FalseValue),
					resource.TestCheckResourceAttr(ResourceType+"."+resourceLabel, "abandon_rate.0.percent", "5"),
					resource.TestCheckResourceAttr(ResourceType+"."+resourceLabel, "impact_override.0.enabled", util.TrueValue),
					resource.TestCheckResourceAttr(ResourceType+"."+resourceLabel, "impact_override.0.impact.0.service_level.0.increase_by_percent", "10"),
				),
			},
			{
				// Import/Read
				ResourceName:      ResourceType + "." + resourceLabel,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
		CheckDestroy: testVerifyServiceGoalTemplateDestroyed,
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

func testVerifyServiceGoalTemplateDestroyed(state *terraform.State) error {
	workforceManagementApi := platformclientv2.NewWorkforceManagementApi()
	for _, res := range state.RootModule().Resources {
		if res.Type != ResourceType {
			continue
		}

		businessUnitId, templateId := SplitServiceGoalTemplateTerraformId(res.Primary.ID)
		template, resp, err := workforceManagementApi.GetWorkforcemanagementBusinessunitServicegoaltemplate(businessUnitId, templateId, nil)
		if template != nil {
			return fmt.Errorf("service goal template (%s) still exists", res.Primary.ID)
		}
		if util.IsStatus404(resp) {
			continue
		}
		return fmt.Errorf("unexpected error: %s", err)
	}

	return nil
}
