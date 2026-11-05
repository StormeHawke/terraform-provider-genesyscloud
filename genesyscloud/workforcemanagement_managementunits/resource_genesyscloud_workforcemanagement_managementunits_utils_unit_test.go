package workforcemanagement_managementunits

import (
	"testing"
	"time"

	"github.com/hashicorp/go-cty/cty"
	"github.com/mypurecloud/platform-client-sdk-go/v200/platformclientv2"
)

func TestUnitModifyActivityCodeIdStateValue(t *testing.T) {
	if got := ModifyActivityCodeIdStateValue("bu-1/code-2"); got != "code-2" {
		t.Errorf("expected code-2, got %s", got)
	}
	if got := ModifyActivityCodeIdStateValue("code-2"); got != "code-2" {
		t.Errorf("expected code-2, got %s", got)
	}
}

func settingsWith(blockName string, block cty.Value) cty.Value {
	return cty.ObjectVal(map[string]cty.Value{blockName: cty.ListVal([]cty.Value{block})})
}

func TestUnitBuildAdherenceOnlyIncludesConfiguredFields(t *testing.T) {
	block := cty.ObjectVal(map[string]cty.Value{
		"adherence_target_percent":       cty.NumberIntVal(85),
		"severe_alert_threshold_minutes": cty.NullVal(cty.Number),
		"track_on_queue_activity":        cty.False,
		"ignored_activity_categories":    cty.SetVal([]cty.Value{cty.StringVal("Break"), cty.StringVal("Meal")}),
		"ignored_activity_code_ids":      cty.SetVal([]cty.Value{cty.StringVal("bu-1/code-2")}),
	})

	adherence := buildAdherence(settingsWith("adherence", block))
	if adherence == nil {
		t.Fatal("expected adherence settings")
	}
	if adherence.AdherenceTargetPercent == nil || *adherence.AdherenceTargetPercent != 85 {
		t.Errorf("expected adherence target percent 85, got %v", adherence.AdherenceTargetPercent)
	}
	if adherence.SevereAlertThresholdMinutes != nil {
		t.Errorf("an unset field must not be sent, got %d", *adherence.SevereAlertThresholdMinutes)
	}
	if adherence.TrackOnQueueActivity == nil || *adherence.TrackOnQueueActivity {
		t.Errorf("an explicit false must be sent, got %v", adherence.TrackOnQueueActivity)
	}
	if adherence.IgnoredActivityCategories == nil || len(*adherence.IgnoredActivityCategories.Values) != 2 {
		t.Errorf("expected 2 ignored categories, got %v", adherence.IgnoredActivityCategories)
	}
	if adherence.IgnoredActivityCodeIds == nil || (*adherence.IgnoredActivityCodeIds.Values)[0] != "code-2" {
		t.Errorf("expected the business unit prefix to be stripped from activity code ids, got %v", adherence.IgnoredActivityCodeIds)
	}
}

func TestUnitBuildAdherenceMissingBlock(t *testing.T) {
	if got := buildAdherence(cty.ObjectVal(map[string]cty.Value{"adherence": cty.ListValEmpty(cty.DynamicPseudoType)})); got != nil {
		t.Errorf("expected nil for an unconfigured block, got %v", got)
	}
	if got := buildAdherence(cty.NullVal(cty.DynamicPseudoType)); got != nil {
		t.Errorf("expected nil for null settings, got %v", got)
	}
}

func TestUnitBuildSchedulingPlanningPeriodAndOverrides(t *testing.T) {
	block := cty.ObjectVal(map[string]cty.Value{
		"default_shrinkage_percent": cty.NumberFloatVal(12.5),
		"planning_period": cty.ListVal([]cty.Value{cty.ObjectVal(map[string]cty.Value{
			"week_count": cty.NumberIntVal(4),
			"start_date": cty.StringVal("2026-11-01"),
		})}),
		"shrinkage_overrides": cty.SetVal([]cty.Value{cty.ObjectVal(map[string]cty.Value{
			"interval_index":    cty.NumberIntVal(3),
			"shrinkage_percent": cty.NumberFloatVal(20),
		})}),
	})

	scheduling := buildScheduling(settingsWith("scheduling", block))
	if scheduling == nil {
		t.Fatal("expected scheduling settings")
	}
	if scheduling.DefaultShrinkagePercent == nil || *scheduling.DefaultShrinkagePercent != 12.5 {
		t.Errorf("expected default shrinkage 12.5, got %v", scheduling.DefaultShrinkagePercent)
	}
	if scheduling.MonthlyPlanningPeriod != nil {
		t.Error("monthly planning period was not configured and must not be sent")
	}
	expectedStart := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	if scheduling.PlanningPeriod == nil || scheduling.PlanningPeriod.Value == nil ||
		*scheduling.PlanningPeriod.Value.WeekCount != 4 || !scheduling.PlanningPeriod.Value.StartDate.Equal(expectedStart) {
		t.Errorf("unexpected planning period %+v", scheduling.PlanningPeriod)
	}
	if scheduling.ShrinkageOverrides == nil || scheduling.ShrinkageOverrides.Values == nil || len(*scheduling.ShrinkageOverrides.Values) != 1 {
		t.Errorf("expected one shrinkage override, got %+v", scheduling.ShrinkageOverrides)
	}
}

func TestUnitBuildShrinkageOverridesEmptySetClears(t *testing.T) {
	scheduling := cty.ObjectVal(map[string]cty.Value{"shrinkage_overrides": cty.SetValEmpty(cty.DynamicPseudoType)})
	overrides := buildShrinkageOverrides(scheduling)
	if overrides == nil || overrides.Clear == nil || !*overrides.Clear {
		t.Errorf("expected an empty configured set to clear the overrides, got %+v", overrides)
	}
}

func TestUnitBuildShiftTradingRules(t *testing.T) {
	block := cty.ObjectVal(map[string]cty.Value{
		"enabled": cty.True,
		"activity_category_rules": cty.ListVal([]cty.Value{cty.ObjectVal(map[string]cty.Value{
			"activity_category":            cty.StringVal("Break"),
			"action":                       cty.StringVal("Replace"),
			"activity_code_id_replacement": cty.StringVal("default-code"),
		})}),
		"external_activity_rules": cty.ListVal([]cty.Value{cty.ObjectVal(map[string]cty.Value{
			"external_activity_type":       cty.StringVal("Coaching"),
			"action":                       cty.StringVal("DoNotAllowTrade"),
			"activity_code_id_replacement": cty.StringVal(""),
		})}),
	})

	shiftTrading := buildShiftTrading(settingsWith("shift_trading", block))
	if shiftTrading == nil || shiftTrading.Enabled == nil || !*shiftTrading.Enabled {
		t.Fatalf("expected enabled shift trading, got %+v", shiftTrading)
	}
	if rule := (*shiftTrading.ActivityCategoryRules)[0]; *rule.ActivityCodeIdReplacement != "default-code" {
		t.Errorf("unexpected activity category rule %+v", rule)
	}
	if rule := (*shiftTrading.ExternalActivityRules)[0]; rule.ActivityCodeIdReplacement != nil {
		t.Errorf("an empty replacement must not be sent, got %s", *rule.ActivityCodeIdReplacement)
	}
}

func TestUnitFlattenTimeOffFormatsDate(t *testing.T) {
	latest := time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC)
	flattened := flattenTimeOff(&platformclientv2.Timeoffsettingsresponse{SubmissionLatestDate: &latest})
	if got := flattened[0].(map[string]interface{})["submission_latest_date"]; got != "2026-12-31" {
		t.Errorf("expected 2026-12-31, got %v", got)
	}
}
