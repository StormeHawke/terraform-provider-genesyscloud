data "genesyscloud_workforcemanagement_businessunits_planninggroup" "voice_support" {
  business_unit_id = genesyscloud_workforcemanagement_businessunits.example_with_settings.id
  name             = "Voice Support"
}
