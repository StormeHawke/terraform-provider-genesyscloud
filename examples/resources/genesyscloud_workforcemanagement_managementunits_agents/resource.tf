resource "genesyscloud_workforcemanagement_managementunits_agents" "contact_center" {
  management_unit_id = genesyscloud_workforcemanagement_managementunits.contact_center.id

  schedulable_user_ids = [
    genesyscloud_user.example_user.id,
  ]

  unschedulable_user_ids = [
    genesyscloud_user.example_user2.id,
  ]
}
