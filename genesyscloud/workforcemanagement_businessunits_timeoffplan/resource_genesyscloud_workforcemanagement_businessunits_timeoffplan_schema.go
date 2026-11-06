package workforcemanagement_businessunits_timeoffplan

import (
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/customdiff"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/mypurecloud/terraform-provider-genesyscloud/genesyscloud/provider"
	resourceExporter "github.com/mypurecloud/terraform-provider-genesyscloud/genesyscloud/resource_exporter"
	registrar "github.com/mypurecloud/terraform-provider-genesyscloud/genesyscloud/resource_register"
)

/*
resource_genesyscloud_workforcemanagement_businessunits_timeoffplan_schema.go holds four functions within it:

1.  The registration code that registers the Datasource, Resource and Exporter for the package.
2.  The resource schema definitions for the workforcemanagement_businessunits_timeoffplan resource.
3.  The datasource schema definitions for the workforcemanagement_businessunits_timeoffplan datasource.
4.  The resource exporter configuration for the workforcemanagement_businessunits_timeoffplan exporter.
*/
const ResourceType = "genesyscloud_workforcemanagement_businessunits_timeoffplan"

var autoApprovalRules = []string{"Never", "Always", "CheckLimits", "CheckActivityCodes", "CheckLimitsAndActivityCodes"}

// SetRegistrar registers all of the resources, datasources and exporters in the package
func SetRegistrar(regInstance registrar.Registrar) {
	regInstance.RegisterResource(ResourceType, ResourceTimeOffPlan())
	regInstance.RegisterDataSource(ResourceType, DataSourceTimeOffPlan())
	regInstance.RegisterExporter(ResourceType, TimeOffPlanExporter())
}

func validateDate(value interface{}, key string) ([]string, []error) {
	if _, err := time.Parse(dateLayout, value.(string)); err != nil {
		return nil, []error{fmt.Errorf("%q must be a date in the format yyyy-MM-dd, got %q", key, value)}
	}
	return nil, nil
}

func idSet(description string, stateFunc schema.SchemaStateFunc) *schema.Schema {
	return &schema.Schema{
		Description: description,
		Type:        schema.TypeSet,
		Optional:    true,
		Elem:        &schema.Schema{Type: schema.TypeString, StateFunc: stateFunc},
	}
}

var overrideDateResource = &schema.Resource{
	Schema: map[string]*schema.Schema{
		"date": {
			Description:  `The date, in the format yyyy-MM-dd and interpreted in the time zone of the business unit.`,
			Type:         schema.TypeString,
			Required:     true,
			ValidateFunc: validateDate,
		},
		"type": {
			Description:  `What happens to a time off request that conflicts with the date. Valid values: Blocked, ManualReview.`,
			Type:         schema.TypeString,
			Required:     true,
			ValidateFunc: validation.StringInSlice([]string{"Blocked", "ManualReview"}, false),
		},
	},
}

var hrisTimeOffTypeResource = &schema.Resource{
	Schema: map[string]*schema.Schema{
		"id": {
			Description:  `The id of the time off type configured in the integration.`,
			Type:         schema.TypeString,
			Required:     true,
			ValidateFunc: validation.StringLenBetween(1, 36),
		},
		"name": {
			Description:  `The name of the time off type configured in the integration.`,
			Type:         schema.TypeString,
			Required:     true,
			ValidateFunc: validation.StringLenBetween(1, 100),
		},
		"hris_integration_id": {
			Description:  `The id of the integration.`,
			Type:         schema.TypeString,
			Required:     true,
			ValidateFunc: validation.StringLenBetween(1, 36),
		},
		"secondary_id": {
			Description: `The secondary id of the time off type, if one is configured in the integration.`,
			Type:        schema.TypeString,
			Optional:    true,
		},
	},
}

var businessUnitAssociationResource = &schema.Resource{
	Schema: map[string]*schema.Schema{
		"management_unit_ids": {
			Description: `The ids of the management units the time off plan applies to. Exactly one of management_unit_ids and staffing_group_ids is required.`,
			Type:        schema.TypeSet,
			Optional:    true,
			Elem:        &schema.Schema{Type: schema.TypeString},
		},
		"staffing_group_ids": {
			Description: `The ids of the staffing groups the time off plan applies to. Exactly one of management_unit_ids and staffing_group_ids is required.`,
			Type:        schema.TypeSet,
			Optional:    true,
			Elem:        &schema.Schema{Type: schema.TypeString},
		},
	},
}

var managementUnitAssociationResource = &schema.Resource{
	Schema: map[string]*schema.Schema{
		"management_unit_id": {
			Description: `The id of the management unit the time off plan belongs to. Changing it will cause the time off plan to be dropped and recreated.`,
			Type:        schema.TypeString,
			Required:    true,
		},
		"staffing_group_ids": {
			Description: `The ids of the staffing groups of the management unit the time off plan applies to. When there are none, the plan applies to the management unit.`,
			Type:        schema.TypeSet,
			Optional:    true,
			Elem:        &schema.Schema{Type: schema.TypeString},
		},
	},
}

// ResourceTimeOffPlan registers the genesyscloud_workforcemanagement_businessunits_timeoffplan resource with Terraform
func ResourceTimeOffPlan() *schema.Resource {
	return &schema.Resource{
		Description: `Genesys Cloud workforce management business unit time off plan, which sets the activity codes agents can request time off with and how those requests are approved.`,

		CreateContext: provider.CreateWithPooledClient(createTimeOffPlan),
		ReadContext:   provider.ReadWithPooledClient(readTimeOffPlan),
		UpdateContext: provider.UpdateWithPooledClient(updateTimeOffPlan),
		DeleteContext: provider.DeleteWithPooledClient(deleteTimeOffPlan),
		CustomizeDiff: customdiff.All(validateAssociations, validateRules, forceNewWhenOwnerChanges),
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		SchemaVersion: 1,
		Schema: map[string]*schema.Schema{
			"business_unit_id": {
				Description: `The id of the business unit this time off plan belongs to. Changing this attribute will cause the time off plan to be dropped and recreated.`,
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
			},
			"name": {
				Description:  `The name of the time off plan.`,
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.StringLenBetween(1, 100),
			},
			"enabled": {
				Description: `Whether agents can use this time off plan. Genesys Cloud enables a time off plan by default.`,
				Type:        schema.TypeBool,
				Optional:    true,
				Computed:    true,
			},
			"activity_code_ids":            idSet(`The ids of the time off activity codes that are part of this time off plan. The default activity codes have the id of their category.`, ModifyActivityCodeIdStateValue),
			"restricted_activity_code_ids": idSet(`The ids of the non time off activity codes to check for conflicts, at most 50. They are required when the auto approval rule checks activity codes, and not allowed otherwise. A request is not approved automatically when one of these activity codes is scheduled during the time off.`, ModifyActivityCodeIdStateValue),
			"auto_approval_rule": {
				Description:  `The rule for approving time off requests automatically. Genesys Cloud uses Never by default. Valid values: Never, Always, CheckLimits, CheckActivityCodes, CheckLimitsAndActivityCodes. The rules that check limits need count_against_time_off_limits to be true, and the rules that check activity codes need restricted_activity_code_ids.`,
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				ValidateFunc: validation.StringInSlice(autoApprovalRules, false),
			},
			"days_before_start_to_expire_from_waitlist": {
				Description:  `The number of days before the start date of a time off request on which the request expires from the waitlist. Genesys Cloud uses 0 by default.`,
				Type:         schema.TypeInt,
				Optional:     true,
				Computed:     true,
				ValidateFunc: validation.IntBetween(0, 365),
			},
			"auto_publish_approved_time_off_requests": {
				Description: `Whether newly approved time off requests with activity codes of this time off plan are published to the schedule automatically.`,
				Type:        schema.TypeBool,
				Optional:    true,
				Computed:    true,
			},
			"count_against_time_off_limits": {
				Description: `Whether this time off plan counts against time off limits. Genesys Cloud uses false by default. Setting it to true needs every management unit or staffing group the plan applies to to already have a time off limit, so the plan has to be created after the time off limits, for example with depends_on. A time off limit that is created or deleted later is added to or removed from the plans that already exist.`,
				Type:        schema.TypeBool,
				Optional:    true,
				Computed:    true,
			},
			"override_dates": {
				Description: `Dates that are checked for conflicts with time off requests. A request that conflicts with one is not approved automatically. Each date can only be listed once, and Genesys Cloud only accepts dates in a limited window around today, which reaches only a short time into the past, so remove dates that have passed from the configuration.`,
				Type:        schema.TypeSet,
				Optional:    true,
				Elem:        overrideDateResource,
			},
			"hris_time_off_type": {
				Description: `The time off type of the HRIS integration this time off plan is associated with.`,
				Type:        schema.TypeList,
				Optional:    true,
				MaxItems:    1,
				Elem:        hrisTimeOffTypeResource,
			},
			"business_unit_association": {
				Description:  `Makes the time off plan belong to the business unit, and applies it to the listed management units or staffing groups. Exactly one of business_unit_association and management_unit_association is required. Genesys Cloud cannot change which of the two a time off plan has once it is created, so changing it will cause the time off plan to be dropped and recreated.`,
				Type:         schema.TypeList,
				Optional:     true,
				MaxItems:     1,
				Elem:         businessUnitAssociationResource,
				ExactlyOneOf: []string{"business_unit_association", "management_unit_association"},
			},
			"management_unit_association": {
				Description:  `Makes the time off plan belong to a management unit. Exactly one of business_unit_association and management_unit_association is required.`,
				Type:         schema.TypeList,
				Optional:     true,
				MaxItems:     1,
				Elem:         managementUnitAssociationResource,
				ExactlyOneOf: []string{"business_unit_association", "management_unit_association"},
			},
			"time_off_limit_ids": {
				Description: `The ids of the time off limits this time off plan counts against.`,
				Type:        schema.TypeSet,
				Computed:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
		},
	}
}

// TimeOffPlanExporter returns the resourceExporter object used to hold the genesyscloud_workforcemanagement_businessunits_timeoffplan exporter's config.
// Activity code ids are exported as plain ids: the default activity codes are not resources, and the activity code resource uses composite ids as exporter keys.
func TimeOffPlanExporter() *resourceExporter.ResourceExporter {
	return &resourceExporter.ResourceExporter{
		GetResourcesFunc: provider.GetAllWithPooledClient(getAllTimeOffPlans),
		RefAttrs: map[string]*resourceExporter.RefAttrSettings{
			"business_unit_id": {RefType: "genesyscloud_workforcemanagement_businessunits"},
			"business_unit_association.management_unit_ids":  {RefType: "genesyscloud_workforcemanagement_managementunits"},
			"management_unit_association.management_unit_id": {RefType: "genesyscloud_workforcemanagement_managementunits"},
		},
	}
}

// DataSourceTimeOffPlan registers the genesyscloud_workforcemanagement_businessunits_timeoffplan data source
func DataSourceTimeOffPlan() *schema.Resource {
	return &schema.Resource{
		Description: `Genesys Cloud workforce management business unit time off plan data source. Select a time off plan by name`,
		ReadContext: provider.ReadWithPooledClient(dataSourceTimeOffPlanRead),
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"business_unit_id": {
				Description: `The id of the business unit the time off plan belongs to`,
				Type:        schema.TypeString,
				Required:    true,
			},
			"name": {
				Description: `Time off plan name`,
				Type:        schema.TypeString,
				Required:    true,
			},
		},
	}
}
