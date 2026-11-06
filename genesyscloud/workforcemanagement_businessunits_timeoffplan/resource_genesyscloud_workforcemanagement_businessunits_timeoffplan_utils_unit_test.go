package workforcemanagement_businessunits_timeoffplan

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/mypurecloud/platform-client-sdk-go/v200/platformclientv2"
)

func resourceGetter(t *testing.T, values map[string]interface{}) func(string) interface{} {
	return schema.TestResourceDataRaw(t, ResourceTimeOffPlan().Schema, values).Get
}

func scalarConfig(values map[string]cty.Value) cty.Value {
	defaults := map[string]cty.Value{
		"name":                          cty.StringVal("Plan"),
		"auto_approval_rule":            cty.NullVal(cty.String),
		"enabled":                       cty.NullVal(cty.Bool),
		"count_against_time_off_limits": cty.NullVal(cty.Bool),
		"auto_publish_approved_time_off_requests":   cty.NullVal(cty.Bool),
		"days_before_start_to_expire_from_waitlist": cty.NullVal(cty.Number),
	}
	for key, value := range values {
		defaults[key] = value
	}
	return cty.ObjectVal(defaults)
}

func overrideDay(date string, dateType string) platformclientv2.Overridedate {
	parsed, _ := time.Parse(dateLayout, date)
	return platformclientv2.Overridedate{Date: &parsed, VarType: &dateType}
}

func TestUnitSplitTimeOffPlanTerraformId(t *testing.T) {
	businessUnitId, planId := SplitTimeOffPlanTerraformId("bu-1/plan-2")
	if businessUnitId != "bu-1" || planId != "plan-2" {
		t.Errorf("expected bu-1 and plan-2, got %s and %s", businessUnitId, planId)
	}
}

func TestUnitModifyActivityCodeIdStateValue(t *testing.T) {
	if got := ModifyActivityCodeIdStateValue("bu-1/code-2"); got != "code-2" {
		t.Errorf("expected code-2, got %s", got)
	}
	if got := ModifyActivityCodeIdStateValue("TimeOff"); got != "TimeOff" {
		t.Errorf("a default activity code id has no prefix and must be kept, got %s", got)
	}
}

func TestUnitBuildCreateRequestOnlyContainsWhatIsConfigured(t *testing.T) {
	get := resourceGetter(t, map[string]interface{}{
		"activity_code_ids":  []interface{}{"bu-1/code-a", "TimeOff"},
		"override_dates":     []interface{}{map[string]interface{}{"date": "2026-12-25", "type": "Blocked"}},
		"hris_time_off_type": []interface{}{map[string]interface{}{"id": "pto", "name": "Paid time off"}},
		"business_unit_association": []interface{}{map[string]interface{}{
			"management_unit_ids": []interface{}{"mu-1", "mu-2"},
		}},
	})
	config := scalarConfig(map[string]cty.Value{"auto_approval_rule": cty.StringVal("CheckLimits"), "enabled": cty.False})

	request := buildCreateRequest(config, get)
	if *request.Name != "Plan" || *request.AutoApprovalRule != "CheckLimits" || request.Enabled == nil || *request.Enabled {
		t.Errorf("expected the configured scalars, including an explicit false, got %+v", request)
	}
	if request.DaysBeforeStartToExpireFromWaitlist != nil || request.CountAgainstTimeOffLimits != nil {
		t.Error("scalars that were not configured must not be sent")
	}
	if !reflect.DeepEqual(*request.ActivityCodeIds, []string{"TimeOff", "code-a"}) {
		t.Errorf("expected the business unit prefix to be stripped from activity code ids, got %v", *request.ActivityCodeIds)
	}
	if request.RestrictedActivityCodeIds != nil {
		t.Error("restricted activity codes that were not configured must not be sent")
	}
	if len(*request.OverrideDates) != 1 || *(*request.OverrideDates)[0].VarType != "Blocked" || *request.HrisTimeOffType.Id != "pto" {
		t.Errorf("unexpected override dates or hris type %+v %+v", request.OverrideDates, request.HrisTimeOffType)
	}
	if request.ManagementUnitAssociation != nil || len(*request.BusinessUnitAssociation.ManagementUnitIds) != 2 || request.BusinessUnitAssociation.StaffingGroupIds != nil {
		t.Errorf("unexpected associations %+v", request)
	}
}

func TestUnitBuildUpdateRequestOnlySendsWhatDiffers(t *testing.T) {
	get := resourceGetter(t, map[string]interface{}{
		"activity_code_ids":            []interface{}{"code-a"},
		"restricted_activity_code_ids": []interface{}{"code-r"},
		"override_dates":               []interface{}{map[string]interface{}{"date": "2026-12-25", "type": "Blocked"}},
	})
	current := &platformclientv2.Butimeoffplanresponse{
		ActivityCodeIds:         &[]string{"code-a"},
		AutoApprovalRule:        platformclientv2.String("CheckActivityCodes"),
		RestrictedActivityCodes: &platformclientv2.Activitycodesreference{Ids: &[]string{"code-other"}},
		OverrideDates:           &[]platformclientv2.Overridedate{overrideDay("2026-12-25", "Blocked")},
		Metadata:                &platformclientv2.Wfmversionedentitymetadata{Version: platformclientv2.Int(5)},
	}

	request := buildUpdateRequest(scalarConfig(nil), get, current)
	if request.ActivityCodeIds != nil || request.OverrideDates != nil {
		t.Errorf("collections that already match must not be sent, got %+v %+v", request.ActivityCodeIds, request.OverrideDates)
	}
	if request.RestrictedActivityCodeIds == nil || !reflect.DeepEqual(*request.RestrictedActivityCodeIds.Values, []string{"code-r"}) {
		t.Errorf("expected the changed restricted activity codes to be sent, got %+v", request.RestrictedActivityCodeIds)
	}
	if *request.Metadata.Version != 5 {
		t.Errorf("expected the current version, got %+v", request.Metadata)
	}
}

func TestUnitBuildUpdateRequestClearsCollectionsRemovedFromTheConfiguration(t *testing.T) {
	get := resourceGetter(t, map[string]interface{}{})
	current := &platformclientv2.Butimeoffplanresponse{
		ActivityCodeIds: &[]string{"code-a"},
		OverrideDates:   &[]platformclientv2.Overridedate{overrideDay("2026-12-25", "Blocked")},
		HrisTimeOffType: &platformclientv2.Hristimeofftype{Id: platformclientv2.String("pto")},
	}

	request := buildUpdateRequest(scalarConfig(nil), get, current)
	if request.ActivityCodeIds == nil || len(*request.ActivityCodeIds.Values) != 0 {
		t.Errorf("expected the activity codes to be cleared with an empty list, got %+v", request.ActivityCodeIds)
	}

	// Genesys Cloud rejects an empty list of override dates, and clears them when the wrapper has no values
	overrideBody, err := json.Marshal(request.OverrideDates)
	if err != nil || request.OverrideDates == nil || string(overrideBody) != `{}` {
		t.Errorf("expected the override dates to be cleared with a wrapper without values, got %s (%v)", overrideBody, err)
	}

	body, err := json.Marshal(request.HrisTimeOffType)
	if err != nil || string(body) != `{"value":null}` {
		t.Errorf("expected the hris time off type to be cleared with a null value, got %s (%v)", body, err)
	}
}

func TestUnitBuildUpdateRequestOnlySendsRestrictedActivityCodesForRulesThatCheckThem(t *testing.T) {
	get := resourceGetter(t, map[string]interface{}{"restricted_activity_code_ids": []interface{}{"code-r"}})
	current := &platformclientv2.Butimeoffplanresponse{AutoApprovalRule: platformclientv2.String("Never")}

	if request := buildUpdateRequest(scalarConfig(nil), get, current); request.RestrictedActivityCodeIds != nil {
		t.Errorf("restricted activity codes must not be sent with a rule that does not check them, got %+v", request.RestrictedActivityCodeIds)
	}

	changesRule := scalarConfig(map[string]cty.Value{"auto_approval_rule": cty.StringVal("CheckLimitsAndActivityCodes")})
	request := buildUpdateRequest(changesRule, get, current)
	if request.RestrictedActivityCodeIds == nil || !reflect.DeepEqual(*request.RestrictedActivityCodeIds.Values, []string{"code-r"}) {
		t.Errorf("expected the restricted activity codes to be sent with a rule that checks them, got %+v", request.RestrictedActivityCodeIds)
	}

	// Changing to a rule that does not check them leaves clearing them to Genesys Cloud
	empty := resourceGetter(t, map[string]interface{}{})
	currentWithCodes := &platformclientv2.Butimeoffplanresponse{
		AutoApprovalRule:        platformclientv2.String("CheckActivityCodes"),
		RestrictedActivityCodes: &platformclientv2.Activitycodesreference{Ids: &[]string{"code-r"}},
	}
	changesToNever := scalarConfig(map[string]cty.Value{"auto_approval_rule": cty.StringVal("Never")})
	if request := buildUpdateRequest(changesToNever, empty, currentWithCodes); request.RestrictedActivityCodeIds != nil {
		t.Errorf("clearing the restricted activity codes is left to Genesys Cloud, got %+v", request.RestrictedActivityCodeIds)
	}
}

func TestUnitValidateRuleSettings(t *testing.T) {
	cases := map[string]struct {
		rule       string
		restricted int
		counts     bool
		message    string
	}{
		"default rule":                   {"", 0, false, ""},
		"never with restricted":          {"Never", 1, false, "must not be set"},
		"check codes without restricted": {"CheckActivityCodes", 0, false, "must be set"},
		"check codes with restricted":    {"CheckActivityCodes", 2, false, ""},
		"check limits without counting":  {"CheckLimits", 0, false, "count_against_time_off_limits must be true"},
		"check limits counting":          {"CheckLimits", 0, true, ""},
		"both rules complete":            {"CheckLimitsAndActivityCodes", 1, true, ""},
		"both rules without counting":    {"CheckLimitsAndActivityCodes", 1, false, "count_against_time_off_limits must be true"},
		"always with restricted":         {"Always", 1, true, "must not be set"},
	}
	for name, test := range cases {
		err := validateRuleSettings(test.rule, test.restricted, test.counts)
		switch {
		case test.message == "" && err != nil:
			t.Errorf("%s: expected no error, got %v", name, err)
		case test.message != "" && (err == nil || !strings.Contains(err.Error(), test.message)):
			t.Errorf("%s: expected an error containing %q, got %v", name, test.message, err)
		}
	}
}

func TestUnitBuildUpdateRequestSwitchesAssociationKindWithASingleWrapper(t *testing.T) {
	staffingGroups := resourceGetter(t, map[string]interface{}{
		"business_unit_association": []interface{}{map[string]interface{}{"staffing_group_ids": []interface{}{"sg-1"}}},
	})
	currentManagementUnits := &platformclientv2.Butimeoffplanresponse{
		BusinessUnitAssociation: &platformclientv2.Timeoffplanbusinessunitassociation{ManagementUnits: &[]platformclientv2.Managementunitreference{{Id: platformclientv2.String("mu-1")}}},
	}
	request := buildUpdateRequest(scalarConfig(nil), staffingGroups, currentManagementUnits)
	if request.BusinessUnitAssociation == nil || request.BusinessUnitAssociation.ManagementUnitIds != nil ||
		!reflect.DeepEqual(*request.BusinessUnitAssociation.StaffingGroupIds.Values, []string{"sg-1"}) {
		t.Errorf("expected only the staffing groups wrapper, got %+v", request.BusinessUnitAssociation)
	}

	// Genesys Cloud ignores the management units wrapper when a staffing groups wrapper is there, even an empty one
	managementUnits := resourceGetter(t, map[string]interface{}{
		"business_unit_association": []interface{}{map[string]interface{}{"management_unit_ids": []interface{}{"mu-1"}}},
	})
	currentStaffingGroups := &platformclientv2.Butimeoffplanresponse{
		BusinessUnitAssociation: &platformclientv2.Timeoffplanbusinessunitassociation{StaffingGroups: &[]platformclientv2.Staffinggroupreference{{Id: platformclientv2.String("sg-1")}}},
	}
	request = buildUpdateRequest(scalarConfig(nil), managementUnits, currentStaffingGroups)
	if request.BusinessUnitAssociation == nil || request.BusinessUnitAssociation.StaffingGroupIds != nil ||
		!reflect.DeepEqual(*request.BusinessUnitAssociation.ManagementUnitIds.Values, []string{"mu-1"}) {
		t.Errorf("expected only the management units wrapper when switching to management units, got %+v", request.BusinessUnitAssociation)
	}
}

func TestUnitBuildUpdateRequestAssociations(t *testing.T) {
	get := resourceGetter(t, map[string]interface{}{
		"business_unit_association": []interface{}{map[string]interface{}{"management_unit_ids": []interface{}{"mu-1", "mu-3"}}},
	})
	current := &platformclientv2.Butimeoffplanresponse{
		BusinessUnitAssociation: &platformclientv2.Timeoffplanbusinessunitassociation{ManagementUnits: &[]platformclientv2.Managementunitreference{
			{Id: platformclientv2.String("mu-1")}, {Id: platformclientv2.String("mu-2")},
		}},
	}

	request := buildUpdateRequest(scalarConfig(nil), get, current)
	if request.BusinessUnitAssociation == nil || !reflect.DeepEqual(*request.BusinessUnitAssociation.ManagementUnitIds.Values, []string{"mu-1", "mu-3"}) {
		t.Errorf("expected the changed management units to be sent, got %+v", request.BusinessUnitAssociation)
	}
	if request.BusinessUnitAssociation.StaffingGroupIds != nil {
		t.Error("staffing groups that were not configured must not be sent")
	}

	same := resourceGetter(t, map[string]interface{}{
		"business_unit_association": []interface{}{map[string]interface{}{"management_unit_ids": []interface{}{"mu-1", "mu-2"}}},
	})
	if request := buildUpdateRequest(scalarConfig(nil), same, current); request.BusinessUnitAssociation != nil {
		t.Errorf("an association that did not change must not be sent, got %+v", request.BusinessUnitAssociation)
	}
}

func TestUnitFlattenAssociationsAreOmittedWhenEmpty(t *testing.T) {
	if got := flattenBusinessUnitAssociation(&platformclientv2.Timeoffplanbusinessunitassociation{}); len(got) != 0 {
		t.Errorf("an empty business unit association must not be in the state, got %v", got)
	}
	if got := flattenManagementUnitAssociation(nil); len(got) != 0 {
		t.Errorf("a missing management unit association must not be in the state, got %v", got)
	}

	flattened := flattenManagementUnitAssociation(&platformclientv2.Timeoffplanmanagementunitassociation{
		ManagementUnit: &platformclientv2.Managementunitreference{Id: platformclientv2.String("mu-1")},
		StaffingGroups: &[]platformclientv2.Staffinggroupreference{{Id: platformclientv2.String("sg-1")}},
	})
	entry := flattened[0].(map[string]interface{})
	if entry["management_unit_id"] != "mu-1" || entry["staffing_group_ids"].(*schema.Set).Len() != 1 {
		t.Errorf("unexpected management unit association %+v", entry)
	}
}

func TestUnitFlattenOverrideDatesAndHrisType(t *testing.T) {
	dates := flattenOverrideDates(&[]platformclientv2.Overridedate{overrideDay("2026-12-25", "Blocked"), overrideDay("2026-01-01", "ManualReview")})
	if dates.Len() != 2 {
		t.Fatalf("expected 2 override dates, got %d", dates.Len())
	}
	hris := flattenHrisType(&platformclientv2.Hristimeofftype{Id: platformclientv2.String("pto"), SecondaryId: platformclientv2.String("2")})
	entry := hris[0].(map[string]interface{})
	if entry["id"] != "pto" || entry["secondary_id"] != "2" {
		t.Errorf("unexpected hris type %+v", entry)
	}
	if got := flattenHrisType(nil); len(got) != 0 {
		t.Errorf("a missing hris type must not be in the state, got %v", got)
	}
}

func TestUnitValidateOverrideDatesAreUnique(t *testing.T) {
	if err := validateOverrideDatesAreUnique([]overrideDate{{date: "2026-12-24", dateTyp: "Blocked"}, {date: "2026-12-25", dateTyp: "Blocked"}}); err != nil {
		t.Errorf("expected different dates to be valid, got %v", err)
	}
	err := validateOverrideDatesAreUnique([]overrideDate{{date: "2026-12-25", dateTyp: "Blocked"}, {date: "2026-12-25", dateTyp: "ManualReview"}})
	if err == nil || !strings.Contains(err.Error(), "2026-12-25") {
		t.Errorf("expected a date listed twice, even with different types, to be rejected, got %v", err)
	}
}

func TestUnitValidateOverrideDatesWindow(t *testing.T) {
	today := time.Date(2026, 11, 6, 15, 30, 0, 0, time.UTC)

	if err := validateOverrideDatesWindow([]overrideDate{{date: "2026-08-05", dateTyp: "Blocked"}, {date: "2030-01-01", dateTyp: "Blocked"}}, today); err != nil {
		t.Errorf("a date exactly 93 days back and a far future date must be accepted, got %v", err)
	}

	err := validateOverrideDatesWindow([]overrideDate{{date: "2026-08-04", dateTyp: "Blocked"}}, today)
	if err == nil || !strings.Contains(err.Error(), "2026-08-04") || !strings.Contains(err.Error(), "remove the dates that have passed") {
		t.Errorf("a date 94 days back must be rejected with advice on what to do, got %v", err)
	}

	tooMany := make([]overrideDate, maxOverrideDates+1)
	for i := range tooMany {
		tooMany[i] = overrideDate{date: today.AddDate(0, 0, i).Format(dateLayout), dateTyp: "Blocked"}
	}
	if err := validateOverrideDatesWindow(tooMany, today); err == nil || !strings.Contains(err.Error(), "at most 550") {
		t.Errorf("more override dates than allowed must be rejected, got %v", err)
	}
}
