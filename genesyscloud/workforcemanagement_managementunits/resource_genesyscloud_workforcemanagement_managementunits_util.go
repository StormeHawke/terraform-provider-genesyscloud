package workforcemanagement_managementunits

import (
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/mypurecloud/platform-client-sdk-go/v200/platformclientv2"
)

const dateLayout = "2006-01-02"

// ModifyActivityCodeIdStateValue strips the business unit from an activity code id before it is saved in state.
// The activity code resource id is <businessUnitId>/<activityCodeId>, but the management unit api only returns the activity code id,
// so saving the composite value would cause a non-empty plan.
func ModifyActivityCodeIdStateValue(id interface{}) string {
	activityCodeId := id.(string)
	if i := strings.LastIndex(activityCodeId, "/"); i >= 0 {
		return activityCodeId[i+1:]
	}
	return activityCodeId
}

/*
Settings requests only include the fields that are present in the configuration. Many settings fields are optional and
computed, and a field the user did not configure must not be sent as a zero value, so presence is read from the raw
configuration instead of from the merged resource data.
*/

var nullValue = cty.NullVal(cty.DynamicPseudoType)

func isPresent(v cty.Value) bool {
	return !v.IsNull() && v.IsKnown()
}

func attrOf(v cty.Value, name string) cty.Value {
	if !isPresent(v) || !v.Type().IsObjectType() || !v.Type().HasAttribute(name) {
		return nullValue
	}
	return v.GetAttr(name)
}

// blockOf returns the single element of a list block with a maximum of one item
func blockOf(v cty.Value, name string) cty.Value {
	list := attrOf(v, name)
	if !isPresent(list) || list.LengthInt() == 0 {
		return nullValue
	}
	return list.AsValueSlice()[0]
}

func stringOf(v cty.Value, name string) *string {
	attr := attrOf(v, name)
	if !isPresent(attr) {
		return nil
	}
	value := attr.AsString()
	return &value
}

func intOf(v cty.Value, name string) *int {
	attr := attrOf(v, name)
	if !isPresent(attr) {
		return nil
	}
	value, _ := attr.AsBigFloat().Int64()
	intValue := int(value)
	return &intValue
}

func floatOf(v cty.Value, name string) *float64 {
	attr := attrOf(v, name)
	if !isPresent(attr) {
		return nil
	}
	value, _ := new(big.Float).Set(attr.AsBigFloat()).Float64()
	return &value
}

func boolOf(v cty.Value, name string) *bool {
	attr := attrOf(v, name)
	if !isPresent(attr) {
		return nil
	}
	value := attr.True()
	return &value
}

func stringSliceOf(v cty.Value, name string, transform func(string) string) *[]string {
	attr := attrOf(v, name)
	if !isPresent(attr) {
		return nil
	}
	values := make([]string, 0, attr.LengthInt())
	for _, element := range attr.AsValueSlice() {
		if isPresent(element) {
			value := element.AsString()
			if transform != nil {
				value = transform(value)
			}
			values = append(values, value)
		}
	}
	return &values
}

func dateOf(v cty.Value, name string) *time.Time {
	value := stringOf(v, name)
	if value == nil {
		return nil
	}
	parsed, err := time.Parse(dateLayout, *value)
	if err != nil {
		return nil
	}
	return &parsed
}

func buildAdherence(settings cty.Value) *platformclientv2.Adherencesettings {
	block := blockOf(settings, "adherence")
	if !isPresent(block) {
		return nil
	}
	adherence := &platformclientv2.Adherencesettings{
		SevereAlertThresholdMinutes:        intOf(block, "severe_alert_threshold_minutes"),
		AdherenceTargetPercent:             intOf(block, "adherence_target_percent"),
		AdherenceExceptionThresholdSeconds: intOf(block, "adherence_exception_threshold_seconds"),
		NonOnQueueActivitiesEquivalent:     boolOf(block, "non_on_queue_activities_equivalent"),
		TrackOnQueueActivity:               boolOf(block, "track_on_queue_activity"),
	}
	if categories := stringSliceOf(block, "ignored_activity_categories", nil); categories != nil {
		adherence.IgnoredActivityCategories = &platformclientv2.Ignoredactivitycategories{Values: categories}
	}
	if codeIds := stringSliceOf(block, "ignored_activity_code_ids", func(id string) string { return ModifyActivityCodeIdStateValue(id) }); codeIds != nil {
		adherence.IgnoredActivityCodeIds = &platformclientv2.Ignoredactivitycodeids{Values: codeIds}
	}
	return adherence
}

func buildTimeOffRequest(settings cty.Value) *platformclientv2.Timeoffsettingsrequest {
	block := blockOf(settings, "time_off")
	if !isPresent(block) {
		return nil
	}
	timeOff := &platformclientv2.Timeoffsettingsrequest{
		SubmissionRangeEnforced:       boolOf(block, "submission_range_enforced"),
		SubmissionRangeType:           stringOf(block, "submission_range_type"),
		SubmissionEarliestDaysFromNow: intOf(block, "submission_earliest_days_from_now"),
		SubmissionLatestDaysFromNow:   intOf(block, "submission_latest_days_from_now"),
	}
	if latestDate := dateOf(block, "submission_latest_date"); latestDate != nil {
		timeOff.SubmissionLatestDate = &platformclientv2.Valuewrapperlocaldate{Value: latestDate}
	}
	return timeOff
}

func buildShrinkageOverrides(scheduling cty.Value) *platformclientv2.Shrinkageoverrides {
	attr := attrOf(scheduling, "shrinkage_overrides")
	if !isPresent(attr) {
		return nil
	}
	if attr.LengthInt() == 0 {
		return &platformclientv2.Shrinkageoverrides{Clear: platformclientv2.Bool(true)}
	}
	values := make([]platformclientv2.Shrinkageoverride, 0, attr.LengthInt())
	for _, element := range attr.AsValueSlice() {
		values = append(values, platformclientv2.Shrinkageoverride{
			IntervalIndex:    intOf(element, "interval_index"),
			ShrinkagePercent: floatOf(element, "shrinkage_percent"),
		})
	}
	return &platformclientv2.Shrinkageoverrides{Values: &values}
}

func buildScheduling(settings cty.Value) *platformclientv2.Schedulingsettingsrequest {
	block := blockOf(settings, "scheduling")
	if !isPresent(block) {
		return nil
	}
	scheduling := &platformclientv2.Schedulingsettingsrequest{
		MaxOccupancyPercentForDeferredWork: intOf(block, "max_occupancy_percent_for_deferred_work"),
		DefaultShrinkagePercent:            floatOf(block, "default_shrinkage_percent"),
		ShrinkageOverrides:                 buildShrinkageOverrides(block),
		StartDayOfWeekend:                  stringOf(block, "start_day_of_weekend"),
	}
	// Only the configured planning period is sent. Switching between planning_period and monthly_planning_period does not clear the
	// other one: that needs the other value wrapper sent with a null value, which is not done here.
	if planningPeriod := blockOf(block, "planning_period"); isPresent(planningPeriod) {
		scheduling.PlanningPeriod = &platformclientv2.Valuewrapperplanningperiodsettings{
			Value: &platformclientv2.Planningperiodsettings{
				WeekCount: intOf(planningPeriod, "week_count"),
				StartDate: dateOf(planningPeriod, "start_date"),
			},
		}
	}
	if monthlyPlanningPeriod := blockOf(block, "monthly_planning_period"); isPresent(monthlyPlanningPeriod) {
		scheduling.MonthlyPlanningPeriod = &platformclientv2.Valuewrappermonthlyplanningperiodsettings{
			Value: &platformclientv2.Monthlyplanningperiodsettings{StartDate: dateOf(monthlyPlanningPeriod, "start_date")},
		}
	}
	if scheduleVisibility := blockOf(block, "schedule_visibility"); isPresent(scheduleVisibility) {
		scheduling.ScheduleVisibility = &platformclientv2.Schedulevisibilitysettingsrequest{
			Enabled:     boolOf(scheduleVisibility, "enabled"),
			FutureWeeks: intOf(scheduleVisibility, "future_weeks"),
		}
	}
	return scheduling
}

func buildShiftTrading(settings cty.Value) *platformclientv2.Shifttradesettings {
	block := blockOf(settings, "shift_trading")
	if !isPresent(block) {
		return nil
	}
	shiftTrading := &platformclientv2.Shifttradesettings{
		Enabled:                           boolOf(block, "enabled"),
		AutoReview:                        boolOf(block, "auto_review"),
		AllowDirectTrades:                 boolOf(block, "allow_direct_trades"),
		MinHoursInFuture:                  intOf(block, "min_hours_in_future"),
		UnequalPaid:                       stringOf(block, "unequal_paid"),
		OneSided:                          stringOf(block, "one_sided"),
		WeeklyMinPaidViolations:           stringOf(block, "weekly_min_paid_violations"),
		WeeklyMaxPaidViolations:           stringOf(block, "weekly_max_paid_violations"),
		RequiresMatchingQueues:            boolOf(block, "requires_matching_queues"),
		RequiresMatchingLanguages:         boolOf(block, "requires_matching_languages"),
		RequiresMatchingSkills:            boolOf(block, "requires_matching_skills"),
		RequiresMatchingPlanningGroups:    boolOf(block, "requires_matching_planning_groups"),
		MaxTradeSpanWeeks:                 intOf(block, "max_trade_span_weeks"),
		MaxTradesPerAgentPerWeek:          intOf(block, "max_trades_per_agent_per_week"),
		MinMinutesBetweenShifts:           intOf(block, "min_minutes_between_shifts"),
		PlanningPeriodMinPaidViolations:   stringOf(block, "planning_period_min_paid_violations"),
		PlanningPeriodMaxPaidViolations:   stringOf(block, "planning_period_max_paid_violations"),
		MinMinutesBetweenShiftsViolations: stringOf(block, "min_minutes_between_shifts_violations"),

		ExternalActivityRuleIgnoresActivityCategoryRule: boolOf(block, "external_activity_rule_ignores_activity_category_rule"),
	}
	if rules := attrOf(block, "activity_category_rules"); isPresent(rules) {
		activityRules := make([]platformclientv2.Shifttradeactivityrule, 0, rules.LengthInt())
		for _, rule := range rules.AsValueSlice() {
			activityRules = append(activityRules, platformclientv2.Shifttradeactivityrule{
				ActivityCategory:          stringOf(rule, "activity_category"),
				Action:                    stringOf(rule, "action"),
				ActivityCodeIdReplacement: nonEmpty(stringOf(rule, "activity_code_id_replacement")),
			})
		}
		shiftTrading.ActivityCategoryRules = &activityRules
	}
	if rules := attrOf(block, "external_activity_rules"); isPresent(rules) {
		externalRules := make([]platformclientv2.Shifttradeexternalactivityrule, 0, rules.LengthInt())
		for _, rule := range rules.AsValueSlice() {
			externalRules = append(externalRules, platformclientv2.Shifttradeexternalactivityrule{
				ExternalActivityType:      stringOf(rule, "external_activity_type"),
				Action:                    stringOf(rule, "action"),
				ActivityCodeIdReplacement: nonEmpty(stringOf(rule, "activity_code_id_replacement")),
			})
		}
		shiftTrading.ExternalActivityRules = &externalRules
	}
	return shiftTrading
}

func nonEmpty(value *string) *string {
	if value == nil || *value == "" {
		return nil
	}
	return value
}

// rawSettings returns the settings block from the raw configuration, or a null value if it was not configured
func rawSettings(d *schema.ResourceData) cty.Value {
	return blockOf(d.GetRawConfig(), "settings")
}

func buildCreateManagementUnitRequest(d *schema.ResourceData) *platformclientv2.Createmanagementunitapirequest {
	request := &platformclientv2.Createmanagementunitapirequest{
		Name:           platformclientv2.String(d.Get("name").(string)),
		BusinessUnitId: platformclientv2.String(d.Get("business_unit_id").(string)),
		DivisionId:     nonEmpty(stringOf(d.GetRawConfig(), "division_id")),
	}

	if settings := rawSettings(d); isPresent(settings) {
		request.Settings = &platformclientv2.Createmanagementunitsettingsrequest{
			Adherence:    buildAdherence(settings),
			TimeOff:      buildTimeOffRequest(settings),
			Scheduling:   buildScheduling(settings),
			ShiftTrading: buildShiftTrading(settings),
		}
	}
	return request
}

func buildUpdateManagementUnitRequest(d *schema.ResourceData, metadata *platformclientv2.Wfmversionedentitymetadata) *platformclientv2.Updatemanagementunitrequest {
	request := &platformclientv2.Updatemanagementunitrequest{
		Name:       platformclientv2.String(d.Get("name").(string)),
		DivisionId: nonEmpty(stringOf(d.GetRawConfig(), "division_id")),
	}

	settings := &platformclientv2.Managementunitsettingsrequest{}
	if rawSettings := rawSettings(d); isPresent(rawSettings) {
		settings.Adherence = buildAdherence(rawSettings)
		settings.TimeOff = buildTimeOffRequest(rawSettings)
		settings.Scheduling = buildScheduling(rawSettings)
		settings.ShiftTrading = buildShiftTrading(rawSettings)
	}
	// The settings carry the current metadata version for optimistic locking
	if metadata.Version != nil {
		settings.Metadata = &platformclientv2.Wfmversionedentitymetadata{Version: metadata.Version}
	}
	request.Settings = settings
	return request
}

func flattenAdherence(adherence *platformclientv2.Adherencesettings) []interface{} {
	if adherence == nil {
		return nil
	}
	block := map[string]interface{}{
		"severe_alert_threshold_minutes":        *adherence.SevereAlertThresholdMinutes,
		"adherence_target_percent":              *adherence.AdherenceTargetPercent,
		"adherence_exception_threshold_seconds": *adherence.AdherenceExceptionThresholdSeconds,
		"non_on_queue_activities_equivalent":    *adherence.NonOnQueueActivitiesEquivalent,
		"track_on_queue_activity":               *adherence.TrackOnQueueActivity,
		"ignored_activity_categories":           stringSet(*adherence.IgnoredActivityCategories.Values),
		"ignored_activity_code_ids":             stringSet(*adherence.IgnoredActivityCodeIds.Values),
	}
	return []interface{}{block}
}

func flattenTimeOff(timeOff *platformclientv2.Timeoffsettingsresponse) []interface{} {
	if timeOff == nil {
		return nil
	}
	block := map[string]interface{}{
		"submission_range_enforced":         *timeOff.SubmissionRangeEnforced,
		"submission_range_type":             *timeOff.SubmissionRangeType,
		"submission_earliest_days_from_now": *timeOff.SubmissionEarliestDaysFromNow,
		"submission_latest_days_from_now":   *timeOff.SubmissionLatestDaysFromNow,
	}
	if timeOff.SubmissionLatestDate != nil {
		block["submission_latest_date"] = timeOff.SubmissionLatestDate.Format(dateLayout)
	}
	return []interface{}{block}
}

func flattenScheduling(scheduling *platformclientv2.Schedulingsettingsresponse) []interface{} {
	if scheduling == nil {
		return nil
	}
	overrides := make([]interface{}, 0)
	for _, override := range *scheduling.ShrinkageOverrides.Values {
		overrides = append(overrides, map[string]interface{}{
			"interval_index":    *override.IntervalIndex,
			"shrinkage_percent": *override.ShrinkagePercent,
		})
	}
	block := map[string]interface{}{
		"max_occupancy_percent_for_deferred_work": *scheduling.MaxOccupancyPercentForDeferredWork,
		"default_shrinkage_percent":               *scheduling.DefaultShrinkagePercent,
		"start_day_of_weekend":                    *scheduling.StartDayOfWeekend,
		"shrinkage_overrides":                     schema.NewSet(schema.HashResource(shrinkageOverrideResource), overrides),
		"schedule_visibility": []interface{}{map[string]interface{}{
			"enabled":      *scheduling.ScheduleVisibility.Enabled,
			"future_weeks": *scheduling.ScheduleVisibility.FutureWeeks,
		}},
	}
	if scheduling.PlanningPeriod != nil {
		block["planning_period"] = []interface{}{map[string]interface{}{
			"week_count": *scheduling.PlanningPeriod.WeekCount,
			"start_date": scheduling.PlanningPeriod.StartDate.Format(dateLayout),
		}}
	}
	if scheduling.MonthlyPlanningPeriod != nil {
		block["monthly_planning_period"] = []interface{}{map[string]interface{}{
			"start_date": scheduling.MonthlyPlanningPeriod.StartDate.Format(dateLayout),
		}}
	}
	return []interface{}{block}
}

func flattenShiftTrading(shiftTrading *platformclientv2.Shifttradesettings) []interface{} {
	if shiftTrading == nil {
		return nil
	}
	activityCategoryRules := make([]interface{}, 0, len(*shiftTrading.ActivityCategoryRules))
	for _, rule := range *shiftTrading.ActivityCategoryRules {
		ruleMap := map[string]interface{}{
			"activity_category": *rule.ActivityCategory,
			"action":            *rule.Action,
		}
		setIfNotNil(ruleMap, "activity_code_id_replacement", rule.ActivityCodeIdReplacement)
		activityCategoryRules = append(activityCategoryRules, ruleMap)
	}
	externalActivityRules := make([]interface{}, 0, len(*shiftTrading.ExternalActivityRules))
	for _, rule := range *shiftTrading.ExternalActivityRules {
		ruleMap := map[string]interface{}{
			"external_activity_type": *rule.ExternalActivityType,
			"action":                 *rule.Action,
		}
		setIfNotNil(ruleMap, "activity_code_id_replacement", rule.ActivityCodeIdReplacement)
		externalActivityRules = append(externalActivityRules, ruleMap)
	}
	block := map[string]interface{}{
		"enabled":                                               *shiftTrading.Enabled,
		"auto_review":                                           *shiftTrading.AutoReview,
		"allow_direct_trades":                                   *shiftTrading.AllowDirectTrades,
		"min_hours_in_future":                                   *shiftTrading.MinHoursInFuture,
		"unequal_paid":                                          *shiftTrading.UnequalPaid,
		"one_sided":                                             *shiftTrading.OneSided,
		"weekly_min_paid_violations":                            *shiftTrading.WeeklyMinPaidViolations,
		"weekly_max_paid_violations":                            *shiftTrading.WeeklyMaxPaidViolations,
		"requires_matching_queues":                              *shiftTrading.RequiresMatchingQueues,
		"requires_matching_languages":                           *shiftTrading.RequiresMatchingLanguages,
		"requires_matching_skills":                              *shiftTrading.RequiresMatchingSkills,
		"requires_matching_planning_groups":                     *shiftTrading.RequiresMatchingPlanningGroups,
		"max_trade_span_weeks":                                  *shiftTrading.MaxTradeSpanWeeks,
		"max_trades_per_agent_per_week":                         *shiftTrading.MaxTradesPerAgentPerWeek,
		"min_minutes_between_shifts":                            *shiftTrading.MinMinutesBetweenShifts,
		"planning_period_min_paid_violations":                   *shiftTrading.PlanningPeriodMinPaidViolations,
		"planning_period_max_paid_violations":                   *shiftTrading.PlanningPeriodMaxPaidViolations,
		"min_minutes_between_shifts_violations":                 *shiftTrading.MinMinutesBetweenShiftsViolations,
		"external_activity_rule_ignores_activity_category_rule": *shiftTrading.ExternalActivityRuleIgnoresActivityCategoryRule,
		"activity_category_rules":                               activityCategoryRules,
		"external_activity_rules":                               externalActivityRules,
	}
	return []interface{}{block}
}

func flattenSettings(settings *platformclientv2.Managementunitsettingsresponse) []interface{} {
	if settings == nil {
		return nil
	}
	block := map[string]interface{}{}
	if adherence := flattenAdherence(settings.Adherence); adherence != nil {
		block["adherence"] = adherence
	}
	if timeOff := flattenTimeOff(settings.TimeOff); timeOff != nil {
		block["time_off"] = timeOff
	}
	if scheduling := flattenScheduling(settings.Scheduling); scheduling != nil {
		block["scheduling"] = scheduling
	}
	if shiftTrading := flattenShiftTrading(settings.ShiftTrading); shiftTrading != nil {
		block["shift_trading"] = shiftTrading
	}
	return []interface{}{block}
}

func setIfNotNil[T any](target map[string]interface{}, key string, value *T) {
	if value != nil {
		target[key] = *value
	}
}

func stringSet(values []string) *schema.Set {
	elements := make([]interface{}, len(values))
	for i, value := range values {
		elements[i] = value
	}
	return schema.NewSet(schema.HashString, elements)
}

// GenerateManagementUnitResource returns terraform config for a management unit, used in tests
func GenerateManagementUnitResource(resourceLabel string, name string, businessUnitId string, attrs ...string) string {
	return fmt.Sprintf(`resource "%s" "%s" {
	name             = "%s"
	business_unit_id = %s
	%s
}
`, ResourceType, resourceLabel, name, businessUnitId, strings.Join(attrs, "\n"))
}
