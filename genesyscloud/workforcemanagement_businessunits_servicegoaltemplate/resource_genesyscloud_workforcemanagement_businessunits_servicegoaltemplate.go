package workforcemanagement_businessunits_servicegoaltemplate

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/mypurecloud/platform-client-sdk-go/v200/platformclientv2"
	"github.com/mypurecloud/terraform-provider-genesyscloud/genesyscloud/consistency_checker"
	"github.com/mypurecloud/terraform-provider-genesyscloud/genesyscloud/provider"
	resourceExporter "github.com/mypurecloud/terraform-provider-genesyscloud/genesyscloud/resource_exporter"
	"github.com/mypurecloud/terraform-provider-genesyscloud/genesyscloud/util"
	"github.com/mypurecloud/terraform-provider-genesyscloud/genesyscloud/util/constants"
)

/*
	NOTE: This resource's Id is in the format <businessUnitId>/<serviceGoalTemplateId> so we can persist the id of the parent business unit.
	The business_unit_id field can not be used for this because the service goal template api does not return the business unit.
*/

/*
The resource_genesyscloud_workforcemanagement_businessunits_servicegoaltemplate.go contains all of the methods that perform the core logic for a resource.
*/

// getAllServiceGoalTemplates retrieves all of the business unit service goal templates via Terraform in the Genesys Cloud and is used for the exporter.
func getAllServiceGoalTemplates(ctx context.Context, clientConfig *platformclientv2.Configuration) (resourceExporter.ResourceIDMetaMap, diag.Diagnostics) {
	proxy := getServiceGoalTemplateProxy(clientConfig)
	resources := make(resourceExporter.ResourceIDMetaMap)

	businessUnits, resp, err := proxy.getAllBusinessUnits(ctx)
	if err != nil {
		return nil, util.BuildAPIDiagnosticError(ResourceType, fmt.Sprintf("Failed to get business units: %v", err), resp)
	}

	for _, businessUnit := range *businessUnits {
		templates, resp, err := proxy.getAllServiceGoalTemplates(ctx, *businessUnit.Id)
		if err != nil {
			return nil, util.BuildAPIDiagnosticError(ResourceType, fmt.Sprintf("Failed to get service goal templates of business unit %s: %v", *businessUnit.Id, err), resp)
		}

		for _, template := range *templates {
			resources[*businessUnit.Id+"/"+*template.Id] = &resourceExporter.ResourceMeta{BlockLabel: *businessUnit.Name + "_" + *template.Name}
		}
	}

	return resources, nil
}

// createServiceGoalTemplate is used by the workforcemanagement_businessunits_servicegoaltemplate resource to create a Genesys Cloud business unit service goal template
func createServiceGoalTemplate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	sdkConfig := meta.(*provider.ProviderMeta).ClientConfig
	proxy := getServiceGoalTemplateProxy(sdkConfig)
	businessUnitId := d.Get("business_unit_id").(string)
	template := buildCreateServiceGoalTemplateRequest(d)

	log.Printf("Creating business unit %s service goal template %s", businessUnitId, *template.Name)
	created, resp, err := proxy.createServiceGoalTemplate(ctx, businessUnitId, template)
	if err != nil {
		return util.BuildAPIDiagnosticError(ResourceType, fmt.Sprintf("Failed to create business unit %s service goal template %s: %s", businessUnitId, *template.Name, err), resp)
	}

	d.SetId(businessUnitId + "/" + *created.Id)
	log.Printf("Created business unit %s service goal template %s %s", businessUnitId, *created.Id, *template.Name)
	return readServiceGoalTemplate(ctx, d, meta)
}

// readServiceGoalTemplate is used by the workforcemanagement_businessunits_servicegoaltemplate resource to read a business unit service goal template from Genesys Cloud
func readServiceGoalTemplate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	sdkConfig := meta.(*provider.ProviderMeta).ClientConfig
	proxy := getServiceGoalTemplateProxy(sdkConfig)
	cc := consistency_checker.NewConsistencyCheck(ctx, d, meta, ResourceServiceGoalTemplate(), constants.ConsistencyChecks(), ResourceType)
	businessUnitId, templateId := SplitServiceGoalTemplateTerraformId(d.Id())

	log.Printf("Reading business unit %s service goal template %s", businessUnitId, templateId)

	return util.WithRetriesForRead(ctx, d, func() *retry.RetryError {
		template, resp, getErr := proxy.getServiceGoalTemplateById(ctx, businessUnitId, templateId)
		if getErr != nil {
			if util.IsStatus404(resp) {
				return retry.RetryableError(util.BuildWithRetriesApiDiagnosticError(ResourceType, fmt.Sprintf("Failed to read business unit %s service goal template %s: %s", businessUnitId, templateId, getErr), resp))
			}
			return retry.NonRetryableError(util.BuildWithRetriesApiDiagnosticError(ResourceType, fmt.Sprintf("Failed to read business unit %s service goal template %s: %s", businessUnitId, templateId, getErr), resp))
		}

		_ = d.Set("business_unit_id", businessUnitId)
		_ = d.Set("name", *template.Name)
		_ = d.Set("service_level", flattenServiceLevel(template.ServiceLevel))
		_ = d.Set("average_speed_of_answer", flattenAverageSpeedOfAnswer(template.AverageSpeedOfAnswer))
		_ = d.Set("abandon_rate", flattenAbandonRate(template.AbandonRate))
		_ = d.Set("impact_override", flattenImpactOverride(template.ImpactOverride))

		log.Printf("Read business unit %s service goal template %s", businessUnitId, templateId)
		return cc.CheckState(d)
	})
}

// updateServiceGoalTemplate is used by the workforcemanagement_businessunits_servicegoaltemplate resource to update a business unit service goal template in Genesys Cloud
func updateServiceGoalTemplate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	sdkConfig := meta.(*provider.ProviderMeta).ClientConfig
	proxy := getServiceGoalTemplateProxy(sdkConfig)
	businessUnitId, templateId := SplitServiceGoalTemplateTerraformId(d.Id())

	log.Printf("Updating business unit %s service goal template %s", businessUnitId, templateId)

	// The update request carries the current metadata version for optimistic locking, so a version conflict re-reads the service goal template and tries again.
	// Rate limiting and server errors are transient, and retrying is safe because every attempt rebuilds the request from the freshly read metadata.
	diagErr := util.WithRetries(ctx, 60*time.Second, func() *retry.RetryError {
		current, resp, err := proxy.getServiceGoalTemplateById(ctx, businessUnitId, templateId)
		if err != nil {
			readDiagErr := util.BuildWithRetriesApiDiagnosticError(ResourceType, fmt.Sprintf("Failed to read business unit %s service goal template %s before update: %s", businessUnitId, templateId, err), resp)
			if isTransientStatus(resp) {
				return retry.RetryableError(readDiagErr)
			}
			return retry.NonRetryableError(readDiagErr)
		}

		_, resp, err = proxy.updateServiceGoalTemplate(ctx, businessUnitId, templateId, buildUpdateServiceGoalTemplateRequest(d, current.Metadata))
		if err == nil {
			return nil
		}
		builtDiagErr := util.BuildWithRetriesApiDiagnosticError(ResourceType, fmt.Sprintf("Failed to update business unit %s service goal template %s: %s", businessUnitId, templateId, err), resp)
		if util.IsStatus409(resp) || isTransientStatus(resp) {
			return retry.RetryableError(builtDiagErr)
		}
		return retry.NonRetryableError(builtDiagErr)
	})
	if diagErr != nil {
		return diagErr
	}

	log.Printf("Updated business unit %s service goal template %s", businessUnitId, templateId)
	return readServiceGoalTemplate(ctx, d, meta)
}

func isTransientStatus(resp *platformclientv2.APIResponse) bool {
	return util.IsStatus429(resp) || (resp != nil && resp.StatusCode >= http.StatusInternalServerError)
}

// deleteServiceGoalTemplate is used by the workforcemanagement_businessunits_servicegoaltemplate resource to delete a business unit service goal template from Genesys Cloud
func deleteServiceGoalTemplate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	sdkConfig := meta.(*provider.ProviderMeta).ClientConfig
	proxy := getServiceGoalTemplateProxy(sdkConfig)
	businessUnitId, templateId := SplitServiceGoalTemplateTerraformId(d.Id())

	log.Printf("Deleting business unit %s service goal template %s", businessUnitId, templateId)
	resp, err := proxy.deleteServiceGoalTemplate(ctx, businessUnitId, templateId)
	if err != nil {
		if util.IsStatus404(resp) {
			log.Printf("Business unit %s service goal template %s already deleted", businessUnitId, templateId)
			return nil
		}
		return util.BuildAPIDiagnosticError(ResourceType, fmt.Sprintf("Failed to delete business unit %s service goal template %s: %s", businessUnitId, templateId, err), resp)
	}

	// The delete is eventually consistent, so poll until the service goal template is gone (404). The 3 minute cap is the upper bound on that wait before the destroy fails
	return util.WithRetries(ctx, 180*time.Second, func() *retry.RetryError {
		_, resp, err := proxy.getServiceGoalTemplateById(ctx, businessUnitId, templateId)
		if err != nil {
			if util.IsStatus404(resp) {
				log.Printf("Deleted business unit %s service goal template %s", businessUnitId, templateId)
				return nil
			}
			return retry.NonRetryableError(util.BuildWithRetriesApiDiagnosticError(ResourceType, fmt.Sprintf("Error deleting business unit %s service goal template %s: %s", businessUnitId, templateId, err), resp))
		}

		return retry.RetryableError(util.BuildWithRetriesApiDiagnosticError(ResourceType, fmt.Sprintf("business unit %s service goal template %s still exists", businessUnitId, templateId), resp))
	})
}
