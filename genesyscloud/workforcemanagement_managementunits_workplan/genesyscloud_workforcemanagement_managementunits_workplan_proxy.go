package workforcemanagement_managementunits_workplan

import (
	"context"
	"fmt"

	"github.com/mypurecloud/platform-client-sdk-go/v200/platformclientv2"
)

/*
The genesyscloud_workforcemanagement_managementunits_workplan_proxy.go file contains the proxy structures and methods that interact
with the Genesys Cloud SDK. We use composition here for each function on the proxy so individual functions can be stubbed
out during testing.
*/

// internalProxy holds a proxy instance that can be used throughout the package
var internalProxy *workPlanProxy

// Type definitions for each func on our proxy so we can easily mock them out later
type createWorkPlanFunc func(ctx context.Context, p *workPlanProxy, managementUnitId string, workPlan *platformclientv2.Createworkplan) (*platformclientv2.Workplan, *platformclientv2.APIResponse, error)
type getAllWorkPlansFunc func(ctx context.Context, p *workPlanProxy, managementUnitId string) (*[]platformclientv2.Workplanlistitemresponse, *platformclientv2.APIResponse, error)
type getWorkPlanIdByNameFunc func(ctx context.Context, p *workPlanProxy, managementUnitId string, name string) (string, *platformclientv2.APIResponse, bool, error)
type getWorkPlanByIdFunc func(ctx context.Context, p *workPlanProxy, managementUnitId string, workPlanId string) (*platformclientv2.Workplan, *platformclientv2.APIResponse, error)
type updateWorkPlanFunc func(ctx context.Context, p *workPlanProxy, managementUnitId string, workPlanId string, workPlan *platformclientv2.Workplan) (*platformclientv2.Workplan, *platformclientv2.APIResponse, error)
type validateWorkPlanFunc func(ctx context.Context, p *workPlanProxy, managementUnitId string, workPlanId string, workPlan *platformclientv2.Workplanvalidationrequest) (*platformclientv2.Validateworkplanresponse, *platformclientv2.APIResponse, error)
type deleteWorkPlanFunc func(ctx context.Context, p *workPlanProxy, managementUnitId string, workPlanId string) (*platformclientv2.APIResponse, error)
type getAllManagementUnitsFunc func(ctx context.Context, p *workPlanProxy) (*[]platformclientv2.Managementunit, *platformclientv2.APIResponse, error)

// workPlanProxy contains all the methods that call genesys cloud APIs.
type workPlanProxy struct {
	clientConfig              *platformclientv2.Configuration
	workforceManagementApi    *platformclientv2.WorkforceManagementApi
	createWorkPlanAttr        createWorkPlanFunc
	getAllWorkPlansAttr       getAllWorkPlansFunc
	getWorkPlanIdByNameAttr   getWorkPlanIdByNameFunc
	getWorkPlanByIdAttr       getWorkPlanByIdFunc
	updateWorkPlanAttr        updateWorkPlanFunc
	validateWorkPlanAttr      validateWorkPlanFunc
	deleteWorkPlanAttr        deleteWorkPlanFunc
	getAllManagementUnitsAttr getAllManagementUnitsFunc
}

// newWorkPlanProxy initializes the work plan proxy with all of the data needed to communicate with Genesys Cloud
func newWorkPlanProxy(clientConfig *platformclientv2.Configuration) *workPlanProxy {
	api := platformclientv2.NewWorkforceManagementApiWithConfig(clientConfig)
	return &workPlanProxy{
		clientConfig:              clientConfig,
		workforceManagementApi:    api,
		createWorkPlanAttr:        createWorkPlanFn,
		getAllWorkPlansAttr:       getAllWorkPlansFn,
		getWorkPlanIdByNameAttr:   getWorkPlanIdByNameFn,
		getWorkPlanByIdAttr:       getWorkPlanByIdFn,
		updateWorkPlanAttr:        updateWorkPlanFn,
		validateWorkPlanAttr:      validateWorkPlanFn,
		deleteWorkPlanAttr:        deleteWorkPlanFn,
		getAllManagementUnitsAttr: getAllManagementUnitsFn,
	}
}

// getWorkPlanProxy acts as a singleton for the internalProxy. It also ensures
// that we can still proxy our tests by directly setting internalProxy package variable
func getWorkPlanProxy(clientConfig *platformclientv2.Configuration) *workPlanProxy {
	if internalProxy == nil {
		internalProxy = newWorkPlanProxy(clientConfig)
	}

	return internalProxy
}

// createWorkPlan creates a Genesys Cloud work plan
func (p *workPlanProxy) createWorkPlan(ctx context.Context, managementUnitId string, workPlan *platformclientv2.Createworkplan) (*platformclientv2.Workplan, *platformclientv2.APIResponse, error) {
	return p.createWorkPlanAttr(ctx, p, managementUnitId, workPlan)
}

// getAllWorkPlans retrieves all Genesys Cloud work plans of a management unit
func (p *workPlanProxy) getAllWorkPlans(ctx context.Context, managementUnitId string) (*[]platformclientv2.Workplanlistitemresponse, *platformclientv2.APIResponse, error) {
	return p.getAllWorkPlansAttr(ctx, p, managementUnitId)
}

// getWorkPlanIdByName returns the id of a Genesys Cloud work plan by name
func (p *workPlanProxy) getWorkPlanIdByName(ctx context.Context, managementUnitId string, name string) (string, *platformclientv2.APIResponse, bool, error) {
	return p.getWorkPlanIdByNameAttr(ctx, p, managementUnitId, name)
}

// getWorkPlanById returns a single Genesys Cloud work plan, including its shifts and agents, by id
func (p *workPlanProxy) getWorkPlanById(ctx context.Context, managementUnitId string, workPlanId string) (*platformclientv2.Workplan, *platformclientv2.APIResponse, error) {
	return p.getWorkPlanByIdAttr(ctx, p, managementUnitId, workPlanId)
}

// updateWorkPlan updates a Genesys Cloud work plan
func (p *workPlanProxy) updateWorkPlan(ctx context.Context, managementUnitId string, workPlanId string, workPlan *platformclientv2.Workplan) (*platformclientv2.Workplan, *platformclientv2.APIResponse, error) {
	return p.updateWorkPlanAttr(ctx, p, managementUnitId, workPlanId, workPlan)
}

// validateWorkPlan validates a work plan without saving it. Use the id "new" to validate a work plan that does not exist yet.
func (p *workPlanProxy) validateWorkPlan(ctx context.Context, managementUnitId string, workPlanId string, workPlan *platformclientv2.Workplanvalidationrequest) (*platformclientv2.Validateworkplanresponse, *platformclientv2.APIResponse, error) {
	return p.validateWorkPlanAttr(ctx, p, managementUnitId, workPlanId, workPlan)
}

// deleteWorkPlan deletes a Genesys Cloud work plan by id
func (p *workPlanProxy) deleteWorkPlan(ctx context.Context, managementUnitId string, workPlanId string) (*platformclientv2.APIResponse, error) {
	return p.deleteWorkPlanAttr(ctx, p, managementUnitId, workPlanId)
}

// getAllManagementUnits retrieves all Genesys Cloud management units. Used by the exporter to walk every management unit.
func (p *workPlanProxy) getAllManagementUnits(ctx context.Context) (*[]platformclientv2.Managementunit, *platformclientv2.APIResponse, error) {
	return p.getAllManagementUnitsAttr(ctx, p)
}

// createWorkPlanFn is an implementation function for creating a Genesys Cloud work plan.
// The validation mode is left unset, so an enabled work plan that fails validation is rejected by the api.
func createWorkPlanFn(ctx context.Context, p *workPlanProxy, managementUnitId string, workPlan *platformclientv2.Createworkplan) (*platformclientv2.Workplan, *platformclientv2.APIResponse, error) {
	return p.workforceManagementApi.PostWorkforcemanagementManagementunitWorkplans(managementUnitId, *workPlan, "")
}

// getAllWorkPlansFn is the implementation for retrieving all work plans of a management unit in Genesys Cloud
func getAllWorkPlansFn(ctx context.Context, p *workPlanProxy, managementUnitId string) (*[]platformclientv2.Workplanlistitemresponse, *platformclientv2.APIResponse, error) {
	listing, resp, err := p.workforceManagementApi.GetWorkforcemanagementManagementunitWorkplans(managementUnitId, nil, nil)
	if err != nil {
		return nil, resp, err
	}

	if listing == nil || listing.Entities == nil {
		return &[]platformclientv2.Workplanlistitemresponse{}, resp, nil
	}

	return listing.Entities, resp, nil
}

// getWorkPlanIdByNameFn is an implementation of the function to get the id of a work plan by name
func getWorkPlanIdByNameFn(ctx context.Context, p *workPlanProxy, managementUnitId string, name string) (string, *platformclientv2.APIResponse, bool, error) {
	workPlans, resp, err := getAllWorkPlansFn(ctx, p, managementUnitId)
	if err != nil {
		return "", resp, false, err
	}

	for _, workPlan := range *workPlans {
		if workPlan.Name != nil && *workPlan.Name == name && workPlan.Id != nil {
			return *workPlan.Id, resp, false, nil
		}
	}

	return "", resp, true, fmt.Errorf("unable to find management unit %s work plan with name %s", managementUnitId, name)
}

// getWorkPlanByIdFn is an implementation of the function to get a Genesys Cloud work plan by id
func getWorkPlanByIdFn(ctx context.Context, p *workPlanProxy, managementUnitId string, workPlanId string) (*platformclientv2.Workplan, *platformclientv2.APIResponse, error) {
	return p.workforceManagementApi.GetWorkforcemanagementManagementunitWorkplan(managementUnitId, workPlanId, nil)
}

// updateWorkPlanFn is an implementation of the function to update a Genesys Cloud work plan
func updateWorkPlanFn(ctx context.Context, p *workPlanProxy, managementUnitId string, workPlanId string, workPlan *platformclientv2.Workplan) (*platformclientv2.Workplan, *platformclientv2.APIResponse, error) {
	return p.workforceManagementApi.PatchWorkforcemanagementManagementunitWorkplan(managementUnitId, workPlanId, *workPlan, "")
}

// validateWorkPlanFn is an implementation of the function to validate a Genesys Cloud work plan, with the validation messages expanded
func validateWorkPlanFn(ctx context.Context, p *workPlanProxy, managementUnitId string, workPlanId string, workPlan *platformclientv2.Workplanvalidationrequest) (*platformclientv2.Validateworkplanresponse, *platformclientv2.APIResponse, error) {
	return p.workforceManagementApi.PostWorkforcemanagementManagementunitWorkplanValidate(managementUnitId, workPlanId, *workPlan, []string{"messages"})
}

// deleteWorkPlanFn is an implementation function for deleting a Genesys Cloud work plan
func deleteWorkPlanFn(ctx context.Context, p *workPlanProxy, managementUnitId string, workPlanId string) (*platformclientv2.APIResponse, error) {
	return p.workforceManagementApi.DeleteWorkforcemanagementManagementunitWorkplan(managementUnitId, workPlanId)
}

// getAllManagementUnitsFn is an implementation of the function to list Genesys Cloud management units.
// The api documents paging as unsupported and returns every management unit, so the paging arguments are placeholders.
func getAllManagementUnitsFn(ctx context.Context, p *workPlanProxy) (*[]platformclientv2.Managementunit, *platformclientv2.APIResponse, error) {
	listing, resp, err := p.workforceManagementApi.GetWorkforcemanagementManagementunits(100, 1, "", "", "")
	if err != nil {
		return nil, resp, err
	}

	if listing == nil || listing.Entities == nil {
		return &[]platformclientv2.Managementunit{}, resp, nil
	}

	return listing.Entities, resp, nil
}
