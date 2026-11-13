package workforcemanagement_businessunits_timeofflimit

import (
	"regexp"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/mypurecloud/terraform-provider-genesyscloud/genesyscloud/provider"
	resourceExporter "github.com/mypurecloud/terraform-provider-genesyscloud/genesyscloud/resource_exporter"
	registrar "github.com/mypurecloud/terraform-provider-genesyscloud/genesyscloud/resource_register"
)

/*
resource_genesyscloud_workforcemanagement_businessunits_timeofflimit_schema.go holds three functions within it:

1.  The registration code that registers the Resource and Exporter for the package.
2.  The resource schema definitions for the workforcemanagement_businessunits_timeofflimit resource.
3.  The resource exporter configuration for the workforcemanagement_businessunits_timeofflimit exporter.

There is no data source: a time off limit has no name to look it up by.
*/
const ResourceType = "genesyscloud_workforcemanagement_businessunits_timeofflimit"

const (
	granularityDaily          = "Daily"
	granularityFifteenMinutes = "FifteenMinutes"
)

// SetRegistrar registers all of the resources and exporters in the package
func SetRegistrar(regInstance registrar.Registrar) {
	regInstance.RegisterResource(ResourceType, ResourceTimeOffLimit())
	regInstance.RegisterExporter(ResourceType, TimeOffLimitExporter())
}

var valueRangeResource = &schema.Resource{
	Schema: map[string]*schema.Schema{
		"start_date": {
			Description:  `The first date of the range, in the format yyyy-MM-dd. The range ends on the last date covered by the list of limit minutes that is set.`,
			Type:         schema.TypeString,
			Required:     true,
			ValidateFunc: validateDate,
		},
		"limit_minutes_per_day": {
			Description: `The time off limit in minutes for each day of the range, starting on start_date. Use it when the granularity is Daily. A day without a limit has a value of 0. A range covers at most 366 days.`,
			Type:        schema.TypeList,
			Optional:    true,
			Elem:        &schema.Schema{Type: schema.TypeInt},
		},
		"limit_minutes_per_fifteen_minutes": {
			Description: `The time off limit in minutes for each fifteen minute interval of the range, starting at midnight of start_date, 96 intervals per day. Use it when the granularity is FifteenMinutes. A range covers at most 366 days.`,
			Type:        schema.TypeList,
			Optional:    true,
			Elem:        &schema.Schema{Type: schema.TypeInt},
		},
	},
}

// ResourceTimeOffLimit registers the genesyscloud_workforcemanagement_businessunits_timeofflimit resource with Terraform
func ResourceTimeOffLimit() *schema.Resource {
	return &schema.Resource{
		Description: `Genesys Cloud workforce management business unit time off limit, which limits the time off agents of a management unit or staffing group can take. Genesys Cloud allows one time off limit for each management unit and one for each staffing group, and it creates the id of the time off limit from theirs, so creating one that already exists fails: import it instead. Genesys Cloud cannot change a time off limit once it is created, so changing any attribute other than value_range drops the time off limit, with all of its values, and creates a new one. Deleting a time off limit purges its values in the background, so after a new time off limit replaces a deleted one the values are checked a few times and set again if they were purged.`,

		CreateContext: provider.CreateWithPooledClient(createTimeOffLimit),
		ReadContext:   provider.ReadWithPooledClient(readTimeOffLimit),
		UpdateContext: provider.UpdateWithPooledClient(updateTimeOffLimit),
		DeleteContext: provider.DeleteWithPooledClient(deleteTimeOffLimit),
		CustomizeDiff: validateValueRanges,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		SchemaVersion: 1,
		Schema: map[string]*schema.Schema{
			"business_unit_id": {
				Description: `The id of the business unit this time off limit belongs to. Changing this attribute will cause the time off limit to be dropped and recreated.`,
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
			},
			"management_unit_id": {
				Description:  `The id of the management unit this time off limit applies to. Exactly one of management_unit_id and staffing_group_id must be set. Changing this attribute will cause the time off limit to be dropped and recreated.`,
				Type:         schema.TypeString,
				Optional:     true,
				ForceNew:     true,
				ExactlyOneOf: []string{"management_unit_id", "staffing_group_id"},
			},
			"staffing_group_id": {
				Description:  `The id of the staffing group, at business unit or management unit level, this time off limit applies to. Exactly one of management_unit_id and staffing_group_id must be set. Changing this attribute will cause the time off limit to be dropped and recreated.`,
				Type:         schema.TypeString,
				Optional:     true,
				ForceNew:     true,
				ExactlyOneOf: []string{"management_unit_id", "staffing_group_id"},
			},
			"granularity": {
				Description: `The granularity of the time off limit. Valid values: Daily, FifteenMinutes. Changing this attribute will cause the time off limit to be dropped and recreated.`,
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
			},
			"full_day_time_off_start_time": {
				Description:  `The start time, in the format HH:mm, of the full day time off requests that count against this time off limit. It can only be set when the granularity is FifteenMinutes. Changing this attribute will cause the time off limit to be dropped and recreated.`,
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				ForceNew:     true,
				ValidateFunc: validation.StringMatch(regexp.MustCompile(`^([01][0-9]|2[0-3]):[0-5][0-9]$`), "must be a time in the format HH:mm"),
			},
			"value_range": {
				Description: `Ranges of limit values to set. This is optional: the limit values can also be managed outside of Terraform, for example by import or in the UI. Only the ranges listed here are managed. Their values are set and checked for changes, and values on dates outside them are neither read nor changed. Removing a range from the configuration stops managing it and leaves its values as they are. Value ranges are not read when the time off limit is imported. Genesys Cloud only accepts and reads dates in a limited window around today. A range that has fallen outside the window can no longer be read, so it is kept as it is and not checked for changes, but it cannot be set again, for example when the time off limit is recreated, so remove ranges that are old from the configuration. At most 366 days of values are set in one request, so more are sent as several requests.`,
				Type:        schema.TypeList,
				Optional:    true,
				Elem:        valueRangeResource,
			},
		},
	}
}

// TimeOffLimitExporter returns the resourceExporter object used to hold the genesyscloud_workforcemanagement_businessunits_timeofflimit exporter's config.
// Limit values are not exported, because the api can only read them for known date ranges.
func TimeOffLimitExporter() *resourceExporter.ResourceExporter {
	return &resourceExporter.ResourceExporter{
		GetResourcesFunc: provider.GetAllWithPooledClient(getAllTimeOffLimits),
		RefAttrs: map[string]*resourceExporter.RefAttrSettings{
			"business_unit_id":   {RefType: "genesyscloud_workforcemanagement_businessunits"},
			"management_unit_id": {RefType: "genesyscloud_workforcemanagement_managementunits"},
		},
	}
}
