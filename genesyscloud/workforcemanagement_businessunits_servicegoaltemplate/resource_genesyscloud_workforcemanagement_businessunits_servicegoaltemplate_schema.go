package workforcemanagement_businessunits_servicegoaltemplate

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/mypurecloud/terraform-provider-genesyscloud/genesyscloud/provider"
	resourceExporter "github.com/mypurecloud/terraform-provider-genesyscloud/genesyscloud/resource_exporter"
	registrar "github.com/mypurecloud/terraform-provider-genesyscloud/genesyscloud/resource_register"
)

/*
resource_genesyscloud_workforcemanagement_businessunits_servicegoaltemplate_schema.go holds four functions within it:

1.  The registration code that registers the Datasource, Resource and Exporter for the package.
2.  The resource schema definitions for the workforcemanagement_businessunits_servicegoaltemplate resource.
3.  The datasource schema definitions for the workforcemanagement_businessunits_servicegoaltemplate datasource.
4.  The resource exporter configuration for the workforcemanagement_businessunits_servicegoaltemplate exporter.
*/
const ResourceType = "genesyscloud_workforcemanagement_businessunits_servicegoaltemplate"

// SetRegistrar registers all of the resources, datasources and exporters in the package
func SetRegistrar(regInstance registrar.Registrar) {
	regInstance.RegisterResource(ResourceType, ResourceServiceGoalTemplate())
	regInstance.RegisterDataSource(ResourceType, DataSourceServiceGoalTemplate())
	regInstance.RegisterExporter(ResourceType, ServiceGoalTemplateExporter())
}

var serviceLevelResource = &schema.Resource{
	Schema: map[string]*schema.Schema{
		"include": {
			Description: `Whether to include service level targets in this service goal template.`,
			Type:        schema.TypeBool,
			Required:    true,
		},
		"percent": {
			Description:  `Service level target percent answered. Required if include is true.`,
			Type:         schema.TypeInt,
			Optional:     true,
			Computed:     true,
			ValidateFunc: validation.IntBetween(0, 100),
		},
		"seconds": {
			Description:  `Service level target answer time in seconds. Required if include is true.`,
			Type:         schema.TypeInt,
			Optional:     true,
			Computed:     true,
			ValidateFunc: validation.IntAtLeast(0),
		},
	},
}

var averageSpeedOfAnswerResource = &schema.Resource{
	Schema: map[string]*schema.Schema{
		"include": {
			Description: `Whether to include average speed of answer (ASA) in this service goal template.`,
			Type:        schema.TypeBool,
			Required:    true,
		},
		"seconds": {
			Description:  `The target average speed of answer (ASA) in seconds. Required if include is true.`,
			Type:         schema.TypeInt,
			Optional:     true,
			Computed:     true,
			ValidateFunc: validation.IntAtLeast(0),
		},
	},
}

var abandonRateResource = &schema.Resource{
	Schema: map[string]*schema.Schema{
		"include": {
			Description: `Whether to include abandon rate in this service goal template.`,
			Type:        schema.TypeBool,
			Required:    true,
		},
		"percent": {
			Description:  `Abandon rate percent goal. Required if include is true.`,
			Type:         schema.TypeInt,
			Optional:     true,
			Computed:     true,
			ValidateFunc: validation.IntBetween(0, 100),
		},
	},
}

var serviceGoalImpactResource = &schema.Resource{
	Schema: map[string]*schema.Schema{
		"increase_by_percent": {
			Description: `The maximum allowed percent increase from the configured goal.`,
			Type:        schema.TypeFloat,
			Required:    true,
		},
		"decrease_by_percent": {
			Description: `The maximum allowed percent decrease from the configured goal.`,
			Type:        schema.TypeFloat,
			Required:    true,
		},
	},
}

var serviceGoalImpactSettingsResource = &schema.Resource{
	Schema: map[string]*schema.Schema{
		"service_level": {
			Description: `Allowed service level percent increase and decrease.`,
			Type:        schema.TypeList,
			Optional:    true,
			MaxItems:    1,
			Elem:        serviceGoalImpactResource,
		},
		"average_speed_of_answer": {
			Description: `Allowed average speed of answer percent increase and decrease.`,
			Type:        schema.TypeList,
			Optional:    true,
			MaxItems:    1,
			Elem:        serviceGoalImpactResource,
		},
		"abandon_rate": {
			Description: `Allowed abandon rate percent increase and decrease.`,
			Type:        schema.TypeList,
			Optional:    true,
			MaxItems:    1,
			Elem:        serviceGoalImpactResource,
		},
	},
}

var impactOverrideResource = &schema.Resource{
	Schema: map[string]*schema.Schema{
		"enabled": {
			Description: `Whether service goal overrides are enabled for this service goal template.`,
			Type:        schema.TypeBool,
			Required:    true,
		},
		"impact": {
			Description: `Settings controlling the max percent increase and decrease of service goals for this service goal template.`,
			Type:        schema.TypeList,
			Optional:    true,
			Computed:    true,
			MaxItems:    1,
			Elem:        serviceGoalImpactSettingsResource,
		},
	},
}

// ResourceServiceGoalTemplate registers the genesyscloud_workforcemanagement_businessunits_servicegoaltemplate resource with Terraform
func ResourceServiceGoalTemplate() *schema.Resource {
	return &schema.Resource{
		Description: `Genesys Cloud workforce management business unit service goal template`,

		CreateContext: provider.CreateWithPooledClient(createServiceGoalTemplate),
		ReadContext:   provider.ReadWithPooledClient(readServiceGoalTemplate),
		UpdateContext: provider.UpdateWithPooledClient(updateServiceGoalTemplate),
		DeleteContext: provider.DeleteWithPooledClient(deleteServiceGoalTemplate),
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		SchemaVersion: 1,
		Schema: map[string]*schema.Schema{
			"business_unit_id": {
				Description: `The id of the business unit this service goal template belongs to. Changing this attribute will cause the service goal template to be dropped and recreated.`,
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
			},
			"name": {
				Description:  `The name of the service goal template.`,
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.StringIsNotEmpty,
			},
			"service_level": {
				Description: `Service level targets for this service goal template.`,
				Type:        schema.TypeList,
				Optional:    true,
				Computed:    true,
				MaxItems:    1,
				Elem:        serviceLevelResource,
			},
			"average_speed_of_answer": {
				Description: `Average speed of answer targets for this service goal template.`,
				Type:        schema.TypeList,
				Optional:    true,
				Computed:    true,
				MaxItems:    1,
				Elem:        averageSpeedOfAnswerResource,
			},
			"abandon_rate": {
				Description: `Abandon rate targets for this service goal template.`,
				Type:        schema.TypeList,
				Optional:    true,
				Computed:    true,
				MaxItems:    1,
				Elem:        abandonRateResource,
			},
			"impact_override": {
				Description: `Settings controlling the max percent increase and decrease of service goals for this service goal template.`,
				Type:        schema.TypeList,
				Optional:    true,
				Computed:    true,
				MaxItems:    1,
				Elem:        impactOverrideResource,
			},
		},
	}
}

// ServiceGoalTemplateExporter returns the resourceExporter object used to hold the genesyscloud_workforcemanagement_businessunits_servicegoaltemplate exporter's config
func ServiceGoalTemplateExporter() *resourceExporter.ResourceExporter {
	return &resourceExporter.ResourceExporter{
		GetResourcesFunc: provider.GetAllWithPooledClient(getAllServiceGoalTemplates),
		RefAttrs: map[string]*resourceExporter.RefAttrSettings{
			"business_unit_id": {RefType: "genesyscloud_workforcemanagement_businessunits"},
		},
	}
}

// DataSourceServiceGoalTemplate registers the genesyscloud_workforcemanagement_businessunits_servicegoaltemplate data source
func DataSourceServiceGoalTemplate() *schema.Resource {
	return &schema.Resource{
		Description: `Genesys Cloud workforce management business unit service goal template data source. Select a service goal template by name`,
		ReadContext: provider.ReadWithPooledClient(dataSourceServiceGoalTemplateRead),
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"business_unit_id": {
				Description: `The id of the business unit the service goal template belongs to`,
				Type:        schema.TypeString,
				Required:    true,
			},
			"name": {
				Description: `Service goal template name`,
				Type:        schema.TypeString,
				Required:    true,
			},
		},
	}
}
