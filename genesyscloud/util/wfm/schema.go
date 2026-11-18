package wfm

import (
	"fmt"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// The service fills in a default for every setting that is not sent, so these attributes are Optional and Computed:
// leaving one out accepts the value the service has, and setting one enforces it.

func OptionalInt(description string) *schema.Schema {
	return &schema.Schema{Description: description, Type: schema.TypeInt, Optional: true, Computed: true}
}

func OptionalFloat(description string) *schema.Schema {
	return &schema.Schema{Description: description, Type: schema.TypeFloat, Optional: true, Computed: true}
}

func OptionalBool(description string) *schema.Schema {
	return &schema.Schema{Description: description, Type: schema.TypeBool, Optional: true, Computed: true}
}

func OptionalString(description string) *schema.Schema {
	return &schema.Schema{Description: description, Type: schema.TypeString, Optional: true, Computed: true}
}

func OptionalBlock(description string, elem *schema.Resource) *schema.Schema {
	return &schema.Schema{Description: description, Type: schema.TypeList, Optional: true, Computed: true, MaxItems: 1, Elem: elem}
}

// StringSet returns the strings as a set
func StringSet(values []string) *schema.Set {
	elements := make([]interface{}, len(values))
	for i, value := range values {
		elements[i] = value
	}
	return schema.NewSet(schema.HashString, elements)
}

// ModifyActivityCodeIdStateValue strips the business unit from an activity code id before it is saved in state.
// The activity code resource id is <businessUnitId>/<activityCodeId>, but the apis that refer to an activity code only return the activity code id,
// so saving the composite value would cause a non-empty plan. The default activity codes have no prefix, and are left as they are.
func ModifyActivityCodeIdStateValue(id interface{}) string {
	activityCodeId := id.(string)
	if i := strings.LastIndex(activityCodeId, "/"); i >= 0 {
		return activityCodeId[i+1:]
	}
	return activityCodeId
}

// ValidateDate validates that a value is a date in the yyyy-MM-dd format
func ValidateDate(value interface{}, key string) ([]string, []error) {
	if _, err := time.Parse(DateLayout, value.(string)); err != nil {
		return nil, []error{fmt.Errorf("%q must be a date in the format yyyy-MM-dd, got %q", key, value)}
	}
	return nil, nil
}
