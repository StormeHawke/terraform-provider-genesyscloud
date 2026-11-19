data "genesyscloud_workforcemanagement_managementunits_workplanrotation" "example_rotation" {
  management_unit_id = genesyscloud_workforcemanagement_managementunits.example_management_unit.id
  name               = "Example Work Plan Rotation"
}
