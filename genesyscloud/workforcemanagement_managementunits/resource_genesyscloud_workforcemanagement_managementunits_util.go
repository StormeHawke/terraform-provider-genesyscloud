package workforcemanagement_managementunits

import (
	"fmt"
	"strings"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/mypurecloud/platform-client-sdk-go/v200/platformclientv2"
	resourceExporter "github.com/mypurecloud/terraform-provider-genesyscloud/genesyscloud/resource_exporter"
	"github.com/mypurecloud/terraform-provider-genesyscloud/genesyscloud/util/wfm"
)

const (
	businessUnitResourceType = "genesyscloud_workforcemanagement_businessunits"
	activityCodeResourceType = "genesyscloud_workforcemanagement_businessunits_activitycode"
)

/*
Settings requests only include the fields that are present in the configuration. Many settings fields are optional and
computed, and a field the user did not configure must not be sent as a zero value, so presence is read from the raw
configuration instead of from the merged resource data.
*/

func buildAdherence(settings cty.Value) *platformclientv2.Adherencesettings {
	block := wfm.BlockOf(settings, "adherence")
	if !wfm.IsPresent(block) {
		return nil
	}
	adherence := &platformclientv2.Adherencesettings{
		SevereAlertThresholdMinutes:        wfm.IntOf(block, "severe_alert_threshold_minutes"),
		AdherenceTargetPercent:             wfm.IntOf(block, "adherence_target_percent"),
		AdherenceExceptionThresholdSeconds: wfm.IntOf(block, "adherence_exception_threshold_seconds"),
		NonOnQueueActivitiesEquivalent:     wfm.BoolOf(block, "non_on_queue_activities_equivalent"),
		TrackOnQueueActivity:               wfm.BoolOf(block, "track_on_queue_activity"),
	}
	if categories := wfm.StringSliceOf(block, "ignored_activity_categories", nil); categories != nil {
		adherence.IgnoredActivityCategories = &platformclientv2.Ignoredactivitycategories{Values: categories}
	}
	if codeIds := wfm.StringSliceOf(block, "ignored_activity_code_ids", func(id string) string { return wfm.ModifyActivityCodeIdStateValue(id) }); codeIds != nil {
		adherence.IgnoredActivityCodeIds = &platformclientv2.Ignoredactivitycodeids{Values: codeIds}
	}
	return adherence
}

func buildTimeOffRequest(settings cty.Value) *platformclientv2.Timeoffsettingsrequest {
	block := wfm.BlockOf(settings, "time_off")
	if !wfm.IsPresent(block) {
		return nil
	}
	timeOff := &platformclientv2.Timeoffsettingsrequest{
		SubmissionRangeEnforced:       wfm.BoolOf(block, "submission_range_enforced"),
		SubmissionRangeType:           wfm.StringOf(block, "submission_range_type"),
		SubmissionEarliestDaysFromNow: wfm.IntOf(block, "submission_earliest_days_from_now"),
		SubmissionLatestDaysFromNow:   wfm.IntOf(block, "submission_latest_days_from_now"),
	}
	if latestDate := wfm.DateOf(block, "submission_latest_date"); latestDate != nil {
		timeOff.SubmissionLatestDate = &platformclientv2.Valuewrapperlocaldate{Value: latestDate}
	}
	return timeOff
}

func buildShrinkageOverrides(scheduling cty.Value) *platformclientv2.Shrinkageoverrides {
	attr := wfm.AttrOf(scheduling, "shrinkage_overrides")
	if !wfm.IsPresent(attr) {
		return nil
	}
	if attr.LengthInt() == 0 {
		return &platformclientv2.Shrinkageoverrides{Clear: platformclientv2.Bool(true)}
	}
	values := make([]platformclientv2.Shrinkageoverride, 0, attr.LengthInt())
	for _, element := range attr.AsValueSlice() {
		values = append(values, platformclientv2.Shrinkageoverride{
			IntervalIndex:    wfm.IntOf(element, "interval_index"),
			ShrinkagePercent: wfm.FloatOf(element, "shrinkage_percent"),
		})
	}
	return &platformclientv2.Shrinkageoverrides{Values: &values}
}

func buildScheduling(settings cty.Value) *platformclientv2.Schedulingsettingsrequest {
	block := wfm.BlockOf(settings, "scheduling")
	if !wfm.IsPresent(block) {
		return nil
	}
	scheduling := &platformclientv2.Schedulingsettingsrequest{
		MaxOccupancyPercentForDeferredWork: wfm.IntOf(block, "max_occupancy_percent_for_deferred_work"),
		DefaultShrinkagePercent:            wfm.FloatOf(block, "default_shrinkage_percent"),
		ShrinkageOverrides:                 buildShrinkageOverrides(block),
		StartDayOfWeekend:                  wfm.StringOf(block, "start_day_of_weekend"),
	}
	// Only the configured planning period is sent. Switching between planning_period and monthly_planning_period does not clear the
	// other one: that needs the other value wrapper sent with a null value, which is not done here.
	if planningPeriod := wfm.BlockOf(block, "planning_period"); wfm.IsPresent(planningPeriod) {
		scheduling.PlanningPeriod = &platformclientv2.Valuewrapperplanningperiodsettings{
			Value: &platformclientv2.Planningperiodsettings{
				WeekCount: wfm.IntOf(planningPeriod, "week_count"),
				StartDate: wfm.DateOf(planningPeriod, "start_date"),
			},
		}
	}
	if monthlyPlanningPeriod := wfm.BlockOf(block, "monthly_planning_period"); wfm.IsPresent(monthlyPlanningPeriod) {
		scheduling.MonthlyPlanningPeriod = &platformclientv2.Valuewrappermonthlyplanningperiodsettings{
			Value: &platformclientv2.Monthlyplanningperiodsettings{StartDate: wfm.DateOf(monthlyPlanningPeriod, "start_date")},
		}
	}
	if scheduleVisibility := wfm.BlockOf(block, "schedule_visibility"); wfm.IsPresent(scheduleVisibility) {
		scheduling.ScheduleVisibility = &platformclientv2.Schedulevisibilitysettingsrequest{
			Enabled:     wfm.BoolOf(scheduleVisibility, "enabled"),
			FutureWeeks: wfm.IntOf(scheduleVisibility, "future_weeks"),
		}
	}
	return scheduling
}

func buildShiftTrading(settings cty.Value) *platformclientv2.Shifttradesettings {
	block := wfm.BlockOf(settings, "shift_trading")
	if !wfm.IsPresent(block) {
		return nil
	}
	shiftTrading := &platformclientv2.Shifttradesettings{
		Enabled:                           wfm.BoolOf(block, "enabled"),
		AutoReview:                        wfm.BoolOf(block, "auto_review"),
		AllowDirectTrades:                 wfm.BoolOf(block, "allow_direct_trades"),
		MinHoursInFuture:                  wfm.IntOf(block, "min_hours_in_future"),
		UnequalPaid:                       wfm.StringOf(block, "unequal_paid"),
		OneSided:                          wfm.StringOf(block, "one_sided"),
		WeeklyMinPaidViolations:           wfm.StringOf(block, "weekly_min_paid_violations"),
		WeeklyMaxPaidViolations:           wfm.StringOf(block, "weekly_max_paid_violations"),
		RequiresMatchingQueues:            wfm.BoolOf(block, "requires_matching_queues"),
		RequiresMatchingLanguages:         wfm.BoolOf(block, "requires_matching_languages"),
		RequiresMatchingSkills:            wfm.BoolOf(block, "requires_matching_skills"),
		RequiresMatchingPlanningGroups:    wfm.BoolOf(block, "requires_matching_planning_groups"),
		MaxTradeSpanWeeks:                 wfm.IntOf(block, "max_trade_span_weeks"),
		MaxTradesPerAgentPerWeek:          wfm.IntOf(block, "max_trades_per_agent_per_week"),
		MinMinutesBetweenShifts:           wfm.IntOf(block, "min_minutes_between_shifts"),
		PlanningPeriodMinPaidViolations:   wfm.StringOf(block, "planning_period_min_paid_violations"),
		PlanningPeriodMaxPaidViolations:   wfm.StringOf(block, "planning_period_max_paid_violations"),
		MinMinutesBetweenShiftsViolations: wfm.StringOf(block, "min_minutes_between_shifts_violations"),

		ExternalActivityRuleIgnoresActivityCategoryRule: wfm.BoolOf(block, "external_activity_rule_ignores_activity_category_rule"),
	}
	if rules := wfm.AttrOf(block, "activity_category_rules"); wfm.IsPresent(rules) {
		activityRules := make([]platformclientv2.Shifttradeactivityrule, 0, rules.LengthInt())
		for _, rule := range rules.AsValueSlice() {
			activityRules = append(activityRules, platformclientv2.Shifttradeactivityrule{
				ActivityCategory:          wfm.StringOf(rule, "activity_category"),
				Action:                    wfm.StringOf(rule, "action"),
				ActivityCodeIdReplacement: activityCodeIdReplacement(rule),
			})
		}
		shiftTrading.ActivityCategoryRules = &activityRules
	}
	if rules := wfm.AttrOf(block, "external_activity_rules"); wfm.IsPresent(rules) {
		externalRules := make([]platformclientv2.Shifttradeexternalactivityrule, 0, rules.LengthInt())
		for _, rule := range rules.AsValueSlice() {
			externalRules = append(externalRules, platformclientv2.Shifttradeexternalactivityrule{
				ExternalActivityType:      wfm.StringOf(rule, "external_activity_type"),
				Action:                    wfm.StringOf(rule, "action"),
				ActivityCodeIdReplacement: activityCodeIdReplacement(rule),
			})
		}
		shiftTrading.ExternalActivityRules = &externalRules
	}
	return shiftTrading
}

// activityCodeIdReplacement reads the id of the activity code that replaces activities in a shift trade rule, without the business unit
// that comes in front of it when it is the id of an activity code resource, because a management unit only uses activity codes of its business unit
func activityCodeIdReplacement(rule cty.Value) *string {
	replacement := nonEmpty(wfm.StringOf(rule, "activity_code_id_replacement"))
	if replacement == nil {
		return nil
	}
	activityCodeId := wfm.ModifyActivityCodeIdStateValue(*replacement)
	return &activityCodeId
}

func nonEmpty(value *string) *string {
	if value == nil || *value == "" {
		return nil
	}
	return value
}

// rawSettings returns the settings block from the raw configuration, or a null value if it was not configured
func rawSettings(d *schema.ResourceData) cty.Value {
	return wfm.BlockOf(d.GetRawConfig(), "settings")
}

func buildCreateManagementUnitRequest(d *schema.ResourceData) *platformclientv2.Createmanagementunitapirequest {
	request := &platformclientv2.Createmanagementunitapirequest{
		Name:           platformclientv2.String(d.Get("name").(string)),
		BusinessUnitId: platformclientv2.String(d.Get("business_unit_id").(string)),
		DivisionId:     nonEmpty(wfm.StringOf(d.GetRawConfig(), "division_id")),
	}

	if settings := rawSettings(d); wfm.IsPresent(settings) {
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
		DivisionId: nonEmpty(wfm.StringOf(d.GetRawConfig(), "division_id")),
	}

	settings := &platformclientv2.Managementunitsettingsrequest{}
	if rawSettings := rawSettings(d); wfm.IsPresent(rawSettings) {
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
		"ignored_activity_categories":           wfm.StringSet(*adherence.IgnoredActivityCategories.Values),
		"ignored_activity_code_ids":             wfm.StringSet(*adherence.IgnoredActivityCodeIds.Values),
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
		block["submission_latest_date"] = timeOff.SubmissionLatestDate.Format(wfm.DateLayout)
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
			"start_date": scheduling.PlanningPeriod.StartDate.Format(wfm.DateLayout),
		}}
	}
	if scheduling.MonthlyPlanningPeriod != nil {
		block["monthly_planning_period"] = []interface{}{map[string]interface{}{
			"start_date": scheduling.MonthlyPlanningPeriod.StartDate.Format(wfm.DateLayout),
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

// GenerateManagementUnitResource returns terraform config for a management unit, used in tests
func GenerateManagementUnitResource(resourceLabel string, name string, businessUnitId string, attrs ...string) string {
	return fmt.Sprintf(`resource "%s" "%s" {
	name             = "%s"
	business_unit_id = %s
	%s
}
`, ResourceType, resourceLabel, name, businessUnitId, strings.Join(attrs, "\n"))
}

// resolveActivityCodeReferences turns the activity code ids of a management unit into references to the exported activity code resources.
// The management unit api only knows an activity code by its id, because it belongs to the business unit of the management unit, but the
// exporter keys activity code resources by <businessUnitId>/<activityCodeId>, because the same id can appear in different business units.
// So the key is built from the business unit of the management unit and the activity code id. An id that matches no exported resource stays as it is.
func resolveActivityCodeReferences(configMap map[string]interface{}, exporters map[string]*resourceExporter.ResourceExporter, _ string) error {
	activityCodes := exporters[activityCodeResourceType]
	if activityCodes == nil || activityCodes.SanitizedResourceMap == nil {
		return nil
	}
	businessUnitId := exportedBusinessUnitId(configMap, exporters)
	if businessUnitId == "" {
		return nil
	}

	reference := func(activityCodeId string) string {
		if meta := activityCodes.SanitizedResourceMap[businessUnitId+"/"+activityCodeId]; meta != nil && meta.BlockLabel != "" {
			return fmt.Sprintf("${%s.%s.id}", activityCodeResourceType, meta.BlockLabel)
		}
		return activityCodeId
	}

	for _, settings := range mapsOf(configMap["settings"]) {
		for _, adherence := range mapsOf(settings["adherence"]) {
			if ids, ok := adherence["ignored_activity_code_ids"].([]interface{}); ok {
				for i, id := range ids {
					if activityCodeId, ok := id.(string); ok {
						ids[i] = reference(activityCodeId)
					}
				}
			}
		}
		for _, shiftTrading := range mapsOf(settings["shift_trading"]) {
			for _, key := range []string{"activity_category_rules", "external_activity_rules"} {
				for _, rule := range mapsOf(shiftTrading[key]) {
					if activityCodeId, ok := rule["activity_code_id_replacement"].(string); ok {
						rule["activity_code_id_replacement"] = reference(activityCodeId)
					}
				}
			}
		}
	}
	return nil
}

// exportedBusinessUnitId returns the id of the business unit of the exported management unit. The exporter may already have replaced it with
// a reference to the exported business unit, in which case the id is found through the label of that reference.
func exportedBusinessUnitId(configMap map[string]interface{}, exporters map[string]*resourceExporter.ResourceExporter) string {
	value, ok := configMap["business_unit_id"].(string)
	if !ok || value == "" {
		return ""
	}
	prefix := "${" + businessUnitResourceType + "."
	if !strings.HasPrefix(value, prefix) || !strings.HasSuffix(value, ".id}") {
		if strings.HasPrefix(value, "${") {
			return ""
		}
		return value
	}
	label := strings.TrimSuffix(strings.TrimPrefix(value, prefix), ".id}")
	businessUnits := exporters[businessUnitResourceType]
	if businessUnits == nil {
		return ""
	}
	for id, meta := range businessUnits.SanitizedResourceMap {
		if meta != nil && meta.BlockLabel == label {
			return id
		}
	}
	return ""
}

// mapsOf returns the maps in an exported list of blocks
func mapsOf(value interface{}) []map[string]interface{} {
	list, ok := value.([]interface{})
	if !ok {
		return nil
	}
	maps := make([]map[string]interface{}, 0, len(list))
	for _, item := range list {
		if entry, ok := item.(map[string]interface{}); ok {
			maps = append(maps, entry)
		}
	}
	return maps
}
