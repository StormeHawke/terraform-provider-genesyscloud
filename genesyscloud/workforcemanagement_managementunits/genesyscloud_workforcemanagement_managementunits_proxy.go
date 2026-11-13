package workforcemanagement_managementunits

import (
	"context"
	"fmt"

	"github.com/mypurecloud/platform-client-sdk-go/v200/platformclientv2"
)

/*
The genesyscloud_workforcemanagement_managementunits_proxy.go file contains the proxy structures and methods that interact
with the Genesys Cloud SDK. We use composition here for each function on the proxy so individual functions can be stubbed
out during testing.
*/

// internalProxy holds a proxy instance that can be used throughout the package
var internalProxy *managementUnitProxy

// Type definitions for each func on our proxy so we can easily mock them out later
type createManagementUnitFunc func(ctx context.Context, p *managementUnitProxy, managementUnit *platformclientv2.Createmanagementunitapirequest) (*platformclientv2.Managementunit, *platformclientv2.APIResponse, error)
type getAllManagementUnitsFunc func(ctx context.Context, p *managementUnitProxy) (*[]platformclientv2.Managementunit, *platformclientv2.APIResponse, error)
type getManagementUnitIdByNameFunc func(ctx context.Context, p *managementUnitProxy, name string) (string, *platformclientv2.APIResponse, bool, error)
type getManagementUnitByIdFunc func(ctx context.Context, p *managementUnitProxy, managementUnitId string) (*platformclientv2.Managementunit, *platformclientv2.APIResponse, error)
type updateManagementUnitFunc func(ctx context.Context, p *managementUnitProxy, managementUnitId string, managementUnit *platformclientv2.Updatemanagementunitrequest) (*platformclientv2.Managementunit, *platformclientv2.APIResponse, error)
type moveManagementUnitFunc func(ctx context.Context, p *managementUnitProxy, managementUnitId string, businessUnitId string) (*platformclientv2.Movemanagementunitresponse, *platformclientv2.APIResponse, error)
type deleteManagementUnitFunc func(ctx context.Context, p *managementUnitProxy, managementUnitId string) (*platformclientv2.APIResponse, error)

// managementUnitProxy contains all the methods that call genesys cloud APIs.
type managementUnitProxy struct {
	clientConfig                  *platformclientv2.Configuration
	workforceManagementApi        *platformclientv2.WorkforceManagementApi
	createManagementUnitAttr      createManagementUnitFunc
	getAllManagementUnitsAttr     getAllManagementUnitsFunc
	getManagementUnitIdByNameAttr getManagementUnitIdByNameFunc
	getManagementUnitByIdAttr     getManagementUnitByIdFunc
	updateManagementUnitAttr      updateManagementUnitFunc
	moveManagementUnitAttr        moveManagementUnitFunc
	deleteManagementUnitAttr      deleteManagementUnitFunc
}

// newManagementUnitProxy initializes the management unit proxy with all of the data needed to communicate with Genesys Cloud
func newManagementUnitProxy(clientConfig *platformclientv2.Configuration) *managementUnitProxy {
	api := platformclientv2.NewWorkforceManagementApiWithConfig(clientConfig)
	return &managementUnitProxy{
		clientConfig:                  clientConfig,
		workforceManagementApi:        api,
		createManagementUnitAttr:      createManagementUnitFn,
		getAllManagementUnitsAttr:     getAllManagementUnitsFn,
		getManagementUnitIdByNameAttr: getManagementUnitIdByNameFn,
		getManagementUnitByIdAttr:     getManagementUnitByIdFn,
		updateManagementUnitAttr:      updateManagementUnitFn,
		moveManagementUnitAttr:        moveManagementUnitFn,
		deleteManagementUnitAttr:      deleteManagementUnitFn,
	}
}

// getManagementUnitProxy acts as a singleton for the internalProxy. It also ensures
// that we can still proxy our tests by directly setting internalProxy package variable
func getManagementUnitProxy(clientConfig *platformclientv2.Configuration) *managementUnitProxy {
	if internalProxy == nil {
		internalProxy = newManagementUnitProxy(clientConfig)
	}

	return internalProxy
}

// createManagementUnit creates a Genesys Cloud management unit
func (p *managementUnitProxy) createManagementUnit(ctx context.Context, managementUnit *platformclientv2.Createmanagementunitapirequest) (*platformclientv2.Managementunit, *platformclientv2.APIResponse, error) {
	return p.createManagementUnitAttr(ctx, p, managementUnit)
}

// getAllManagementUnits retrieves all Genesys Cloud management units
func (p *managementUnitProxy) getAllManagementUnits(ctx context.Context) (*[]platformclientv2.Managementunit, *platformclientv2.APIResponse, error) {
	return p.getAllManagementUnitsAttr(ctx, p)
}

// getManagementUnitIdByName returns the id of a Genesys Cloud management unit by name
func (p *managementUnitProxy) getManagementUnitIdByName(ctx context.Context, name string) (string, *platformclientv2.APIResponse, bool, error) {
	return p.getManagementUnitIdByNameAttr(ctx, p, name)
}

// getManagementUnitById returns a single Genesys Cloud management unit, including its settings, by id
func (p *managementUnitProxy) getManagementUnitById(ctx context.Context, managementUnitId string) (*platformclientv2.Managementunit, *platformclientv2.APIResponse, error) {
	return p.getManagementUnitByIdAttr(ctx, p, managementUnitId)
}

// updateManagementUnit updates a Genesys Cloud management unit
func (p *managementUnitProxy) updateManagementUnit(ctx context.Context, managementUnitId string, managementUnit *platformclientv2.Updatemanagementunitrequest) (*platformclientv2.Managementunit, *platformclientv2.APIResponse, error) {
	return p.updateManagementUnitAttr(ctx, p, managementUnitId, managementUnit)
}

// moveManagementUnit moves a Genesys Cloud management unit to another business unit
func (p *managementUnitProxy) moveManagementUnit(ctx context.Context, managementUnitId string, businessUnitId string) (*platformclientv2.Movemanagementunitresponse, *platformclientv2.APIResponse, error) {
	return p.moveManagementUnitAttr(ctx, p, managementUnitId, businessUnitId)
}

// deleteManagementUnit deletes a Genesys Cloud management unit by id
func (p *managementUnitProxy) deleteManagementUnit(ctx context.Context, managementUnitId string) (*platformclientv2.APIResponse, error) {
	return p.deleteManagementUnitAttr(ctx, p, managementUnitId)
}

// createManagementUnitFn is an implementation function for creating a Genesys Cloud management unit
func createManagementUnitFn(ctx context.Context, p *managementUnitProxy, managementUnit *platformclientv2.Createmanagementunitapirequest) (*platformclientv2.Managementunit, *platformclientv2.APIResponse, error) {
	return p.workforceManagementApi.PostWorkforcemanagementManagementunits(*managementUnit)
}

// getAllManagementUnitsFn is the implementation for retrieving all management units in Genesys Cloud.
// The api documents paging as unsupported and returns every management unit, so the paging arguments are placeholders.
func getAllManagementUnitsFn(ctx context.Context, p *managementUnitProxy) (*[]platformclientv2.Managementunit, *platformclientv2.APIResponse, error) {
	listing, resp, err := p.workforceManagementApi.GetWorkforcemanagementManagementunits(100, 1, "", "", "")
	if err != nil {
		return nil, resp, err
	}

	return listing.Entities, resp, nil
}

// getManagementUnitIdByNameFn is an implementation of the function to get the id of a management unit by name
func getManagementUnitIdByNameFn(ctx context.Context, p *managementUnitProxy, name string) (string, *platformclientv2.APIResponse, bool, error) {
	managementUnits, resp, err := getAllManagementUnitsFn(ctx, p)
	if err != nil {
		return "", resp, false, err
	}

	for _, managementUnit := range *managementUnits {
		if *managementUnit.Name == name {
			return *managementUnit.Id, resp, false, nil
		}
	}

	return "", resp, true, fmt.Errorf("unable to find management unit with name %s", name)
}

// settingsExpands requests every settings section, because settings are only returned when expanded and this resource manages all of them.
// settings.shortTermForecasting is deliberately not requested: it is deprecated on management units and now lives on the business unit.
var settingsExpands = []string{"settings", "settings.adherence", "settings.timeOff", "settings.scheduling", "settings.shiftTrading"}

// getManagementUnitByIdFn is an implementation of the function to get a Genesys Cloud management unit by id, with all of its settings expanded
func getManagementUnitByIdFn(ctx context.Context, p *managementUnitProxy, managementUnitId string) (*platformclientv2.Managementunit, *platformclientv2.APIResponse, error) {
	return p.workforceManagementApi.GetWorkforcemanagementManagementunit(managementUnitId, settingsExpands)
}

// updateManagementUnitFn is an implementation of the function to update a Genesys Cloud management unit
func updateManagementUnitFn(ctx context.Context, p *managementUnitProxy, managementUnitId string, managementUnit *platformclientv2.Updatemanagementunitrequest) (*platformclientv2.Managementunit, *platformclientv2.APIResponse, error) {
	return p.workforceManagementApi.PatchWorkforcemanagementManagementunit(managementUnitId, *managementUnit)
}

// moveManagementUnitFn is an implementation of the function to move a Genesys Cloud management unit to another business unit
func moveManagementUnitFn(ctx context.Context, p *managementUnitProxy, managementUnitId string, businessUnitId string) (*platformclientv2.Movemanagementunitresponse, *platformclientv2.APIResponse, error) {
	return p.workforceManagementApi.PostWorkforcemanagementManagementunitMove(managementUnitId, platformclientv2.Movemanagementunitrequest{BusinessUnitId: &businessUnitId})
}

// deleteManagementUnitFn is an implementation function for deleting a Genesys Cloud management unit
func deleteManagementUnitFn(ctx context.Context, p *managementUnitProxy, managementUnitId string) (*platformclientv2.APIResponse, error) {
	return p.workforceManagementApi.DeleteWorkforcemanagementManagementunit(managementUnitId)
}
