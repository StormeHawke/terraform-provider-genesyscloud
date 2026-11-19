resource "genesyscloud_workforcemanagement_managementunits_workplanrotation" "example_rotation" {
  management_unit_id = genesyscloud_workforcemanagement_managementunits.example_management_unit.id
  name               = "Example Work Plan Rotation"
  start_date         = "2026-01-05"
  end_date           = "2026-12-28"

  # The work plans of the pattern rotate weekly, in this order
  work_plan_ids = [
    genesyscloud_workforcemanagement_managementunits_workplan.example_work_plan.id,
  ]

  agents {
    user_id    = genesyscloud_user.example_user.id
    start_date = "2026-01-05"
    position   = 0
  }
}
