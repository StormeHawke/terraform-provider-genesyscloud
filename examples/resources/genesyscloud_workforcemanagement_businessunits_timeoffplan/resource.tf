resource "genesyscloud_workforcemanagement_businessunits_timeoffplan" "vacation" {
  business_unit_id = genesyscloud_workforcemanagement_businessunits.example_with_settings.id
  name             = "Vacation"
  enabled          = true

  activity_code_ids = [genesyscloud_workforcemanagement_businessunits_activitycode.team_meeting.id]

  # Checking limits needs the plan to count against them, and every management unit it applies to needs a time off limit first
  auto_approval_rule            = "CheckLimits"
  count_against_time_off_limits = true

  days_before_start_to_expire_from_waitlist = 7
  auto_publish_approved_time_off_requests   = true

  override_dates {
    date = "2026-12-25"
    type = "Blocked"
  }

  business_unit_association {
    management_unit_ids = [genesyscloud_workforcemanagement_managementunits.contact_center.id]
  }

  depends_on = [genesyscloud_workforcemanagement_businessunits_timeofflimit.contact_center]
}
