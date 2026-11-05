resource "genesyscloud_workforcemanagement_businessunits_servicegoaltemplate" "standard_goals" {
  business_unit_id = genesyscloud_workforcemanagement_businessunits.example_with_settings.id
  name             = "Standard Service Goals"

  service_level {
    include = true
    percent = 80
    seconds = 20
  }

  average_speed_of_answer {
    include = true
    seconds = 30
  }

  abandon_rate {
    include = true
    percent = 5
  }

  impact_override {
    enabled = true
    impact {
      service_level {
        increase_by_percent = 10
        decrease_by_percent = 5
      }
      average_speed_of_answer {
        increase_by_percent = 20
        decrease_by_percent = 10
      }
      abandon_rate {
        increase_by_percent = 15
        decrease_by_percent = 5
      }
    }
  }
}
