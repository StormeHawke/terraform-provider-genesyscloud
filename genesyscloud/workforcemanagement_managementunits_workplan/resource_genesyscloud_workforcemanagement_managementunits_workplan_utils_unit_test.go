package workforcemanagement_managementunits_workplan

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/mypurecloud/platform-client-sdk-go/v200/platformclientv2"
)

func TestUnitSplitWorkPlanTerraformId(t *testing.T) {
	managementUnitId, workPlanId := SplitWorkPlanTerraformId("mu-1/plan-2")
	if managementUnitId != "mu-1" || workPlanId != "plan-2" {
		t.Errorf("expected mu-1 and plan-2, got %s and %s", managementUnitId, workPlanId)
	}
}

func TestUnitModifyActivityCodeIdStateValue(t *testing.T) {
	if got := ModifyActivityCodeIdStateValue("bu-1/code-2"); got != "code-2" {
		t.Errorf("expected code-2, got %s", got)
	}
}

func hasField(sdkType interface{}, key string) bool {
	return reflect.ValueOf(sdkType).Elem().FieldByName(snakeToPascal(key)).IsValid()
}

// Every attribute that is copied by name must exist on every sdk type it is copied to, or it would be silently dropped
func TestUnitScalarKeysExistOnTheSdkTypes(t *testing.T) {
	checks := []struct {
		keys     []string
		sdkTypes []interface{}
	}{
		{workPlanScalarKeys, []interface{}{&platformclientv2.Createworkplan{}, &platformclientv2.Workplan{}, &platformclientv2.Workplanvalidationrequest{}}},
		{shiftScalarKeys, []interface{}{&platformclientv2.Createworkplanshift{}, &platformclientv2.Workplanshift{}}},
		{activityScalarKeys, []interface{}{&platformclientv2.Createworkplanactivity{}, &platformclientv2.Workplanactivity{}}},
		{planningPeriodConstraintKeys, []interface{}{&platformclientv2.Planningperiodshiftconstraints{}}},
	}
	for _, check := range checks {
		for _, key := range check.keys {
			for _, sdkType := range check.sdkTypes {
				if !hasField(sdkType, key) {
					t.Errorf("%T has no field for attribute %q", sdkType, key)
				}
			}
		}
	}
}

// Every attribute of the schema must be handled, so adding an attribute without wiring it up fails here
func TestUnitEverySchemaAttributeIsHandled(t *testing.T) {
	covered := func(keys []string, special ...string) map[string]bool {
		set := map[string]bool{}
		for _, key := range append(append([]string{}, keys...), special...) {
			set[key] = true
		}
		return set
	}
	checks := map[string]struct {
		attributes map[string]*schema.Schema
		handled    map[string]bool
	}{
		"work plan": {ResourceWorkPlan().Schema, covered(workPlanScalarKeys, "management_unit_id", "valid", "optional_days", "shift_start_variances", "shifts", "agent_ids")},
		"shift":     {shiftResource.Schema, covered(shiftScalarKeys, "days", "planning_period_constraints", "activities")},
		"activity":  {activityResource.Schema, covered(activityScalarKeys)},
	}
	for name, check := range checks {
		for key := range check.attributes {
			if !check.handled[key] {
				t.Errorf("%s attribute %q is not handled by the request and read code", name, key)
			}
		}
		for key := range check.handled {
			if _, exists := check.attributes[key]; !exists && name != "work plan" {
				t.Errorf("%s handles %q, which is not in the schema", name, key)
			}
		}
	}
}

func TestUnitAssignScalarsOnlySetsConfiguredValues(t *testing.T) {
	config := cty.ObjectVal(map[string]cty.Value{
		"name":                        cty.StringVal("Plan"),
		"enabled":                     cty.False,
		"maximum_days":                cty.NumberIntVal(5),
		"weekly_exact_paid_minutes":   cty.NullVal(cty.Number),
		"shift_start_variance_period": cty.StringVal("Weekly"),
	})
	request := &platformclientv2.Createworkplan{}
	assignScalars(request, config, workPlanScalarKeys)

	if request.Name == nil || *request.Name != "Plan" || request.MaximumDays == nil || *request.MaximumDays != 5 {
		t.Errorf("expected the configured values, got %+v", request)
	}
	if request.Enabled == nil || *request.Enabled {
		t.Error("an explicit false must be sent")
	}
	if request.WeeklyExactPaidMinutes != nil || request.MaximumConsecutiveWorkingDays != nil {
		t.Error("values that were not configured must not be sent")
	}
}

func shift(name string, id string, activities ...platformclientv2.Workplanactivity) platformclientv2.Workplanshift {
	result := platformclientv2.Workplanshift{Name: platformclientv2.String(name)}
	if id != "" {
		result.Id = platformclientv2.String(id)
	}
	if activities != nil {
		result.Activities = &activities
	}
	return result
}

func activity(id string, code string, length int) platformclientv2.Workplanactivity {
	result := platformclientv2.Workplanactivity{ActivityCodeId: platformclientv2.String(code), LengthMinutes: platformclientv2.Int(length)}
	if id != "" {
		result.Id = platformclientv2.String(id)
	}
	return result
}

func TestUnitBuildShiftUpdatesMatchesByNameAndAddsAndDeletes(t *testing.T) {
	existing := []platformclientv2.Workplanshift{shift("Early", "0"), shift("Late", "1"), shift("Old", "2")}
	configured := []platformclientv2.Workplanshift{shift("Late", ""), shift("New", ""), shift("Early", "")}

	updates, err := buildShiftUpdates(existing, configured)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	byName := map[string]platformclientv2.Workplanshift{}
	var deleted []string
	for _, update := range updates {
		if update.Delete != nil && *update.Delete {
			deleted = append(deleted, *update.Id)
			continue
		}
		byName[*update.Name] = update
	}
	if byName["Early"].Id == nil || *byName["Early"].Id != "0" || byName["Late"].Id == nil || *byName["Late"].Id != "1" {
		t.Errorf("existing shifts must be edited by id, got %+v", byName)
	}
	if byName["New"].Id != nil {
		t.Error("a shift with a new name must be added without an id")
	}
	if !reflect.DeepEqual(deleted, []string{"2"}) {
		t.Errorf("expected the shift that is no longer configured to be deleted, got %v", deleted)
	}
}

func TestUnitBuildShiftUpdatesLeavesActivitiesAloneWhenTheyMatch(t *testing.T) {
	existing := []platformclientv2.Workplanshift{shift("Day", "0", activity("a", "break", 15), activity("b", "meal", 30))}
	// configured in a different order than they are stored, and without the values the api filled in
	configured := []platformclientv2.Workplanshift{shift("Day", "", activity("", "meal", 30), activity("", "break", 15))}

	updates, err := buildShiftUpdates(existing, configured)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(updates) != 1 || updates[0].Activities != nil {
		t.Errorf("activities that already match must not be sent, got %+v", updates[0].Activities)
	}
}

func TestUnitBuildShiftUpdatesReplacesActivitiesThatDiffer(t *testing.T) {
	existing := []platformclientv2.Workplanshift{shift("Day", "0", activity("a", "break", 15))}
	configured := []platformclientv2.Workplanshift{shift("Day", "", activity("", "break", 20))}

	updates, err := buildShiftUpdates(existing, configured)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	activities := *updates[0].Activities
	if len(activities) != 2 || activities[0].Delete == nil || !*activities[0].Delete || *activities[0].Id != "a" {
		t.Errorf("expected the existing activity to be deleted first, got %+v", activities)
	}
	if activities[1].Id != nil || *activities[1].LengthMinutes != 20 {
		t.Errorf("expected the configured activity to be added without an id, got %+v", activities[1])
	}
}

func TestUnitBuildShiftUpdatesClearsActivitiesWhenConfiguredEmpty(t *testing.T) {
	existing := []platformclientv2.Workplanshift{shift("Day", "0", activity("a", "break", 15))}
	configured := []platformclientv2.Workplanshift{shift("Day", "", []platformclientv2.Workplanactivity{}...)}
	empty := []platformclientv2.Workplanactivity{}
	configured[0].Activities = &empty

	updates, _ := buildShiftUpdates(existing, configured)
	if updates[0].Activities == nil || len(*updates[0].Activities) != 1 || !*(*updates[0].Activities)[0].Delete {
		t.Errorf("expected the only activity to be deleted, got %+v", updates[0].Activities)
	}
}

func TestUnitBuildShiftUpdatesRejectsAmbiguousExistingNames(t *testing.T) {
	existing := []platformclientv2.Workplanshift{shift("Day", "0"), shift("Day", "1")}
	_, err := buildShiftUpdates(existing, []platformclientv2.Workplanshift{shift("Day", "")})
	if err == nil || !strings.Contains(err.Error(), "cannot be told apart") {
		t.Errorf("expected an error about duplicate existing names, got %v", err)
	}
}

func TestUnitBuildAgentUpdates(t *testing.T) {
	updates := buildAgentUpdates([]string{"a", "b"}, []string{"b", "c"})
	if len(updates) != 2 || *updates[0].Id != "c" || updates[0].Delete != nil || *updates[1].Id != "a" || !*updates[1].Delete {
		t.Errorf("expected c assigned and a unassigned, got %+v", updates)
	}
	if got := buildAgentUpdates([]string{"a"}, []string{"a"}); len(got) != 0 {
		t.Errorf("expected no updates when the agents already match, got %+v", got)
	}
}

func TestUnitBuildUpdateWorkPlanRequestLeavesUnconfiguredPartsAlone(t *testing.T) {
	current := &platformclientv2.Workplan{
		Shifts:   &[]platformclientv2.Workplanshift{shift("Day", "0")},
		Agents:   &[]platformclientv2.Deletableuserreference{{Id: platformclientv2.String("a")}},
		Metadata: &platformclientv2.Wfmversionedentitymetadata{Version: platformclientv2.Int(7)},
	}
	config := cty.ObjectVal(map[string]cty.Value{
		"name":      cty.StringVal("Renamed"),
		"shifts":    cty.NullVal(cty.DynamicPseudoType),
		"agent_ids": cty.NullVal(cty.DynamicPseudoType),
	})

	request, err := buildUpdateWorkPlanRequest(config, current)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if request.Shifts != nil || request.Agents != nil {
		t.Errorf("shifts and agents that are not configured must not be sent, got %+v %+v", request.Shifts, request.Agents)
	}
	if *request.Name != "Renamed" || request.Metadata == nil || *request.Metadata.Version != 7 {
		t.Errorf("expected the name and the current version, got %+v", request)
	}
}

func TestUnitBuildCreateWorkPlanRequestConvertsShifts(t *testing.T) {
	config := cty.ObjectVal(map[string]cty.Value{
		"name": cty.StringVal("Plan"),
		"shifts": cty.ListVal([]cty.Value{cty.ObjectVal(map[string]cty.Value{
			"name": cty.StringVal("Day"),
			"days": cty.SetVal([]cty.Value{cty.StringVal("Monday"), cty.StringVal("Tuesday")}),
			"planning_period_constraints": cty.ListVal([]cty.Value{cty.ObjectVal(map[string]cty.Value{
				"enabled": cty.True, "minimum_count": cty.NumberIntVal(2),
			})}),
			"activities": cty.ListVal([]cty.Value{cty.ObjectVal(map[string]cty.Value{
				"activity_code_id": cty.StringVal("bu-1/break-code"),
				"length_minutes":   cty.NumberIntVal(15),
			})}),
		})}),
		"agent_ids": cty.SetVal([]cty.Value{cty.StringVal("agent-1")}),
	})

	request := buildCreateWorkPlanRequest(config)
	created := (*request.Shifts)[0]
	if *created.Name != "Day" || len(*created.Days.Values) != 2 {
		t.Errorf("unexpected shift %+v", created)
	}
	if *created.PlanningPeriodConstraints.MinimumCount != 2 {
		t.Errorf("unexpected planning period constraints %+v", created.PlanningPeriodConstraints)
	}
	if code := *(*created.Activities)[0].ActivityCodeId; code != "break-code" {
		t.Errorf("expected the business unit prefix to be stripped from the activity code id, got %s", code)
	}
	if len(*request.Agents) != 1 || *(*request.Agents)[0].Id != "agent-1" {
		t.Errorf("unexpected agents %+v", request.Agents)
	}
}

func TestUnitFlattenShiftsKeepsThePreviousOrder(t *testing.T) {
	api := []platformclientv2.Workplanshift{shift("Alpha", "0"), shift("Zulu", "1"), shift("Mike", "2"), shift("Brand new", "3")}
	prior := []interface{}{
		map[string]interface{}{"name": "Zulu"},
		map[string]interface{}{"name": "Alpha"},
		map[string]interface{}{"name": "Mike"},
	}

	flattened := flattenShifts(&api, prior)
	var names []string
	for _, item := range flattened {
		names = append(names, item.(map[string]interface{})["name"].(string))
	}
	if !reflect.DeepEqual(names, []string{"Zulu", "Alpha", "Mike", "Brand new"}) {
		t.Errorf("expected the previous order followed by new shifts, got %v", names)
	}
}

func TestUnitFlattenShiftsSortsByNameWithoutPriorState(t *testing.T) {
	api := []platformclientv2.Workplanshift{shift("Zulu", "1"), shift("Alpha", "0")}
	flattened := flattenShifts(&api, nil)
	names := []string{flattened[0].(map[string]interface{})["name"].(string), flattened[1].(map[string]interface{})["name"].(string)}
	if !sort.StringsAreSorted(names) {
		t.Errorf("expected a stable order when importing, got %v", names)
	}
}

func TestUnitFlattenActivitiesKeepsThePreviousOrder(t *testing.T) {
	api := []platformclientv2.Workplanactivity{activity("1", "meal", 30), activity("2", "break", 15)}
	prior := []map[string]interface{}{
		{"activity_code_id": "break", "description": "", "length_minutes": 15},
		{"activity_code_id": "meal", "description": "", "length_minutes": 30},
	}

	flattened := flattenActivities(&api, prior)
	if flattened[0].(map[string]interface{})["activity_code_id"] != "break" {
		t.Errorf("expected break first, got %+v", flattened)
	}
}

func TestUnitFormatValidationMessages(t *testing.T) {
	response := &platformclientv2.Validateworkplanresponse{Messages: &platformclientv2.Validateworkplanmessages{
		ViolationMessages: &[]platformclientv2.Workplanconfigurationviolationmessage{{
			Severity: platformclientv2.String("Error"),
			VarType:  platformclientv2.String("DailyShiftHasNoDaysSelected"),
			Arguments: &[]platformclientv2.Workplanvalidationmessageargument{
				{VarType: platformclientv2.String("ShiftName"), Value: platformclientv2.String("Day")},
			},
		}},
	}}

	got := formatValidationMessages(response)
	if got != "- Error: DailyShiftHasNoDaysSelected (ShiftName=Day)" {
		t.Errorf("unexpected message %q", got)
	}
	if formatValidationMessages(nil) != "" || formatValidationMessages(&platformclientv2.Validateworkplanresponse{}) != "" {
		t.Error("expected no text when there are no messages")
	}
}

func stubProxy() (*workPlanProxy, *[]string) {
	calls := &[]string{}
	return &workPlanProxy{}, calls
}

func TestUnitValidationDetailsOnlyAskedAfterABadRequest(t *testing.T) {
	asked := 0
	proxy, _ := stubProxy()
	proxy.validateWorkPlanAttr = func(ctx context.Context, p *workPlanProxy, managementUnitId string, workPlanId string, workPlan *platformclientv2.Workplanvalidationrequest) (*platformclientv2.Validateworkplanresponse, *platformclientv2.APIResponse, error) {
		asked++
		return &platformclientv2.Validateworkplanresponse{Messages: &platformclientv2.Validateworkplanmessages{
			ViolationMessages: &[]platformclientv2.Workplanconfigurationviolationmessage{{Severity: platformclientv2.String("Error"), VarType: platformclientv2.String("NoShifts")}},
		}}, nil, nil
	}
	request := func() *platformclientv2.Workplan {
		return &platformclientv2.Workplan{Name: platformclientv2.String("Plan")}
	}

	if got := validationDetails(context.Background(), proxy, "mu", "plan", &platformclientv2.APIResponse{StatusCode: http.StatusInternalServerError}, request); got != "" || asked != 0 {
		t.Errorf("a server error must not trigger validation, got %q after %d call(s)", got, asked)
	}
	got := validationDetails(context.Background(), proxy, "mu", "plan", &platformclientv2.APIResponse{StatusCode: http.StatusBadRequest}, request)
	if asked != 1 || !strings.Contains(got, "Validation messages:") || !strings.Contains(got, "NoShifts") {
		t.Errorf("expected the validation messages after a bad request, got %q", got)
	}
}

func TestUnitValidationDetailsIgnoresAFailedValidationCall(t *testing.T) {
	proxy, _ := stubProxy()
	proxy.validateWorkPlanAttr = func(ctx context.Context, p *workPlanProxy, managementUnitId string, workPlanId string, workPlan *platformclientv2.Workplanvalidationrequest) (*platformclientv2.Validateworkplanresponse, *platformclientv2.APIResponse, error) {
		return nil, nil, fmt.Errorf("validate is down")
	}
	got := validationDetails(context.Background(), proxy, "mu", newWorkPlanId, &platformclientv2.APIResponse{StatusCode: http.StatusBadRequest},
		func() *platformclientv2.Workplan { return &platformclientv2.Workplan{} })
	if got != "" {
		t.Errorf("the original error must stand when the messages cannot be read, got %q", got)
	}
}

func TestUnitDeleteUnassignsAgentsBeforeDeleting(t *testing.T) {
	var order []string
	var patched *platformclientv2.Workplan
	proxy, _ := stubProxy()
	proxy.getWorkPlanByIdAttr = func(ctx context.Context, p *workPlanProxy, managementUnitId string, workPlanId string) (*platformclientv2.Workplan, *platformclientv2.APIResponse, error) {
		return &platformclientv2.Workplan{
			Agents:   &[]platformclientv2.Deletableuserreference{{Id: platformclientv2.String("a")}, {Id: platformclientv2.String("b")}},
			Metadata: &platformclientv2.Wfmversionedentitymetadata{Version: platformclientv2.Int(4)},
		}, nil, nil
	}
	proxy.updateWorkPlanAttr = func(ctx context.Context, p *workPlanProxy, managementUnitId string, workPlanId string, workPlan *platformclientv2.Workplan) (*platformclientv2.Workplan, *platformclientv2.APIResponse, error) {
		order = append(order, "unassign")
		patched = workPlan
		return nil, nil, nil
	}
	proxy.deleteWorkPlanAttr = func(ctx context.Context, p *workPlanProxy, managementUnitId string, workPlanId string) (*platformclientv2.APIResponse, error) {
		order = append(order, "delete")
		return nil, nil
	}

	if diagErr := deleteWorkPlanAndUnassignAgents(context.Background(), proxy, "mu", "plan"); diagErr != nil {
		t.Fatalf("unexpected error: %v", diagErr)
	}
	if !reflect.DeepEqual(order, []string{"unassign", "delete"}) {
		t.Errorf("expected the agents to be unassigned before the delete, got %v", order)
	}
	if len(*patched.Agents) != 2 || !*(*patched.Agents)[0].Delete || *patched.Metadata.Version != 4 {
		t.Errorf("expected both agents to be unassigned with the current version, got %+v", patched)
	}
}

func TestUnitDeleteSkipsUnassigningWhenThereAreNoAgents(t *testing.T) {
	updated := false
	deleted := false
	proxy, _ := stubProxy()
	proxy.getWorkPlanByIdAttr = func(ctx context.Context, p *workPlanProxy, managementUnitId string, workPlanId string) (*platformclientv2.Workplan, *platformclientv2.APIResponse, error) {
		return &platformclientv2.Workplan{}, nil, nil
	}
	proxy.updateWorkPlanAttr = func(ctx context.Context, p *workPlanProxy, managementUnitId string, workPlanId string, workPlan *platformclientv2.Workplan) (*platformclientv2.Workplan, *platformclientv2.APIResponse, error) {
		updated = true
		return nil, nil, nil
	}
	proxy.deleteWorkPlanAttr = func(ctx context.Context, p *workPlanProxy, managementUnitId string, workPlanId string) (*platformclientv2.APIResponse, error) {
		deleted = true
		return nil, nil
	}

	if diagErr := deleteWorkPlanAndUnassignAgents(context.Background(), proxy, "mu", "plan"); diagErr != nil {
		t.Fatalf("unexpected error: %v", diagErr)
	}
	if updated || !deleted {
		t.Errorf("expected only the delete, got update %v delete %v", updated, deleted)
	}
}

func TestUnitDeleteTreatsAMissingWorkPlanAsDeleted(t *testing.T) {
	proxy, _ := stubProxy()
	proxy.getWorkPlanByIdAttr = func(ctx context.Context, p *workPlanProxy, managementUnitId string, workPlanId string) (*platformclientv2.Workplan, *platformclientv2.APIResponse, error) {
		return nil, &platformclientv2.APIResponse{StatusCode: http.StatusNotFound}, fmt.Errorf("not found")
	}
	if diagErr := deleteWorkPlanAndUnassignAgents(context.Background(), proxy, "mu", "plan"); diagErr != nil {
		t.Errorf("a work plan that is already gone must not fail the delete, got %v", diagErr)
	}
}

func invalidResponse() *platformclientv2.Validateworkplanresponse {
	return &platformclientv2.Validateworkplanresponse{
		Valid: platformclientv2.Bool(false),
		Messages: &platformclientv2.Validateworkplanmessages{
			ViolationMessages: &[]platformclientv2.Workplanconfigurationviolationmessage{{Severity: platformclientv2.String("Error"), VarType: platformclientv2.String("NoShifts")}},
		},
	}
}

func knownConfig() cty.Value {
	return cty.ObjectVal(map[string]cty.Value{"name": cty.StringVal("Plan"), "enabled": cty.True})
}

func TestUnitValidateConfiguredWorkPlanFailsAnInvalidNewWorkPlan(t *testing.T) {
	var validatedId string
	proxy, _ := stubProxy()
	proxy.validateWorkPlanAttr = func(ctx context.Context, p *workPlanProxy, managementUnitId string, workPlanId string, workPlan *platformclientv2.Workplanvalidationrequest) (*platformclientv2.Validateworkplanresponse, *platformclientv2.APIResponse, error) {
		validatedId = workPlanId
		return invalidResponse(), nil, nil
	}

	err := validateConfiguredWorkPlan(context.Background(), proxy, "mu", newWorkPlanId, knownConfig())
	if err == nil || !strings.Contains(err.Error(), "NoShifts") {
		t.Errorf("expected the validation messages in the error, got %v", err)
	}
	if validatedId != newWorkPlanId {
		t.Errorf("a new work plan must be validated as %q, got %q", newWorkPlanId, validatedId)
	}
}

func TestUnitValidateConfiguredWorkPlanValidatesChangesOnTopOfTheCurrentWorkPlan(t *testing.T) {
	var request *platformclientv2.Workplanvalidationrequest
	proxy, _ := stubProxy()
	proxy.getWorkPlanByIdAttr = func(ctx context.Context, p *workPlanProxy, managementUnitId string, workPlanId string) (*platformclientv2.Workplan, *platformclientv2.APIResponse, error) {
		return &platformclientv2.Workplan{Shifts: &[]platformclientv2.Workplanshift{shift("Day", "0")}}, nil, nil
	}
	proxy.validateWorkPlanAttr = func(ctx context.Context, p *workPlanProxy, managementUnitId string, workPlanId string, workPlan *platformclientv2.Workplanvalidationrequest) (*platformclientv2.Validateworkplanresponse, *platformclientv2.APIResponse, error) {
		request = workPlan
		return &platformclientv2.Validateworkplanresponse{Valid: platformclientv2.Bool(true)}, nil, nil
	}
	config := cty.ObjectVal(map[string]cty.Value{
		"name":   cty.StringVal("Plan"),
		"shifts": cty.ListValEmpty(cty.Object(map[string]cty.Type{"name": cty.String})),
	})

	if err := validateConfiguredWorkPlan(context.Background(), proxy, "mu", "plan-1", config); err != nil {
		t.Fatalf("expected a valid work plan to pass, got %v", err)
	}
	if request == nil || request.Shifts == nil || len(*request.Shifts) != 1 || !*(*request.Shifts)[0].Delete {
		t.Errorf("expected the removal of the configured-away shift to be part of the validated changes, got %+v", request)
	}
}

func TestUnitValidateConfiguredWorkPlanSkipsWhatItCannotCheck(t *testing.T) {
	asked := false
	proxy, _ := stubProxy()
	proxy.getWorkPlanByIdAttr = func(ctx context.Context, p *workPlanProxy, managementUnitId string, workPlanId string) (*platformclientv2.Workplan, *platformclientv2.APIResponse, error) {
		return nil, nil, fmt.Errorf("cannot read")
	}
	proxy.validateWorkPlanAttr = func(ctx context.Context, p *workPlanProxy, managementUnitId string, workPlanId string, workPlan *platformclientv2.Workplanvalidationrequest) (*platformclientv2.Validateworkplanresponse, *platformclientv2.APIResponse, error) {
		asked = true
		return invalidResponse(), nil, nil
	}

	unknown := cty.ObjectVal(map[string]cty.Value{"name": cty.UnknownVal(cty.String)})
	if err := validateConfiguredWorkPlan(context.Background(), proxy, "mu", newWorkPlanId, unknown); err != nil || asked {
		t.Errorf("a configuration with unknown values must not be validated, got %v (asked %v)", err, asked)
	}
	if err := validateConfiguredWorkPlan(context.Background(), proxy, "", newWorkPlanId, knownConfig()); err != nil || asked {
		t.Errorf("an unknown management unit must not be validated, got %v (asked %v)", err, asked)
	}
	if err := validateConfiguredWorkPlan(context.Background(), proxy, "mu", "plan-1", knownConfig()); err != nil || asked {
		t.Errorf("a work plan that cannot be read must not fail the plan, got %v (asked %v)", err, asked)
	}
}

func TestUnitValidateConfiguredWorkPlanDoesNotFailThePlanWhenValidationFails(t *testing.T) {
	proxy, _ := stubProxy()
	proxy.validateWorkPlanAttr = func(ctx context.Context, p *workPlanProxy, managementUnitId string, workPlanId string, workPlan *platformclientv2.Workplanvalidationrequest) (*platformclientv2.Validateworkplanresponse, *platformclientv2.APIResponse, error) {
		return nil, &platformclientv2.APIResponse{StatusCode: http.StatusInternalServerError}, fmt.Errorf("validate is down")
	}
	if err := validateConfiguredWorkPlan(context.Background(), proxy, "mu", newWorkPlanId, knownConfig()); err != nil {
		t.Errorf("an unavailable validation must not fail the plan, got %v", err)
	}
}

func TestUnitConfigurableKeysExcludeReadOnlyAttributes(t *testing.T) {
	keys := configurableKeys()
	for _, key := range keys {
		if key == "valid" {
			t.Error("the read only valid attribute is not configurable")
		}
	}
	if !reflect.DeepEqual(append([]string(nil), keys...), keys) || len(keys) < 30 {
		t.Errorf("expected the configurable attributes, got %v", keys)
	}
}
