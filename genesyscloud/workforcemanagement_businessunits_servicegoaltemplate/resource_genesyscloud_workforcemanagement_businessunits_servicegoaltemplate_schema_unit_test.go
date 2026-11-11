package workforcemanagement_businessunits_servicegoaltemplate

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUnitValidateAtMostTwoDecimals(t *testing.T) {
	for _, valid := range []float64{0, 5, 10.5, 10.1, 10.13, 99.99, 100} {
		_, errs := validateAtMostTwoDecimals(valid, "increase_by_percent")
		assert.Empty(t, errs, "expected %v to be accepted", valid)
	}
	for _, invalid := range []float64{10.126, 0.001, 33.333} {
		_, errs := validateAtMostTwoDecimals(invalid, "increase_by_percent")
		assert.NotEmpty(t, errs, "expected %v to be rejected", invalid)
	}
}

func TestUnitServiceGoalLimits(t *testing.T) {
	assert.Equal(t, 31622400, maxServiceLevelSeconds)
	assert.Equal(t, 432000, maxAverageSpeedOfAnswerSeconds)
}
