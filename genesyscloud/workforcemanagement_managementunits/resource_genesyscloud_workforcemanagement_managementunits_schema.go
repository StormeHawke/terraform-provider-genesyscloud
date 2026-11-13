package workforcemanagement_managementunits

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/mypurecloud/terraform-provider-genesyscloud/genesyscloud/provider"
	resourceExporter "github.com/mypurecloud/terraform-provider-genesyscloud/genesyscloud/resource_exporter"
	registrar "github.com/mypurecloud/terraform-provider-genesyscloud/genesyscloud/resource_register"
)

/*
resource_genesyscloud_workforcemanagement_managementunits_schema.go holds four functions within it:

1.  The registration code that registers the Datasource, Resource and Exporter for the package.
2.  The resource schema definitions for the workforcemanagement_managementunits resource.
3.  The datasource schema definitions for the workforcemanagement_managementunits datasource.
4.  The resource exporter configuration for the workforcemanagement_managementunits exporter.
*/
const ResourceType = "genesyscloud_workforcemanagement_managementunits"

// SetRegistrar registers all of the resources, datasources and exporters in the package
func SetRegistrar(regInstance registrar.Registrar) {
	regInstance.RegisterResource(ResourceType, ResourceManagementUnit())
	regInstance.RegisterDataSource(ResourceType, DataSourceManagementUnit())
	regInstance.RegisterExporter(ResourceType, ManagementUnitExporter())
}

func optionalInt(description string) *schema.Schema {
	return &schema.Schema{Description: description, Type: schema.TypeInt, Optional: true, Computed: true}
}

func optionalFloat(description string) *schema.Schema {
	return &schema.Schema{Description: description, Type: schema.TypeFloat, Optional: true, Computed: true}
}

func optionalBool(description string) *schema.Schema {
	return &schema.Schema{Description: description, Type: schema.TypeBool, Optional: true, Computed: true}
}

func optionalString(description string) *schema.Schema {
	return &schema.Schema{Description: description, Type: schema.TypeString, Optional: true, Computed: true}
}

func optionalBlock(description string, elem *schema.Resource) *schema.Schema {
	return &schema.Schema{Description: description, Type: schema.TypeList, Optional: true, Computed: true, MaxItems: 1, Elem: elem}
}

var adherenceResource = &schema.Resource{
	Schema: map[string]*schema.Schema{
		"severe_alert_threshold_minutes":        optionalInt(`The threshold in minutes where an alert is triggered when an agent is considered severely out of adherence.`),
		"adherence_target_percent":              optionalInt(`Target adherence percentage.`),
		"adherence_exception_threshold_seconds": optionalInt(`The threshold in seconds for which agents should not be penalized for being momentarily out of adherence.`),
		"non_on_queue_activities_equivalent":    optionalBool(`Whether to treat all non-on-queue activities as equivalent for adherence purposes.`),
		"track_on_queue_activity":               optionalBool(`Whether to track on-queue activities.`),
		"ignored_activity_categories": {
			Description: `Activity categories that are ignored for adherence purposes. Valid values: OnQueueWork, Break, Meal, Meeting, OffQueueWork, TimeOff, Training, Unavailable.`,
			Type:        schema.TypeSet,
			Optional:    true,
			Computed:    true,
			Elem:        &schema.Schema{Type: schema.TypeString},
		},
		"ignored_activity_code_ids": {
			Description: `Activity code ids that are ignored for adherence purposes.`,
			Type:        schema.TypeSet,
			Optional:    true,
			Computed:    true,
			Elem:        &schema.Schema{Type: schema.TypeString, StateFunc: ModifyActivityCodeIdStateValue},
		},
	},
}

var timeOffResource = &schema.Resource{
	Schema: map[string]*schema.Schema{
		"submission_range_enforced":         optionalBool(`Whether to enforce a submission range for agent time off requests.`),
		"submission_range_type":             optionalString(`The type of the submission range. Valid values: Relative, FixedEnd.`),
		"submission_earliest_days_from_now": optionalInt(`The earliest number of days from now for which an agent can submit a time off request. Use negative numbers to indicate days in the past.`),
		"submission_latest_days_from_now":   optionalInt(`The latest number of days from now for which an agent can submit a time off request.`),
		"submission_latest_date":            optionalString(`The latest date for time off request submission, interpreted in the business unit time zone, in yyyy-MM-dd format.`),
	},
}

var planningPeriodResource = &schema.Resource{
	Schema: map[string]*schema.Schema{
		"week_count": {
			Description: `Planning period length in weeks.`,
			Type:        schema.TypeInt,
			Required:    true,
		},
		"start_date": {
			Description: `Start date of the planning period in yyyy-MM-dd format.`,
			Type:        schema.TypeString,
			Required:    true,
		},
	},
}

var monthlyPlanningPeriodResource = &schema.Resource{
	Schema: map[string]*schema.Schema{
		"start_date": {
			Description: `Start date of the monthly planning period in yyyy-MM-dd format. The date must be the first day of a month.`,
			Type:        schema.TypeString,
			Required:    true,
		},
	},
}

var scheduleVisibilityResource = &schema.Resource{
	Schema: map[string]*schema.Schema{
		"enabled":      optionalBool(`Whether schedule visibility controls are enabled for this management unit.`),
		"future_weeks": optionalInt(`The number of weeks into the future that agents can see schedules in this management unit. 0 means the current week.`),
	},
}

var shrinkageOverrideResource = &schema.Resource{
	Schema: map[string]*schema.Schema{
		"interval_index": {
			Description: `Index of the shrinkage override interval. The starting index is 0 and indexes are based on 15 minute intervals for a 7 day week.`,
			Type:        schema.TypeInt,
			Required:    true,
		},
		"shrinkage_percent": {
			Description: `Shrinkage override percent.`,
			Type:        schema.TypeFloat,
			Required:    true,
		},
	},
}

var schedulingResource = &schema.Resource{
	Schema: map[string]*schema.Schema{
		"max_occupancy_percent_for_deferred_work": optionalInt(`Max occupancy percent for deferred work.`),
		"default_shrinkage_percent":               optionalFloat(`Default shrinkage percent for scheduling.`),
		"shrinkage_overrides": {
			Description: `Shrinkage overrides for scheduling. Set to an empty list to clear existing overrides.`,
			Type:        schema.TypeSet,
			Optional:    true,
			Computed:    true,
			Elem:        shrinkageOverrideResource,
		},
		"planning_period":         optionalBlock(`Planning period settings for scheduling. Only one of planning_period or monthly_planning_period may be defined.`, planningPeriodResource),
		"monthly_planning_period": optionalBlock(`Monthly planning period settings for scheduling. Only one of planning_period or monthly_planning_period may be defined.`, monthlyPlanningPeriodResource),
		"start_day_of_weekend":    optionalString(`Start day of the weekend for scheduling.`),
		"schedule_visibility":     optionalBlock(`Schedule visibility settings for agents.`, scheduleVisibilityResource),
	},
}

var shiftTradeActivityRuleResource = &schema.Resource{
	Schema: map[string]*schema.Schema{
		"activity_category": {
			Description: `The activity category to which this rule applies. Valid values: OnQueueWork, Break, Meal, Meeting, OffQueueWork, TimeOff, Training, Unavailable.`,
			Type:        schema.TypeString,
			Required:    true,
		},
		"action": {
			Description: `The action this rule invokes. Valid values: Replace, DoNotAllowTrade, KeepWithSchedule.`,
			Type:        schema.TypeString,
			Required:    true,
		},
		"activity_code_id_replacement": {
			Description: `The id of the default activity code that replaces activities of the original category. Required if action is Replace.`,
			Type:        schema.TypeString,
			Optional:    true,
		},
	},
}

var shiftTradeExternalActivityRuleResource = &schema.Resource{
	Schema: map[string]*schema.Schema{
		"external_activity_type": {
			Description: `The external activity type to which this rule applies. Valid values: ActivityPlan, Coaching, Learning, Opportunity.`,
			Type:        schema.TypeString,
			Required:    true,
		},
		"action": {
			Description: `The action this rule invokes. Valid values: DoNotAllowTrade, KeepWithAgent.`,
			Type:        schema.TypeString,
			Required:    true,
		},
		"activity_code_id_replacement": {
			Description: `The id of the default activity code that replaces external activities. Required if action is KeepWithAgent.`,
			Type:        schema.TypeString,
			Optional:    true,
		},
	},
}

var shiftTradingResource = &schema.Resource{
	Schema: map[string]*schema.Schema{
		"enabled":                           optionalBool(`Whether shift trading is enabled for this management unit.`),
		"auto_review":                       optionalBool(`Whether automatic shift trade review is enabled according to the rules defined for this management unit.`),
		"allow_direct_trades":               optionalBool(`Whether direct shift trades between agents are allowed.`),
		"min_hours_in_future":               optionalInt(`The minimum number of hours in the future that shift trades are allowed.`),
		"unequal_paid":                      optionalString(`How to handle shift trades that involve unequal paid times. Valid values: Allow, Disallow, AdminReview.`),
		"one_sided":                         optionalString(`How to handle one-sided shift trades. Valid values: Allow, Disallow, AdminReview.`),
		"weekly_min_paid_violations":        optionalString(`How to handle shift trades that violate the weekly minimum paid time constraint. Valid values: Allow, Disallow, AdminReview.`),
		"weekly_max_paid_violations":        optionalString(`How to handle shift trades that violate the weekly maximum paid time constraint. Valid values: Allow, Disallow, AdminReview.`),
		"requires_matching_queues":          optionalBool(`Whether to constrain shift trades to agents with matching queues.`),
		"requires_matching_languages":       optionalBool(`Whether to constrain shift trades to agents with matching languages.`),
		"requires_matching_skills":          optionalBool(`Whether to constrain shift trades to agents with matching skills.`),
		"requires_matching_planning_groups": optionalBool(`Whether to constrain shift trades to agents with matching planning groups.`),
		"external_activity_rule_ignores_activity_category_rule": optionalBool(`Whether external activity rules ignore activity category rules.`),
		"max_trade_span_weeks":                  optionalInt(`The maximum number of weeks a shift trade can span.`),
		"max_trades_per_agent_per_week":         optionalInt(`The maximum number of shift trades an agent can submit per week.`),
		"min_minutes_between_shifts":            optionalInt(`The minimum number of minutes between shifts.`),
		"planning_period_min_paid_violations":   optionalString(`How to handle shift trades that violate the planning period minimum paid time constraint. Valid values: Allow, Disallow, AdminReview.`),
		"planning_period_max_paid_violations":   optionalString(`How to handle shift trades that violate the planning period maximum paid time constraint. Valid values: Allow, Disallow, AdminReview.`),
		"min_minutes_between_shifts_violations": optionalString(`How to handle shift trades that violate the minimum minutes between shifts constraint. Valid values: Allow, Disallow, AdminReview.`),
		"activity_category_rules": {
			Description: `Rules that specify what to do with activity categories that are part of a shift in a trade.`,
			Type:        schema.TypeList,
			Optional:    true,
			Computed:    true,
			Elem:        shiftTradeActivityRuleResource,
		},
		"external_activity_rules": {
			Description: `Rules that specify what to do with external activities that are part of a shift.`,
			Type:        schema.TypeList,
			Optional:    true,
			Computed:    true,
			Elem:        shiftTradeExternalActivityRuleResource,
		},
	},
}

var settingsResource = &schema.Resource{
	Schema: map[string]*schema.Schema{
		"adherence":     optionalBlock(`Adherence settings for this management unit.`, adherenceResource),
		"time_off":      optionalBlock(`Time off request settings for this management unit.`, timeOffResource),
		"scheduling":    optionalBlock(`Scheduling settings for this management unit.`, schedulingResource),
		"shift_trading": optionalBlock(`Shift trade settings for this management unit.`, shiftTradingResource),
	},
}

// ResourceManagementUnit registers the genesyscloud_workforcemanagement_managementunits resource with Terraform
func ResourceManagementUnit() *schema.Resource {
	return &schema.Resource{
		Description: `Genesys Cloud workforce management management unit`,

		CreateContext: provider.CreateWithPooledClient(createManagementUnit),
		ReadContext:   provider.ReadWithPooledClient(readManagementUnit),
		UpdateContext: provider.UpdateWithPooledClient(updateManagementUnit),
		DeleteContext: provider.DeleteWithPooledClient(deleteManagementUnit),
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		SchemaVersion: 1,
		Schema: map[string]*schema.Schema{
			"name": {
				Description:  `The name of the management unit.`,
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.StringIsNotEmpty,
			},
			"business_unit_id": {
				Description: `The id of the business unit this management unit belongs to. Changing this attribute moves the management unit to the new business unit.`,
				Type:        schema.TypeString,
				Required:    true,
			},
			"division_id": {
				Description: `The id of the division this management unit belongs to. Defaults to the home division.`,
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
			},
			"settings": {
				Description: `The configuration settings for this management unit. Short term forecasting settings are not included because Genesys Cloud has moved them to the business unit.`,
				Type:        schema.TypeList,
				Optional:    true,
				Computed:    true,
				MaxItems:    1,
				Elem:        settingsResource,
			},
		},
	}
}

// ManagementUnitExporter returns the resourceExporter object used to hold the genesyscloud_workforcemanagement_managementunits exporter's config
func ManagementUnitExporter() *resourceExporter.ResourceExporter {
	return &resourceExporter.ResourceExporter{
		GetResourcesFunc: provider.GetAllWithPooledClient(getAllManagementUnits),
		// Ignored activity code ids are exported as plain ids: the activity code resource uses composite
		// ids (businessUnitId/activityCodeId) as exporter keys, so they cannot be matched to references.
		RefAttrs: map[string]*resourceExporter.RefAttrSettings{
			"business_unit_id": {RefType: "genesyscloud_workforcemanagement_businessunits"},
			"division_id":      {RefType: "genesyscloud_auth_division"},
		},
	}
}

// DataSourceManagementUnit registers the genesyscloud_workforcemanagement_managementunits data source
func DataSourceManagementUnit() *schema.Resource {
	return &schema.Resource{
		Description: `Genesys Cloud workforce management management unit data source. Select a management unit by name`,
		ReadContext: provider.ReadWithPooledClient(dataSourceManagementUnitRead),
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"name": {
				Description: `Management unit name`,
				Type:        schema.TypeString,
				Required:    true,
			},
		},
	}
}
