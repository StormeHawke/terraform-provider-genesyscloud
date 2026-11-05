data "genesyscloud_workforcemanagement_businessunits_servicegoaltemplate" "standard_goals" {
  business_unit_id = genesyscloud_workforcemanagement_businessunits.example_with_settings.id
  name             = "Standard Service Goals"
}
