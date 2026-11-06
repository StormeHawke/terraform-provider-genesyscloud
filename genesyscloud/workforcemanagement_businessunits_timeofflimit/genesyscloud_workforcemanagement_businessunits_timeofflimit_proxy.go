package workforcemanagement_businessunits_timeofflimit

import (
	"context"

	"github.com/mypurecloud/platform-client-sdk-go/v200/platformclientv2"
)

/*
The genesyscloud_workforcemanagement_businessunits_timeofflimit_proxy.go file contains the proxy structures and methods that interact
with the Genesys Cloud SDK. We use composition here for each function on the proxy so individual functions can be stubbed
out during testing.
*/

// internalProxy holds a proxy instance that can be used throughout the package
var internalProxy *timeOffLimitProxy

// Type definitions for each func on our proxy so we can easily mock them out later
type createTimeOffLimitFunc func(ctx context.Context, p *timeOffLimitProxy, businessUnitId string, limit *platformclientv2.Bucreatetimeofflimitrequest) (*platformclientv2.Butimeofflimitresponse, *platformclientv2.APIResponse, error)
type getAllTimeOffLimitsFunc func(ctx context.Context, p *timeOffLimitProxy, businessUnitId string) (*[]platformclientv2.Butimeofflimitresponse, *platformclientv2.APIResponse, error)
type getTimeOffLimitByIdFunc func(ctx context.Context, p *timeOffLimitProxy, businessUnitId string, limitId string) (*platformclientv2.Butimeofflimitresponse, *platformclientv2.APIResponse, error)
type deleteTimeOffLimitFunc func(ctx context.Context, p *timeOffLimitProxy, businessUnitId string, limitId string) (*platformclientv2.APIResponse, error)
type setLimitValuesFunc func(ctx context.Context, p *timeOffLimitProxy, businessUnitId string, limitId string, values *platformclientv2.Busettimeofflimitvaluesrequest) (*platformclientv2.Butimeofflimitresponse, *platformclientv2.APIResponse, error)
type queryLimitValuesFunc func(ctx context.Context, p *timeOffLimitProxy, businessUnitId string, limitId string, dateRanges []platformclientv2.Localdaterange) (*platformclientv2.Butimeofflimitvaluesforgranularityresponse, *platformclientv2.APIResponse, error)
type getAllBusinessUnitsFunc func(ctx context.Context, p *timeOffLimitProxy) (*[]platformclientv2.Businessunitlistitem, *platformclientv2.APIResponse, error)
type getAllManagementUnitsFunc func(ctx context.Context, p *timeOffLimitProxy) (*[]platformclientv2.Managementunit, *platformclientv2.APIResponse, error)

// timeOffLimitProxy contains all the methods that call genesys cloud APIs.
type timeOffLimitProxy struct {
	clientConfig              *platformclientv2.Configuration
	workforceManagementApi    *platformclientv2.WorkforceManagementApi
	createTimeOffLimitAttr    createTimeOffLimitFunc
	getAllTimeOffLimitsAttr   getAllTimeOffLimitsFunc
	getTimeOffLimitByIdAttr   getTimeOffLimitByIdFunc
	deleteTimeOffLimitAttr    deleteTimeOffLimitFunc
	setLimitValuesAttr        setLimitValuesFunc
	queryLimitValuesAttr      queryLimitValuesFunc
	getAllBusinessUnitsAttr   getAllBusinessUnitsFunc
	getAllManagementUnitsAttr getAllManagementUnitsFunc
}

// newTimeOffLimitProxy initializes the time off limit proxy with all of the data needed to communicate with Genesys Cloud
func newTimeOffLimitProxy(clientConfig *platformclientv2.Configuration) *timeOffLimitProxy {
	api := platformclientv2.NewWorkforceManagementApiWithConfig(clientConfig)
	return &timeOffLimitProxy{
		clientConfig:              clientConfig,
		workforceManagementApi:    api,
		createTimeOffLimitAttr:    createTimeOffLimitFn,
		getAllTimeOffLimitsAttr:   getAllTimeOffLimitsFn,
		getTimeOffLimitByIdAttr:   getTimeOffLimitByIdFn,
		deleteTimeOffLimitAttr:    deleteTimeOffLimitFn,
		setLimitValuesAttr:        setLimitValuesFn,
		queryLimitValuesAttr:      queryLimitValuesFn,
		getAllBusinessUnitsAttr:   getAllBusinessUnitsFn,
		getAllManagementUnitsAttr: getAllManagementUnitsFn,
	}
}

// getTimeOffLimitProxy acts as a singleton for the internalProxy. It also ensures
// that we can still proxy our tests by directly setting internalProxy package variable
func getTimeOffLimitProxy(clientConfig *platformclientv2.Configuration) *timeOffLimitProxy {
	if internalProxy == nil {
		internalProxy = newTimeOffLimitProxy(clientConfig)
	}

	return internalProxy
}

// createTimeOffLimit creates a Genesys Cloud time off limit
func (p *timeOffLimitProxy) createTimeOffLimit(ctx context.Context, businessUnitId string, limit *platformclientv2.Bucreatetimeofflimitrequest) (*platformclientv2.Butimeofflimitresponse, *platformclientv2.APIResponse, error) {
	return p.createTimeOffLimitAttr(ctx, p, businessUnitId, limit)
}

// getAllTimeOffLimits retrieves all Genesys Cloud time off limits of a business unit
func (p *timeOffLimitProxy) getAllTimeOffLimits(ctx context.Context, businessUnitId string) (*[]platformclientv2.Butimeofflimitresponse, *platformclientv2.APIResponse, error) {
	return p.getAllTimeOffLimitsAttr(ctx, p, businessUnitId)
}

// getTimeOffLimitById returns a single Genesys Cloud time off limit by id. The daily values are not part of it.
func (p *timeOffLimitProxy) getTimeOffLimitById(ctx context.Context, businessUnitId string, limitId string) (*platformclientv2.Butimeofflimitresponse, *platformclientv2.APIResponse, error) {
	return p.getTimeOffLimitByIdAttr(ctx, p, businessUnitId, limitId)
}

// deleteTimeOffLimit deletes a Genesys Cloud time off limit by id
func (p *timeOffLimitProxy) deleteTimeOffLimit(ctx context.Context, businessUnitId string, limitId string) (*platformclientv2.APIResponse, error) {
	return p.deleteTimeOffLimitAttr(ctx, p, businessUnitId, limitId)
}

// setLimitValues sets the daily or fifteen minute limit values of a time off limit for date ranges
func (p *timeOffLimitProxy) setLimitValues(ctx context.Context, businessUnitId string, limitId string, values *platformclientv2.Busettimeofflimitvaluesrequest) (*platformclientv2.Butimeofflimitresponse, *platformclientv2.APIResponse, error) {
	return p.setLimitValuesAttr(ctx, p, businessUnitId, limitId, values)
}

// queryLimitValues reads the limit values of a time off limit for date ranges
func (p *timeOffLimitProxy) queryLimitValues(ctx context.Context, businessUnitId string, limitId string, dateRanges []platformclientv2.Localdaterange) (*platformclientv2.Butimeofflimitvaluesforgranularityresponse, *platformclientv2.APIResponse, error) {
	return p.queryLimitValuesAttr(ctx, p, businessUnitId, limitId, dateRanges)
}

// getAllBusinessUnits retrieves all Genesys Cloud business units. Used by the exporter.
func (p *timeOffLimitProxy) getAllBusinessUnits(ctx context.Context) (*[]platformclientv2.Businessunitlistitem, *platformclientv2.APIResponse, error) {
	return p.getAllBusinessUnitsAttr(ctx, p)
}

// getAllManagementUnits retrieves all Genesys Cloud management units. Used by the exporter to name limits after their management unit.
func (p *timeOffLimitProxy) getAllManagementUnits(ctx context.Context) (*[]platformclientv2.Managementunit, *platformclientv2.APIResponse, error) {
	return p.getAllManagementUnitsAttr(ctx, p)
}

// createTimeOffLimitFn is an implementation function for creating a Genesys Cloud time off limit
func createTimeOffLimitFn(ctx context.Context, p *timeOffLimitProxy, businessUnitId string, limit *platformclientv2.Bucreatetimeofflimitrequest) (*platformclientv2.Butimeofflimitresponse, *platformclientv2.APIResponse, error) {
	return p.workforceManagementApi.PostWorkforcemanagementBusinessunitTimeofflimits(businessUnitId, *limit)
}

// getAllTimeOffLimitsFn is the implementation for retrieving all time off limits of a business unit in Genesys Cloud
func getAllTimeOffLimitsFn(ctx context.Context, p *timeOffLimitProxy, businessUnitId string) (*[]platformclientv2.Butimeofflimitresponse, *platformclientv2.APIResponse, error) {
	listing, resp, err := p.workforceManagementApi.GetWorkforcemanagementBusinessunitTimeofflimits(businessUnitId, "")
	if err != nil {
		return nil, resp, err
	}

	if listing == nil || listing.Entities == nil {
		return &[]platformclientv2.Butimeofflimitresponse{}, resp, nil
	}

	return listing.Entities, resp, nil
}

// getTimeOffLimitByIdFn is an implementation of the function to get a Genesys Cloud time off limit by id
func getTimeOffLimitByIdFn(ctx context.Context, p *timeOffLimitProxy, businessUnitId string, limitId string) (*platformclientv2.Butimeofflimitresponse, *platformclientv2.APIResponse, error) {
	return p.workforceManagementApi.GetWorkforcemanagementBusinessunitTimeofflimit(businessUnitId, limitId)
}

// deleteTimeOffLimitFn is an implementation function for deleting a Genesys Cloud time off limit
func deleteTimeOffLimitFn(ctx context.Context, p *timeOffLimitProxy, businessUnitId string, limitId string) (*platformclientv2.APIResponse, error) {
	return p.workforceManagementApi.DeleteWorkforcemanagementBusinessunitTimeofflimit(businessUnitId, limitId)
}

// setLimitValuesFn is an implementation function for setting the limit values of a Genesys Cloud time off limit
func setLimitValuesFn(ctx context.Context, p *timeOffLimitProxy, businessUnitId string, limitId string, values *platformclientv2.Busettimeofflimitvaluesrequest) (*platformclientv2.Butimeofflimitresponse, *platformclientv2.APIResponse, error) {
	return p.workforceManagementApi.PutWorkforcemanagementBusinessunitTimeofflimitValues(businessUnitId, limitId, *values)
}

// queryLimitValuesFn is an implementation function for reading the limit values of a Genesys Cloud time off limit
func queryLimitValuesFn(ctx context.Context, p *timeOffLimitProxy, businessUnitId string, limitId string, dateRanges []platformclientv2.Localdaterange) (*platformclientv2.Butimeofflimitvaluesforgranularityresponse, *platformclientv2.APIResponse, error) {
	return p.workforceManagementApi.PostWorkforcemanagementBusinessunitTimeofflimitValuesQuery(businessUnitId, limitId, platformclientv2.Querytimeofflimitvaluesforgranularityrequest{DateRanges: &dateRanges})
}

// getAllBusinessUnitsFn is an implementation of the function to list Genesys Cloud business units
func getAllBusinessUnitsFn(ctx context.Context, p *timeOffLimitProxy) (*[]platformclientv2.Businessunitlistitem, *platformclientv2.APIResponse, error) {
	listing, resp, err := p.workforceManagementApi.GetWorkforcemanagementBusinessunits("", "")
	if err != nil {
		return nil, resp, err
	}

	if listing == nil || listing.Entities == nil {
		return &[]platformclientv2.Businessunitlistitem{}, resp, nil
	}

	return listing.Entities, resp, nil
}

// getAllManagementUnitsFn is an implementation of the function to list Genesys Cloud management units.
// The api documents paging as unsupported and returns every management unit, so the paging arguments are placeholders.
func getAllManagementUnitsFn(ctx context.Context, p *timeOffLimitProxy) (*[]platformclientv2.Managementunit, *platformclientv2.APIResponse, error) {
	listing, resp, err := p.workforceManagementApi.GetWorkforcemanagementManagementunits(100, 1, "", "", "")
	if err != nil {
		return nil, resp, err
	}

	if listing == nil || listing.Entities == nil {
		return &[]platformclientv2.Managementunit{}, resp, nil
	}

	return listing.Entities, resp, nil
}
