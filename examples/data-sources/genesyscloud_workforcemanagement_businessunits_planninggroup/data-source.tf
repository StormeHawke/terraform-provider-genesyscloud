data "genesyscloud_workforcemanagement_businessunits_planninggroup" "example_planning_group" {
  business_unit_id = genesyscloud_workforcemanagement_businessunits.example_with_settings.id
  name             = "Example Planning Group"
}
