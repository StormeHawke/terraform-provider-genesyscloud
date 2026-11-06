data "genesyscloud_workforcemanagement_managementunits_workplanrotation" "weekly_rotation" {
  management_unit_id = genesyscloud_workforcemanagement_managementunits.contact_center.id
  name               = "Weekly Rotation"
}
