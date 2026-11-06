package workforcemanagement_businessunits_timeofflimit

import (
	"context"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/mypurecloud/platform-client-sdk-go/v200/platformclientv2"
)

const (
	dateLayout       = "2006-01-02"
	oneDay           = 24 * time.Hour
	fifteenMinuteKey = "limit_minutes_per_fifteen_minutes"
	dailyKey         = "limit_minutes_per_day"

	// A day has 96 fifteen minute slots, except on the days of a daylight saving change in the time zone of the business unit, which have
	// 92 or 100. How many days a list of fifteen minute values covers depends on the time zone, so ranges are measured with the
	// bound that is safe for each purpose.
	minSlotsPerDay = 92
	maxSlotsPerDay = 100

	maxDaysPerRange   = 366 // the most days one range covers
	maxDaysPerRequest = 366 // the most days all the ranges of one write or one query cover together
	maxRangesPerQuery = 30  // the most date ranges one query reads
)

// SplitTimeOffLimitTerraformId splits the resource id, which is in the form
// <businessUnitId>/<timeOffLimitId>, into the businessUnitId and timeOffLimitId
func SplitTimeOffLimitTerraformId(id string) (businessUnitId string, limitId string) {
	parts := strings.SplitN(id, "/", 2)
	if len(parts) != 2 {
		return id, ""
	}
	return parts[0], parts[1]
}

func validateDate(value interface{}, key string) ([]string, []error) {
	if _, err := time.Parse(dateLayout, value.(string)); err != nil {
		return nil, []error{fmt.Errorf("%q must be a date in the format yyyy-MM-dd, got %q", key, value)}
	}
	return nil, nil
}

// valueRange is a range of limit values: the minutes per day, or the minutes per fifteen minute interval, from a start date
type valueRange struct {
	start      time.Time
	perDay     []int
	perFifteen []int
}

// days returns the number of days the range covers, given how many fifteen minute slots a day has. Fifteen minute values that run
// past midnight belong to the following days. Fewer slots per day means more days, so minSlotsPerDay gives the longest the range can
// be and maxSlotsPerDay the shortest.
func (r valueRange) days(slotsPerDay int) int {
	if len(r.perDay) > 0 {
		return len(r.perDay)
	}
	return (len(r.perFifteen) + slotsPerDay - 1) / slotsPerDay
}

// end returns the last date the range covers, given how many fifteen minute slots a day has
func (r valueRange) end(slotsPerDay int) time.Time {
	return r.start.Add(time.Duration(maxInt(r.days(slotsPerDay), 1)-1) * oneDay)
}

func (r valueRange) equals(other valueRange) bool {
	return r.start.Equal(other.start) && equalInts(r.perDay, other.perDay) && equalInts(r.perFifteen, other.perFifteen)
}

func maxInt(a int, b int) int {
	if a > b {
		return a
	}
	return b
}

func equalInts(a []int, b []int) bool {
	if len(a) == 0 && len(b) == 0 {
		return true
	}
	return reflect.DeepEqual(a, b)
}

func toInts(value interface{}) []int {
	list, ok := value.([]interface{})
	if !ok {
		return nil
	}
	ints := make([]int, 0, len(list))
	for _, item := range list {
		if number, ok := item.(int); ok {
			ints = append(ints, number)
		}
	}
	return ints
}

// readValueRanges reads the value ranges of a configuration, plan or state
func readValueRanges(value interface{}) []valueRange {
	items, ok := value.([]interface{})
	if !ok {
		return nil
	}
	ranges := make([]valueRange, 0, len(items))
	for _, item := range items {
		entry, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		start, err := time.Parse(dateLayout, fmt.Sprint(entry["start_date"]))
		if err != nil {
			continue
		}
		ranges = append(ranges, valueRange{start: start, perDay: toInts(entry[dailyKey]), perFifteen: toInts(entry[fifteenMinuteKey])})
	}
	return ranges
}

// validateRanges checks the value ranges against the granularity of the limit. A range sets the values for the granularity of the limit,
// covers at most 366 days, and must not overlap another range.
func validateRanges(granularity string, ranges []valueRange) error {
	for _, r := range ranges {
		date := r.start.Format(dateLayout)
		switch {
		case len(r.perDay) == 0 && len(r.perFifteen) == 0:
			return fmt.Errorf("the value range starting on %s must set %s or %s", date, dailyKey, fifteenMinuteKey)
		case len(r.perDay) > 0 && len(r.perFifteen) > 0:
			return fmt.Errorf("the value range starting on %s sets both %s and %s, only one can be set", date, dailyKey, fifteenMinuteKey)
		case len(r.perDay) > 0 && granularity != granularityDaily:
			return fmt.Errorf("the value range starting on %s sets %s, which needs the Daily granularity, but the granularity is %s", date, dailyKey, granularity)
		case len(r.perFifteen) > 0 && granularity != granularityFifteenMinutes:
			return fmt.Errorf("the value range starting on %s sets %s, which needs the FifteenMinutes granularity, but the granularity is %s", date, fifteenMinuteKey, granularity)
		case r.days(maxSlotsPerDay) > maxDaysPerRange:
			return fmt.Errorf("the value range starting on %s covers %d days, but a range covers at most %d days", date, r.days(maxSlotsPerDay), maxDaysPerRange)
		}
	}

	sorted := append([]valueRange(nil), ranges...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].start.Before(sorted[j].start) })
	for i := 1; i < len(sorted); i++ {
		if !sorted[i].start.After(sorted[i-1].end(maxSlotsPerDay)) {
			return fmt.Errorf("the value ranges starting on %s and %s overlap", sorted[i-1].start.Format(dateLayout), sorted[i].start.Format(dateLayout))
		}
	}
	return nil
}

// validateValueRanges rejects value ranges that do not fit the granularity of the limit, or that overlap
func validateValueRanges(_ context.Context, diff *schema.ResourceDiff, _ interface{}) error {
	if !diff.NewValueKnown("value_range") || !diff.NewValueKnown("granularity") {
		return nil
	}
	return validateRanges(diff.Get("granularity").(string), readValueRanges(diff.Get("value_range")))
}

// changedRanges returns the ranges that are new or whose values differ from the same range in the old ranges
func changedRanges(old []valueRange, current []valueRange) []valueRange {
	var changed []valueRange
	for _, candidate := range current {
		found := false
		for _, existing := range old {
			if existing.equals(candidate) {
				found = true
				break
			}
		}
		if !found {
			changed = append(changed, candidate)
		}
	}
	return changed
}

// buildSetValuesRequest builds the request that sets the values of the ranges, with the version of the limit
func buildSetValuesRequest(ranges []valueRange, metadata *platformclientv2.Wfmversionedentitymetadata) *platformclientv2.Busettimeofflimitvaluesrequest {
	values := make([]platformclientv2.Butimeofflimitrange, 0, len(ranges))
	for _, r := range ranges {
		start := r.start
		entry := platformclientv2.Butimeofflimitrange{StartDate: &start}
		if len(r.perDay) > 0 {
			perDay := append([]int(nil), r.perDay...)
			entry.LimitMinutesPerDay = &perDay
		}
		if len(r.perFifteen) > 0 {
			perFifteen := append([]int(nil), r.perFifteen...)
			entry.LimitMinutesPerFifteenMinutes = &perFifteen
		}
		values = append(values, entry)
	}

	request := &platformclientv2.Busettimeofflimitvaluesrequest{Values: &values}
	if metadata != nil && metadata.Version != nil {
		request.Metadata = &platformclientv2.Wfmversionedentitymetadata{Version: metadata.Version}
	}
	return request
}

// chunkRanges groups ranges so that each group stays within what one request accepts: at most maxDaysPerRequest days in total, measured
// with the longest span the ranges can have, and at most maxRanges ranges. A range that is too long for a group on its own is sent alone.
func chunkRanges(ranges []valueRange, maxRanges int) [][]valueRange {
	var chunks [][]valueRange
	var current []valueRange
	currentDays := 0
	for _, r := range ranges {
		days := r.days(minSlotsPerDay)
		if len(current) > 0 && (currentDays+days > maxDaysPerRequest || len(current) >= maxRanges) {
			chunks = append(chunks, current)
			current = nil
			currentDays = 0
		}
		current = append(current, r)
		currentDays += days
	}
	if len(current) > 0 {
		chunks = append(chunks, current)
	}
	return chunks
}

// buildQueryRanges returns the date ranges to query to read the values of the ranges back. They cover the longest span the ranges can
// have, and the values are cut to the length of each range afterwards.
func buildQueryRanges(ranges []valueRange) []platformclientv2.Localdaterange {
	queries := make([]platformclientv2.Localdaterange, 0, len(ranges))
	for _, r := range ranges {
		start := r.start
		end := r.end(minSlotsPerDay)
		queries = append(queries, platformclientv2.Localdaterange{StartDate: &start, EndDate: &end})
	}
	return queries
}

// valuesMatch reports whether the values Genesys Cloud returned are the configured values of every range
func valuesMatch(read *platformclientv2.Butimeofflimitvaluesforgranularityresponse, configured []valueRange) bool {
	for _, r := range configured {
		if !readRangeValues(read, r).equals(r) {
			return false
		}
	}
	return true
}

// readRangeValues returns the values the api has for a configured range, cut to the length of the range.
// The values are looked up by start date, because that is how the api returns the requested ranges.
func readRangeValues(response *platformclientv2.Butimeofflimitvaluesforgranularityresponse, configured valueRange) valueRange {
	result := valueRange{start: configured.start}
	if response == nil || response.LimitValues == nil {
		return result
	}
	for _, values := range *response.LimitValues {
		if values.StartDate == nil || !values.StartDate.Equal(configured.start) {
			continue
		}
		if values.ValuesPerDay != nil && values.ValuesPerDay.LimitMinutes != nil && len(configured.perDay) > 0 {
			result.perDay = firstN(*values.ValuesPerDay.LimitMinutes, len(configured.perDay))
		}
		if values.ValuesPerFifteenMinutes != nil && values.ValuesPerFifteenMinutes.LimitMinutes != nil && len(configured.perFifteen) > 0 {
			result.perFifteen = firstN(*values.ValuesPerFifteenMinutes.LimitMinutes, len(configured.perFifteen))
		}
	}
	return result
}

func firstN(values []int, n int) []int {
	if len(values) > n {
		return append([]int(nil), values[:n]...)
	}
	return append([]int(nil), values...)
}

func flattenValueRanges(ranges []valueRange) []interface{} {
	flattened := make([]interface{}, 0, len(ranges))
	for _, r := range ranges {
		entry := map[string]interface{}{"start_date": r.start.Format(dateLayout)}
		if len(r.perDay) > 0 {
			entry[dailyKey] = intsToInterfaces(r.perDay)
		}
		if len(r.perFifteen) > 0 {
			entry[fifteenMinuteKey] = intsToInterfaces(r.perFifteen)
		}
		flattened = append(flattened, entry)
	}
	return flattened
}

func intsToInterfaces(values []int) []interface{} {
	items := make([]interface{}, len(values))
	for i, value := range values {
		items[i] = value
	}
	return items
}

// GenerateTimeOffLimitResource returns terraform config for a time off limit, used in tests
func GenerateTimeOffLimitResource(resourceLabel string, businessUnitId string, managementUnitId string, granularity string, attrs ...string) string {
	return fmt.Sprintf(`resource "%s" "%s" {
	business_unit_id   = %s
	management_unit_id = %s
	granularity        = "%s"
	%s
}
`, ResourceType, resourceLabel, businessUnitId, managementUnitId, granularity, strings.Join(attrs, "\n"))
}
