---
page_title: "genesyscloud_workforcemanagement_managementunits Resource - terraform-provider-genesyscloud"
subcategory: ""
description: |-
  Genesys Cloud workforce management management unit
---
# genesyscloud_workforcemanagement_managementunits (Resource)

<!-- This document is automatically generated. Do not edit manually. Make changes to the schema, examples, or apis.md files in examples/resources/ and run 'make docs' to regenerate. -->

Genesys Cloud workforce management management unit

## API Usage

The following Genesys Cloud APIs are used by this resource. Ensure your OAuth Client has been granted the necessary scopes and permissions to perform these operations:

* [GET /api/v2/workforcemanagement/managementunits](https://developer.genesys.cloud/devapps/api-explorer#get-api-v2-workforcemanagement-managementunits)
* [POST /api/v2/workforcemanagement/managementunits](https://developer.genesys.cloud/devapps/api-explorer#post-api-v2-workforcemanagement-managementunits)
* [DELETE /api/v2/workforcemanagement/managementunits/{managementUnitId}](https://developer.genesys.cloud/devapps/api-explorer#delete-api-v2-workforcemanagement-managementunits--managementUnitId-)
* [GET /api/v2/workforcemanagement/managementunits/{managementUnitId}](https://developer.genesys.cloud/devapps/api-explorer#get-api-v2-workforcemanagement-managementunits--managementUnitId-)
* [PATCH /api/v2/workforcemanagement/managementunits/{managementUnitId}](https://developer.genesys.cloud/devapps/api-explorer#patch-api-v2-workforcemanagement-managementunits--managementUnitId-)
* [POST /api/v2/workforcemanagement/managementunits/{managementUnitId}/move](https://developer.genesys.cloud/devapps/api-explorer#post-api-v2-workforcemanagement-managementunits--managementUnitId--move)

## Permissions and Scopes

The following permissions are required to use this resource:

* `wfm:activityCode:add`
* `wfm:activityCode:delete`
* `wfm:activityCode:edit`
* `wfm:activityCode:view`
* `wfm:agent:edit`
* `wfm:agent:view`
* `wfm:agentSchedule:view`
* `wfm:agentSchedulingPreferences:edit`
* `wfm:agentSchedulingPreferencesQuery:view`
* `wfm:agentSchedulingPreferencesSettings:view`
* `wfm:agentShiftTradeRequest:participate`
* `wfm:agentTimeOffRequest:submit`
* `wfm:businessUnit:add`
* `wfm:businessUnit:delete`
* `wfm:businessUnit:edit`
* `wfm:businessUnit:view`
* `wfm:historicalAdherence:view`
* `wfm:intraday:view`
* `wfm:managementUnit:add`
* `wfm:managementUnit:delete`
* `wfm:managementUnit:edit`
* `wfm:managementUnit:view`
* `wfm:planningGroup:add`
* `wfm:planningGroup:delete`
* `wfm:planningGroup:edit`
* `wfm:planningGroup:view`
* `wfm:publishedSchedule:view`
* `wfm:realtimeAdherence:view`
* `wfm:schedule:add`
* `wfm:schedule:delete`
* `wfm:schedule:edit`
* `wfm:schedule:generate`
* `wfm:schedule:view`
* `wfm:schedulingPreferencesQuery:view`
* `wfm:schedulingPreferencesSchedulesQuery:view`
* `wfm:schedulingPreferencesSettings:edit`
* `wfm:schedulingPreferencesSettings:view`
* `wfm:schedulingPreferencesSettingsQuery:view`
* `wfm:serviceGoalTemplate:add`
* `wfm:serviceGoalTemplate:delete`
* `wfm:serviceGoalTemplate:edit`
* `wfm:serviceGoalTemplate:view`
* `wfm:shiftTradeRequest:edit`
* `wfm:shiftTradeRequest:view`
* `wfm:shortTermForecast:add`
* `wfm:shortTermForecast:delete`
* `wfm:shortTermForecast:edit`
* `wfm:shortTermForecast:view`
* `wfm:shrinkage:view`
* `wfm:staffingGroup:add`
* `wfm:staffingGroup:delete`
* `wfm:staffingGroup:edit`
* `wfm:staffingGroup:view`
* `wfm:timeOffLimit:add`
* `wfm:timeOffLimit:delete`
* `wfm:timeOffLimit:edit`
* `wfm:timeOffLimit:view`
* `wfm:timeOffPlan:add`
* `wfm:timeOffPlan:delete`
* `wfm:timeOffPlan:edit`
* `wfm:timeOffPlan:view`
* `wfm:timeOffRequest:add`
* `wfm:timeOffRequest:edit`
* `wfm:timeOffRequest:view`
* `wfm:workPlan:add`
* `wfm:workPlan:delete`
* `wfm:workPlan:edit`
* `wfm:workPlan:view`
* `wfm:workPlanRotation:add`
* `wfm:workPlanRotation:delete`
* `wfm:workPlanRotation:edit`
* `wfm:workPlanRotation:view`

The following OAuth scopes are required to use this resource:

* `workforce-management`
* `workforce-management:readonly`


## Example Usage

```terraform
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
```

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `business_unit_id` (String) The id of the business unit this management unit belongs to. Changing this attribute moves the management unit to the new business unit.
- `name` (String) The name of the management unit.

### Optional

- `division_id` (String) The id of the division this management unit belongs to. Defaults to the home division.
- `settings` (Block List, Max: 1) The configuration settings for this management unit. Short term forecasting settings are not included because Genesys Cloud has moved them to the business unit. (see [below for nested schema](#nestedblock--settings))

### Read-Only

- `id` (String) The ID of this resource.

<a id="nestedblock--settings"></a>
### Nested Schema for `settings`

Optional:

- `adherence` (Block List, Max: 1) Adherence settings for this management unit. (see [below for nested schema](#nestedblock--settings--adherence))
- `scheduling` (Block List, Max: 1) Scheduling settings for this management unit. (see [below for nested schema](#nestedblock--settings--scheduling))
- `shift_trading` (Block List, Max: 1) Shift trade settings for this management unit. (see [below for nested schema](#nestedblock--settings--shift_trading))
- `time_off` (Block List, Max: 1) Time off request settings for this management unit. (see [below for nested schema](#nestedblock--settings--time_off))

<a id="nestedblock--settings--adherence"></a>
### Nested Schema for `settings.adherence`

Optional:

- `adherence_exception_threshold_seconds` (Number) The threshold in seconds for which agents should not be penalized for being momentarily out of adherence.
- `adherence_target_percent` (Number) Target adherence percentage.
- `ignored_activity_categories` (Set of String) Activity categories that are ignored for adherence purposes. Valid values: OnQueueWork, Break, Meal, Meeting, OffQueueWork, TimeOff, Training, Unavailable.
- `ignored_activity_code_ids` (Set of String) Activity code ids that are ignored for adherence purposes.
- `non_on_queue_activities_equivalent` (Boolean) Whether to treat all non-on-queue activities as equivalent for adherence purposes.
- `severe_alert_threshold_minutes` (Number) The threshold in minutes where an alert is triggered when an agent is considered severely out of adherence.
- `track_on_queue_activity` (Boolean) Whether to track on-queue activities.


<a id="nestedblock--settings--scheduling"></a>
### Nested Schema for `settings.scheduling`

Optional:

- `default_shrinkage_percent` (Number) Default shrinkage percent for scheduling.
- `max_occupancy_percent_for_deferred_work` (Number) Max occupancy percent for deferred work.
- `monthly_planning_period` (Block List, Max: 1) Monthly planning period settings for scheduling. Only one of planning_period or monthly_planning_period may be defined. (see [below for nested schema](#nestedblock--settings--scheduling--monthly_planning_period))
- `planning_period` (Block List, Max: 1) Planning period settings for scheduling. Only one of planning_period or monthly_planning_period may be defined. (see [below for nested schema](#nestedblock--settings--scheduling--planning_period))
- `schedule_visibility` (Block List, Max: 1) Schedule visibility settings for agents. (see [below for nested schema](#nestedblock--settings--scheduling--schedule_visibility))
- `shrinkage_overrides` (Block Set) Shrinkage overrides for scheduling. Set to an empty list to clear existing overrides. (see [below for nested schema](#nestedblock--settings--scheduling--shrinkage_overrides))
- `start_day_of_weekend` (String) Start day of the weekend for scheduling.

<a id="nestedblock--settings--scheduling--monthly_planning_period"></a>
### Nested Schema for `settings.scheduling.monthly_planning_period`

Required:

- `start_date` (String) Start date of the monthly planning period in yyyy-MM-dd format. The date must be the first day of a month.


<a id="nestedblock--settings--scheduling--planning_period"></a>
### Nested Schema for `settings.scheduling.planning_period`

Required:

- `start_date` (String) Start date of the planning period in yyyy-MM-dd format.
- `week_count` (Number) Planning period length in weeks.


<a id="nestedblock--settings--scheduling--schedule_visibility"></a>
### Nested Schema for `settings.scheduling.schedule_visibility`

Optional:

- `enabled` (Boolean) Whether schedule visibility controls are enabled for this management unit.
- `future_weeks` (Number) The number of weeks into the future that agents can see schedules in this management unit. 0 means the current week.


<a id="nestedblock--settings--scheduling--shrinkage_overrides"></a>
### Nested Schema for `settings.scheduling.shrinkage_overrides`

Required:

- `interval_index` (Number) Index of the shrinkage override interval. The starting index is 0 and indexes are based on 15 minute intervals for a 7 day week.
- `shrinkage_percent` (Number) Shrinkage override percent.



<a id="nestedblock--settings--shift_trading"></a>
### Nested Schema for `settings.shift_trading`

Optional:

- `activity_category_rules` (Block List) Rules that specify what to do with activity categories that are part of a shift in a trade. (see [below for nested schema](#nestedblock--settings--shift_trading--activity_category_rules))
- `allow_direct_trades` (Boolean) Whether direct shift trades between agents are allowed.
- `auto_review` (Boolean) Whether automatic shift trade review is enabled according to the rules defined for this management unit.
- `enabled` (Boolean) Whether shift trading is enabled for this management unit.
- `external_activity_rule_ignores_activity_category_rule` (Boolean) Whether external activity rules ignore activity category rules.
- `external_activity_rules` (Block List) Rules that specify what to do with external activities that are part of a shift. (see [below for nested schema](#nestedblock--settings--shift_trading--external_activity_rules))
- `max_trade_span_weeks` (Number) The maximum number of weeks a shift trade can span.
- `max_trades_per_agent_per_week` (Number) The maximum number of shift trades an agent can submit per week.
- `min_hours_in_future` (Number) The minimum number of hours in the future that shift trades are allowed.
- `min_minutes_between_shifts` (Number) The minimum number of minutes between shifts.
- `min_minutes_between_shifts_violations` (String) How to handle shift trades that violate the minimum minutes between shifts constraint. Valid values: Allow, Disallow, AdminReview.
- `one_sided` (String) How to handle one-sided shift trades. Valid values: Allow, Disallow, AdminReview.
- `planning_period_max_paid_violations` (String) How to handle shift trades that violate the planning period maximum paid time constraint. Valid values: Allow, Disallow, AdminReview.
- `planning_period_min_paid_violations` (String) How to handle shift trades that violate the planning period minimum paid time constraint. Valid values: Allow, Disallow, AdminReview.
- `requires_matching_languages` (Boolean) Whether to constrain shift trades to agents with matching languages.
- `requires_matching_planning_groups` (Boolean) Whether to constrain shift trades to agents with matching planning groups.
- `requires_matching_queues` (Boolean) Whether to constrain shift trades to agents with matching queues.
- `requires_matching_skills` (Boolean) Whether to constrain shift trades to agents with matching skills.
- `unequal_paid` (String) How to handle shift trades that involve unequal paid times. Valid values: Allow, Disallow, AdminReview.
- `weekly_max_paid_violations` (String) How to handle shift trades that violate the weekly maximum paid time constraint. Valid values: Allow, Disallow, AdminReview.
- `weekly_min_paid_violations` (String) How to handle shift trades that violate the weekly minimum paid time constraint. Valid values: Allow, Disallow, AdminReview.

<a id="nestedblock--settings--shift_trading--activity_category_rules"></a>
### Nested Schema for `settings.shift_trading.activity_category_rules`

Required:

- `action` (String) The action this rule invokes. Valid values: Replace, DoNotAllowTrade, KeepWithSchedule.
- `activity_category` (String) The activity category to which this rule applies. Valid values: OnQueueWork, Break, Meal, Meeting, OffQueueWork, TimeOff, Training, Unavailable.

Optional:

- `activity_code_id_replacement` (String) The id of the default activity code that replaces activities of the original category. Required if action is Replace.


<a id="nestedblock--settings--shift_trading--external_activity_rules"></a>
### Nested Schema for `settings.shift_trading.external_activity_rules`

Required:

- `action` (String) The action this rule invokes. Valid values: DoNotAllowTrade, KeepWithAgent.
- `external_activity_type` (String) The external activity type to which this rule applies. Valid values: ActivityPlan, Coaching, Learning, Opportunity.

Optional:

- `activity_code_id_replacement` (String) The id of the default activity code that replaces external activities. Required if action is KeepWithAgent.



<a id="nestedblock--settings--time_off"></a>
### Nested Schema for `settings.time_off`

Optional:

- `submission_earliest_days_from_now` (Number) The earliest number of days from now for which an agent can submit a time off request. Use negative numbers to indicate days in the past.
- `submission_latest_date` (String) The latest date for time off request submission, interpreted in the business unit time zone, in yyyy-MM-dd format.
- `submission_latest_days_from_now` (Number) The latest number of days from now for which an agent can submit a time off request.
- `submission_range_enforced` (Boolean) Whether to enforce a submission range for agent time off requests.
- `submission_range_type` (String) The type of the submission range. Valid values: Relative, FixedEnd.

