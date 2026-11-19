resource "genesyscloud_workforcemanagement_managementunits_agents" "example_agents" {
  management_unit_id = genesyscloud_workforcemanagement_managementunits.example_management_unit.id

  user_ids = [
    genesyscloud_user.example_user.id,
    genesyscloud_user.example_user2.id,
  ]
}
