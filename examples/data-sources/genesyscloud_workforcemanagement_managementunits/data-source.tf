data "genesyscloud_workforcemanagement_managementunits" "example_management_unit" {
  business_unit_id = genesyscloud_workforcemanagement_businessunits.example_with_settings.id
  name             = "Example Management Unit"
}
