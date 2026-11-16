data "genesyscloud_workforcemanagement_managementunits_workplan" "example_work_plan" {
  management_unit_id = genesyscloud_workforcemanagement_managementunits.example_management_unit.id
  name               = "Example Work Plan"
}
