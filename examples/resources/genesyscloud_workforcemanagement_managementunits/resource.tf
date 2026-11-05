resource "genesyscloud_workforcemanagement_managementunits" "contact_center" {
  name             = "Contact Center MU"
  business_unit_id = genesyscloud_workforcemanagement_businessunits.example_with_settings.id

  settings {
    adherence {
      adherence_target_percent       = 85
      severe_alert_threshold_minutes = 10
      track_on_queue_activity        = true
      ignored_activity_categories    = ["Break"]
    }

    time_off {
      submission_range_enforced         = true
      submission_range_type             = "Relative"
      submission_earliest_days_from_now = 1
      submission_latest_days_from_now   = 90
    }

    scheduling {
      max_occupancy_percent_for_deferred_work = 80
      default_shrinkage_percent               = 10
      start_day_of_weekend                    = "Saturday"

      planning_period {
        week_count = 4
        start_date = "2026-11-01"
      }

      schedule_visibility {
        enabled      = true
        future_weeks = 2
      }
    }

    shift_trading {
      enabled             = true
      allow_direct_trades = true
      min_hours_in_future = 24
      unequal_paid        = "AdminReview"

      activity_category_rules {
        activity_category = "Meeting"
        action            = "KeepWithSchedule"
      }
    }
  }
}
