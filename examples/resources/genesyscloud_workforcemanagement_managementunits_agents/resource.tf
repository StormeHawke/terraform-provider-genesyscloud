resource "genesyscloud_workforcemanagement_managementunits_agents" "example_agents" {
  management_unit_id = genesyscloud_workforcemanagement_managementunits.example_management_unit.id

  schedulable_user_ids = [
    genesyscloud_user.example_user.id,
  ]

  unschedulable_user_ids = [
    genesyscloud_user.example_user2.id,
  ]
}
