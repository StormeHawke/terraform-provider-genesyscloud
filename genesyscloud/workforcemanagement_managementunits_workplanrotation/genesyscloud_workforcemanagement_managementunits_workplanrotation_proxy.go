package workforcemanagement_managementunits_workplanrotation

import (
	"context"
	"fmt"

	"github.com/mypurecloud/platform-client-sdk-go/v200/platformclientv2"
)

/*
The genesyscloud_workforcemanagement_managementunits_workplanrotation_proxy.go file contains the proxy structures and methods that interact
with the Genesys Cloud SDK. We use composition here for each function on the proxy so individual functions can be stubbed
out during testing.
*/

// internalProxy holds a proxy instance that can be used throughout the package
var internalProxy *workPlanRotationProxy

// Type definitions for each func on our proxy so we can easily mock them out later
type createWorkPlanRotationFunc func(ctx context.Context, p *workPlanRotationProxy, managementUnitId string, rotation *platformclientv2.Addworkplanrotationrequest) (*platformclientv2.Workplanrotationresponse, *platformclientv2.APIResponse, error)
type getAllWorkPlanRotationsFunc func(ctx context.Context, p *workPlanRotationProxy, managementUnitId string) (*[]platformclientv2.Workplanrotationresponse, *platformclientv2.APIResponse, error)
type getWorkPlanRotationIdByNameFunc func(ctx context.Context, p *workPlanRotationProxy, managementUnitId string, name string) (string, *platformclientv2.APIResponse, bool, error)
type getWorkPlanRotationByIdFunc func(ctx context.Context, p *workPlanRotationProxy, managementUnitId string, rotationId string) (*platformclientv2.Workplanrotationresponse, *platformclientv2.APIResponse, error)
type updateWorkPlanRotationFunc func(ctx context.Context, p *workPlanRotationProxy, managementUnitId string, rotationId string, rotation *platformclientv2.Updateworkplanrotationrequest) (*platformclientv2.Workplanrotationresponse, *platformclientv2.APIResponse, error)
type deleteWorkPlanRotationFunc func(ctx context.Context, p *workPlanRotationProxy, managementUnitId string, rotationId string) (*platformclientv2.APIResponse, error)
type getAllManagementUnitsFunc func(ctx context.Context, p *workPlanRotationProxy) (*[]platformclientv2.Managementunit, *platformclientv2.APIResponse, error)

// workPlanRotationProxy contains all the methods that call genesys cloud APIs.
type workPlanRotationProxy struct {
	clientConfig                    *platformclientv2.Configuration
	workforceManagementApi          *platformclientv2.WorkforceManagementApi
	createWorkPlanRotationAttr      createWorkPlanRotationFunc
	getAllWorkPlanRotationsAttr     getAllWorkPlanRotationsFunc
	getWorkPlanRotationIdByNameAttr getWorkPlanRotationIdByNameFunc
	getWorkPlanRotationByIdAttr     getWorkPlanRotationByIdFunc
	updateWorkPlanRotationAttr      updateWorkPlanRotationFunc
	deleteWorkPlanRotationAttr      deleteWorkPlanRotationFunc
	getAllManagementUnitsAttr       getAllManagementUnitsFunc
}

// newWorkPlanRotationProxy initializes the work plan rotation proxy with all of the data needed to communicate with Genesys Cloud
func newWorkPlanRotationProxy(clientConfig *platformclientv2.Configuration) *workPlanRotationProxy {
	api := platformclientv2.NewWorkforceManagementApiWithConfig(clientConfig)
	return &workPlanRotationProxy{
		clientConfig:                    clientConfig,
		workforceManagementApi:          api,
		createWorkPlanRotationAttr:      createWorkPlanRotationFn,
		getAllWorkPlanRotationsAttr:     getAllWorkPlanRotationsFn,
		getWorkPlanRotationIdByNameAttr: getWorkPlanRotationIdByNameFn,
		getWorkPlanRotationByIdAttr:     getWorkPlanRotationByIdFn,
		updateWorkPlanRotationAttr:      updateWorkPlanRotationFn,
		deleteWorkPlanRotationAttr:      deleteWorkPlanRotationFn,
		getAllManagementUnitsAttr:       getAllManagementUnitsFn,
	}
}

// getWorkPlanRotationProxy acts as a singleton for the internalProxy. It also ensures
// that we can still proxy our tests by directly setting internalProxy package variable
func getWorkPlanRotationProxy(clientConfig *platformclientv2.Configuration) *workPlanRotationProxy {
	if internalProxy == nil {
		internalProxy = newWorkPlanRotationProxy(clientConfig)
	}

	return internalProxy
}

// createWorkPlanRotation creates a Genesys Cloud work plan rotation
func (p *workPlanRotationProxy) createWorkPlanRotation(ctx context.Context, managementUnitId string, rotation *platformclientv2.Addworkplanrotationrequest) (*platformclientv2.Workplanrotationresponse, *platformclientv2.APIResponse, error) {
	return p.createWorkPlanRotationAttr(ctx, p, managementUnitId, rotation)
}

// getAllWorkPlanRotations retrieves all Genesys Cloud work plan rotations of a management unit
func (p *workPlanRotationProxy) getAllWorkPlanRotations(ctx context.Context, managementUnitId string) (*[]platformclientv2.Workplanrotationresponse, *platformclientv2.APIResponse, error) {
	return p.getAllWorkPlanRotationsAttr(ctx, p, managementUnitId)
}

// getWorkPlanRotationIdByName returns the id of a Genesys Cloud work plan rotation by name
func (p *workPlanRotationProxy) getWorkPlanRotationIdByName(ctx context.Context, managementUnitId string, name string) (string, *platformclientv2.APIResponse, bool, error) {
	return p.getWorkPlanRotationIdByNameAttr(ctx, p, managementUnitId, name)
}

// getWorkPlanRotationById returns a single Genesys Cloud work plan rotation, including its agents, by id
func (p *workPlanRotationProxy) getWorkPlanRotationById(ctx context.Context, managementUnitId string, rotationId string) (*platformclientv2.Workplanrotationresponse, *platformclientv2.APIResponse, error) {
	return p.getWorkPlanRotationByIdAttr(ctx, p, managementUnitId, rotationId)
}

// updateWorkPlanRotation updates a Genesys Cloud work plan rotation
func (p *workPlanRotationProxy) updateWorkPlanRotation(ctx context.Context, managementUnitId string, rotationId string, rotation *platformclientv2.Updateworkplanrotationrequest) (*platformclientv2.Workplanrotationresponse, *platformclientv2.APIResponse, error) {
	return p.updateWorkPlanRotationAttr(ctx, p, managementUnitId, rotationId, rotation)
}

// deleteWorkPlanRotation deletes a Genesys Cloud work plan rotation by id
func (p *workPlanRotationProxy) deleteWorkPlanRotation(ctx context.Context, managementUnitId string, rotationId string) (*platformclientv2.APIResponse, error) {
	return p.deleteWorkPlanRotationAttr(ctx, p, managementUnitId, rotationId)
}

// getAllManagementUnits retrieves all Genesys Cloud management units. Used by the exporter to walk every management unit.
func (p *workPlanRotationProxy) getAllManagementUnits(ctx context.Context) (*[]platformclientv2.Managementunit, *platformclientv2.APIResponse, error) {
	return p.getAllManagementUnitsAttr(ctx, p)
}

// createWorkPlanRotationFn is an implementation function for creating a Genesys Cloud work plan rotation. Genesys Cloud always creates it enabled.
func createWorkPlanRotationFn(ctx context.Context, p *workPlanRotationProxy, managementUnitId string, rotation *platformclientv2.Addworkplanrotationrequest) (*platformclientv2.Workplanrotationresponse, *platformclientv2.APIResponse, error) {
	return p.workforceManagementApi.PostWorkforcemanagementManagementunitWorkplanrotations(managementUnitId, *rotation)
}

// getAllWorkPlanRotationsFn is the implementation for retrieving all work plan rotations of a management unit in Genesys Cloud
func getAllWorkPlanRotationsFn(ctx context.Context, p *workPlanRotationProxy, managementUnitId string) (*[]platformclientv2.Workplanrotationresponse, *platformclientv2.APIResponse, error) {
	listing, resp, err := p.workforceManagementApi.GetWorkforcemanagementManagementunitWorkplanrotations(managementUnitId, nil)
	if err != nil {
		return nil, resp, err
	}

	if listing == nil || listing.Entities == nil {
		return &[]platformclientv2.Workplanrotationresponse{}, resp, nil
	}

	return listing.Entities, resp, nil
}

// getWorkPlanRotationIdByNameFn is an implementation of the function to get the id of a work plan rotation by name
func getWorkPlanRotationIdByNameFn(ctx context.Context, p *workPlanRotationProxy, managementUnitId string, name string) (string, *platformclientv2.APIResponse, bool, error) {
	rotations, resp, err := getAllWorkPlanRotationsFn(ctx, p, managementUnitId)
	if err != nil {
		return "", resp, false, err
	}

	for _, rotation := range *rotations {
		if rotation.Name != nil && *rotation.Name == name && rotation.Id != nil {
			return *rotation.Id, resp, false, nil
		}
	}

	return "", resp, true, fmt.Errorf("unable to find management unit %s work plan rotation with name %s", managementUnitId, name)
}

// getWorkPlanRotationByIdFn is an implementation of the function to get a Genesys Cloud work plan rotation by id
func getWorkPlanRotationByIdFn(ctx context.Context, p *workPlanRotationProxy, managementUnitId string, rotationId string) (*platformclientv2.Workplanrotationresponse, *platformclientv2.APIResponse, error) {
	return p.workforceManagementApi.GetWorkforcemanagementManagementunitWorkplanrotation(managementUnitId, rotationId)
}

// updateWorkPlanRotationFn is an implementation of the function to update a Genesys Cloud work plan rotation
func updateWorkPlanRotationFn(ctx context.Context, p *workPlanRotationProxy, managementUnitId string, rotationId string, rotation *platformclientv2.Updateworkplanrotationrequest) (*platformclientv2.Workplanrotationresponse, *platformclientv2.APIResponse, error) {
	return p.workforceManagementApi.PatchWorkforcemanagementManagementunitWorkplanrotation(managementUnitId, rotationId, *rotation)
}

// deleteWorkPlanRotationFn is an implementation function for deleting a Genesys Cloud work plan rotation
func deleteWorkPlanRotationFn(ctx context.Context, p *workPlanRotationProxy, managementUnitId string, rotationId string) (*platformclientv2.APIResponse, error) {
	return p.workforceManagementApi.DeleteWorkforcemanagementManagementunitWorkplanrotation(managementUnitId, rotationId)
}

// getAllManagementUnitsFn is an implementation of the function to list Genesys Cloud management units.
// The api documents paging as unsupported and returns every management unit, so the paging arguments are placeholders.
func getAllManagementUnitsFn(ctx context.Context, p *workPlanRotationProxy) (*[]platformclientv2.Managementunit, *platformclientv2.APIResponse, error) {
	listing, resp, err := p.workforceManagementApi.GetWorkforcemanagementManagementunits(100, 1, "", "", "")
	if err != nil {
		return nil, resp, err
	}

	if listing == nil || listing.Entities == nil {
		return &[]platformclientv2.Managementunit{}, resp, nil
	}

	return listing.Entities, resp, nil
}
