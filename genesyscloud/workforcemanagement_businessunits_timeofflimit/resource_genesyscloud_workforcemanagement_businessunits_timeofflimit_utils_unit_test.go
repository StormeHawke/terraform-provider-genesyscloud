package workforcemanagement_businessunits_timeofflimit

import (
	"context"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/mypurecloud/platform-client-sdk-go/v200/platformclientv2"
)

func day(value string) time.Time {
	parsed, _ := time.Parse(dateLayout, value)
	return parsed
}

func dailyRange(start string, values ...int) valueRange {
	return valueRange{start: day(start), perDay: values}
}

func TestUnitSplitTimeOffLimitTerraformId(t *testing.T) {
	businessUnitId, limitId := SplitTimeOffLimitTerraformId("bu-1/limit-2")
	if businessUnitId != "bu-1" || limitId != "limit-2" {
		t.Errorf("expected bu-1 and limit-2, got %s and %s", businessUnitId, limitId)
	}
}

func TestUnitValueRangeCoversTheRightDates(t *testing.T) {
	daily := dailyRange("2026-03-01", 10, 20, 30)
	if daily.days(minSlotsPerDay) != 3 || daily.end(maxSlotsPerDay).Format(dateLayout) != "2026-03-03" {
		t.Errorf("expected 3 days ending 2026-03-03, got %d ending %s", daily.days(minSlotsPerDay), daily.end(maxSlotsPerDay).Format(dateLayout))
	}

	// 190 fifteen minute slots cover 2 days of 96 slots, but 3 days when the days are daylight saving days of 92 slots,
	// and 2 days when they are days of 100 slots
	fifteen := valueRange{start: day("2026-03-01"), perFifteen: make([]int, 190)}
	if fifteen.days(maxSlotsPerDay) != 2 || fifteen.days(minSlotsPerDay) != 3 {
		t.Errorf("expected the shortest span to be 2 days and the longest 3, got %d and %d", fifteen.days(maxSlotsPerDay), fifteen.days(minSlotsPerDay))
	}
	if fifteen.end(minSlotsPerDay).Format(dateLayout) != "2026-03-03" {
		t.Errorf("expected the longest span to end on 2026-03-03, got %s", fifteen.end(minSlotsPerDay).Format(dateLayout))
	}
}

func TestUnitValidateRanges(t *testing.T) {
	valid := []valueRange{dailyRange("2026-03-01", 1, 2), dailyRange("2026-03-03", 3)}
	if err := validateRanges(granularityDaily, valid); err != nil {
		t.Errorf("expected touching ranges that do not overlap to be valid, got %v", err)
	}

	cases := map[string]struct {
		granularity string
		ranges      []valueRange
		message     string
	}{
		"overlap":                {granularityDaily, []valueRange{dailyRange("2026-03-01", 1, 2, 3), dailyRange("2026-03-03", 4)}, "overlap"},
		"empty":                  {granularityDaily, []valueRange{{start: day("2026-03-01")}}, "must set"},
		"both lists":             {granularityDaily, []valueRange{{start: day("2026-03-01"), perDay: []int{1}, perFifteen: []int{1}}}, "only one can be set"},
		"daily on 15":            {granularityFifteenMinutes, []valueRange{dailyRange("2026-03-01", 1)}, "needs the Daily granularity"},
		"15 on daily":            {granularityDaily, []valueRange{{start: day("2026-03-01"), perFifteen: []int{1}}}, "needs the FifteenMinutes granularity"},
		"too many days":          {granularityDaily, []valueRange{{start: day("2026-03-01"), perDay: make([]int, 367)}}, "at most 366"},
		"too many 15 mins":       {granularityFifteenMinutes, []valueRange{{start: day("2026-03-01"), perFifteen: make([]int, 367*maxSlotsPerDay)}}, "at most 366"},
		"15 mins that could fit": {granularityFifteenMinutes, []valueRange{{start: day("2026-03-01"), perFifteen: make([]int, 366*96)}}, ""},
	}
	for name, test := range cases {
		err := validateRanges(test.granularity, test.ranges)
		if test.message == "" {
			if err != nil {
				t.Errorf("%s: expected no error, got %v", name, err)
			}
			continue
		}
		if err == nil || !strings.Contains(err.Error(), test.message) {
			t.Errorf("%s: expected an error containing %q, got %v", name, test.message, err)
		}
	}
}

func TestUnitChangedRangesOnlyReturnsNewOrDifferentRanges(t *testing.T) {
	old := []valueRange{dailyRange("2026-03-01", 1, 2), dailyRange("2026-04-01", 5)}
	current := []valueRange{dailyRange("2026-03-01", 1, 2), dailyRange("2026-04-01", 6), dailyRange("2026-05-01", 7)}

	changed := changedRanges(old, current)
	if len(changed) != 2 || changed[0].start.Format(dateLayout) != "2026-04-01" || changed[1].start.Format(dateLayout) != "2026-05-01" {
		t.Errorf("expected only the changed and the new range, got %+v", changed)
	}
}

func TestUnitBuildSetValuesRequestKeepsTheGranularityOfEachRange(t *testing.T) {
	request := buildSetValuesRequest(
		[]valueRange{dailyRange("2026-03-01", 10, 20), {start: day("2026-03-05"), perFifteen: []int{15, 30}}},
		&platformclientv2.Wfmversionedentitymetadata{Version: platformclientv2.Int(12)},
	)

	values := *request.Values
	if len(values) != 2 || !reflect.DeepEqual(*values[0].LimitMinutesPerDay, []int{10, 20}) || values[0].LimitMinutesPerFifteenMinutes != nil {
		t.Errorf("unexpected daily range %+v", values[0])
	}
	if !reflect.DeepEqual(*values[1].LimitMinutesPerFifteenMinutes, []int{15, 30}) || values[1].LimitMinutesPerDay != nil {
		t.Errorf("unexpected fifteen minute range %+v", values[1])
	}
	if *request.Metadata.Version != 12 {
		t.Errorf("expected the version of the limit to be sent, got %+v", request.Metadata)
	}
}

func TestUnitBuildQueryRangesCoverTheLongestSpan(t *testing.T) {
	queries := buildQueryRanges([]valueRange{dailyRange("2026-03-01", 1, 2, 3), {start: day("2026-06-01"), perFifteen: make([]int, 96)}})
	if queries[0].EndDate.Format(dateLayout) != "2026-03-03" {
		t.Errorf("unexpected query for the daily range %+v", queries[0])
	}
	// 96 slots take 2 days when the first day has 92 slots
	if queries[1].EndDate.Format(dateLayout) != "2026-06-02" {
		t.Errorf("expected the fifteen minute range to be read over 2 days, got %+v", queries[1])
	}
}

func TestUnitChunkRangesKeepsEachRequestWithinTheLimits(t *testing.T) {
	ranges := []valueRange{
		{start: day("2026-01-01"), perDay: make([]int, 200)},
		{start: day("2026-08-01"), perDay: make([]int, 100)},
		{start: day("2026-12-01"), perDay: make([]int, 100)},
		{start: day("2027-04-01"), perDay: make([]int, 5)},
	}

	chunks := chunkRanges(ranges, maxRangesPerQuery)
	if len(chunks) != 2 || len(chunks[0]) != 2 || len(chunks[1]) != 2 {
		t.Fatalf("expected 2 requests of 2 ranges (300 and 105 days), got %d chunks", len(chunks))
	}

	var many []valueRange
	for i := 0; i < 35; i++ {
		many = append(many, dailyRange("2026-01-01", 1))
	}
	if chunks := chunkRanges(many, maxRangesPerQuery); len(chunks) != 2 || len(chunks[0]) != 30 {
		t.Errorf("expected a query to read at most 30 ranges, got %d chunks", len(chunks))
	}

	if chunks := chunkRanges([]valueRange{{start: day("2026-01-01"), perDay: make([]int, 366)}, dailyRange("2027-06-01", 1)}, 10); len(chunks) != 2 {
		t.Errorf("a range of a full 366 days must be sent on its own, got %d chunks", len(chunks))
	}
}

func TestUnitReadRangeValuesMatchesByStartDateAndCutsToTheConfiguredLength(t *testing.T) {
	response := &platformclientv2.Butimeofflimitvaluesforgranularityresponse{LimitValues: &[]platformclientv2.Butimeofflimitvalues{
		{StartDate: timePointer("2026-06-01"), ValuesPerDay: &platformclientv2.Timeofflimitvalues{LimitMinutes: &[]int{1, 1, 1}}},
		{StartDate: timePointer("2026-03-01"), ValuesPerDay: &platformclientv2.Timeofflimitvalues{LimitMinutes: &[]int{10, 20, 30, 40}}},
	}}

	got := readRangeValues(response, dailyRange("2026-03-01", 0, 0, 0))
	if !reflect.DeepEqual(got.perDay, []int{10, 20, 30}) {
		t.Errorf("expected the first 3 values of the range that starts on 2026-03-01, got %v", got.perDay)
	}
	if missing := readRangeValues(response, dailyRange("2027-01-01", 0)); len(missing.perDay) != 0 {
		t.Errorf("a range the api did not return must have no values, got %v", missing.perDay)
	}
}

func timePointer(value string) *time.Time {
	parsed := day(value)
	return &parsed
}

func TestUnitReadAndFlattenValueRangesRoundTrip(t *testing.T) {
	flattened := flattenValueRanges([]valueRange{dailyRange("2026-03-01", 10, 20), {start: day("2026-03-05"), perFifteen: []int{15}}})
	ranges := readValueRanges(flattened)
	if len(ranges) != 2 || !ranges[0].equals(dailyRange("2026-03-01", 10, 20)) || !reflect.DeepEqual(ranges[1].perFifteen, []int{15}) {
		t.Errorf("unexpected ranges after a round trip %+v", ranges)
	}
}

func TestUnitLimitLabelNamesTheManagementUnitOrStaffingGroup(t *testing.T) {
	names := map[string]string{"mu-1": "Contact Center"}
	managementUnit := platformclientv2.Butimeofflimitresponse{Id: platformclientv2.String("limit"), ManagementUnit: &platformclientv2.Managementunitreference{Id: platformclientv2.String("mu-1")}}
	staffingGroup := platformclientv2.Butimeofflimitresponse{Id: platformclientv2.String("limit"),
		StaffingGroup:  &platformclientv2.Staffinggroupreference{Id: platformclientv2.String("sg-1")},
		ManagementUnit: &platformclientv2.Managementunitreference{Id: platformclientv2.String("mu-1")}}

	if got := limitLabel(managementUnit, names); got != "Contact Center" {
		t.Errorf("expected the management unit name, got %s", got)
	}
	if got := limitLabel(staffingGroup, names); got != "staffinggroup_sg-1" {
		t.Errorf("a limit of a staffing group must be named after the staffing group, got %s", got)
	}
}

func TestUnitSetValuesReadsTheVersionAgainAfterAConflict(t *testing.T) {
	versions := []int{1, 2}
	reads := 0
	var sent []int
	proxy := &timeOffLimitProxy{
		getTimeOffLimitByIdAttr: func(ctx context.Context, p *timeOffLimitProxy, businessUnitId string, limitId string) (*platformclientv2.Butimeofflimitresponse, *platformclientv2.APIResponse, error) {
			version := versions[reads]
			reads++
			return &platformclientv2.Butimeofflimitresponse{Metadata: &platformclientv2.Wfmversionedentitymetadata{Version: &version}}, nil, nil
		},
		setLimitValuesAttr: func(ctx context.Context, p *timeOffLimitProxy, businessUnitId string, limitId string, values *platformclientv2.Busettimeofflimitvaluesrequest) (*platformclientv2.Butimeofflimitresponse, *platformclientv2.APIResponse, error) {
			sent = append(sent, *values.Metadata.Version)
			if len(sent) == 1 {
				return nil, &platformclientv2.APIResponse{StatusCode: http.StatusConflict}, context.DeadlineExceeded
			}
			return nil, nil, nil
		},
	}

	if diagErr := setValues(context.Background(), proxy, "bu", "limit", []valueRange{dailyRange("2026-03-01", 1)}); diagErr != nil {
		t.Fatalf("unexpected error: %v", diagErr)
	}
	if !reflect.DeepEqual(sent, []int{1, 2}) {
		t.Errorf("expected the values to be sent again with the new version, got %v", sent)
	}
}

func TestUnitSetValuesSplitsRequestsAtTheDayLimit(t *testing.T) {
	var sent [][]int
	proxy := &timeOffLimitProxy{
		getTimeOffLimitByIdAttr: func(ctx context.Context, p *timeOffLimitProxy, businessUnitId string, limitId string) (*platformclientv2.Butimeofflimitresponse, *platformclientv2.APIResponse, error) {
			return &platformclientv2.Butimeofflimitresponse{Metadata: &platformclientv2.Wfmversionedentitymetadata{Version: platformclientv2.Int(1)}}, nil, nil
		},
		setLimitValuesAttr: func(ctx context.Context, p *timeOffLimitProxy, businessUnitId string, limitId string, values *platformclientv2.Busettimeofflimitvaluesrequest) (*platformclientv2.Butimeofflimitresponse, *platformclientv2.APIResponse, error) {
			lengths := []int{}
			for _, entry := range *values.Values {
				lengths = append(lengths, len(*entry.LimitMinutesPerDay))
			}
			sent = append(sent, lengths)
			return nil, nil, nil
		},
	}

	ranges := []valueRange{
		{start: day("2026-01-01"), perDay: make([]int, 200)},
		{start: day("2026-08-01"), perDay: make([]int, 200)},
		{start: day("2027-03-01"), perDay: make([]int, 100)},
	}
	if diagErr := setValues(context.Background(), proxy, "bu", "limit", ranges); diagErr != nil {
		t.Fatalf("unexpected error: %v", diagErr)
	}
	if !reflect.DeepEqual(sent, [][]int{{200}, {200, 100}}) {
		t.Errorf("expected the values to be sent in 2 requests of at most 366 days, got %v", sent)
	}
}

func TestUnitQueryManagedValuesCombinesTheQueries(t *testing.T) {
	queries := 0
	proxy := &timeOffLimitProxy{
		queryLimitValuesAttr: func(ctx context.Context, p *timeOffLimitProxy, businessUnitId string, limitId string, dateRanges []platformclientv2.Localdaterange) (*platformclientv2.Butimeofflimitvaluesforgranularityresponse, *platformclientv2.APIResponse, error) {
			queries++
			values := make([]platformclientv2.Butimeofflimitvalues, 0, len(dateRanges))
			for _, dateRange := range dateRanges {
				values = append(values, platformclientv2.Butimeofflimitvalues{StartDate: dateRange.StartDate, ValuesPerDay: &platformclientv2.Timeofflimitvalues{LimitMinutes: &[]int{queries}}})
			}
			return &platformclientv2.Butimeofflimitvaluesforgranularityresponse{LimitValues: &values}, nil, nil
		},
	}

	managed := []valueRange{
		{start: day("2026-01-01"), perDay: make([]int, 300)},
		{start: day("2026-12-01"), perDay: make([]int, 100)},
	}
	combined, _, diagErr := queryManagedValues(context.Background(), proxy, "bu", "limit", managed)
	if diagErr != nil {
		t.Fatalf("unexpected error: %v", diagErr)
	}
	if queries != 2 || len(*combined.LimitValues) != 2 {
		t.Errorf("expected 2 queries whose results are combined, got %d queries and %d results", queries, len(*combined.LimitValues))
	}
	if got := readRangeValues(combined, valueRange{start: day("2026-12-01"), perDay: make([]int, 100)}); !reflect.DeepEqual(got.perDay, []int{2}) {
		t.Errorf("expected the range to take its values from the second query, got %v", got.perDay)
	}
}

// verificationProxy simulates a time off limit whose values are purged after they are set, a number of times, and records how often they are set
func verificationProxy(purgedSets int, sets *int, configured valueRange) *timeOffLimitProxy {
	return &timeOffLimitProxy{
		getTimeOffLimitByIdAttr: func(ctx context.Context, p *timeOffLimitProxy, businessUnitId string, limitId string) (*platformclientv2.Butimeofflimitresponse, *platformclientv2.APIResponse, error) {
			return &platformclientv2.Butimeofflimitresponse{Metadata: &platformclientv2.Wfmversionedentitymetadata{Version: platformclientv2.Int(1)}}, nil, nil
		},
		setLimitValuesAttr: func(ctx context.Context, p *timeOffLimitProxy, businessUnitId string, limitId string, values *platformclientv2.Busettimeofflimitvaluesrequest) (*platformclientv2.Butimeofflimitresponse, *platformclientv2.APIResponse, error) {
			*sets++
			return nil, nil, nil
		},
		queryLimitValuesAttr: func(ctx context.Context, p *timeOffLimitProxy, businessUnitId string, limitId string, dateRanges []platformclientv2.Localdaterange) (*platformclientv2.Butimeofflimitvaluesforgranularityresponse, *platformclientv2.APIResponse, error) {
			values := append([]int(nil), configured.perDay...)
			if *sets <= purgedSets {
				values = make([]int, len(configured.perDay)) // purged: every day is back to 0
			}
			return &platformclientv2.Butimeofflimitvaluesforgranularityresponse{LimitValues: &[]platformclientv2.Butimeofflimitvalues{
				{StartDate: dateRanges[0].StartDate, ValuesPerDay: &platformclientv2.Timeofflimitvalues{LimitMinutes: &values}},
			}}, nil, nil
		},
	}
}

func withoutVerificationDelay(t *testing.T) {
	previousDelay := valueVerificationDelay
	valueVerificationDelay = 0
	t.Cleanup(func() { valueVerificationDelay = previousDelay })
}

func TestUnitVerifyValuesSettledSetsPurgedValuesAgain(t *testing.T) {
	withoutVerificationDelay(t)
	configured := dailyRange("2026-03-01", 480, 480, 960)
	sets := 1 // the values were just set once by the create; the purge of the previous limit then removes them
	proxy := verificationProxy(1, &sets, configured)

	if diagErr := verifyValuesSettled(context.Background(), proxy, "bu", "limit", []valueRange{configured}); diagErr != nil {
		t.Fatalf("unexpected error: %v", diagErr)
	}
	if sets != 2 {
		t.Errorf("expected the purged values to be set once more, got %d sets in total", sets)
	}
}

func TestUnitVerifyValuesSettledLeavesValuesThatAreThere(t *testing.T) {
	withoutVerificationDelay(t)
	configured := dailyRange("2026-03-01", 480, 480, 960)
	sets := 1
	proxy := verificationProxy(0, &sets, configured)

	if diagErr := verifyValuesSettled(context.Background(), proxy, "bu", "limit", []valueRange{configured}); diagErr != nil {
		t.Fatalf("unexpected error: %v", diagErr)
	}
	if sets != 1 {
		t.Errorf("values that are there must not be set again, got %d sets in total", sets)
	}
}

func TestUnitVerifyValuesSettledGivesUpAfterTheAttempts(t *testing.T) {
	withoutVerificationDelay(t)
	configured := dailyRange("2026-03-01", 480)
	sets := 1
	proxy := verificationProxy(100, &sets, configured) // the values never stay

	if diagErr := verifyValuesSettled(context.Background(), proxy, "bu", "limit", []valueRange{configured}); diagErr != nil {
		t.Fatalf("giving up must not fail the apply, the next plan shows the missing values: %v", diagErr)
	}
	if sets != 1+valueVerificationAttempts {
		t.Errorf("expected the values to be set again after every failed check, got %d sets in total", sets)
	}
}

func TestUnitQueryManagedValuesKeepsRangesOutsideTheWindowInsteadOfFailing(t *testing.T) {
	proxy := &timeOffLimitProxy{
		queryLimitValuesAttr: func(ctx context.Context, p *timeOffLimitProxy, businessUnitId string, limitId string, dateRanges []platformclientv2.Localdaterange) (*platformclientv2.Butimeofflimitvaluesforgranularityresponse, *platformclientv2.APIResponse, error) {
			values := make([]platformclientv2.Butimeofflimitvalues, 0, len(dateRanges))
			for _, dateRange := range dateRanges {
				// Genesys Cloud rejects the whole query when one of its ranges is outside the window
				if dateRange.StartDate.Before(day("2024-01-01")) {
					return nil, &platformclientv2.APIResponse{StatusCode: http.StatusBadRequest, ErrorMessage: "Date range should not go past (1834) days back and (1100) days forward from current date"}, context.DeadlineExceeded
				}
				values = append(values, platformclientv2.Butimeofflimitvalues{StartDate: dateRange.StartDate, ValuesPerDay: &platformclientv2.Timeofflimitvalues{LimitMinutes: &[]int{7}}})
			}
			return &platformclientv2.Butimeofflimitvaluesforgranularityresponse{LimitValues: &values}, nil, nil
		},
	}

	managed := []valueRange{dailyRange("2020-01-01", 1), dailyRange("2026-03-01", 1)}
	combined, unreadable, diagErr := queryManagedValues(context.Background(), proxy, "bu", "limit", managed)
	if diagErr != nil {
		t.Fatalf("a range outside the window must not fail the read, got %v", diagErr)
	}
	if !unreadable["2020-01-01"] || unreadable["2026-03-01"] {
		t.Errorf("expected only the old range to be unreadable, got %v", unreadable)
	}
	if got := readRangeValues(combined, managed[1]); !reflect.DeepEqual(got.perDay, []int{7}) {
		t.Errorf("expected the range inside the window to still be read, got %v", got.perDay)
	}
}

func TestUnitQueryManagedValuesStillFailsOnOtherErrors(t *testing.T) {
	proxy := &timeOffLimitProxy{
		queryLimitValuesAttr: func(ctx context.Context, p *timeOffLimitProxy, businessUnitId string, limitId string, dateRanges []platformclientv2.Localdaterange) (*platformclientv2.Butimeofflimitvaluesforgranularityresponse, *platformclientv2.APIResponse, error) {
			return nil, &platformclientv2.APIResponse{StatusCode: http.StatusBadRequest, ErrorMessage: "Time-off limit is invalid"}, context.DeadlineExceeded
		},
	}
	if _, _, diagErr := queryManagedValues(context.Background(), proxy, "bu", "limit", []valueRange{dailyRange("2026-03-01", 1)}); diagErr == nil || !diagErr.HasError() {
		t.Error("an error that is not about the window must still fail the read")
	}
}
