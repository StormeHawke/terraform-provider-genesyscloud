resource "genesyscloud_workforcemanagement_managementunits_workplanrotation" "weekly_rotation" {
  management_unit_id = genesyscloud_workforcemanagement_managementunits.contact_center.id
  name               = "Weekly Rotation"
  start_date         = "2026-01-05"
  end_date           = "2026-12-28"

  # The work plans of the pattern rotate weekly, in this order
  work_plan_ids = [
    genesyscloud_workforcemanagement_managementunits_workplan.weekday_support.id,
  ]

  agents {
    user_id    = genesyscloud_user.example_user.id
    start_date = "2026-01-05"
    position   = 0
  }
}
