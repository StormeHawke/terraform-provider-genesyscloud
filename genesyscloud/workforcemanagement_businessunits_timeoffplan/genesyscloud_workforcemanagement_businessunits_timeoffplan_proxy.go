package workforcemanagement_businessunits_timeoffplan

import (
	"context"
	"fmt"

	"github.com/mypurecloud/platform-client-sdk-go/v200/platformclientv2"
)

/*
The genesyscloud_workforcemanagement_businessunits_timeoffplan_proxy.go file contains the proxy structures and methods that interact
with the Genesys Cloud SDK. We use composition here for each function on the proxy so individual functions can be stubbed
out during testing.
*/

// internalProxy holds a proxy instance that can be used throughout the package
var internalProxy *timeOffPlanProxy

// Type definitions for each func on our proxy so we can easily mock them out later
type createTimeOffPlanFunc func(ctx context.Context, p *timeOffPlanProxy, businessUnitId string, plan *platformclientv2.Bucreatetimeoffplanrequest) (*platformclientv2.Butimeoffplanresponse, *platformclientv2.APIResponse, error)
type getAllTimeOffPlansFunc func(ctx context.Context, p *timeOffPlanProxy, businessUnitId string) (*[]platformclientv2.Butimeoffplanresponse, *platformclientv2.APIResponse, error)
type getTimeOffPlanIdByNameFunc func(ctx context.Context, p *timeOffPlanProxy, businessUnitId string, name string) (string, *platformclientv2.APIResponse, bool, error)
type getTimeOffPlanByIdFunc func(ctx context.Context, p *timeOffPlanProxy, businessUnitId string, planId string) (*platformclientv2.Butimeoffplanresponse, *platformclientv2.APIResponse, error)
type updateTimeOffPlanFunc func(ctx context.Context, p *timeOffPlanProxy, businessUnitId string, planId string, plan *platformclientv2.Buupdatetimeoffplanrequest) (*platformclientv2.Butimeoffplanresponse, *platformclientv2.APIResponse, error)
type deleteTimeOffPlanFunc func(ctx context.Context, p *timeOffPlanProxy, businessUnitId string, planId string) (*platformclientv2.APIResponse, error)
type getAllBusinessUnitsFunc func(ctx context.Context, p *timeOffPlanProxy) (*[]platformclientv2.Businessunitlistitem, *platformclientv2.APIResponse, error)

// timeOffPlanProxy contains all the methods that call genesys cloud APIs.
type timeOffPlanProxy struct {
	clientConfig               *platformclientv2.Configuration
	workforceManagementApi     *platformclientv2.WorkforceManagementApi
	createTimeOffPlanAttr      createTimeOffPlanFunc
	getAllTimeOffPlansAttr     getAllTimeOffPlansFunc
	getTimeOffPlanIdByNameAttr getTimeOffPlanIdByNameFunc
	getTimeOffPlanByIdAttr     getTimeOffPlanByIdFunc
	updateTimeOffPlanAttr      updateTimeOffPlanFunc
	deleteTimeOffPlanAttr      deleteTimeOffPlanFunc
	getAllBusinessUnitsAttr    getAllBusinessUnitsFunc
}

// newTimeOffPlanProxy initializes the time off plan proxy with all of the data needed to communicate with Genesys Cloud
func newTimeOffPlanProxy(clientConfig *platformclientv2.Configuration) *timeOffPlanProxy {
	api := platformclientv2.NewWorkforceManagementApiWithConfig(clientConfig)
	return &timeOffPlanProxy{
		clientConfig:               clientConfig,
		workforceManagementApi:     api,
		createTimeOffPlanAttr:      createTimeOffPlanFn,
		getAllTimeOffPlansAttr:     getAllTimeOffPlansFn,
		getTimeOffPlanIdByNameAttr: getTimeOffPlanIdByNameFn,
		getTimeOffPlanByIdAttr:     getTimeOffPlanByIdFn,
		updateTimeOffPlanAttr:      updateTimeOffPlanFn,
		deleteTimeOffPlanAttr:      deleteTimeOffPlanFn,
		getAllBusinessUnitsAttr:    getAllBusinessUnitsFn,
	}
}

// getTimeOffPlanProxy acts as a singleton for the internalProxy. It also ensures
// that we can still proxy our tests by directly setting internalProxy package variable
func getTimeOffPlanProxy(clientConfig *platformclientv2.Configuration) *timeOffPlanProxy {
	if internalProxy == nil {
		internalProxy = newTimeOffPlanProxy(clientConfig)
	}

	return internalProxy
}

// createTimeOffPlan creates a Genesys Cloud time off plan
func (p *timeOffPlanProxy) createTimeOffPlan(ctx context.Context, businessUnitId string, plan *platformclientv2.Bucreatetimeoffplanrequest) (*platformclientv2.Butimeoffplanresponse, *platformclientv2.APIResponse, error) {
	return p.createTimeOffPlanAttr(ctx, p, businessUnitId, plan)
}

// getAllTimeOffPlans retrieves all Genesys Cloud time off plans of a business unit
func (p *timeOffPlanProxy) getAllTimeOffPlans(ctx context.Context, businessUnitId string) (*[]platformclientv2.Butimeoffplanresponse, *platformclientv2.APIResponse, error) {
	return p.getAllTimeOffPlansAttr(ctx, p, businessUnitId)
}

// getTimeOffPlanIdByName returns the id of a Genesys Cloud time off plan by name
func (p *timeOffPlanProxy) getTimeOffPlanIdByName(ctx context.Context, businessUnitId string, name string) (string, *platformclientv2.APIResponse, bool, error) {
	return p.getTimeOffPlanIdByNameAttr(ctx, p, businessUnitId, name)
}

// getTimeOffPlanById returns a single Genesys Cloud time off plan by id
func (p *timeOffPlanProxy) getTimeOffPlanById(ctx context.Context, businessUnitId string, planId string) (*platformclientv2.Butimeoffplanresponse, *platformclientv2.APIResponse, error) {
	return p.getTimeOffPlanByIdAttr(ctx, p, businessUnitId, planId)
}

// updateTimeOffPlan updates a Genesys Cloud time off plan
func (p *timeOffPlanProxy) updateTimeOffPlan(ctx context.Context, businessUnitId string, planId string, plan *platformclientv2.Buupdatetimeoffplanrequest) (*platformclientv2.Butimeoffplanresponse, *platformclientv2.APIResponse, error) {
	return p.updateTimeOffPlanAttr(ctx, p, businessUnitId, planId, plan)
}

// deleteTimeOffPlan deletes a Genesys Cloud time off plan by id
func (p *timeOffPlanProxy) deleteTimeOffPlan(ctx context.Context, businessUnitId string, planId string) (*platformclientv2.APIResponse, error) {
	return p.deleteTimeOffPlanAttr(ctx, p, businessUnitId, planId)
}

// getAllBusinessUnits retrieves all Genesys Cloud business units. Used by the exporter.
func (p *timeOffPlanProxy) getAllBusinessUnits(ctx context.Context) (*[]platformclientv2.Businessunitlistitem, *platformclientv2.APIResponse, error) {
	return p.getAllBusinessUnitsAttr(ctx, p)
}

// createTimeOffPlanFn is an implementation function for creating a Genesys Cloud time off plan
func createTimeOffPlanFn(ctx context.Context, p *timeOffPlanProxy, businessUnitId string, plan *platformclientv2.Bucreatetimeoffplanrequest) (*platformclientv2.Butimeoffplanresponse, *platformclientv2.APIResponse, error) {
	return p.workforceManagementApi.PostWorkforcemanagementBusinessunitTimeoffplans(businessUnitId, *plan)
}

// getAllTimeOffPlansFn is the implementation for retrieving all time off plans of a business unit in Genesys Cloud
func getAllTimeOffPlansFn(ctx context.Context, p *timeOffPlanProxy, businessUnitId string) (*[]platformclientv2.Butimeoffplanresponse, *platformclientv2.APIResponse, error) {
	listing, resp, err := p.workforceManagementApi.GetWorkforcemanagementBusinessunitTimeoffplans(businessUnitId, "", false, nil)
	if err != nil {
		return nil, resp, err
	}

	return listing.Entities, resp, nil
}

// getTimeOffPlanIdByNameFn is an implementation of the function to get the id of a time off plan by name
func getTimeOffPlanIdByNameFn(ctx context.Context, p *timeOffPlanProxy, businessUnitId string, name string) (string, *platformclientv2.APIResponse, bool, error) {
	plans, resp, err := getAllTimeOffPlansFn(ctx, p, businessUnitId)
	if err != nil {
		return "", resp, false, err
	}

	for _, plan := range *plans {
		if *plan.Name == name {
			return *plan.Id, resp, false, nil
		}
	}

	return "", resp, true, fmt.Errorf("unable to find business unit %s time off plan with name %s", businessUnitId, name)
}

// getTimeOffPlanByIdFn is an implementation of the function to get a Genesys Cloud time off plan by id
func getTimeOffPlanByIdFn(ctx context.Context, p *timeOffPlanProxy, businessUnitId string, planId string) (*platformclientv2.Butimeoffplanresponse, *platformclientv2.APIResponse, error) {
	return p.workforceManagementApi.GetWorkforcemanagementBusinessunitTimeoffplan(businessUnitId, planId)
}

// updateTimeOffPlanFn is an implementation of the function to update a Genesys Cloud time off plan
func updateTimeOffPlanFn(ctx context.Context, p *timeOffPlanProxy, businessUnitId string, planId string, plan *platformclientv2.Buupdatetimeoffplanrequest) (*platformclientv2.Butimeoffplanresponse, *platformclientv2.APIResponse, error) {
	return p.workforceManagementApi.PatchWorkforcemanagementBusinessunitTimeoffplan(businessUnitId, planId, *plan)
}

// deleteTimeOffPlanFn is an implementation function for deleting a Genesys Cloud time off plan
func deleteTimeOffPlanFn(ctx context.Context, p *timeOffPlanProxy, businessUnitId string, planId string) (*platformclientv2.APIResponse, error) {
	return p.workforceManagementApi.DeleteWorkforcemanagementBusinessunitTimeoffplan(businessUnitId, planId)
}

// getAllBusinessUnitsFn is an implementation of the function to list Genesys Cloud business units
func getAllBusinessUnitsFn(ctx context.Context, p *timeOffPlanProxy) (*[]platformclientv2.Businessunitlistitem, *platformclientv2.APIResponse, error) {
	listing, resp, err := p.workforceManagementApi.GetWorkforcemanagementBusinessunits("", "")
	if err != nil {
		return nil, resp, err
	}

	return listing.Entities, resp, nil
}
