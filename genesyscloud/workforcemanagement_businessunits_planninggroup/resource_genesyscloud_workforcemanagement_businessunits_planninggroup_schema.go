package workforcemanagement_businessunits_planninggroup

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/mypurecloud/terraform-provider-genesyscloud/genesyscloud/provider"
	resourceExporter "github.com/mypurecloud/terraform-provider-genesyscloud/genesyscloud/resource_exporter"
	registrar "github.com/mypurecloud/terraform-provider-genesyscloud/genesyscloud/resource_register"
)

/*
resource_genesyscloud_workforcemanagement_businessunits_planninggroup_schema.go holds four functions within it:

1.  The registration code that registers the Datasource, Resource and Exporter for the package.
2.  The resource schema definitions for the workforcemanagement_businessunits_planninggroup resource.
3.  The datasource schema definitions for the workforcemanagement_businessunits_planninggroup datasource.
4.  The resource exporter configuration for the workforcemanagement_businessunits_planninggroup exporter.
*/
const ResourceType = "genesyscloud_workforcemanagement_businessunits_planninggroup"

// serviceGoalTemplateResourceType is hard-coded to avoid an import cycle between the exporter packages
const serviceGoalTemplateResourceType = "genesyscloud_workforcemanagement_businessunits_servicegoaltemplate"

// SetRegistrar registers all of the resources, datasources and exporters in the package
func SetRegistrar(regInstance registrar.Registrar) {
	regInstance.RegisterResource(ResourceType, ResourcePlanningGroup())
	regInstance.RegisterDataSource(ResourceType, DataSourcePlanningGroup())
	regInstance.RegisterExporter(ResourceType, PlanningGroupExporter())
}

var routePathResource = &schema.Resource{
	Schema: map[string]*schema.Schema{
		"queue_id": {
			Description: `The id of the queue to associate with the route path.`,
			Type:        schema.TypeString,
			Required:    true,
		},
		"media_type": {
			Description: `The media type of the queue to associate with the route path. Valid values: Voice, Chat, Email, Callback, Message, Workitem.`,
			Type:        schema.TypeString,
			Required:    true,
		},
		"language_id": {
			Description: `The id of the language to associate with the route path.`,
			Type:        schema.TypeString,
			Optional:    true,
		},
		"skill_ids": {
			Description: `The ids of the skills to associate with the route path.`,
			Type:        schema.TypeSet,
			Optional:    true,
			Elem:        &schema.Schema{Type: schema.TypeString},
		},
	},
}

// ResourcePlanningGroup registers the genesyscloud_workforcemanagement_businessunits_planninggroup resource with Terraform
func ResourcePlanningGroup() *schema.Resource {
	return &schema.Resource{
		Description: `Genesys Cloud workforce management business unit planning group. A route path that already belongs to another planning group must be removed from that planning group first.`,

		CreateContext: provider.CreateWithPooledClient(createPlanningGroup),
		ReadContext:   provider.ReadWithPooledClient(readPlanningGroup),
		UpdateContext: provider.UpdateWithPooledClient(updatePlanningGroup),
		DeleteContext: provider.DeleteWithPooledClient(deletePlanningGroup),
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		SchemaVersion: 1,
		Schema: map[string]*schema.Schema{
			"business_unit_id": {
				Description: `The id of the business unit this planning group belongs to. Changing this attribute will cause the planning group to be dropped and recreated.`,
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
			},
			"name": {
				Description:  `The name of the planning group.`,
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.StringIsNotEmpty,
			},
			"service_goal_template_id": {
				Description: `The id of the service goal template to associate with this planning group.`,
				Type:        schema.TypeString,
				Optional:    true,
				StateFunc:   ModifyServiceGoalTemplateIdStateValue,
			},
			"route_paths": {
				Description: `The set of route paths to associate with the planning group.`,
				Type:        schema.TypeSet,
				Optional:    true,
				Elem:        routePathResource,
			},
		},
	}
}

// PlanningGroupExporter returns the resourceExporter object used to hold the genesyscloud_workforcemanagement_businessunits_planninggroup exporter's config
func PlanningGroupExporter() *resourceExporter.ResourceExporter {
	return &resourceExporter.ResourceExporter{
		GetResourcesFunc: provider.GetAllWithPooledClient(getAllPlanningGroups),
		// The service goal template resource uses composite ids (businessUnitId/serviceGoalTemplateId) as map keys, but the
		// planning group stores only the bare template id, so a custom resolver finishes the reference.
		RefAttrs: map[string]*resourceExporter.RefAttrSettings{
			"business_unit_id":         {RefType: "genesyscloud_workforcemanagement_businessunits"},
			"service_goal_template_id": {RefType: serviceGoalTemplateResourceType},
			"route_paths.queue_id":     {RefType: "genesyscloud_routing_queue"},
			"route_paths.language_id":  {RefType: "genesyscloud_routing_language"},
			"route_paths.skill_ids":    {RefType: "genesyscloud_routing_skill"},
		},
		CustomAttributeResolver: map[string]*resourceExporter.RefAttrCustomResolver{
			"service_goal_template_id": {ResolverFunc: ServiceGoalTemplateRefResolver("service_goal_template_id")},
		},
	}
}

// DataSourcePlanningGroup registers the genesyscloud_workforcemanagement_businessunits_planninggroup data source
func DataSourcePlanningGroup() *schema.Resource {
	return &schema.Resource{
		Description: `Genesys Cloud workforce management business unit planning group data source. Select a planning group by name`,
		ReadContext: provider.ReadWithPooledClient(dataSourcePlanningGroupRead),
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"business_unit_id": {
				Description: `The id of the business unit the planning group belongs to`,
				Type:        schema.TypeString,
				Required:    true,
			},
			"name": {
				Description: `Planning group name`,
				Type:        schema.TypeString,
				Required:    true,
			},
		},
	}
}
