package workforcemanagement_managementunits_agents

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/mypurecloud/terraform-provider-genesyscloud/genesyscloud/provider"
	resourceExporter "github.com/mypurecloud/terraform-provider-genesyscloud/genesyscloud/resource_exporter"
	registrar "github.com/mypurecloud/terraform-provider-genesyscloud/genesyscloud/resource_register"
)

/*
resource_genesyscloud_workforcemanagement_managementunits_agents_schema.go holds three functions within it:

1.  The registration code that registers the Resource and Exporter for the package.
2.  The resource schema definitions for the workforcemanagement_managementunits_agents resource.
3.  The resource exporter configuration for the workforcemanagement_managementunits_agents exporter.

There is no data source: the resource represents the membership of a management unit rather than a named entity.
*/
const ResourceType = "genesyscloud_workforcemanagement_managementunits_agents"

// SetRegistrar registers all of the resources and exporters in the package
func SetRegistrar(regInstance registrar.Registrar) {
	regInstance.RegisterResource(ResourceType, ResourceManagementUnitAgents())
	regInstance.RegisterExporter(ResourceType, ManagementUnitAgentsExporter())
}

// ResourceManagementUnitAgents registers the genesyscloud_workforcemanagement_managementunits_agents resource with Terraform
func ResourceManagementUnitAgents() *schema.Resource {
	return &schema.Resource{
		Description: `Manages the agents of a Genesys Cloud workforce management management unit. This resource is authoritative for the management unit: any user that is a member of the management unit but is not listed in schedulable_user_ids or unschedulable_user_ids is removed from it, and an agent can only belong to one management unit, so listing a user here moves them out of their current management unit. Do not manage the same user from more than one of these resources. Destroying this resource removes the agents it manages from the management unit; it does not delete the management unit or the users. Agents are moved asynchronously, so applying can take a few minutes.`,

		CreateContext: provider.CreateWithPooledClient(createManagementUnitAgents),
		ReadContext:   provider.ReadWithPooledClient(readManagementUnitAgents),
		UpdateContext: provider.UpdateWithPooledClient(updateManagementUnitAgents),
		DeleteContext: provider.DeleteWithPooledClient(deleteManagementUnitAgents),
		CustomizeDiff: validateDistinctUserSets,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		SchemaVersion: 1,
		Schema: map[string]*schema.Schema{
			"management_unit_id": {
				Description: `The id of the management unit whose agents are managed by this resource. Changing this attribute will cause the resource to be dropped and recreated.`,
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
			},
			"schedulable_user_ids": {
				Description: `The ids of the users that are members of the management unit and can be included in schedule generation. A user can only be listed in one of schedulable_user_ids and unschedulable_user_ids.`,
				Type:        schema.TypeSet,
				Optional:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
			"unschedulable_user_ids": {
				Description: `The ids of the users that are members of the management unit but cannot be included in schedule generation. A user can only be listed in one of schedulable_user_ids and unschedulable_user_ids.`,
				Type:        schema.TypeSet,
				Optional:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
		},
	}
}

// ManagementUnitAgentsExporter returns the resourceExporter object used to hold the genesyscloud_workforcemanagement_managementunits_agents exporter's config
func ManagementUnitAgentsExporter() *resourceExporter.ResourceExporter {
	return &resourceExporter.ResourceExporter{
		GetResourcesFunc: provider.GetAllWithPooledClient(getAllManagementUnitAgents),
		RefAttrs: map[string]*resourceExporter.RefAttrSettings{
			"management_unit_id":     {RefType: "genesyscloud_workforcemanagement_managementunits"},
			"schedulable_user_ids":   {RefType: "genesyscloud_user"},
			"unschedulable_user_ids": {RefType: "genesyscloud_user"},
		},
	}
}
