package wfm

import (
	"math/big"
	"time"

	"github.com/hashicorp/go-cty/cty"
)

/*
Resources with many Optional and Computed attributes send only the values that are present in the configuration, because a value
the user did not configure must not be sent as a zero value. Presence is read from the raw configuration instead of from the merged
resource data, and these functions read it.
*/

// DateLayout is the layout of the dates in the configuration and in the api
const DateLayout = "2006-01-02"

// NullValue is the value of an attribute that is not in the configuration
var NullValue = cty.NullVal(cty.DynamicPseudoType)

// IsPresent returns whether the value is set in the configuration and known
func IsPresent(v cty.Value) bool {
	return !v.IsNull() && v.IsKnown()
}

// AttrOf returns the attribute of an object value, or a null value when the object does not have it
func AttrOf(v cty.Value, name string) cty.Value {
	if !IsPresent(v) || !v.Type().IsObjectType() || !v.Type().HasAttribute(name) {
		return NullValue
	}
	return v.GetAttr(name)
}

// BlockOf returns the single element of a list block with a maximum of one item
func BlockOf(v cty.Value, name string) cty.Value {
	list := AttrOf(v, name)
	if !IsPresent(list) || list.LengthInt() == 0 {
		return NullValue
	}
	return list.AsValueSlice()[0]
}

func StringOf(v cty.Value, name string) *string {
	attr := AttrOf(v, name)
	if !IsPresent(attr) {
		return nil
	}
	value := attr.AsString()
	return &value
}

func IntOf(v cty.Value, name string) *int {
	attr := AttrOf(v, name)
	if !IsPresent(attr) {
		return nil
	}
	value, _ := attr.AsBigFloat().Int64()
	intValue := int(value)
	return &intValue
}

func FloatOf(v cty.Value, name string) *float64 {
	attr := AttrOf(v, name)
	if !IsPresent(attr) {
		return nil
	}
	value, _ := new(big.Float).Set(attr.AsBigFloat()).Float64()
	return &value
}

func BoolOf(v cty.Value, name string) *bool {
	attr := AttrOf(v, name)
	if !IsPresent(attr) {
		return nil
	}
	value := attr.True()
	return &value
}

// StringSliceOf returns the strings of a list or set attribute, each one changed by transform when it is not nil
func StringSliceOf(v cty.Value, name string, transform func(string) string) *[]string {
	attr := AttrOf(v, name)
	if !IsPresent(attr) {
		return nil
	}
	values := make([]string, 0, attr.LengthInt())
	for _, element := range attr.AsValueSlice() {
		if IsPresent(element) {
			value := element.AsString()
			if transform != nil {
				value = transform(value)
			}
			values = append(values, value)
		}
	}
	return &values
}

// DateOf returns the date of a yyyy-MM-dd attribute, or nil when it is not present, empty or not a date
func DateOf(v cty.Value, name string) *time.Time {
	value := StringOf(v, name)
	if value == nil || *value == "" {
		return nil
	}
	parsed, err := time.Parse(DateLayout, *value)
	if err != nil {
		return nil
	}
	return &parsed
}
