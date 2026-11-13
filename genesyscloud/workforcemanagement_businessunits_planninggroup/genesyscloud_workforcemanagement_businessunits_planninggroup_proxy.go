package workforcemanagement_businessunits_planninggroup

import (
	"context"
	"fmt"

	"github.com/mypurecloud/platform-client-sdk-go/v200/platformclientv2"
)

/*
The genesyscloud_workforcemanagement_businessunits_planninggroup_proxy.go file contains the proxy structures and methods that interact
with the Genesys Cloud SDK. We use composition here for each function on the proxy so individual functions can be stubbed
out during testing.
*/

// internalProxy holds a proxy instance that can be used throughout the package
var internalProxy *planningGroupProxy

// Type definitions for each func on our proxy so we can easily mock them out later
type createPlanningGroupFunc func(ctx context.Context, p *planningGroupProxy, businessUnitId string, planningGroup *platformclientv2.Createplanninggrouprequest) (*platformclientv2.Planninggroup, *platformclientv2.APIResponse, error)
type getAllPlanningGroupsFunc func(ctx context.Context, p *planningGroupProxy, businessUnitId string) (*[]platformclientv2.Planninggroup, *platformclientv2.APIResponse, error)
type getPlanningGroupIdByNameFunc func(ctx context.Context, p *planningGroupProxy, businessUnitId string, name string) (string, *platformclientv2.APIResponse, bool, error)
type getPlanningGroupByIdFunc func(ctx context.Context, p *planningGroupProxy, businessUnitId string, planningGroupId string) (*platformclientv2.Planninggroup, *platformclientv2.APIResponse, error)
type updatePlanningGroupFunc func(ctx context.Context, p *planningGroupProxy, businessUnitId string, planningGroupId string, planningGroup *platformclientv2.Updateplanninggrouprequest) (*platformclientv2.Planninggroup, *platformclientv2.APIResponse, error)
type deletePlanningGroupFunc func(ctx context.Context, p *planningGroupProxy, businessUnitId string, planningGroupId string) (*platformclientv2.APIResponse, error)
type getAllBusinessUnitsFunc func(ctx context.Context, p *planningGroupProxy) (*[]platformclientv2.Businessunitlistitem, *platformclientv2.APIResponse, error)

// planningGroupProxy contains all the methods that call genesys cloud APIs.
type planningGroupProxy struct {
	clientConfig                 *platformclientv2.Configuration
	workforceManagementApi       *platformclientv2.WorkforceManagementApi
	createPlanningGroupAttr      createPlanningGroupFunc
	getAllPlanningGroupsAttr     getAllPlanningGroupsFunc
	getPlanningGroupIdByNameAttr getPlanningGroupIdByNameFunc
	getPlanningGroupByIdAttr     getPlanningGroupByIdFunc
	updatePlanningGroupAttr      updatePlanningGroupFunc
	deletePlanningGroupAttr      deletePlanningGroupFunc
	getAllBusinessUnitsAttr      getAllBusinessUnitsFunc
}

// newPlanningGroupProxy initializes the planning group proxy with all of the data needed to communicate with Genesys Cloud
func newPlanningGroupProxy(clientConfig *platformclientv2.Configuration) *planningGroupProxy {
	api := platformclientv2.NewWorkforceManagementApiWithConfig(clientConfig)
	return &planningGroupProxy{
		clientConfig:                 clientConfig,
		workforceManagementApi:       api,
		createPlanningGroupAttr:      createPlanningGroupFn,
		getAllPlanningGroupsAttr:     getAllPlanningGroupsFn,
		getPlanningGroupIdByNameAttr: getPlanningGroupIdByNameFn,
		getPlanningGroupByIdAttr:     getPlanningGroupByIdFn,
		updatePlanningGroupAttr:      updatePlanningGroupFn,
		deletePlanningGroupAttr:      deletePlanningGroupFn,
		getAllBusinessUnitsAttr:      getAllBusinessUnitsFn,
	}
}

// getPlanningGroupProxy acts as a singleton for the internalProxy. It also ensures
// that we can still proxy our tests by directly setting internalProxy package variable
func getPlanningGroupProxy(clientConfig *platformclientv2.Configuration) *planningGroupProxy {
	if internalProxy == nil {
		internalProxy = newPlanningGroupProxy(clientConfig)
	}

	return internalProxy
}

// createPlanningGroup creates a Genesys Cloud planning group
func (p *planningGroupProxy) createPlanningGroup(ctx context.Context, businessUnitId string, planningGroup *platformclientv2.Createplanninggrouprequest) (*platformclientv2.Planninggroup, *platformclientv2.APIResponse, error) {
	return p.createPlanningGroupAttr(ctx, p, businessUnitId, planningGroup)
}

// getAllPlanningGroups retrieves all Genesys Cloud planning groups of a business unit
func (p *planningGroupProxy) getAllPlanningGroups(ctx context.Context, businessUnitId string) (*[]platformclientv2.Planninggroup, *platformclientv2.APIResponse, error) {
	return p.getAllPlanningGroupsAttr(ctx, p, businessUnitId)
}

// getPlanningGroupIdByName returns the id of a Genesys Cloud planning group by name
func (p *planningGroupProxy) getPlanningGroupIdByName(ctx context.Context, businessUnitId string, name string) (string, *platformclientv2.APIResponse, bool, error) {
	return p.getPlanningGroupIdByNameAttr(ctx, p, businessUnitId, name)
}

// getPlanningGroupById returns a single Genesys Cloud planning group by id
func (p *planningGroupProxy) getPlanningGroupById(ctx context.Context, businessUnitId string, planningGroupId string) (*platformclientv2.Planninggroup, *platformclientv2.APIResponse, error) {
	return p.getPlanningGroupByIdAttr(ctx, p, businessUnitId, planningGroupId)
}

// updatePlanningGroup updates a Genesys Cloud planning group
func (p *planningGroupProxy) updatePlanningGroup(ctx context.Context, businessUnitId string, planningGroupId string, planningGroup *platformclientv2.Updateplanninggrouprequest) (*platformclientv2.Planninggroup, *platformclientv2.APIResponse, error) {
	return p.updatePlanningGroupAttr(ctx, p, businessUnitId, planningGroupId, planningGroup)
}

// deletePlanningGroup deletes a Genesys Cloud planning group by id
func (p *planningGroupProxy) deletePlanningGroup(ctx context.Context, businessUnitId string, planningGroupId string) (*platformclientv2.APIResponse, error) {
	return p.deletePlanningGroupAttr(ctx, p, businessUnitId, planningGroupId)
}

// getAllBusinessUnits retrieves all Genesys Cloud business units. Used by the exporter to walk every business unit.
func (p *planningGroupProxy) getAllBusinessUnits(ctx context.Context) (*[]platformclientv2.Businessunitlistitem, *platformclientv2.APIResponse, error) {
	return p.getAllBusinessUnitsAttr(ctx, p)
}

// createPlanningGroupFn is an implementation function for creating a Genesys Cloud planning group
func createPlanningGroupFn(ctx context.Context, p *planningGroupProxy, businessUnitId string, planningGroup *platformclientv2.Createplanninggrouprequest) (*platformclientv2.Planninggroup, *platformclientv2.APIResponse, error) {
	return p.workforceManagementApi.PostWorkforcemanagementBusinessunitPlanninggroups(businessUnitId, *planningGroup)
}

// getAllPlanningGroupsFn is the implementation for retrieving all planning groups of a business unit in Genesys Cloud
func getAllPlanningGroupsFn(ctx context.Context, p *planningGroupProxy, businessUnitId string) (*[]platformclientv2.Planninggroup, *platformclientv2.APIResponse, error) {
	listing, resp, err := p.workforceManagementApi.GetWorkforcemanagementBusinessunitPlanninggroups(businessUnitId)
	if err != nil {
		return nil, resp, err
	}

	return listing.Entities, resp, nil
}

// getPlanningGroupIdByNameFn is an implementation of the function to get the id of a planning group by name
func getPlanningGroupIdByNameFn(ctx context.Context, p *planningGroupProxy, businessUnitId string, name string) (string, *platformclientv2.APIResponse, bool, error) {
	planningGroups, resp, err := getAllPlanningGroupsFn(ctx, p, businessUnitId)
	if err != nil {
		return "", resp, false, err
	}

	for _, planningGroup := range *planningGroups {
		if *planningGroup.Name == name {
			return *planningGroup.Id, resp, false, nil
		}
	}

	return "", resp, true, fmt.Errorf("unable to find business unit %s planning group with name %s", businessUnitId, name)
}

// getPlanningGroupByIdFn is an implementation of the function to get a Genesys Cloud planning group by id
func getPlanningGroupByIdFn(ctx context.Context, p *planningGroupProxy, businessUnitId string, planningGroupId string) (*platformclientv2.Planninggroup, *platformclientv2.APIResponse, error) {
	return p.workforceManagementApi.GetWorkforcemanagementBusinessunitPlanninggroup(businessUnitId, planningGroupId)
}

// updatePlanningGroupFn is an implementation of the function to update a Genesys Cloud planning group
func updatePlanningGroupFn(ctx context.Context, p *planningGroupProxy, businessUnitId string, planningGroupId string, planningGroup *platformclientv2.Updateplanninggrouprequest) (*platformclientv2.Planninggroup, *platformclientv2.APIResponse, error) {
	return p.workforceManagementApi.PatchWorkforcemanagementBusinessunitPlanninggroup(businessUnitId, planningGroupId, *planningGroup)
}

// deletePlanningGroupFn is an implementation function for deleting a Genesys Cloud planning group
func deletePlanningGroupFn(ctx context.Context, p *planningGroupProxy, businessUnitId string, planningGroupId string) (*platformclientv2.APIResponse, error) {
	return p.workforceManagementApi.DeleteWorkforcemanagementBusinessunitPlanninggroup(businessUnitId, planningGroupId)
}

// getAllBusinessUnitsFn is an implementation of the function to list Genesys Cloud business units
func getAllBusinessUnitsFn(ctx context.Context, p *planningGroupProxy) (*[]platformclientv2.Businessunitlistitem, *platformclientv2.APIResponse, error) {
	listing, resp, err := p.workforceManagementApi.GetWorkforcemanagementBusinessunits("", "")
	if err != nil {
		return nil, resp, err
	}

	return listing.Entities, resp, nil
}
