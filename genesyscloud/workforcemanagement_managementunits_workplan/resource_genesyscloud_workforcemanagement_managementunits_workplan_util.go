package workforcemanagement_managementunits_workplan

import (
	"context"
	"fmt"
	"math/big"
	"reflect"
	"sort"
	"strings"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/mypurecloud/platform-client-sdk-go/v200/platformclientv2"
	"github.com/mypurecloud/terraform-provider-genesyscloud/genesyscloud/provider"
	"github.com/mypurecloud/terraform-provider-genesyscloud/genesyscloud/util/wfm"
)

// SplitWorkPlanTerraformId splits the resource id, which is in the form
// <managementUnitId>/<workPlanId>, into the managementUnitId and workPlanId
func SplitWorkPlanTerraformId(id string) (managementUnitId string, workPlanId string) {
	parts := strings.SplitN(id, "/", 2)
	if len(parts) != 2 {
		return id, ""
	}
	return parts[0], parts[1]
}

/*
Most work plan fields are optional and computed, so a field that was not configured must not be sent as a zero value.
Requests are therefore built from the raw configuration, which says whether a field was set, instead of from the merged resource data.
*/

// Terraform attributes that map one to one, by name, onto a field of the work plan, shift and activity sdk types.
// The tests check that every one of them exists on every sdk type it is copied to.
var (
	workPlanScalarKeys = []string{
		"name", "enabled", "constrain_weekly_paid_time", "flexible_weekly_paid_time", "weekly_exact_paid_minutes",
		"weekly_minimum_paid_minutes", "weekly_maximum_paid_minutes", "constrain_paid_time_granularity", "paid_time_granularity_minutes",
		"constrain_minimum_time_between_shifts", "minimum_time_between_shifts_minutes", "maximum_days",
		"minimum_consecutive_non_working_minutes_per_week", "constrain_maximum_consecutive_working_weekends",
		"maximum_consecutive_working_weekends", "minimum_working_days_per_week", "constrain_maximum_consecutive_working_days",
		"maximum_consecutive_working_days", "minimum_shift_start_distance_minutes", "minimum_days_off_per_planning_period",
		"maximum_days_off_per_planning_period", "minimum_paid_minutes_per_planning_period", "maximum_paid_minutes_per_planning_period",
		"constrain_maximum_working_weekends_per_planning_period", "maximum_working_weekends_per_planning_period",
		"shift_start_variance_type", "shift_start_variance_period",
	}
	shiftScalarKeys = []string{
		"name", "flexible_start_time", "exact_start_time_minutes_from_midnight", "earliest_start_time_minutes_from_midnight",
		"latest_start_time_minutes_from_midnight", "start_increment_minutes", "constrain_latest_stop_time",
		"latest_stop_time_minutes_from_midnight", "constrain_earliest_stop_time", "earliest_stop_time_minutes_from_midnight",
		"flexible_paid_time", "exact_paid_time_minutes", "minimum_paid_time_minutes", "maximum_paid_time_minutes",
		"constrain_contiguous_work_time", "minimum_contiguous_work_time_minutes", "maximum_contiguous_work_time_minutes",
		"constrain_day_off", "day_off_rule",
	}
	activityScalarKeys = []string{
		"activity_code_id", "description", "length_minutes", "start_time_is_relative_to_shift_start", "flexible_start_time",
		"earliest_start_time_minutes", "latest_start_time_minutes", "exact_start_time_minutes", "start_time_increment_minutes",
		"counts_as_paid_time", "counts_as_contiguous_work_time", "minimum_length_from_shift_start_minutes",
		"minimum_length_from_shift_end_minutes", "maximum_length_from_shift_end_minutes",
	}
	planningPeriodConstraintKeys = []string{"enabled", "minimum_count", "maximum_count"}
)

func snakeToPascal(key string) string {
	var builder strings.Builder
	for _, part := range strings.Split(key, "_") {
		if part != "" {
			builder.WriteString(strings.ToUpper(part[:1]) + part[1:])
		}
	}
	return builder.String()
}

var nullValue = cty.NullVal(cty.DynamicPseudoType)

// assignScalars copies the configured values of the listed attributes onto the same-named fields of target, a pointer to an sdk struct
func assignScalars(target interface{}, config cty.Value, keys []string) {
	structValue := reflect.ValueOf(target).Elem()
	for _, key := range keys {
		attr := wfm.AttrOf(config, key)
		if !wfm.IsPresent(attr) {
			continue
		}
		field := structValue.FieldByName(snakeToPascal(key))
		if !field.IsValid() {
			continue
		}
		switch field.Type().String() {
		case "*string":
			value := attr.AsString()
			field.Set(reflect.ValueOf(&value))
		case "*int":
			parsed, _ := attr.AsBigFloat().Int64()
			value := int(parsed)
			field.Set(reflect.ValueOf(&value))
		case "*bool":
			value := attr.True()
			field.Set(reflect.ValueOf(&value))
		case "*float64":
			value, _ := new(big.Float).Set(attr.AsBigFloat()).Float64()
			field.Set(reflect.ValueOf(&value))
		}
	}
}

// copyScalars copies the listed fields from one sdk struct to another with the same field names
func copyScalars(target interface{}, source interface{}, keys []string) {
	targetValue := reflect.ValueOf(target).Elem()
	sourceValue := reflect.ValueOf(source)
	if sourceValue.Kind() == reflect.Ptr {
		sourceValue = sourceValue.Elem()
	}
	for _, key := range keys {
		name := snakeToPascal(key)
		targetField := targetValue.FieldByName(name)
		sourceField := sourceValue.FieldByName(name)
		if targetField.IsValid() && sourceField.IsValid() && targetField.Type() == sourceField.Type() {
			targetField.Set(sourceField)
		}
	}
}

// scalarsToMap returns the non-nil fields of an sdk struct under their terraform attribute names
func scalarsToMap(source interface{}, keys []string) map[string]interface{} {
	result := make(map[string]interface{})
	sourceValue := reflect.ValueOf(source)
	if sourceValue.Kind() == reflect.Ptr {
		sourceValue = sourceValue.Elem()
	}
	for _, key := range keys {
		field := sourceValue.FieldByName(snakeToPascal(key))
		if field.IsValid() && field.Kind() == reflect.Ptr && !field.IsNil() {
			result[key] = field.Elem().Interface()
		}
	}
	return result
}

func dayWrapper(config cty.Value, name string) *platformclientv2.Setwrapperdayofweek {
	days := wfm.SortedStringSliceOf(config, name)
	if days == nil {
		return nil
	}
	return &platformclientv2.Setwrapperdayofweek{Values: days}
}

func readActivity(config cty.Value) platformclientv2.Workplanactivity {
	activity := platformclientv2.Workplanactivity{}
	assignScalars(&activity, config, activityScalarKeys)
	if activity.ActivityCodeId != nil {
		activity.ActivityCodeId = platformclientv2.String(wfm.ModifyActivityCodeIdStateValue(*activity.ActivityCodeId))
	}
	return activity
}

// readShift reads a configured shift. Activities are only set when the configuration has them, so a shift without
// configured activities leaves the activities of the matching existing shift alone.
func readShift(config cty.Value) platformclientv2.Workplanshift {
	shift := platformclientv2.Workplanshift{}
	assignScalars(&shift, config, shiftScalarKeys)
	shift.Days = dayWrapper(config, "days")

	if constraints := wfm.BlockOf(config, "planning_period_constraints"); wfm.IsPresent(constraints) {
		planningPeriod := platformclientv2.Planningperiodshiftconstraints{}
		assignScalars(&planningPeriod, constraints, planningPeriodConstraintKeys)
		shift.PlanningPeriodConstraints = &planningPeriod
	}

	if activities := wfm.AttrOf(config, "activities"); wfm.IsPresent(activities) {
		configured := make([]platformclientv2.Workplanactivity, 0, activities.LengthInt())
		for _, activity := range activities.AsValueSlice() {
			configured = append(configured, readActivity(activity))
		}
		shift.Activities = &configured
	}
	return shift
}

// readShifts reads the configured shifts, or returns nil when the configuration does not have any shifts attribute
func readShifts(config cty.Value) *[]platformclientv2.Workplanshift {
	attr := wfm.AttrOf(config, "shifts")
	if !wfm.IsPresent(attr) {
		return nil
	}
	shifts := make([]platformclientv2.Workplanshift, 0, attr.LengthInt())
	for _, shift := range attr.AsValueSlice() {
		shifts = append(shifts, readShift(shift))
	}
	return &shifts
}

func readShiftStartVariances(config cty.Value) *platformclientv2.Listwrappershiftstartvariance {
	attr := wfm.AttrOf(config, "shift_start_variances")
	if !wfm.IsPresent(attr) {
		return nil
	}
	variances := make([]platformclientv2.Shiftstartvariance, 0, attr.LengthInt())
	for _, variance := range attr.AsValueSlice() {
		entry := platformclientv2.Shiftstartvariance{ApplicableDays: wfm.SortedStringSliceOf(variance, "applicable_days")}
		assignScalars(&entry, variance, []string{"max_shift_start_variance_minutes"})
		variances = append(variances, entry)
	}
	return &platformclientv2.Listwrappershiftstartvariance{Values: &variances}
}

func readAgentIds(config cty.Value) *[]string {
	return wfm.SortedStringSliceOf(config, "agent_ids")
}

// activityCovers reports whether an existing activity has every value that is set on the configured activity
func activityCovers(existing platformclientv2.Workplanactivity, configured platformclientv2.Workplanactivity) bool {
	for _, key := range activityScalarKeys {
		name := snakeToPascal(key)
		wanted := reflect.ValueOf(configured).FieldByName(name)
		if wanted.IsNil() {
			continue
		}
		actual := reflect.ValueOf(existing).FieldByName(name)
		if actual.IsNil() || actual.Elem().Interface() != wanted.Elem().Interface() {
			return false
		}
	}
	return true
}

// activitiesMatch reports whether the existing activities are exactly the configured activities, in any order
func activitiesMatch(existing []platformclientv2.Workplanactivity, configured []platformclientv2.Workplanactivity) bool {
	if len(existing) != len(configured) {
		return false
	}
	used := make([]bool, len(existing))
	for _, wanted := range configured {
		found := false
		for i, candidate := range existing {
			if !used[i] && activityCovers(candidate, wanted) {
				used[i] = true
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

// buildShiftUpdates works out the shifts to send so the work plan ends up with the configured shifts.
// Shifts are matched to the existing ones by name: a matching shift is edited, an unmatched one is added and an existing
// shift that is not configured is deleted. The activities of an edited shift are only sent when they differ, in which case
// the existing activities are deleted and the configured ones added.
func buildShiftUpdates(existing []platformclientv2.Workplanshift, configured []platformclientv2.Workplanshift) ([]platformclientv2.Workplanshift, error) {
	existingByName := make(map[string][]platformclientv2.Workplanshift)
	for _, shift := range existing {
		if shift.Name != nil {
			existingByName[*shift.Name] = append(existingByName[*shift.Name], shift)
		}
	}

	updates := make([]platformclientv2.Workplanshift, 0, len(configured))
	matched := make(map[string]bool)
	for _, shift := range configured {
		name := ""
		if shift.Name != nil {
			name = *shift.Name
		}
		candidates := existingByName[name]
		switch {
		case len(candidates) > 1:
			return nil, fmt.Errorf("the work plan has %d shifts named %q, so the shift to update cannot be told apart; rename them in Genesys Cloud so the names are unique", len(candidates), name)
		case len(candidates) == 1:
			matched[name] = true
			current := candidates[0]
			shift.Id = current.Id
			if shift.Activities != nil {
				currentActivities := *current.Activities
				if activitiesMatch(currentActivities, *shift.Activities) {
					shift.Activities = nil
				} else {
					shift.Activities = replacementActivities(currentActivities, *shift.Activities)
				}
			}
		}
		updates = append(updates, shift)
	}

	for _, shift := range existing {
		if shift.Name == nil || !matched[*shift.Name] {
			updates = append(updates, platformclientv2.Workplanshift{Id: shift.Id, Delete: platformclientv2.Bool(true)})
		}
	}
	return updates, nil
}

func replacementActivities(existing []platformclientv2.Workplanactivity, configured []platformclientv2.Workplanactivity) *[]platformclientv2.Workplanactivity {
	replacement := make([]platformclientv2.Workplanactivity, 0, len(existing)+len(configured))
	for _, activity := range existing {
		replacement = append(replacement, platformclientv2.Workplanactivity{Id: activity.Id, Delete: platformclientv2.Bool(true)})
	}
	replacement = append(replacement, configured...)
	return &replacement
}

// buildAgentUpdates returns the agents to assign to the work plan and the agents to unassign from it
func buildAgentUpdates(existing []string, configured []string) []platformclientv2.Deletableuserreference {
	existingSet := make(map[string]bool, len(existing))
	for _, id := range existing {
		existingSet[id] = true
	}
	configuredSet := make(map[string]bool, len(configured))
	for _, id := range configured {
		configuredSet[id] = true
	}

	var updates []platformclientv2.Deletableuserreference
	for _, id := range wfm.SortedKeys(configuredSet) {
		if !existingSet[id] {
			updates = append(updates, platformclientv2.Deletableuserreference{Id: platformclientv2.String(id)})
		}
	}
	for _, id := range wfm.SortedKeys(existingSet) {
		if !configuredSet[id] {
			updates = append(updates, platformclientv2.Deletableuserreference{Id: platformclientv2.String(id), Delete: platformclientv2.Bool(true)})
		}
	}
	return updates
}

func existingAgentIds(workPlan *platformclientv2.Workplan) []string {
	ids := []string{}
	for _, agent := range *workPlan.Agents {
		ids = append(ids, *agent.Id)
	}
	return ids
}

func buildCreateWorkPlanRequest(config cty.Value) *platformclientv2.Createworkplan {
	request := &platformclientv2.Createworkplan{}
	assignScalars(request, config, workPlanScalarKeys)
	request.OptionalDays = dayWrapper(config, "optional_days")
	request.ShiftStartVariances = readShiftStartVariances(config)

	if shifts := readShifts(config); shifts != nil {
		created := make([]platformclientv2.Createworkplanshift, 0, len(*shifts))
		for _, shift := range *shifts {
			created = append(created, shiftToCreate(shift))
		}
		request.Shifts = &created
	}
	if agentIds := readAgentIds(config); agentIds != nil {
		agents := make([]platformclientv2.Userreference, 0, len(*agentIds))
		for _, id := range *agentIds {
			agents = append(agents, platformclientv2.Userreference{Id: platformclientv2.String(id)})
		}
		request.Agents = &agents
	}
	return request
}

func shiftToCreate(shift platformclientv2.Workplanshift) platformclientv2.Createworkplanshift {
	created := platformclientv2.Createworkplanshift{Days: shift.Days, PlanningPeriodConstraints: shift.PlanningPeriodConstraints}
	copyScalars(&created, shift, shiftScalarKeys)
	if shift.Activities != nil {
		activities := make([]platformclientv2.Createworkplanactivity, 0, len(*shift.Activities))
		for _, activity := range *shift.Activities {
			createdActivity := platformclientv2.Createworkplanactivity{}
			copyScalars(&createdActivity, activity, activityScalarKeys)
			activities = append(activities, createdActivity)
		}
		created.Activities = &activities
	}
	return created
}

// buildUpdateWorkPlanRequest builds the patch for a work plan from the configuration and the current state of the work plan.
// Attributes that are not configured are left out, so the work plan keeps their current values.
func buildUpdateWorkPlanRequest(config cty.Value, current *platformclientv2.Workplan) (*platformclientv2.Workplan, error) {
	request := &platformclientv2.Workplan{}
	assignScalars(request, config, workPlanScalarKeys)
	request.OptionalDays = dayWrapper(config, "optional_days")
	request.ShiftStartVariances = readShiftStartVariances(config)

	if configuredShifts := readShifts(config); configuredShifts != nil {
		updates, err := buildShiftUpdates(*current.Shifts, *configuredShifts)
		if err != nil {
			return nil, err
		}
		request.Shifts = &updates
	}

	if configuredAgents := readAgentIds(config); configuredAgents != nil {
		if updates := buildAgentUpdates(existingAgentIds(current), *configuredAgents); len(updates) > 0 {
			request.Agents = &updates
		}
	}

	// The version of the work plan is sent for optimistic locking
	if current.Metadata != nil && current.Metadata.Version != nil {
		request.Metadata = &platformclientv2.Wfmversionedentitymetadata{Version: current.Metadata.Version}
	}
	return request, nil
}

func flattenActivities(activities *[]platformclientv2.Workplanactivity, priorOrder []map[string]interface{}) []interface{} {
	remaining := append([]platformclientv2.Workplanactivity(nil), *activities...)
	sort.SliceStable(remaining, func(i, j int) bool { return activitySortKey(remaining[i]) < activitySortKey(remaining[j]) })

	ordered := make([]platformclientv2.Workplanactivity, 0, len(remaining))
	for _, prior := range priorOrder {
		for i, candidate := range remaining {
			if activityMatchesPrior(candidate, prior) {
				ordered = append(ordered, candidate)
				remaining = append(remaining[:i], remaining[i+1:]...)
				break
			}
		}
	}
	ordered = append(ordered, remaining...)

	flattened := make([]interface{}, 0, len(ordered))
	for _, activity := range ordered {
		flattened = append(flattened, scalarsToMap(activity, activityScalarKeys))
	}
	return flattened
}

func activitySortKey(activity platformclientv2.Workplanactivity) string {
	return fmt.Sprintf("%s|%s|%d|%d", wfm.StringValue(activity.ActivityCodeId), wfm.StringValue(activity.Description), intValue(activity.LengthMinutes), intValue(activity.ExactStartTimeMinutes))
}

// activityMatchesPrior compares the identifying values of an activity with the same activity of the previous state
func activityMatchesPrior(activity platformclientv2.Workplanactivity, prior map[string]interface{}) bool {
	return wfm.StringValue(activity.ActivityCodeId) == fmt.Sprint(prior["activity_code_id"]) &&
		wfm.StringValue(activity.Description) == fmt.Sprint(prior["description"]) &&
		intValue(activity.LengthMinutes) == intOfInterface(prior["length_minutes"])
}

func flattenShifts(shifts *[]platformclientv2.Workplanshift, priorShifts []interface{}) []interface{} {
	remaining := append([]platformclientv2.Workplanshift(nil), *shifts...)
	sort.SliceStable(remaining, func(i, j int) bool { return wfm.StringValue(remaining[i].Name) < wfm.StringValue(remaining[j].Name) })

	priorByName := make(map[string][]map[string]interface{})
	var priorNames []string
	for _, prior := range priorShifts {
		priorMap, ok := prior.(map[string]interface{})
		if !ok {
			continue
		}
		name := fmt.Sprint(priorMap["name"])
		priorNames = append(priorNames, name)
		var priorActivities []map[string]interface{}
		if activities, ok := priorMap["activities"].([]interface{}); ok {
			for _, activity := range activities {
				if activityMap, ok := activity.(map[string]interface{}); ok {
					priorActivities = append(priorActivities, activityMap)
				}
			}
		}
		priorByName[name] = priorActivities
	}

	ordered := make([]platformclientv2.Workplanshift, 0, len(remaining))
	for _, name := range priorNames {
		for i, candidate := range remaining {
			if wfm.StringValue(candidate.Name) == name {
				ordered = append(ordered, candidate)
				remaining = append(remaining[:i], remaining[i+1:]...)
				break
			}
		}
	}
	ordered = append(ordered, remaining...)

	flattened := make([]interface{}, 0, len(ordered))
	for _, shift := range ordered {
		shiftMap := scalarsToMap(shift, shiftScalarKeys)
		if shift.Days != nil && shift.Days.Values != nil {
			shiftMap["days"] = wfm.StringSet(*shift.Days.Values)
		}
		if shift.PlanningPeriodConstraints != nil {
			shiftMap["planning_period_constraints"] = []interface{}{scalarsToMap(shift.PlanningPeriodConstraints, planningPeriodConstraintKeys)}
		}
		shiftMap["activities"] = flattenActivities(shift.Activities, priorByName[wfm.StringValue(shift.Name)])
		flattened = append(flattened, shiftMap)
	}
	return flattened
}

func flattenShiftStartVariances(variances *platformclientv2.Listwrappershiftstartvariance) *schema.Set {
	flattened := make([]interface{}, 0)
	if variances != nil {
		for _, variance := range *variances.Values {
			flattened = append(flattened, map[string]interface{}{
				"applicable_days":                  wfm.StringSet(*variance.ApplicableDays),
				"max_shift_start_variance_minutes": *variance.MaxShiftStartVarianceMinutes,
			})
		}
	}
	return schema.NewSet(schema.HashResource(shiftStartVarianceResource), flattened)
}

func intValue(value *int) int {
	if value == nil {
		return 0
	}
	return *value
}

func intOfInterface(value interface{}) int {
	if number, ok := value.(int); ok {
		return number
	}
	return 0
}

// validateShiftNames rejects a configuration with two shifts of the same name, because shifts are matched to existing shifts by name
func validateShiftNames(_ context.Context, diff *schema.ResourceDiff, _ interface{}) error {
	if !diff.NewValueKnown("shifts") {
		return nil
	}

	seen := make(map[string]bool)
	for _, shift := range diff.Get("shifts").([]interface{}) {
		shiftMap, ok := shift.(map[string]interface{})
		if !ok {
			continue
		}
		name := fmt.Sprint(shiftMap["name"])
		if seen[name] {
			return fmt.Errorf("shift names must be unique within a work plan, %q is used more than once", name)
		}
		seen[name] = true
	}
	return nil
}

// toValidationRequest converts a work plan patch into the request of the validate route, which takes the same changes
func toValidationRequest(workPlan *platformclientv2.Workplan) *platformclientv2.Workplanvalidationrequest {
	request := &platformclientv2.Workplanvalidationrequest{
		OptionalDays:        workPlan.OptionalDays,
		ShiftStartVariances: workPlan.ShiftStartVariances,
		Shifts:              workPlan.Shifts,
		Agents:              workPlan.Agents,
	}
	copyScalars(request, workPlan, workPlanScalarKeys)
	return request
}

// formatValidationMessages describes why a work plan is invalid, in a form that can be added to an error
func formatValidationMessages(response *platformclientv2.Validateworkplanresponse) string {
	if response.Messages == nil {
		return ""
	}

	var lines []string
	for _, message := range *response.Messages.ViolationMessages {
		lines = append(lines, fmt.Sprintf("- %s: %s%s", wfm.StringValue(message.Severity), wfm.StringValue(message.VarType), formatArguments(message.Arguments)))
	}
	if conflict := response.Messages.ConstraintConflictMessage; conflict != nil {
		if conflict.Message != nil {
			lines = append(lines, fmt.Sprintf("- Constraint conflict: %s%s", wfm.StringValue(conflict.Message.VarType), formatArguments(conflict.Message.Arguments)))
		}
		if conflict.ConflictedConstraintMessages != nil && len(*conflict.ConflictedConstraintMessages) > 0 {
			lines = append(lines, fmt.Sprintf("  (%d conflicting constraints)", len(*conflict.ConflictedConstraintMessages)))
		}
	}
	return strings.Join(lines, "\n")
}

func formatArguments(arguments *[]platformclientv2.Workplanvalidationmessageargument) string {
	if arguments == nil || len(*arguments) == 0 {
		return ""
	}
	parts := make([]string, 0, len(*arguments))
	for _, argument := range *arguments {
		parts = append(parts, fmt.Sprintf("%s=%s", wfm.StringValue(argument.VarType), wfm.StringValue(argument.Value)))
	}
	return " (" + strings.Join(parts, ", ") + ")"
}

// validateWorkPlanDuringPlan asks Genesys Cloud to validate an enabled work plan while planning, so an invalid work plan fails the plan
// with the reasons instead of failing the apply. Genesys Cloud only rejects enabled work plans that are invalid, so disabled ones are
// not checked. It does nothing when the configuration is not fully known yet, which is the case when it refers to resources that are created
// in the same run, and it never fails the plan because validation could not be performed.
func validateWorkPlanDuringPlan(ctx context.Context, diff *schema.ResourceDiff, meta interface{}) error {
	providerMeta, ok := meta.(*provider.ProviderMeta)
	if !ok || providerMeta == nil {
		return nil
	}
	if diff.Id() != "" && !diff.HasChanges(configurableKeys()...) {
		return nil
	}
	if enabled, _ := diff.Get("enabled").(bool); !enabled {
		return nil
	}

	managementUnitId := diff.Get("management_unit_id").(string)
	proxy := getWorkPlanProxy(providerMeta.ClientConfig)
	if diff.Id() == "" {
		return validateNewWorkPlan(ctx, proxy, managementUnitId, diff.GetRawConfig())
	}
	_, workPlanId := SplitWorkPlanTerraformId(diff.Id())
	return validateExistingWorkPlan(ctx, proxy, managementUnitId, workPlanId, diff.GetRawConfig())
}

// validateNewWorkPlan validates the configuration as a work plan that does not exist yet
func validateNewWorkPlan(ctx context.Context, proxy *workPlanProxy, managementUnitId string, config cty.Value) error {
	if !canValidateWorkPlan(managementUnitId, config) {
		return nil
	}
	return validateWorkPlanRequest(ctx, proxy, managementUnitId, newWorkPlanId, &platformclientv2.Workplan{}, config)
}

// validateExistingWorkPlan validates the configured changes on top of the current work plan
func validateExistingWorkPlan(ctx context.Context, proxy *workPlanProxy, managementUnitId string, workPlanId string, config cty.Value) error {
	if !canValidateWorkPlan(managementUnitId, config) {
		return nil
	}
	current, _, err := proxy.getWorkPlanById(ctx, managementUnitId, workPlanId)
	if err != nil {
		return nil
	}
	return validateWorkPlanRequest(ctx, proxy, managementUnitId, workPlanId, current, config)
}

// canValidateWorkPlan reports whether there is enough to validate: the management unit has to be known, and so does the whole configuration
func canValidateWorkPlan(managementUnitId string, config cty.Value) bool {
	return managementUnitId != "" && config.IsWhollyKnown()
}

// validateWorkPlanRequest asks Genesys Cloud to validate the configured changes on top of the current work plan, and fails with the reasons if it is not valid
func validateWorkPlanRequest(ctx context.Context, proxy *workPlanProxy, managementUnitId string, workPlanId string, current *platformclientv2.Workplan, config cty.Value) error {
	request, err := buildUpdateWorkPlanRequest(config, current)
	if err != nil {
		// Ambiguous shift names on the work plan are reported when it is applied
		return nil
	}

	validation, _, err := proxy.validateWorkPlan(ctx, managementUnitId, workPlanId, toValidationRequest(request))
	if err != nil || *validation.Valid {
		return nil
	}

	messages := formatValidationMessages(validation)
	if messages == "" {
		return fmt.Errorf("the work plan is not valid according to Genesys Cloud, and an enabled work plan that is not valid cannot be saved")
	}
	return fmt.Errorf("the work plan is not valid according to Genesys Cloud, and an enabled work plan that is not valid cannot be saved:\n%s", messages)
}

// configurableKeys returns the attributes that can be set in the configuration, which are all of them except the read only ones
func configurableKeys() []string {
	keys := make([]string, 0)
	for key, attribute := range ResourceWorkPlan().Schema {
		if attribute.Optional || attribute.Required {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	return keys
}

// GenerateWorkPlanResource returns terraform config for a work plan, used in tests
func GenerateWorkPlanResource(resourceLabel string, managementUnitId string, name string, attrs ...string) string {
	return fmt.Sprintf(`resource "%s" "%s" {
	management_unit_id = %s
	name               = "%s"
	%s
}
`, ResourceType, resourceLabel, managementUnitId, name, strings.Join(attrs, "\n"))
}
