data "genesyscloud_workforcemanagement_businessunits_timeoffplan" "vacation" {
  business_unit_id = genesyscloud_workforcemanagement_businessunits.example_with_settings.id
  name             = "Vacation"
}
