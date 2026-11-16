data "genesyscloud_workforcemanagement_businessunits_timeoffplan" "example_time_off_plan" {
  business_unit_id = genesyscloud_workforcemanagement_businessunits.example_with_settings.id
  name             = "Example Time Off Plan"
}
