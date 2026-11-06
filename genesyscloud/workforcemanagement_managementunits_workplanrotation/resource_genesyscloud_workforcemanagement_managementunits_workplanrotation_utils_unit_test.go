package workforcemanagement_managementunits_workplanrotation

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
	"testing"
	"time"

	"github.com/hashicorp/go-cty/cty"
	"github.com/mypurecloud/platform-client-sdk-go/v200/platformclientv2"
	resourceExporter "github.com/mypurecloud/terraform-provider-genesyscloud/genesyscloud/resource_exporter"
)

func date(value string) *time.Time {
	parsed, _ := time.Parse(dateLayout, value)
	return &parsed
}

func currentAgent(userId string, start string, end string, position int) platformclientv2.Workplanrotationagentresponse {
	agent := platformclientv2.Workplanrotationagentresponse{
		User:      &platformclientv2.Userreference{Id: platformclientv2.String(userId)},
		DateRange: &platformclientv2.Daterangewithoptionalend{StartBusinessUnitDate: date(start)},
		Position:  platformclientv2.Int(position),
	}
	if end != "" {
		agent.DateRange.EndBusinessUnitDate = date(end)
	}
	return agent
}

func configuredAgent(userId string, start string, end string, position int) rotationAgent {
	agent := rotationAgent{userId: userId, start: date(start), position: platformclientv2.Int(position)}
	if end != "" {
		agent.end = date(end)
	}
	return agent
}

func TestUnitSplitWorkPlanRotationTerraformId(t *testing.T) {
	managementUnitId, rotationId := SplitWorkPlanRotationTerraformId("mu-1/rotation-2")
	if managementUnitId != "mu-1" || rotationId != "rotation-2" {
		t.Errorf("expected mu-1 and rotation-2, got %s and %s", managementUnitId, rotationId)
	}
}

func TestUnitModifyWorkPlanIdStateValue(t *testing.T) {
	if got := ModifyWorkPlanIdStateValue("mu-1/plan-2"); got != "plan-2" {
		t.Errorf("expected plan-2, got %s", got)
	}
	if got := ModifyWorkPlanIdStateValue("plan-2"); got != "plan-2" {
		t.Errorf("expected plan-2, got %s", got)
	}
}

func TestUnitBuildAgentUpdates(t *testing.T) {
	current := []platformclientv2.Workplanrotationagentresponse{
		currentAgent("same", "2026-01-05", "", 0),
		currentAgent("moved", "2026-01-05", "", 1),
		currentAgent("gone", "2026-01-05", "2026-06-01", 2),
	}
	configured := []rotationAgent{
		configuredAgent("same", "2026-01-05", "", 0),
		configuredAgent("moved", "2026-01-05", "", 3),
		configuredAgent("new", "2026-02-02", "2026-12-28", 1),
	}

	updates := buildAgentUpdates(current, configured)
	byUser := map[string]platformclientv2.Updateworkplanrotationagentrequest{}
	for _, update := range updates {
		byUser[*update.UserId] = update
	}

	if _, found := byUser["same"]; found {
		t.Error("an agent that did not change must not be sent")
	}
	if *byUser["moved"].Position != 3 || byUser["moved"].Delete != nil {
		t.Errorf("expected the changed position to be sent, got %+v", byUser["moved"])
	}
	if byUser["new"].DateRange == nil || byUser["new"].DateRange.EndBusinessUnitDate == nil || *byUser["new"].Position != 1 {
		t.Errorf("expected a new agent to be sent with its date range and position, got %+v", byUser["new"])
	}
	if byUser["gone"].Delete == nil || !*byUser["gone"].Delete {
		t.Errorf("expected an agent that is no longer configured to be removed, got %+v", byUser["gone"])
	}
	if len(updates) != 3 {
		t.Errorf("expected 3 updates, got %d", len(updates))
	}
}

func TestUnitBuildAgentUpdatesDetectsAChangedEndDate(t *testing.T) {
	current := []platformclientv2.Workplanrotationagentresponse{currentAgent("a", "2026-01-05", "2026-06-01", 0)}

	if updates := buildAgentUpdates(current, []rotationAgent{configuredAgent("a", "2026-01-05", "2026-06-01", 0)}); len(updates) != 0 {
		t.Errorf("expected no update for an unchanged agent, got %+v", updates)
	}
	updates := buildAgentUpdates(current, []rotationAgent{configuredAgent("a", "2026-01-05", "", 0)})
	if len(updates) != 1 || updates[0].DateRange.EndBusinessUnitDate != nil {
		t.Errorf("expected removing the end date to be sent as an open ended range, got %+v", updates)
	}
}

func rotationConfig() cty.Value {
	return cty.ObjectVal(map[string]cty.Value{
		"name":       cty.StringVal("Rotation"),
		"enabled":    cty.NullVal(cty.Bool),
		"start_date": cty.StringVal("2026-01-05"),
		"end_date":   cty.NullVal(cty.String),
		"work_plan_ids": cty.ListVal([]cty.Value{
			cty.StringVal("mu-1/plan-b"), cty.StringVal("plan-a"), cty.StringVal("plan-b"),
		}),
		"agents": cty.SetVal([]cty.Value{cty.ObjectVal(map[string]cty.Value{
			"user_id":    cty.StringVal("user-1"),
			"start_date": cty.StringVal("2026-01-05"),
			"end_date":   cty.NullVal(cty.String),
			"position":   cty.NumberIntVal(1),
		})}),
	})
}

func TestUnitBuildAddRequestKeepsThePatternOrder(t *testing.T) {
	request := buildAddRequest(rotationConfig())

	if !reflect.DeepEqual(*request.Pattern.WorkPlanIds, []string{"plan-b", "plan-a", "plan-b"}) {
		t.Errorf("the pattern order, repeats and bare work plan ids must be kept, got %v", *request.Pattern.WorkPlanIds)
	}
	if request.DateRange == nil || request.DateRange.EndBusinessUnitDate != nil || request.DateRange.StartBusinessUnitDate.Format(dateLayout) != "2026-01-05" {
		t.Errorf("unexpected date range %+v", request.DateRange)
	}
	agents := *request.Agents
	if len(agents) != 1 || *agents[0].UserId != "user-1" || *agents[0].Position != 1 {
		t.Errorf("unexpected agents %+v", agents)
	}
}

func TestUnitBuildUpdateRequestOnlyContainsConfiguredParts(t *testing.T) {
	current := &platformclientv2.Workplanrotationresponse{
		Agents:   &[]platformclientv2.Workplanrotationagentresponse{currentAgent("user-1", "2026-01-05", "", 1)},
		Metadata: &platformclientv2.Wfmversionedentitymetadata{Version: platformclientv2.Int(9)},
	}

	request := buildUpdateRequest(rotationConfig(), current)
	if request.Enabled != nil {
		t.Error("enabled was not configured and must not be sent")
	}
	if request.Agents != nil {
		t.Errorf("an agent that did not change must not be sent, got %+v", request.Agents)
	}
	if *request.Metadata.Version != 9 || *request.Name != "Rotation" {
		t.Errorf("expected the name and the current version, got %+v", request)
	}

	withoutAgents := cty.ObjectVal(map[string]cty.Value{
		"name":          cty.StringVal("Rotation"),
		"start_date":    cty.StringVal("2026-01-05"),
		"agents":        cty.NullVal(cty.DynamicPseudoType),
		"work_plan_ids": cty.NullVal(cty.DynamicPseudoType),
	})
	request = buildUpdateRequest(withoutAgents, current)
	if request.Agents != nil || request.Pattern != nil {
		t.Errorf("agents and a pattern that are not configured must be left alone, got %+v %+v", request.Agents, request.Pattern)
	}
}

func TestUnitFlattenKeepsThePatternOrderAndAgentDates(t *testing.T) {
	pattern := &platformclientv2.Workplanpatternresponse{WorkPlans: &[]platformclientv2.Workplanreference{
		{Id: platformclientv2.String("b")}, {Id: platformclientv2.String("a")}, {Id: platformclientv2.String("b")},
	}}
	if got := flattenWorkPlanIds(pattern); !reflect.DeepEqual(got, []interface{}{"b", "a", "b"}) {
		t.Errorf("expected the pattern order, got %v", got)
	}

	agents := []platformclientv2.Workplanrotationagentresponse{currentAgent("user-1", "2026-01-05", "2026-06-01", 2), currentAgent("user-2", "2026-02-02", "", 0)}
	set := flattenAgents(&agents)
	if set.Len() != 2 {
		t.Fatalf("expected 2 agents, got %d", set.Len())
	}
	for _, item := range set.List() {
		entry := item.(map[string]interface{})
		if entry["user_id"] == "user-2" {
			if _, hasEnd := entry["end_date"]; hasEnd {
				t.Error("an open ended agent must not have an end date")
			}
		} else if entry["end_date"] != "2026-06-01" || entry["position"] != 2 {
			t.Errorf("unexpected agent %+v", entry)
		}
	}
}

func TestUnitWorkPlanArrayRefResolverFindsCompositeKeys(t *testing.T) {
	exporters := map[string]*resourceExporter.ResourceExporter{
		workPlanResourceType: {SanitizedResourceMap: resourceExporter.ResourceIDMetaMap{
			"mu-1/plan-a": {BlockLabel: "Contact_Center_Day"},
		}},
	}
	configMap := map[string]interface{}{"work_plan_ids": []interface{}{"plan-a", "unknown-plan", "${already.resolved.id}"}}

	if err := WorkPlanArrayRefResolver("work_plan_ids")(configMap, exporters, "rotation"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := []interface{}{"${genesyscloud_workforcemanagement_managementunits_workplan.Contact_Center_Day.id}", "unknown-plan", "${already.resolved.id}"}
	if !reflect.DeepEqual(configMap["work_plan_ids"], expected) {
		t.Errorf("unexpected references %v", configMap["work_plan_ids"])
	}
}

func TestUnitDeleteRemovesAgentsBeforeDeleting(t *testing.T) {
	var order []string
	var patched *platformclientv2.Updateworkplanrotationrequest
	proxy := &workPlanRotationProxy{
		getWorkPlanRotationByIdAttr: func(ctx context.Context, p *workPlanRotationProxy, managementUnitId string, rotationId string) (*platformclientv2.Workplanrotationresponse, *platformclientv2.APIResponse, error) {
			return &platformclientv2.Workplanrotationresponse{
				Agents:   &[]platformclientv2.Workplanrotationagentresponse{currentAgent("a", "2026-01-05", "", 0), currentAgent("b", "2026-01-05", "", 1)},
				Metadata: &platformclientv2.Wfmversionedentitymetadata{Version: platformclientv2.Int(3)},
			}, nil, nil
		},
		updateWorkPlanRotationAttr: func(ctx context.Context, p *workPlanRotationProxy, managementUnitId string, rotationId string, rotation *platformclientv2.Updateworkplanrotationrequest) (*platformclientv2.Workplanrotationresponse, *platformclientv2.APIResponse, error) {
			order = append(order, "remove agents")
			patched = rotation
			return nil, nil, nil
		},
		deleteWorkPlanRotationAttr: func(ctx context.Context, p *workPlanRotationProxy, managementUnitId string, rotationId string) (*platformclientv2.APIResponse, error) {
			order = append(order, "delete")
			return nil, nil
		},
	}

	if diagErr := deleteWorkPlanRotationAndRemoveAgents(context.Background(), proxy, "mu", "rotation"); diagErr != nil {
		t.Fatalf("unexpected error: %v", diagErr)
	}
	if !reflect.DeepEqual(order, []string{"remove agents", "delete"}) {
		t.Errorf("expected the agents to be removed before the delete, got %v", order)
	}
	if len(*patched.Agents) != 2 || !*(*patched.Agents)[0].Delete || *patched.Metadata.Version != 3 {
		t.Errorf("expected both agents to be removed with the current version, got %+v", patched)
	}
}

func TestUnitDeleteTreatsAMissingRotationAsDeleted(t *testing.T) {
	proxy := &workPlanRotationProxy{
		getWorkPlanRotationByIdAttr: func(ctx context.Context, p *workPlanRotationProxy, managementUnitId string, rotationId string) (*platformclientv2.Workplanrotationresponse, *platformclientv2.APIResponse, error) {
			return nil, &platformclientv2.APIResponse{StatusCode: http.StatusNotFound}, fmt.Errorf("not found")
		},
	}
	if diagErr := deleteWorkPlanRotationAndRemoveAgents(context.Background(), proxy, "mu", "rotation"); diagErr != nil {
		t.Errorf("a rotation that is already gone must not fail the delete, got %v", diagErr)
	}
}

func TestUnitDisableSendsEnabledFalseWithTheCurrentVersion(t *testing.T) {
	var patched *platformclientv2.Updateworkplanrotationrequest
	proxy := &workPlanRotationProxy{
		getWorkPlanRotationByIdAttr: func(ctx context.Context, p *workPlanRotationProxy, managementUnitId string, rotationId string) (*platformclientv2.Workplanrotationresponse, *platformclientv2.APIResponse, error) {
			return &platformclientv2.Workplanrotationresponse{Metadata: &platformclientv2.Wfmversionedentitymetadata{Version: platformclientv2.Int(1)}}, nil, nil
		},
		updateWorkPlanRotationAttr: func(ctx context.Context, p *workPlanRotationProxy, managementUnitId string, rotationId string, rotation *platformclientv2.Updateworkplanrotationrequest) (*platformclientv2.Workplanrotationresponse, *platformclientv2.APIResponse, error) {
			patched = rotation
			return nil, nil, nil
		},
	}

	if diagErr := disableWorkPlanRotation(context.Background(), proxy, "mu", "rotation"); diagErr != nil {
		t.Fatalf("unexpected error: %v", diagErr)
	}
	if patched.Enabled == nil || *patched.Enabled || *patched.Metadata.Version != 1 || patched.Name != nil || patched.Pattern != nil {
		t.Errorf("expected only enabled=false and the version to be sent, got %+v", patched)
	}
}
