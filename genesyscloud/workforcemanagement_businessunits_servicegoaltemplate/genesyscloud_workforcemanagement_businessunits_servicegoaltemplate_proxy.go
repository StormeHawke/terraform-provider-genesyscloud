package workforcemanagement_businessunits_servicegoaltemplate

import (
	"context"
	"fmt"

	"github.com/mypurecloud/platform-client-sdk-go/v200/platformclientv2"
)

/*
The genesyscloud_workforcemanagement_businessunits_servicegoaltemplate_proxy.go file contains the proxy structures and methods that interact
with the Genesys Cloud SDK. We use composition here for each function on the proxy so individual functions can be stubbed
out during testing.
*/

// internalProxy holds a proxy instance that can be used throughout the package
var internalProxy *serviceGoalTemplateProxy

// Type definitions for each func on our proxy so we can easily mock them out later
type createServiceGoalTemplateFunc func(ctx context.Context, p *serviceGoalTemplateProxy, businessUnitId string, template *platformclientv2.Createservicegoaltemplate) (*platformclientv2.Servicegoaltemplate, *platformclientv2.APIResponse, error)
type getAllServiceGoalTemplatesFunc func(ctx context.Context, p *serviceGoalTemplateProxy, businessUnitId string) (*[]platformclientv2.Servicegoaltemplate, *platformclientv2.APIResponse, error)
type getServiceGoalTemplateIdByNameFunc func(ctx context.Context, p *serviceGoalTemplateProxy, businessUnitId string, name string) (string, *platformclientv2.APIResponse, bool, error)
type getServiceGoalTemplateByIdFunc func(ctx context.Context, p *serviceGoalTemplateProxy, businessUnitId string, templateId string) (*platformclientv2.Servicegoaltemplate, *platformclientv2.APIResponse, error)
type updateServiceGoalTemplateFunc func(ctx context.Context, p *serviceGoalTemplateProxy, businessUnitId string, templateId string, template *platformclientv2.Updateservicegoaltemplate) (*platformclientv2.Servicegoaltemplate, *platformclientv2.APIResponse, error)
type deleteServiceGoalTemplateFunc func(ctx context.Context, p *serviceGoalTemplateProxy, businessUnitId string, templateId string) (*platformclientv2.APIResponse, error)
type getAllBusinessUnitsFunc func(ctx context.Context, p *serviceGoalTemplateProxy) (*[]platformclientv2.Businessunitlistitem, *platformclientv2.APIResponse, error)

// serviceGoalTemplateProxy contains all the methods that call genesys cloud APIs.
type serviceGoalTemplateProxy struct {
	clientConfig                       *platformclientv2.Configuration
	workforceManagementApi             *platformclientv2.WorkforceManagementApi
	createServiceGoalTemplateAttr      createServiceGoalTemplateFunc
	getAllServiceGoalTemplatesAttr     getAllServiceGoalTemplatesFunc
	getServiceGoalTemplateIdByNameAttr getServiceGoalTemplateIdByNameFunc
	getServiceGoalTemplateByIdAttr     getServiceGoalTemplateByIdFunc
	updateServiceGoalTemplateAttr      updateServiceGoalTemplateFunc
	deleteServiceGoalTemplateAttr      deleteServiceGoalTemplateFunc
	getAllBusinessUnitsAttr            getAllBusinessUnitsFunc
}

// newServiceGoalTemplateProxy initializes the service goal template proxy with all of the data needed to communicate with Genesys Cloud
func newServiceGoalTemplateProxy(clientConfig *platformclientv2.Configuration) *serviceGoalTemplateProxy {
	api := platformclientv2.NewWorkforceManagementApiWithConfig(clientConfig)
	return &serviceGoalTemplateProxy{
		clientConfig:                       clientConfig,
		workforceManagementApi:             api,
		createServiceGoalTemplateAttr:      createServiceGoalTemplateFn,
		getAllServiceGoalTemplatesAttr:     getAllServiceGoalTemplatesFn,
		getServiceGoalTemplateIdByNameAttr: getServiceGoalTemplateIdByNameFn,
		getServiceGoalTemplateByIdAttr:     getServiceGoalTemplateByIdFn,
		updateServiceGoalTemplateAttr:      updateServiceGoalTemplateFn,
		deleteServiceGoalTemplateAttr:      deleteServiceGoalTemplateFn,
		getAllBusinessUnitsAttr:            getAllBusinessUnitsFn,
	}
}

// getServiceGoalTemplateProxy acts as a singleton for the internalProxy. It also ensures
// that we can still proxy our tests by directly setting internalProxy package variable
func getServiceGoalTemplateProxy(clientConfig *platformclientv2.Configuration) *serviceGoalTemplateProxy {
	if internalProxy == nil {
		internalProxy = newServiceGoalTemplateProxy(clientConfig)
	}

	return internalProxy
}

// createServiceGoalTemplate creates a Genesys Cloud service goal template
func (p *serviceGoalTemplateProxy) createServiceGoalTemplate(ctx context.Context, businessUnitId string, template *platformclientv2.Createservicegoaltemplate) (*platformclientv2.Servicegoaltemplate, *platformclientv2.APIResponse, error) {
	return p.createServiceGoalTemplateAttr(ctx, p, businessUnitId, template)
}

// getAllServiceGoalTemplates retrieves all Genesys Cloud service goal templates of a business unit
func (p *serviceGoalTemplateProxy) getAllServiceGoalTemplates(ctx context.Context, businessUnitId string) (*[]platformclientv2.Servicegoaltemplate, *platformclientv2.APIResponse, error) {
	return p.getAllServiceGoalTemplatesAttr(ctx, p, businessUnitId)
}

// getServiceGoalTemplateIdByName returns the id of a Genesys Cloud service goal template by name
func (p *serviceGoalTemplateProxy) getServiceGoalTemplateIdByName(ctx context.Context, businessUnitId string, name string) (string, *platformclientv2.APIResponse, bool, error) {
	return p.getServiceGoalTemplateIdByNameAttr(ctx, p, businessUnitId, name)
}

// getServiceGoalTemplateById returns a single Genesys Cloud service goal template by id
func (p *serviceGoalTemplateProxy) getServiceGoalTemplateById(ctx context.Context, businessUnitId string, templateId string) (*platformclientv2.Servicegoaltemplate, *platformclientv2.APIResponse, error) {
	return p.getServiceGoalTemplateByIdAttr(ctx, p, businessUnitId, templateId)
}

// updateServiceGoalTemplate updates a Genesys Cloud service goal template
func (p *serviceGoalTemplateProxy) updateServiceGoalTemplate(ctx context.Context, businessUnitId string, templateId string, template *platformclientv2.Updateservicegoaltemplate) (*platformclientv2.Servicegoaltemplate, *platformclientv2.APIResponse, error) {
	return p.updateServiceGoalTemplateAttr(ctx, p, businessUnitId, templateId, template)
}

// deleteServiceGoalTemplate deletes a Genesys Cloud service goal template by id
func (p *serviceGoalTemplateProxy) deleteServiceGoalTemplate(ctx context.Context, businessUnitId string, templateId string) (*platformclientv2.APIResponse, error) {
	return p.deleteServiceGoalTemplateAttr(ctx, p, businessUnitId, templateId)
}

// getAllBusinessUnits retrieves all Genesys Cloud business units. Used by the exporter to walk every business unit.
func (p *serviceGoalTemplateProxy) getAllBusinessUnits(ctx context.Context) (*[]platformclientv2.Businessunitlistitem, *platformclientv2.APIResponse, error) {
	return p.getAllBusinessUnitsAttr(ctx, p)
}

// createServiceGoalTemplateFn is an implementation function for creating a Genesys Cloud service goal template
func createServiceGoalTemplateFn(ctx context.Context, p *serviceGoalTemplateProxy, businessUnitId string, template *platformclientv2.Createservicegoaltemplate) (*platformclientv2.Servicegoaltemplate, *platformclientv2.APIResponse, error) {
	return p.workforceManagementApi.PostWorkforcemanagementBusinessunitServicegoaltemplates(businessUnitId, *template)
}

// getAllServiceGoalTemplatesFn is the implementation for retrieving all service goal templates of a business unit in Genesys Cloud
func getAllServiceGoalTemplatesFn(ctx context.Context, p *serviceGoalTemplateProxy, businessUnitId string) (*[]platformclientv2.Servicegoaltemplate, *platformclientv2.APIResponse, error) {
	listing, resp, err := p.workforceManagementApi.GetWorkforcemanagementBusinessunitServicegoaltemplates(businessUnitId, nil)
	if err != nil {
		return nil, resp, err
	}

	if listing == nil || listing.Entities == nil {
		return &[]platformclientv2.Servicegoaltemplate{}, resp, nil
	}

	return listing.Entities, resp, nil
}

// getServiceGoalTemplateIdByNameFn is an implementation of the function to get the id of a service goal template by name
func getServiceGoalTemplateIdByNameFn(ctx context.Context, p *serviceGoalTemplateProxy, businessUnitId string, name string) (string, *platformclientv2.APIResponse, bool, error) {
	templates, resp, err := getAllServiceGoalTemplatesFn(ctx, p, businessUnitId)
	if err != nil {
		return "", resp, false, err
	}

	for _, template := range *templates {
		// The contract only guarantees the id, not the name
		if template.Name != nil && *template.Name == name {
			return *template.Id, resp, false, nil
		}
	}

	return "", resp, true, fmt.Errorf("unable to find business unit %s service goal template with name %s", businessUnitId, name)
}

// getServiceGoalTemplateByIdFn is an implementation of the function to get a Genesys Cloud service goal template by id
func getServiceGoalTemplateByIdFn(ctx context.Context, p *serviceGoalTemplateProxy, businessUnitId string, templateId string) (*platformclientv2.Servicegoaltemplate, *platformclientv2.APIResponse, error) {
	return p.workforceManagementApi.GetWorkforcemanagementBusinessunitServicegoaltemplate(businessUnitId, templateId, nil)
}

// updateServiceGoalTemplateFn is an implementation of the function to update a Genesys Cloud service goal template
func updateServiceGoalTemplateFn(ctx context.Context, p *serviceGoalTemplateProxy, businessUnitId string, templateId string, template *platformclientv2.Updateservicegoaltemplate) (*platformclientv2.Servicegoaltemplate, *platformclientv2.APIResponse, error) {
	return p.workforceManagementApi.PatchWorkforcemanagementBusinessunitServicegoaltemplate(businessUnitId, templateId, *template)
}

// deleteServiceGoalTemplateFn is an implementation function for deleting a Genesys Cloud service goal template
func deleteServiceGoalTemplateFn(ctx context.Context, p *serviceGoalTemplateProxy, businessUnitId string, templateId string) (*platformclientv2.APIResponse, error) {
	return p.workforceManagementApi.DeleteWorkforcemanagementBusinessunitServicegoaltemplate(businessUnitId, templateId)
}

// getAllBusinessUnitsFn is an implementation of the function to list Genesys Cloud business units
func getAllBusinessUnitsFn(ctx context.Context, p *serviceGoalTemplateProxy) (*[]platformclientv2.Businessunitlistitem, *platformclientv2.APIResponse, error) {
	listing, resp, err := p.workforceManagementApi.GetWorkforcemanagementBusinessunits("", "")
	if err != nil {
		return nil, resp, err
	}

	if listing == nil || listing.Entities == nil {
		return &[]platformclientv2.Businessunitlistitem{}, resp, nil
	}

	return listing.Entities, resp, nil
}
