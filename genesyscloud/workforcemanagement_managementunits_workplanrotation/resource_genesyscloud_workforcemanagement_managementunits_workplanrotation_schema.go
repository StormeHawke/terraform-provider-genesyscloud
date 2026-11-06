package workforcemanagement_managementunits_workplanrotation

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
resource_genesyscloud_workforcemanagement_managementunits_workplanrotation_schema.go holds four functions within it:

1.  The registration code that registers the Datasource, Resource and Exporter for the package.
2.  The resource schema definitions for the workforcemanagement_managementunits_workplanrotation resource.
3.  The datasource schema definitions for the workforcemanagement_managementunits_workplanrotation datasource.
4.  The resource exporter configuration for the workforcemanagement_managementunits_workplanrotation exporter.
*/
const ResourceType = "genesyscloud_workforcemanagement_managementunits_workplanrotation"

// workPlanResourceType is hard-coded to avoid an import cycle between the exporter packages
const workPlanResourceType = "genesyscloud_workforcemanagement_managementunits_workplan"

// SetRegistrar registers all of the resources, datasources and exporters in the package
func SetRegistrar(regInstance registrar.Registrar) {
	regInstance.RegisterResource(ResourceType, ResourceWorkPlanRotation())
	regInstance.RegisterDataSource(ResourceType, DataSourceWorkPlanRotation())
	regInstance.RegisterExporter(ResourceType, WorkPlanRotationExporter())
}

func validateDate(value interface{}, key string) ([]string, []error) {
	if _, err := time.Parse(dateLayout, value.(string)); err != nil {
		return nil, []error{fmt.Errorf("%q must be a date in the format yyyy-MM-dd, got %q", key, value)}
	}
	return nil, nil
}

var rotationAgentResource = &schema.Resource{
	Schema: map[string]*schema.Schema{
		"user_id": {
			Description: `The id of an agent in this work plan rotation.`,
			Type:        schema.TypeString,
			Required:    true,
		},
		"start_date": {
			Description:  `The first date in the business unit's time zone on which the agent follows the rotation, in the format yyyy-MM-dd.`,
			Type:         schema.TypeString,
			Required:     true,
			ValidateFunc: validateDate,
		},
		"end_date": {
			Description:  `The last date in the business unit's time zone on which the agent follows the rotation, in the format yyyy-MM-dd. Leave it out for an open ended date range.`,
			Type:         schema.TypeString,
			Optional:     true,
			ValidateFunc: validateDate,
		},
		"position": {
			Description:  `The start position of the agent in the pattern of the rotation. Positions start at 0, which is the first work plan of the pattern.`,
			Type:         schema.TypeInt,
			Required:     true,
			ValidateFunc: validation.IntAtLeast(0),
		},
	},
}

// ResourceWorkPlanRotation registers the genesyscloud_workforcemanagement_managementunits_workplanrotation resource with Terraform
func ResourceWorkPlanRotation() *schema.Resource {
	return &schema.Resource{
		Description: `Genesys Cloud workforce management management unit work plan rotation: an ordered pattern of work plans that rotates weekly, with the agents that follow it. An agent can only be in one work plan rotation, so adding an agent to this rotation removes them from their current one.`,

		CreateContext: provider.CreateWithPooledClient(createWorkPlanRotation),
		ReadContext:   provider.ReadWithPooledClient(readWorkPlanRotation),
		UpdateContext: provider.UpdateWithPooledClient(updateWorkPlanRotation),
		DeleteContext: provider.DeleteWithPooledClient(deleteWorkPlanRotation),
		CustomizeDiff: customdiff.All(validateUniqueAgents),
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		SchemaVersion: 1,
		Schema: map[string]*schema.Schema{
			"management_unit_id": {
				Description: `The id of the management unit this work plan rotation belongs to. Changing this attribute will cause the work plan rotation to be dropped and recreated.`,
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
			},
			"name": {
				Description:  `The name of the work plan rotation. Names must be unique within a management unit.`,
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.StringIsNotEmpty,
			},
			"enabled": {
				Description: `Whether the work plan rotation is enabled for scheduling. Genesys Cloud always creates a work plan rotation enabled, so creating one that is disabled takes a second request.`,
				Type:        schema.TypeBool,
				Optional:    true,
				Computed:    true,
			},
			"start_date": {
				Description:  `The first date in the business unit's time zone on which the work plan rotation applies, in the format yyyy-MM-dd.`,
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validateDate,
			},
			"end_date": {
				Description:  `The last date in the business unit's time zone on which the work plan rotation applies, in the format yyyy-MM-dd. Leave it out for an open ended work plan rotation.`,
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validateDate,
			},
			"work_plan_ids": {
				Description: `The ids of the work plans of the pattern, in the order in which they rotate weekly. Every work plan must be enabled, and a work plan that is used here cannot be disabled or deleted.`,
				Type:        schema.TypeList,
				Required:    true,
				MinItems:    1,
				Elem:        &schema.Schema{Type: schema.TypeString, StateFunc: ModifyWorkPlanIdStateValue},
			},
			"agents": {
				Description: `The agents that follow this work plan rotation. This is authoritative for the work plan rotation: agents in it that are not listed are removed. An agent can only be in one work plan rotation, so listing an agent here removes them from their current rotation.`,
				Type:        schema.TypeSet,
				Optional:    true,
				Computed:    true,
				Elem:        rotationAgentResource,
			},
		},
	}
}

// WorkPlanRotationExporter returns the resourceExporter object used to hold the genesyscloud_workforcemanagement_managementunits_workplanrotation exporter's config
func WorkPlanRotationExporter() *resourceExporter.ResourceExporter {
	return &resourceExporter.ResourceExporter{
		GetResourcesFunc: provider.GetAllWithPooledClient(getAllWorkPlanRotations),
		// The work plan resource uses composite ids (managementUnitId/workPlanId) as exporter keys, but a rotation stores only the bare
		// work plan ids, so a custom resolver finishes those references.
		RefAttrs: map[string]*resourceExporter.RefAttrSettings{
			"management_unit_id": {RefType: "genesyscloud_workforcemanagement_managementunits"},
			"work_plan_ids":      {RefType: workPlanResourceType},
			"agents.user_id":     {RefType: "genesyscloud_user"},
		},
		CustomAttributeResolver: map[string]*resourceExporter.RefAttrCustomResolver{
			"work_plan_ids": {ResolverFunc: WorkPlanArrayRefResolver("work_plan_ids")},
		},
	}
}

// DataSourceWorkPlanRotation registers the genesyscloud_workforcemanagement_managementunits_workplanrotation data source
func DataSourceWorkPlanRotation() *schema.Resource {
	return &schema.Resource{
		Description: `Genesys Cloud workforce management management unit work plan rotation data source. Select a work plan rotation by name`,
		ReadContext: provider.ReadWithPooledClient(dataSourceWorkPlanRotationRead),
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"management_unit_id": {
				Description: `The id of the management unit the work plan rotation belongs to`,
				Type:        schema.TypeString,
				Required:    true,
			},
			"name": {
				Description: `Work plan rotation name`,
				Type:        schema.TypeString,
				Required:    true,
			},
		},
	}
}
