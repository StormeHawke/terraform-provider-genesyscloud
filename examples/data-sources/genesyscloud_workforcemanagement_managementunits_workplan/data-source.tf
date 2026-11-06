data "genesyscloud_workforcemanagement_managementunits_workplan" "weekday_support" {
  management_unit_id = genesyscloud_workforcemanagement_managementunits.contact_center.id
  name               = "Weekday Support"
}
