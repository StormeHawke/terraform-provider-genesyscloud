resource "genesyscloud_workforcemanagement_businessunits_timeofflimit" "example_limit" {
  business_unit_id   = genesyscloud_workforcemanagement_businessunits.example_with_settings.id
  management_unit_id = genesyscloud_workforcemanagement_managementunits.example_management_unit.id
  granularity        = "Daily"

  # Optional: set the limit values of date ranges. Only these ranges are managed by Terraform.
  value_range {
    start_date            = "2026-12-21"
    limit_minutes_per_day = [480, 480, 0, 0, 960, 960, 960]
  }
}
