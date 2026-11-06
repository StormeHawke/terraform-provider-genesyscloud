package workforcemanagement_managementunits_workplanrotation

import (
	"context"
	"fmt"
	"math/big"
	"sort"
	"strings"
	"time"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/mypurecloud/platform-client-sdk-go/v200/platformclientv2"
	resourceExporter "github.com/mypurecloud/terraform-provider-genesyscloud/genesyscloud/resource_exporter"
)

const dateLayout = "2006-01-02"

// SplitWorkPlanRotationTerraformId splits the resource id, which is in the form
// <managementUnitId>/<workPlanRotationId>, into the managementUnitId and workPlanRotationId
func SplitWorkPlanRotationTerraformId(id string) (managementUnitId string, rotationId string) {
	parts := strings.SplitN(id, "/", 2)
	if len(parts) != 2 {
		return id, ""
	}
	return parts[0], parts[1]
}

// ModifyWorkPlanIdStateValue strips the management unit from a work plan id before it is saved in state.
// The work plan resource id is <managementUnitId>/<workPlanId>, but the rotation api only returns the work plan id,
// so saving the composite value would cause a non-empty plan.
func ModifyWorkPlanIdStateValue(id interface{}) string {
	workPlanId := id.(string)
	if i := strings.LastIndex(workPlanId, "/"); i >= 0 {
		return workPlanId[i+1:]
	}
	return workPlanId
}

// WorkPlanArrayRefResolver resolves bare work plan ids to Terraform references by searching the work plan exporter's resource map for a
// composite key ending with "/<workPlanId>".
func WorkPlanArrayRefResolver(attrName string) func(configMap map[string]interface{}, exporters map[string]*resourceExporter.ResourceExporter, resourceLabel string) error {
	return func(configMap map[string]interface{}, exporters map[string]*resourceExporter.ResourceExporter, resourceLabel string) error {
		ids, ok := configMap[attrName].([]interface{})
		if !ok || len(ids) == 0 {
			return nil
		}
		exporter := exporters[workPlanResourceType]
		if exporter == nil || exporter.SanitizedResourceMap == nil {
			return nil
		}

		resolved := make([]interface{}, 0, len(ids))
		for _, item := range ids {
			workPlanId, ok := item.(string)
			if !ok || workPlanId == "" || strings.HasPrefix(workPlanId, "${") {
				resolved = append(resolved, item)
				continue
			}
			reference := workPlanId
			suffix := "/" + workPlanId
			for compositeId, meta := range exporter.SanitizedResourceMap {
				if strings.HasSuffix(compositeId, suffix) && meta != nil && meta.BlockLabel != "" {
					reference = fmt.Sprintf("${%s.%s.id}", workPlanResourceType, meta.BlockLabel)
					break
				}
			}
			resolved = append(resolved, reference)
		}
		configMap[attrName] = resolved
		return nil
	}
}

/*
Requests only contain the values that are in the configuration, so they are built from the raw configuration instead of the merged resource data.
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

func stringOf(v cty.Value, name string) *string {
	attr := attrOf(v, name)
	if !isPresent(attr) {
		return nil
	}
	value := attr.AsString()
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

func intOf(v cty.Value, name string) *int {
	attr := attrOf(v, name)
	if !isPresent(attr) {
		return nil
	}
	value, _ := new(big.Float).Set(attr.AsBigFloat()).Int64()
	intValue := int(value)
	return &intValue
}

func dateOf(v cty.Value, name string) *time.Time {
	value := stringOf(v, name)
	if value == nil || *value == "" {
		return nil
	}
	parsed, err := time.Parse(dateLayout, *value)
	if err != nil {
		return nil
	}
	return &parsed
}

// rotationAgent is an agent as it is configured, or as it is returned by Genesys Cloud
type rotationAgent struct {
	userId   string
	start    *time.Time
	end      *time.Time
	position *int
}

func readAgents(config cty.Value) *[]rotationAgent {
	attr := attrOf(config, "agents")
	if !isPresent(attr) {
		return nil
	}
	agents := make([]rotationAgent, 0, attr.LengthInt())
	for _, element := range attr.AsValueSlice() {
		userId := stringOf(element, "user_id")
		if userId == nil {
			continue
		}
		agents = append(agents, rotationAgent{userId: *userId, start: dateOf(element, "start_date"), end: dateOf(element, "end_date"), position: intOf(element, "position")})
	}
	sort.Slice(agents, func(i, j int) bool { return agents[i].userId < agents[j].userId })
	return &agents
}

func readWorkPlanIds(config cty.Value) *[]string {
	attr := attrOf(config, "work_plan_ids")
	if !isPresent(attr) {
		return nil
	}
	ids := make([]string, 0, attr.LengthInt())
	for _, element := range attr.AsValueSlice() {
		if isPresent(element) {
			ids = append(ids, ModifyWorkPlanIdStateValue(element.AsString()))
		}
	}
	return &ids
}

// buildDateRange returns the configured date range. It always has both ends, because Genesys Cloud replaces the whole range
// and a missing end means open ended.
func buildDateRange(config cty.Value) *platformclientv2.Daterangewithoptionalend {
	start := dateOf(config, "start_date")
	if start == nil {
		return nil
	}
	return &platformclientv2.Daterangewithoptionalend{StartBusinessUnitDate: start, EndBusinessUnitDate: dateOf(config, "end_date")}
}

func agentDateRange(agent rotationAgent) *platformclientv2.Daterangewithoptionalend {
	return &platformclientv2.Daterangewithoptionalend{StartBusinessUnitDate: agent.start, EndBusinessUnitDate: agent.end}
}

func buildAddRequest(config cty.Value) *platformclientv2.Addworkplanrotationrequest {
	request := &platformclientv2.Addworkplanrotationrequest{
		Name:      stringOf(config, "name"),
		DateRange: buildDateRange(config),
	}
	if workPlanIds := readWorkPlanIds(config); workPlanIds != nil {
		request.Pattern = &platformclientv2.Workplanpatternrequest{WorkPlanIds: workPlanIds}
	}
	if agents := readAgents(config); agents != nil {
		added := make([]platformclientv2.Addworkplanrotationagentrequest, 0, len(*agents))
		for _, agent := range *agents {
			added = append(added, platformclientv2.Addworkplanrotationagentrequest{UserId: platformclientv2.String(agent.userId), DateRange: agentDateRange(agent), Position: agent.position})
		}
		request.Agents = &added
	}
	return request
}

func sameDate(a *time.Time, b *time.Time) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return a.Format(dateLayout) == b.Format(dateLayout)
}

// buildAgentUpdates returns the agents to add, the agents whose date range or position changed, and the agents to remove
func buildAgentUpdates(current []platformclientv2.Workplanrotationagentresponse, configured []rotationAgent) []platformclientv2.Updateworkplanrotationagentrequest {
	currentByUser := make(map[string]platformclientv2.Workplanrotationagentresponse, len(current))
	for _, agent := range current {
		if agent.User != nil && agent.User.Id != nil {
			currentByUser[*agent.User.Id] = agent
		}
	}

	var updates []platformclientv2.Updateworkplanrotationagentrequest
	configuredUsers := make(map[string]bool, len(configured))
	for _, agent := range configured {
		configuredUsers[agent.userId] = true
		existing, found := currentByUser[agent.userId]
		if found && existing.DateRange != nil && existing.Position != nil &&
			sameDate(existing.DateRange.StartBusinessUnitDate, agent.start) && sameDate(existing.DateRange.EndBusinessUnitDate, agent.end) &&
			agent.position != nil && *existing.Position == *agent.position {
			continue
		}
		updates = append(updates, platformclientv2.Updateworkplanrotationagentrequest{UserId: platformclientv2.String(agent.userId), DateRange: agentDateRange(agent), Position: agent.position})
	}

	var removed []string
	for userId := range currentByUser {
		if !configuredUsers[userId] {
			removed = append(removed, userId)
		}
	}
	sort.Strings(removed)
	for _, userId := range removed {
		updates = append(updates, platformclientv2.Updateworkplanrotationagentrequest{UserId: platformclientv2.String(userId), Delete: platformclientv2.Bool(true)})
	}
	return updates
}

// buildUpdateRequest builds the patch for a work plan rotation from the configuration and its current state.
// Attributes that are not configured are left out, so the work plan rotation keeps their current values.
func buildUpdateRequest(config cty.Value, current *platformclientv2.Workplanrotationresponse) *platformclientv2.Updateworkplanrotationrequest {
	request := &platformclientv2.Updateworkplanrotationrequest{
		Name:      stringOf(config, "name"),
		Enabled:   boolOf(config, "enabled"),
		DateRange: buildDateRange(config),
	}
	if workPlanIds := readWorkPlanIds(config); workPlanIds != nil {
		request.Pattern = &platformclientv2.Workplanpatternrequest{WorkPlanIds: workPlanIds}
	}
	if configured := readAgents(config); configured != nil {
		var existing []platformclientv2.Workplanrotationagentresponse
		if current.Agents != nil {
			existing = *current.Agents
		}
		if updates := buildAgentUpdates(existing, *configured); len(updates) > 0 {
			request.Agents = &updates
		}
	}
	// The version of the work plan rotation is sent for optimistic locking
	if current.Metadata != nil && current.Metadata.Version != nil {
		request.Metadata = &platformclientv2.Wfmversionedentitymetadata{Version: current.Metadata.Version}
	}
	return request
}

func flattenAgents(agents *[]platformclientv2.Workplanrotationagentresponse) *schema.Set {
	flattened := make([]interface{}, 0)
	if agents != nil {
		for _, agent := range *agents {
			if agent.User == nil || agent.User.Id == nil {
				continue
			}
			entry := map[string]interface{}{"user_id": *agent.User.Id}
			if agent.DateRange != nil {
				if agent.DateRange.StartBusinessUnitDate != nil {
					entry["start_date"] = agent.DateRange.StartBusinessUnitDate.Format(dateLayout)
				}
				if agent.DateRange.EndBusinessUnitDate != nil {
					entry["end_date"] = agent.DateRange.EndBusinessUnitDate.Format(dateLayout)
				}
			}
			if agent.Position != nil {
				entry["position"] = *agent.Position
			}
			flattened = append(flattened, entry)
		}
	}
	return schema.NewSet(schema.HashResource(rotationAgentResource), flattened)
}

func flattenWorkPlanIds(pattern *platformclientv2.Workplanpatternresponse) []interface{} {
	ids := make([]interface{}, 0)
	if pattern != nil && pattern.WorkPlans != nil {
		for _, workPlan := range *pattern.WorkPlans {
			if workPlan.Id != nil {
				ids = append(ids, *workPlan.Id)
			}
		}
	}
	return ids
}

// validateUniqueAgents rejects a configuration that lists an agent more than once, because agents are matched by user id
func validateUniqueAgents(_ context.Context, diff *schema.ResourceDiff, _ interface{}) error {
	if !diff.NewValueKnown("agents") {
		return nil
	}
	agents, ok := diff.Get("agents").(*schema.Set)
	if !ok {
		return nil
	}
	seen := make(map[string]bool)
	for _, item := range agents.List() {
		entry, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		userId := fmt.Sprint(entry["user_id"])
		if seen[userId] {
			return fmt.Errorf("an agent can only be listed once in a work plan rotation, %q is listed more than once", userId)
		}
		seen[userId] = true
	}
	return nil
}

// GenerateWorkPlanRotationResource returns terraform config for a work plan rotation, used in tests
func GenerateWorkPlanRotationResource(resourceLabel string, managementUnitId string, name string, startDate string, workPlanIds string, attrs ...string) string {
	return fmt.Sprintf(`resource "%s" "%s" {
	management_unit_id = %s
	name               = "%s"
	start_date         = "%s"
	work_plan_ids      = %s
	%s
}
`, ResourceType, resourceLabel, managementUnitId, name, startDate, workPlanIds, strings.Join(attrs, "\n"))
}
