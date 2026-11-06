package workforcemanagement_managementunits_workplan

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/customdiff"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/mypurecloud/terraform-provider-genesyscloud/genesyscloud/provider"
	resourceExporter "github.com/mypurecloud/terraform-provider-genesyscloud/genesyscloud/resource_exporter"
	registrar "github.com/mypurecloud/terraform-provider-genesyscloud/genesyscloud/resource_register"
)

/*
resource_genesyscloud_workforcemanagement_managementunits_workplan_schema.go holds four functions within it:

1.  The registration code that registers the Datasource, Resource and Exporter for the package.
2.  The resource schema definitions for the workforcemanagement_managementunits_workplan resource.
3.  The datasource schema definitions for the workforcemanagement_managementunits_workplan datasource.
4.  The resource exporter configuration for the workforcemanagement_managementunits_workplan exporter.
*/
const ResourceType = "genesyscloud_workforcemanagement_managementunits_workplan"

var (
	daysOfWeek     = []string{"Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"}
	varianceTypes  = []string{"ShiftStart", "ShiftStartAndPaidDuration"}
	variancePeriod = []string{"Weekly", "PlanningPeriod"}
	dayOffRules    = []string{"NextDayOff", "PreviousDayOff"}
)

// SetRegistrar registers all of the resources, datasources and exporters in the package
func SetRegistrar(regInstance registrar.Registrar) {
	regInstance.RegisterResource(ResourceType, ResourceWorkPlan())
	regInstance.RegisterDataSource(ResourceType, DataSourceWorkPlan())
	regInstance.RegisterExporter(ResourceType, WorkPlanExporter())
}

func optionalInt(description string) *schema.Schema {
	return &schema.Schema{Description: description, Type: schema.TypeInt, Optional: true, Computed: true, ValidateFunc: validation.IntAtLeast(0)}
}

func optionalBool(description string) *schema.Schema {
	return &schema.Schema{Description: description, Type: schema.TypeBool, Optional: true, Computed: true}
}

func optionalString(description string, allowed ...string) *schema.Schema {
	s := &schema.Schema{Description: description, Type: schema.TypeString, Optional: true, Computed: true}
	if len(allowed) > 0 {
		s.ValidateFunc = validation.StringInSlice(allowed, false)
	}
	return s
}

func daySet(description string) *schema.Schema {
	return &schema.Schema{
		Description: description,
		Type:        schema.TypeSet,
		Optional:    true,
		Computed:    true,
		Elem:        &schema.Schema{Type: schema.TypeString, ValidateFunc: validation.StringInSlice(daysOfWeek, false)},
	}
}

var shiftStartVarianceResource = &schema.Resource{
	Schema: map[string]*schema.Schema{
		"applicable_days": {
			Description: `The days for which the shift start variance is configured. Valid values: Sunday, Monday, Tuesday, Wednesday, Thursday, Friday, Saturday.`,
			Type:        schema.TypeSet,
			Required:    true,
			Elem:        &schema.Schema{Type: schema.TypeString, ValidateFunc: validation.StringInSlice(daysOfWeek, false)},
		},
		"max_shift_start_variance_minutes": {
			Description:  `The maximum variance in minutes across shift starts.`,
			Type:         schema.TypeInt,
			Required:     true,
			ValidateFunc: validation.IntAtLeast(0),
		},
	},
}

var planningPeriodConstraintsResource = &schema.Resource{
	Schema: map[string]*schema.Schema{
		"enabled":       optionalBool(`Whether shifts per planning period are constrained.`),
		"minimum_count": optionalInt(`The minimum number of shifts required per planning period.`),
		"maximum_count": optionalInt(`The maximum number of shifts allowed per planning period.`),
	},
}

var activityResource = &schema.Resource{
	Schema: map[string]*schema.Schema{
		"activity_code_id": {
			Description: `The id of the activity code of this activity. The default activity codes that every management unit has use the id of their category rather than a UUID, so the same id works in every management unit.`,
			Type:        schema.TypeString,
			Required:    true,
			StateFunc:   ModifyActivityCodeIdStateValue,
		},
		"description":                             optionalString(`The description of the activity.`),
		"length_minutes":                          optionalInt(`The length of the activity in minutes.`),
		"start_time_is_relative_to_shift_start":   optionalBool(`Whether the start time of the activity is relative to the start time of the shift it belongs to.`),
		"flexible_start_time":                     optionalBool(`Whether the start time of the activity is flexible.`),
		"earliest_start_time_minutes":             optionalInt(`The earliest activity start in minutes, relative to the shift start if start_time_is_relative_to_shift_start is true and to midnight otherwise. Used if flexible_start_time is true.`),
		"latest_start_time_minutes":               optionalInt(`The latest activity start in minutes, relative to the shift start if start_time_is_relative_to_shift_start is true and to midnight otherwise. Used if flexible_start_time is true.`),
		"exact_start_time_minutes":                optionalInt(`The exact activity start in minutes, relative to the shift start if start_time_is_relative_to_shift_start is true and to midnight otherwise. Used if flexible_start_time is false.`),
		"start_time_increment_minutes":            optionalInt(`The increment in minutes that contributes to the different possible start times for the activity.`),
		"counts_as_paid_time":                     optionalBool(`Whether the activity is paid.`),
		"counts_as_contiguous_work_time":          optionalBool(`Whether the activity duration counts towards contiguous work time.`),
		"minimum_length_from_shift_start_minutes": optionalInt(`The minimum duration in minutes between the shift start and the start of the activity (for example a break or meal).`),
		"minimum_length_from_shift_end_minutes":   optionalInt(`The minimum duration in minutes between the end of the activity (for example a break or meal) and the shift end.`),
		"maximum_length_from_shift_end_minutes":   optionalInt(`The maximum duration in minutes between the end of the activity (for example a break or meal) and the shift end.`),
	},
}

var shiftResource = &schema.Resource{
	Schema: map[string]*schema.Schema{
		"name": {
			Description:  `The name of the shift. Shift names must be unique within a work plan, because shifts are matched to existing shifts by name.`,
			Type:         schema.TypeString,
			Required:     true,
			ValidateFunc: validation.StringIsNotEmpty,
		},
		"days":                                   daySet(`The days of the week this shift applies to. Valid values: Sunday, Monday, Tuesday, Wednesday, Thursday, Friday, Saturday.`),
		"flexible_start_time":                    optionalBool(`Whether the start time of the shift is flexible.`),
		"exact_start_time_minutes_from_midnight": optionalInt(`The exact start time of the shift as minutes from midnight. Used if flexible_start_time is false.`),
		"earliest_start_time_minutes_from_midnight": optionalInt(`The earliest start time of the shift as minutes from midnight. Used if flexible_start_time is true.`),
		"latest_start_time_minutes_from_midnight":   optionalInt(`The latest start time of the shift as minutes from midnight. Used if flexible_start_time is true.`),
		"start_increment_minutes":                   optionalInt(`The increment in minutes that contributes to the different possible start times for the shift. Used if flexible_start_time is true.`),
		"constrain_latest_stop_time":                optionalBool(`Whether the latest stop time constraint for the shift is enabled.`),
		"latest_stop_time_minutes_from_midnight":    optionalInt(`The latest stop time of the shift as minutes from midnight. Used if constrain_latest_stop_time is true.`),
		"constrain_earliest_stop_time":              optionalBool(`Whether the earliest stop time constraint for the shift is enabled.`),
		"earliest_stop_time_minutes_from_midnight":  optionalInt(`The earliest time the shift can end, as minutes from midnight. Used if constrain_earliest_stop_time is true.`),
		"flexible_paid_time":                        optionalBool(`Whether the paid time setting for the shift is flexible.`),
		"exact_paid_time_minutes":                   optionalInt(`The exact paid time in minutes of the shift. Used if flexible_paid_time is false.`),
		"minimum_paid_time_minutes":                 optionalInt(`The minimum paid time in minutes of the shift. Used if flexible_paid_time is true.`),
		"maximum_paid_time_minutes":                 optionalInt(`The maximum paid time in minutes of the shift. Used if flexible_paid_time is true.`),
		"constrain_contiguous_work_time":            optionalBool(`Whether the contiguous work time constraint for the shift is enabled.`),
		"minimum_contiguous_work_time_minutes":      optionalInt(`The minimum contiguous work time in minutes. Used if constrain_contiguous_work_time is true.`),
		"maximum_contiguous_work_time_minutes":      optionalInt(`The maximum contiguous work time in minutes. Used if constrain_contiguous_work_time is true.`),
		"constrain_day_off":                         optionalBool(`Whether the day off rule is enabled.`),
		"day_off_rule":                              optionalString(`The day off rule for agents to have the next or previous day off. Used if constrain_day_off is true. Valid values: NextDayOff, PreviousDayOff.`, dayOffRules...),
		"planning_period_constraints": {
			Description: `Constraints on the number of times this shift is worked per planning period.`,
			Type:        schema.TypeList,
			Optional:    true,
			Computed:    true,
			MaxItems:    1,
			Elem:        planningPeriodConstraintsResource,
		},
		"activities": {
			Description: `The activities of this shift. Genesys Cloud does not keep activities in a stable order, so they are compared as a group: when they differ from what is configured, all of the shift's activities are replaced.`,
			Type:        schema.TypeList,
			Optional:    true,
			Computed:    true,
			Elem:        activityResource,
		},
	},
}

// ResourceWorkPlan registers the genesyscloud_workforcemanagement_managementunits_workplan resource with Terraform
func ResourceWorkPlan() *schema.Resource {
	return &schema.Resource{
		Description: `Genesys Cloud workforce management management unit work plan, with its shifts and the agents assigned to it. An agent can only have one work plan, so assigning an agent to this work plan unassigns the agent from their current work plan. An enabled work plan is validated by Genesys Cloud while planning, so a work plan that would be rejected fails the plan with the validation messages. This needs every value to be known, so a work plan that depends on resources created in the same run is only validated when it is applied.`,

		CreateContext: provider.CreateWithPooledClient(createWorkPlan),
		ReadContext:   provider.ReadWithPooledClient(readWorkPlan),
		UpdateContext: provider.UpdateWithPooledClient(updateWorkPlan),
		DeleteContext: provider.DeleteWithPooledClient(deleteWorkPlan),
		CustomizeDiff: customdiff.All(validateShiftNames, validateWorkPlanDuringPlan),
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		SchemaVersion: 1,
		Schema: map[string]*schema.Schema{
			"management_unit_id": {
				Description: `The id of the management unit this work plan belongs to. Changing this attribute will cause the work plan to be dropped and recreated.`,
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
			},
			"name": {
				Description:  `The name of the work plan.`,
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.StringIsNotEmpty,
			},
			"enabled": optionalBool(`Whether the work plan is enabled for scheduling. An enabled work plan that fails validation is rejected by Genesys Cloud.`),
			"valid": {
				Description: `Whether the work plan is valid.`,
				Type:        schema.TypeBool,
				Computed:    true,
			},
			"constrain_weekly_paid_time":                             optionalBool(`Whether the weekly paid time constraint is enabled.`),
			"flexible_weekly_paid_time":                              optionalBool(`Whether the weekly paid time constraint is flexible.`),
			"weekly_exact_paid_minutes":                              optionalInt(`The exact weekly paid time in minutes. Used if flexible_weekly_paid_time is false.`),
			"weekly_minimum_paid_minutes":                            optionalInt(`The minimum weekly paid time in minutes. Used if flexible_weekly_paid_time is true.`),
			"weekly_maximum_paid_minutes":                            optionalInt(`The maximum weekly paid time in minutes. Used if flexible_weekly_paid_time is true.`),
			"constrain_paid_time_granularity":                        optionalBool(`Whether the paid time granularity of shifts is constrained.`),
			"paid_time_granularity_minutes":                          optionalInt(`The granularity in minutes allowed for the paid time of a shift. Used if constrain_paid_time_granularity is true.`),
			"constrain_minimum_time_between_shifts":                  optionalBool(`Whether the minimum time between shifts constraint is enabled.`),
			"minimum_time_between_shifts_minutes":                    optionalInt(`The minimum time between shifts in minutes. Used if constrain_minimum_time_between_shifts is true.`),
			"maximum_days":                                           optionalInt(`The maximum number of days in a week that can be scheduled.`),
			"minimum_consecutive_non_working_minutes_per_week":       optionalInt(`The minimum consecutive non working minutes per week that agents are allowed to have off.`),
			"constrain_maximum_consecutive_working_weekends":         optionalBool(`Whether to constrain the maximum consecutive working weekends.`),
			"maximum_consecutive_working_weekends":                   optionalInt(`The maximum number of consecutive weekends that agents are allowed to work.`),
			"minimum_working_days_per_week":                          optionalInt(`The minimum number of days agents must work per week.`),
			"constrain_maximum_consecutive_working_days":             optionalBool(`Whether to constrain the maximum consecutive working days.`),
			"maximum_consecutive_working_days":                       optionalInt(`The maximum number of consecutive days agents are allowed to work. Used if constrain_maximum_consecutive_working_days is true.`),
			"minimum_shift_start_distance_minutes":                   optionalInt(`The minimum duration in minutes between the start times of two consecutive working days.`),
			"minimum_days_off_per_planning_period":                   optionalInt(`The minimum days off in the planning period.`),
			"maximum_days_off_per_planning_period":                   optionalInt(`The maximum days off in the planning period.`),
			"minimum_paid_minutes_per_planning_period":               optionalInt(`The minimum paid minutes in the planning period.`),
			"maximum_paid_minutes_per_planning_period":               optionalInt(`The maximum paid minutes in the planning period.`),
			"constrain_maximum_working_weekends_per_planning_period": optionalBool(`Whether to constrain the maximum working weekends in the planning period.`),
			"maximum_working_weekends_per_planning_period":           optionalInt(`The maximum working weekends in the planning period.`),
			"optional_days":                                          daySet(`The optional days to schedule for this work plan. Valid values: Sunday, Monday, Tuesday, Wednesday, Thursday, Friday, Saturday.`),
			"shift_start_variance_type":                              optionalString(`Ensures an agent starts each workday within a threshold. Valid values: ShiftStart, ShiftStartAndPaidDuration.`, varianceTypes...),
			"shift_start_variance_period":                            optionalString(`The period over which the maximum shift start variance is applied. Valid values: Weekly, PlanningPeriod.`, variancePeriod...),
			"shift_start_variances": {
				Description: `The variance in minutes among the start times of shifts, per set of days.`,
				Type:        schema.TypeSet,
				Optional:    true,
				Computed:    true,
				Elem:        shiftStartVarianceResource,
			},
			"shifts": {
				Description: `The shifts of this work plan. Shifts are matched to the shifts already on the work plan by name, so names must be unique. Genesys Cloud does not keep shifts in a stable order, so the order of the configuration is kept but has no meaning.`,
				Type:        schema.TypeList,
				Optional:    true,
				Computed:    true,
				Elem:        shiftResource,
			},
			"agent_ids": {
				Description: `The ids of the agents assigned to this work plan. This is authoritative for the work plan: agents assigned to it but not listed are unassigned. An agent can only have one work plan, so listing an agent here removes them from their current work plan.`,
				Type:        schema.TypeSet,
				Optional:    true,
				Computed:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
		},
	}
}

// WorkPlanExporter returns the resourceExporter object used to hold the genesyscloud_workforcemanagement_managementunits_workplan exporter's config
func WorkPlanExporter() *resourceExporter.ResourceExporter {
	return &resourceExporter.ResourceExporter{
		GetResourcesFunc: provider.GetAllWithPooledClient(getAllWorkPlans),
		// Activity code ids are exported as plain ids: many shifts use default activity codes, which are not resources,
		// and the activity code resource uses composite ids as exporter keys, so they cannot be matched to references.
		RefAttrs: map[string]*resourceExporter.RefAttrSettings{
			"management_unit_id": {RefType: "genesyscloud_workforcemanagement_managementunits"},
			"agent_ids":          {RefType: "genesyscloud_user"},
		},
	}
}

// DataSourceWorkPlan registers the genesyscloud_workforcemanagement_managementunits_workplan data source
func DataSourceWorkPlan() *schema.Resource {
	return &schema.Resource{
		Description: `Genesys Cloud workforce management management unit work plan data source. Select a work plan by name`,
		ReadContext: provider.ReadWithPooledClient(dataSourceWorkPlanRead),
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"management_unit_id": {
				Description: `The id of the management unit the work plan belongs to`,
				Type:        schema.TypeString,
				Required:    true,
			},
			"name": {
				Description: `Work plan name`,
				Type:        schema.TypeString,
				Required:    true,
			},
		},
	}
}
