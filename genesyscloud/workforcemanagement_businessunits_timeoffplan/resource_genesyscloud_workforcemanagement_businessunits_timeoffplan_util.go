package workforcemanagement_businessunits_timeoffplan

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
)

const dateLayout = "2006-01-02"

// SplitTimeOffPlanTerraformId splits the resource id, which is in the form
// <businessUnitId>/<timeOffPlanId>, into the businessUnitId and timeOffPlanId
func SplitTimeOffPlanTerraformId(id string) (businessUnitId string, planId string) {
	parts := strings.SplitN(id, "/", 2)
	if len(parts) != 2 {
		return id, ""
	}
	return parts[0], parts[1]
}

// ModifyActivityCodeIdStateValue strips the business unit from an activity code id before it is saved in state.
// The activity code resource id is <businessUnitId>/<activityCodeId>, but the time off plan api only returns the activity code id,
// so saving the composite value would cause a non-empty plan. The default activity codes have no prefix, and are left as they are.
func ModifyActivityCodeIdStateValue(id interface{}) string {
	activityCodeId := id.(string)
	if i := strings.LastIndex(activityCodeId, "/"); i >= 0 {
		return activityCodeId[i+1:]
	}
	return activityCodeId
}

/*
The scalar settings have defaults in Genesys Cloud, so they are optional and computed and are built from the raw configuration, which says
whether they were set. The collections and blocks are optional without being computed: what is in the configuration is what the plan has.
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

func setToStrings(value interface{}, transform func(string) string) []string {
	set, ok := value.(*schema.Set)
	if !ok || set == nil {
		return []string{}
	}
	values := make([]string, 0, set.Len())
	for _, item := range set.List() {
		text := fmt.Sprint(item)
		if transform != nil {
			text = transform(text)
		}
		values = append(values, text)
	}
	sort.Strings(values)
	return values
}

func stripActivityCodePrefix(id string) string {
	return ModifyActivityCodeIdStateValue(id)
}

func sameStrings(a []string, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	left := append([]string(nil), a...)
	right := append([]string(nil), b...)
	sort.Strings(left)
	sort.Strings(right)
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}

func stringWrapper(values []string) *platformclientv2.Setwrapperstring {
	copied := append([]string{}, values...)
	return &platformclientv2.Setwrapperstring{Values: &copied}
}

// overrideDate is a date that is checked for conflicts, as it is configured or returned
type overrideDate struct {
	date    string
	dateTyp string
}

func readOverrideDates(value interface{}) []overrideDate {
	set, ok := value.(*schema.Set)
	if !ok || set == nil {
		return nil
	}
	dates := make([]overrideDate, 0, set.Len())
	for _, item := range set.List() {
		entry, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		dates = append(dates, overrideDate{date: fmt.Sprint(entry["date"]), dateTyp: fmt.Sprint(entry["type"])})
	}
	sort.Slice(dates, func(i, j int) bool { return dates[i].date < dates[j].date })
	return dates
}

func currentOverrideDates(dates *[]platformclientv2.Overridedate) []overrideDate {
	current := []overrideDate{}
	if dates == nil {
		return current
	}
	for _, date := range *dates {
		if date.Date != nil && date.VarType != nil {
			current = append(current, overrideDate{date: date.Date.Format(dateLayout), dateTyp: *date.VarType})
		}
	}
	sort.Slice(current, func(i, j int) bool { return current[i].date < current[j].date })
	return current
}

func sameOverrideDates(a []overrideDate, b []overrideDate) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func toOverrideDates(dates []overrideDate) []platformclientv2.Overridedate {
	result := make([]platformclientv2.Overridedate, 0, len(dates))
	for _, entry := range dates {
		parsed, err := time.Parse(dateLayout, entry.date)
		if err != nil {
			continue
		}
		dateType := entry.dateTyp
		result = append(result, platformclientv2.Overridedate{Date: &parsed, VarType: &dateType})
	}
	return result
}

func readHrisType(value interface{}) *platformclientv2.Hristimeofftype {
	list, ok := value.([]interface{})
	if !ok || len(list) == 0 {
		return nil
	}
	entry, ok := list[0].(map[string]interface{})
	if !ok {
		return nil
	}
	hris := &platformclientv2.Hristimeofftype{Id: platformclientv2.String(fmt.Sprint(entry["id"]))}
	if name, ok := entry["name"].(string); ok && name != "" {
		hris.Name = &name
	}
	if integrationId, ok := entry["hris_integration_id"].(string); ok && integrationId != "" {
		hris.HrisIntegrationId = &integrationId
	}
	if secondaryId, ok := entry["secondary_id"].(string); ok && secondaryId != "" {
		hris.SecondaryId = &secondaryId
	}
	return hris
}

func sameHrisType(a *platformclientv2.Hristimeofftype, b *platformclientv2.Hristimeofftype) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return stringValue(a.Id) == stringValue(b.Id) && stringValue(a.Name) == stringValue(b.Name) &&
		stringValue(a.HrisIntegrationId) == stringValue(b.HrisIntegrationId) && stringValue(a.SecondaryId) == stringValue(b.SecondaryId)
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

// associations reads the association blocks of the configuration
type associations struct {
	businessUnit   *businessUnitAssociation
	managementUnit *managementUnitAssociation
}

type businessUnitAssociation struct {
	managementUnitIds []string
	staffingGroupIds  []string
}

type managementUnitAssociation struct {
	managementUnitId string
	staffingGroupIds []string
}

func readAssociations(get func(string) interface{}) associations {
	var result associations
	if list, ok := get("business_unit_association").([]interface{}); ok && len(list) > 0 {
		entry, _ := list[0].(map[string]interface{})
		result.businessUnit = &businessUnitAssociation{
			managementUnitIds: setToStrings(entry["management_unit_ids"], nil),
			staffingGroupIds:  setToStrings(entry["staffing_group_ids"], nil),
		}
	}
	if list, ok := get("management_unit_association").([]interface{}); ok && len(list) > 0 {
		entry, _ := list[0].(map[string]interface{})
		result.managementUnit = &managementUnitAssociation{
			managementUnitId: fmt.Sprint(entry["management_unit_id"]),
			staffingGroupIds: setToStrings(entry["staffing_group_ids"], nil),
		}
	}
	return result
}

func buildCreateRequest(config cty.Value, get func(string) interface{}) *platformclientv2.Bucreatetimeoffplanrequest {
	request := &platformclientv2.Bucreatetimeoffplanrequest{
		Name:                                stringOf(config, "name"),
		AutoApprovalRule:                    stringOf(config, "auto_approval_rule"),
		DaysBeforeStartToExpireFromWaitlist: intOf(config, "days_before_start_to_expire_from_waitlist"),
		AutoPublishApprovedTimeOffRequests:  boolOf(config, "auto_publish_approved_time_off_requests"),
		Enabled:                             boolOf(config, "enabled"),
		CountAgainstTimeOffLimits:           boolOf(config, "count_against_time_off_limits"),
		HrisTimeOffType:                     readHrisType(get("hris_time_off_type")),
	}
	if ids := setToStrings(get("activity_code_ids"), stripActivityCodePrefix); len(ids) > 0 {
		request.ActivityCodeIds = &ids
	}
	if ids := setToStrings(get("restricted_activity_code_ids"), stripActivityCodePrefix); len(ids) > 0 {
		request.RestrictedActivityCodeIds = &ids
	}
	if dates := toOverrideDates(readOverrideDates(get("override_dates"))); len(dates) > 0 {
		request.OverrideDates = &dates
	}

	configured := readAssociations(get)
	if configured.businessUnit != nil {
		association := &platformclientv2.Createtimeoffplanbusinessunitassociation{}
		if len(configured.businessUnit.managementUnitIds) > 0 {
			association.ManagementUnitIds = &configured.businessUnit.managementUnitIds
		}
		if len(configured.businessUnit.staffingGroupIds) > 0 {
			association.StaffingGroupIds = &configured.businessUnit.staffingGroupIds
		}
		request.BusinessUnitAssociation = association
	}
	if configured.managementUnit != nil {
		association := &platformclientv2.Createtimeoffplanmanagementunitassociation{ManagementUnitId: &configured.managementUnit.managementUnitId}
		if len(configured.managementUnit.staffingGroupIds) > 0 {
			association.StaffingGroupIds = &configured.managementUnit.staffingGroupIds
		}
		request.ManagementUnitAssociation = association
	}
	return request
}

func currentIds(references []string) []string {
	if references == nil {
		return []string{}
	}
	return references
}

func managementUnitReferenceIds(references *[]platformclientv2.Managementunitreference) []string {
	ids := []string{}
	if references != nil {
		for _, reference := range *references {
			if reference.Id != nil {
				ids = append(ids, *reference.Id)
			}
		}
	}
	return ids
}

func staffingGroupReferenceIds(references *[]platformclientv2.Staffinggroupreference) []string {
	ids := []string{}
	if references != nil {
		for _, reference := range *references {
			if reference.Id != nil {
				ids = append(ids, *reference.Id)
			}
		}
	}
	return ids
}

// buildUpdateRequest builds the patch for a time off plan from the configuration and its current state. Collections and blocks are only sent
// when they differ from the current plan, because each of them replaces what the plan has.
func buildUpdateRequest(config cty.Value, get func(string) interface{}, current *platformclientv2.Butimeoffplanresponse) *platformclientv2.Buupdatetimeoffplanrequest {
	request := &platformclientv2.Buupdatetimeoffplanrequest{
		Name:                                stringOf(config, "name"),
		AutoApprovalRule:                    stringOf(config, "auto_approval_rule"),
		DaysBeforeStartToExpireFromWaitlist: intOf(config, "days_before_start_to_expire_from_waitlist"),
		AutoPublishApprovedTimeOffRequests:  boolOf(config, "auto_publish_approved_time_off_requests"),
		Enabled:                             boolOf(config, "enabled"),
		CountAgainstTimeOffLimits:           boolOf(config, "count_against_time_off_limits"),
	}

	if ids := setToStrings(get("activity_code_ids"), stripActivityCodePrefix); !sameStrings(ids, currentIds(derefStrings(current.ActivityCodeIds))) {
		request.ActivityCodeIds = stringWrapper(ids)
	}
	// Genesys Cloud keeps restricted activity codes only for the rules that check activity codes: it clears them itself when the rule changes
	// to one that does not, and rejects them when they are sent with such a rule, so they are only sent for the rules that check them.
	rule := stringValue(stringOf(config, "auto_approval_rule"))
	if rule == "" {
		rule = stringValue(current.AutoApprovalRule)
	}
	if checksActivityCodes(rule) {
		var currentRestricted []string
		if current.RestrictedActivityCodes != nil {
			currentRestricted = derefStrings(current.RestrictedActivityCodes.Ids)
		}
		if ids := setToStrings(get("restricted_activity_code_ids"), stripActivityCodePrefix); len(ids) > 0 && !sameStrings(ids, currentIds(currentRestricted)) {
			request.RestrictedActivityCodeIds = stringWrapper(ids)
		}
	}

	if dates := readOverrideDates(get("override_dates")); !sameOverrideDates(dates, currentOverrideDates(current.OverrideDates)) {
		if len(dates) == 0 {
			// Genesys Cloud rejects an empty list of override dates, and clears them when the list is left out of the wrapper
			request.OverrideDates = &platformclientv2.Listwrapperoverridedate{}
		} else {
			converted := toOverrideDates(dates)
			request.OverrideDates = &platformclientv2.Listwrapperoverridedate{Values: &converted}
		}
	}
	if hris := readHrisType(get("hris_time_off_type")); !sameHrisType(hris, current.HrisTimeOffType) {
		wrapper := &platformclientv2.Valuewrapperhristimeofftype{Value: hris}
		if hris == nil {
			// A wrapper whose value is null clears the HRIS time off type, which an omitted value would not
			wrapper.SetField("Value", nil)
		}
		request.HrisTimeOffType = wrapper
	}

	configured := readAssociations(get)
	if configured.businessUnit != nil {
		var currentManagementUnits, currentStaffingGroups []string
		if current.BusinessUnitAssociation != nil {
			currentManagementUnits = managementUnitReferenceIds(current.BusinessUnitAssociation.ManagementUnits)
			currentStaffingGroups = staffingGroupReferenceIds(current.BusinessUnitAssociation.StaffingGroups)
		}
		// Only the wrapper of the kind that is configured is sent: Genesys Cloud uses the staffing groups wrapper whenever it is there,
		// even when it is empty, and ignores the management units wrapper next to it. Sending one replaces the other kind.
		association := &platformclientv2.Updatetimeoffplanbusinessunitassociation{}
		switch {
		case len(configured.businessUnit.staffingGroupIds) > 0:
			if !sameStrings(configured.businessUnit.staffingGroupIds, currentStaffingGroups) || len(currentManagementUnits) > 0 {
				association.StaffingGroupIds = stringWrapper(configured.businessUnit.staffingGroupIds)
			}
		case len(configured.businessUnit.managementUnitIds) > 0:
			if !sameStrings(configured.businessUnit.managementUnitIds, currentManagementUnits) || len(currentStaffingGroups) > 0 {
				association.ManagementUnitIds = stringWrapper(configured.businessUnit.managementUnitIds)
			}
		}
		if association.ManagementUnitIds != nil || association.StaffingGroupIds != nil {
			request.BusinessUnitAssociation = association
		}
	}
	if configured.managementUnit != nil {
		var currentStaffingGroups []string
		if current.ManagementUnitAssociation != nil {
			currentStaffingGroups = staffingGroupReferenceIds(current.ManagementUnitAssociation.StaffingGroups)
		}
		if !sameStrings(configured.managementUnit.staffingGroupIds, currentStaffingGroups) {
			request.ManagementUnitAssociation = &platformclientv2.Updatetimeoffplanmanagementunitassociation{StaffingGroupIds: stringWrapper(configured.managementUnit.staffingGroupIds)}
		}
	}

	// The version of the time off plan is sent for optimistic locking
	if current.Metadata != nil && current.Metadata.Version != nil {
		request.Metadata = &platformclientv2.Wfmversionedentitymetadata{Version: current.Metadata.Version}
	}
	return request
}

func derefStrings(values *[]string) []string {
	if values == nil {
		return nil
	}
	return *values
}

func flattenOverrideDates(dates *[]platformclientv2.Overridedate) *schema.Set {
	flattened := make([]interface{}, 0)
	for _, date := range currentOverrideDates(dates) {
		flattened = append(flattened, map[string]interface{}{"date": date.date, "type": date.dateTyp})
	}
	return schema.NewSet(schema.HashResource(overrideDateResource), flattened)
}

func flattenHrisType(hris *platformclientv2.Hristimeofftype) []interface{} {
	if hris == nil || hris.Id == nil {
		return []interface{}{}
	}
	entry := map[string]interface{}{"id": *hris.Id}
	if hris.Name != nil {
		entry["name"] = *hris.Name
	}
	if hris.HrisIntegrationId != nil {
		entry["hris_integration_id"] = *hris.HrisIntegrationId
	}
	if hris.SecondaryId != nil {
		entry["secondary_id"] = *hris.SecondaryId
	}
	return []interface{}{entry}
}

func flattenStringSet(values []string) *schema.Set {
	items := make([]interface{}, len(values))
	for i, value := range values {
		items[i] = value
	}
	return schema.NewSet(schema.HashString, items)
}

func flattenBusinessUnitAssociation(association *platformclientv2.Timeoffplanbusinessunitassociation) []interface{} {
	if association == nil {
		return []interface{}{}
	}
	managementUnits := managementUnitReferenceIds(association.ManagementUnits)
	staffingGroups := staffingGroupReferenceIds(association.StaffingGroups)
	if len(managementUnits) == 0 && len(staffingGroups) == 0 {
		return []interface{}{}
	}
	return []interface{}{map[string]interface{}{
		"management_unit_ids": flattenStringSet(managementUnits),
		"staffing_group_ids":  flattenStringSet(staffingGroups),
	}}
}

func flattenManagementUnitAssociation(association *platformclientv2.Timeoffplanmanagementunitassociation) []interface{} {
	if association == nil || association.ManagementUnit == nil || association.ManagementUnit.Id == nil {
		return []interface{}{}
	}
	return []interface{}{map[string]interface{}{
		"management_unit_id": *association.ManagementUnit.Id,
		"staffing_group_ids": flattenStringSet(staffingGroupReferenceIds(association.StaffingGroups)),
	}}
}

func flattenLimitIds(limits *[]platformclientv2.Butimeofflimitreference) *schema.Set {
	ids := []string{}
	if limits != nil {
		for _, limit := range *limits {
			if limit.Id != nil {
				ids = append(ids, *limit.Id)
			}
		}
	}
	return flattenStringSet(ids)
}

// checksActivityCodes reports whether the auto approval rule checks activity codes
func checksActivityCodes(rule string) bool {
	return strings.Contains(rule, "ActivityCodes")
}

// checksLimits reports whether the auto approval rule checks time off limits
func checksLimits(rule string) bool {
	return strings.Contains(rule, "Limits")
}

// validateRuleSettings checks the settings that Genesys Cloud ties to the auto approval rule: the rules that check activity codes need
// restricted activity codes and the others do not accept them, and the rules that check limits need the plan to count against limits.
// A rule that is not set is the default, which is Never.
func validateRuleSettings(rule string, restrictedActivityCodes int, countAgainstTimeOffLimits bool) error {
	if rule == "" {
		rule = "Never"
	}
	if checksActivityCodes(rule) && restrictedActivityCodes == 0 {
		return fmt.Errorf("restricted_activity_code_ids must be set when auto_approval_rule is %s", rule)
	}
	if !checksActivityCodes(rule) && restrictedActivityCodes > 0 {
		return fmt.Errorf("restricted_activity_code_ids must not be set when auto_approval_rule is %s, which does not check activity codes", rule)
	}
	if checksLimits(rule) && !countAgainstTimeOffLimits {
		return fmt.Errorf("count_against_time_off_limits must be true when auto_approval_rule is %s", rule)
	}
	return nil
}

const (
	// maxOverrideDatesPastDays and maxOverrideDates are the limits Genesys Cloud enforces for the override dates of a time off plan in a customer
	// organization (timeoff.plan.past.override.days.max and timeoff.plan.override.days.max, both fixed). Organizations used for testing have
	// tighter limits, which Genesys Cloud reports when the plan is saved.
	maxOverrideDatesPastDays = 93
	maxOverrideDates         = 550
)

// validateOverrideDatesWindow rejects override dates that Genesys Cloud would refuse when the plan is saved: too many of them, or a date that is
// too far in the past. It is only used when the override dates change, because dates that were saved earlier and have since aged out are only
// refused when the list is sent again, so a configuration that did not change must keep planning.
func validateOverrideDatesWindow(dates []overrideDate, today time.Time) error {
	if len(dates) > maxOverrideDates {
		return fmt.Errorf("a time off plan can have at most %d override dates, but %d are configured", maxOverrideDates, len(dates))
	}
	earliest := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, time.UTC).AddDate(0, 0, -maxOverrideDatesPastDays)
	for _, entry := range dates {
		parsed, err := time.Parse(dateLayout, entry.date)
		if err != nil {
			continue
		}
		if parsed.Before(earliest) {
			return fmt.Errorf("override date %s is more than %d days in the past, which Genesys Cloud does not accept: remove the dates that have passed from override_dates", entry.date, maxOverrideDatesPastDays)
		}
	}
	return nil
}

// validateOverrideDatesAreUnique rejects an override date that is listed twice, which Genesys Cloud rejects, even with different types
func validateOverrideDatesAreUnique(dates []overrideDate) error {
	seen := make(map[string]bool, len(dates))
	for _, entry := range dates {
		if seen[entry.date] {
			return fmt.Errorf("override date %s is listed more than once, and each date can only be listed once", entry.date)
		}
		seen[entry.date] = true
	}
	return nil
}

// validateRules checks the settings tied to the auto approval rule while planning
func validateRules(_ context.Context, diff *schema.ResourceDiff, _ interface{}) error {
	for _, key := range []string{"auto_approval_rule", "restricted_activity_code_ids", "count_against_time_off_limits"} {
		if !diff.NewValueKnown(key) {
			return nil
		}
	}
	if diff.NewValueKnown("override_dates") {
		dates := readOverrideDates(diff.Get("override_dates"))
		if err := validateOverrideDatesAreUnique(dates); err != nil {
			return err
		}
		if diff.Id() == "" || diff.HasChange("override_dates") {
			if err := validateOverrideDatesWindow(dates, time.Now()); err != nil {
				return err
			}
		}
	}
	restricted := setToStrings(diff.Get("restricted_activity_code_ids"), nil)
	return validateRuleSettings(diff.Get("auto_approval_rule").(string), len(restricted), diff.Get("count_against_time_off_limits").(bool))
}

// validateAssociations checks the associations: a business unit association needs exactly one of management units and staffing groups
func validateAssociations(_ context.Context, diff *schema.ResourceDiff, _ interface{}) error {
	if !diff.NewValueKnown("business_unit_association") {
		return nil
	}
	list, ok := diff.Get("business_unit_association").([]interface{})
	if !ok || len(list) == 0 {
		return nil
	}
	entry, _ := list[0].(map[string]interface{})
	managementUnits := len(setToStrings(entry["management_unit_ids"], nil))
	staffingGroups := len(setToStrings(entry["staffing_group_ids"], nil))
	switch {
	case managementUnits > 0 && staffingGroups > 0:
		return fmt.Errorf("business_unit_association can set either management_unit_ids or staffing_group_ids, not both")
	case managementUnits == 0 && staffingGroups == 0:
		return fmt.Errorf("business_unit_association must set one of management_unit_ids and staffing_group_ids")
	}
	return nil
}

// forceNewWhenOwnerChanges recreates the time off plan when it would change owner. Genesys Cloud fixes whether a time off plan belongs to the
// business unit or to a management unit when it is created, and cannot move it to another management unit.
func forceNewWhenOwnerChanges(_ context.Context, diff *schema.ResourceDiff, _ interface{}) error {
	if diff.Id() == "" {
		return nil
	}

	const managementUnitKey = "management_unit_association.0.management_unit_id"
	if diff.HasChange(managementUnitKey) {
		if oldValue, _ := diff.GetChange(managementUnitKey); oldValue.(string) != "" {
			return diff.ForceNew(managementUnitKey)
		}
	}

	hasBlock := func(value interface{}) bool {
		list, ok := value.([]interface{})
		return ok && len(list) > 0
	}
	oldBusinessUnit, newBusinessUnit := diff.GetChange("business_unit_association")
	oldManagementUnit, newManagementUnit := diff.GetChange("management_unit_association")
	switchedToManagementUnit := hasBlock(oldBusinessUnit) && hasBlock(newManagementUnit)
	switchedToBusinessUnit := hasBlock(oldManagementUnit) && hasBlock(newBusinessUnit)
	if switchedToManagementUnit && diff.HasChange("management_unit_association") {
		return diff.ForceNew("management_unit_association")
	}
	if switchedToBusinessUnit && diff.HasChange("business_unit_association") {
		return diff.ForceNew("business_unit_association")
	}
	return nil
}

// GenerateTimeOffPlanResource returns terraform config for a time off plan, used in tests
func GenerateTimeOffPlanResource(resourceLabel string, businessUnitId string, name string, attrs ...string) string {
	return fmt.Sprintf(`resource "%s" "%s" {
	business_unit_id = %s
	name             = "%s"
	%s
}
`, ResourceType, resourceLabel, businessUnitId, name, strings.Join(attrs, "\n"))
}
