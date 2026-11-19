resource "genesyscloud_workforcemanagement_managementunits_workplan" "example_work_plan" {
  management_unit_id = genesyscloud_workforcemanagement_managementunits.example_management_unit.id
  name               = "Example Work Plan"
  enabled            = true

  constrain_weekly_paid_time = true
  flexible_weekly_paid_time  = false
  weekly_exact_paid_minutes  = 2400
  maximum_days               = 5
  optional_days              = ["Saturday"]

  shifts {
    name                                   = "Day"
    days                                   = ["Monday", "Tuesday", "Wednesday", "Thursday", "Friday"]
    flexible_start_time                    = false
    exact_start_time_minutes_from_midnight = 480
    flexible_paid_time                     = false
    exact_paid_time_minutes                = 480

    activities {
      activity_code_id                      = genesyscloud_workforcemanagement_businessunits_activitycode.example_activity_code.id
      length_minutes                        = 30
      start_time_is_relative_to_shift_start = true
      flexible_start_time                   = false
      exact_start_time_minutes              = 240
      counts_as_paid_time                   = false
    }
  }

  agent_ids = [
    genesyscloud_user.example_user.id,
  ]
}
