package workforcemanagement_managementunits_agents

import (
	"context"

	"github.com/mypurecloud/platform-client-sdk-go/v200/platformclientv2"
)

/*
The genesyscloud_workforcemanagement_managementunits_agents_proxy.go file contains the proxy structures and methods that interact
with the Genesys Cloud SDK. We use composition here for each function on the proxy so individual functions can be stubbed
out during testing.
*/

// internalProxy holds a proxy instance that can be used throughout the package
var internalProxy *managementUnitAgentsProxy

// Type definitions for each func on our proxy so we can easily mock them out later
type getMembersFunc func(ctx context.Context, p *managementUnitAgentsProxy, managementUnitId string) ([]string, *platformclientv2.APIResponse, error)
type getAgentSchedulableFunc func(ctx context.Context, p *managementUnitAgentsProxy, managementUnitId string, userId string) (*bool, *platformclientv2.APIResponse, error)
type moveAgentsFunc func(ctx context.Context, p *managementUnitAgentsProxy, userIds []string, destinationManagementUnitId *string) (*platformclientv2.Moveagentsresponse, *platformclientv2.APIResponse, error)
type updateAgentsFunc func(ctx context.Context, p *managementUnitAgentsProxy, managementUnitId string, agents []platformclientv2.Updatemuagentrequest) (*platformclientv2.APIResponse, error)
type getAllManagementUnitsFunc func(ctx context.Context, p *managementUnitAgentsProxy) (*[]platformclientv2.Managementunit, *platformclientv2.APIResponse, error)

// managementUnitAgentsProxy contains all the methods that call genesys cloud APIs.
type managementUnitAgentsProxy struct {
	clientConfig              *platformclientv2.Configuration
	workforceManagementApi    *platformclientv2.WorkforceManagementApi
	getMembersAttr            getMembersFunc
	getAgentSchedulableAttr   getAgentSchedulableFunc
	moveAgentsAttr            moveAgentsFunc
	updateAgentsAttr          updateAgentsFunc
	getAllManagementUnitsAttr getAllManagementUnitsFunc
}

// newManagementUnitAgentsProxy initializes the management unit agents proxy with all of the data needed to communicate with Genesys Cloud
func newManagementUnitAgentsProxy(clientConfig *platformclientv2.Configuration) *managementUnitAgentsProxy {
	api := platformclientv2.NewWorkforceManagementApiWithConfig(clientConfig)
	return &managementUnitAgentsProxy{
		clientConfig:              clientConfig,
		workforceManagementApi:    api,
		getMembersAttr:            getMembersFn,
		getAgentSchedulableAttr:   getAgentSchedulableFn,
		moveAgentsAttr:            moveAgentsFn,
		updateAgentsAttr:          updateAgentsFn,
		getAllManagementUnitsAttr: getAllManagementUnitsFn,
	}
}

// getManagementUnitAgentsProxy acts as a singleton for the internalProxy. It also ensures
// that we can still proxy our tests by directly setting internalProxy package variable
func getManagementUnitAgentsProxy(clientConfig *platformclientv2.Configuration) *managementUnitAgentsProxy {
	if internalProxy == nil {
		internalProxy = newManagementUnitAgentsProxy(clientConfig)
	}

	return internalProxy
}

// getMembers returns the ids of the users that are members of a management unit
func (p *managementUnitAgentsProxy) getMembers(ctx context.Context, managementUnitId string) ([]string, *platformclientv2.APIResponse, error) {
	return p.getMembersAttr(ctx, p, managementUnitId)
}

// getAgentSchedulable returns whether an agent of a management unit can be included in schedule generation
func (p *managementUnitAgentsProxy) getAgentSchedulable(ctx context.Context, managementUnitId string, userId string) (*bool, *platformclientv2.APIResponse, error) {
	return p.getAgentSchedulableAttr(ctx, p, managementUnitId, userId)
}

// moveAgents moves users into a management unit, or out of their management unit when the destination is nil
func (p *managementUnitAgentsProxy) moveAgents(ctx context.Context, userIds []string, destinationManagementUnitId *string) (*platformclientv2.Moveagentsresponse, *platformclientv2.APIResponse, error) {
	return p.moveAgentsAttr(ctx, p, userIds, destinationManagementUnitId)
}

// updateAgents updates the configuration of agents that are already members of a management unit
func (p *managementUnitAgentsProxy) updateAgents(ctx context.Context, managementUnitId string, agents []platformclientv2.Updatemuagentrequest) (*platformclientv2.APIResponse, error) {
	return p.updateAgentsAttr(ctx, p, managementUnitId, agents)
}

// getAllManagementUnits retrieves all Genesys Cloud management units. Used by the exporter.
func (p *managementUnitAgentsProxy) getAllManagementUnits(ctx context.Context) (*[]platformclientv2.Managementunit, *platformclientv2.APIResponse, error) {
	return p.getAllManagementUnitsAttr(ctx, p)
}

// getMembersFn is an implementation of the function to get the member user ids of a management unit
func getMembersFn(ctx context.Context, p *managementUnitAgentsProxy, managementUnitId string) ([]string, *platformclientv2.APIResponse, error) {
	listing, resp, err := p.workforceManagementApi.GetWorkforcemanagementManagementunitUsers(managementUnitId)
	if err != nil {
		return nil, resp, err
	}

	userIds := make([]string, 0)
	for _, user := range *listing.Entities {
		userIds = append(userIds, *user.Id)
	}
	return userIds, resp, nil
}

// getAgentSchedulableFn is an implementation of the function to read the schedulable flag of an agent.
// The api has no public route that lists agents with their configuration, so each agent is read individually without its capabilities.
func getAgentSchedulableFn(ctx context.Context, p *managementUnitAgentsProxy, managementUnitId string, userId string) (*bool, *platformclientv2.APIResponse, error) {
	agent, resp, err := p.workforceManagementApi.GetWorkforcemanagementManagementunitAgent(managementUnitId, userId, true, nil)
	if err != nil {
		return nil, resp, err
	}
	return agent.Schedulable, resp, nil
}

// moveAgentsFn is an implementation of the function to move agents in and out of a management unit
func moveAgentsFn(ctx context.Context, p *managementUnitAgentsProxy, userIds []string, destinationManagementUnitId *string) (*platformclientv2.Moveagentsresponse, *platformclientv2.APIResponse, error) {
	return p.workforceManagementApi.PostWorkforcemanagementAgents(platformclientv2.Moveagentsrequest{
		UserIds:                     &userIds,
		DestinationManagementUnitId: destinationManagementUnitId,
	})
}

// updateAgentsFn is an implementation of the function to update the configuration of agents in a management unit
func updateAgentsFn(ctx context.Context, p *managementUnitAgentsProxy, managementUnitId string, agents []platformclientv2.Updatemuagentrequest) (*platformclientv2.APIResponse, error) {
	return p.workforceManagementApi.PatchWorkforcemanagementManagementunitAgents(managementUnitId, platformclientv2.Updatemuagentsrequest{Entities: &agents})
}

// getAllManagementUnitsFn is an implementation of the function to list Genesys Cloud management units.
// The api documents paging as unsupported and returns every management unit, so the paging arguments are placeholders.
func getAllManagementUnitsFn(ctx context.Context, p *managementUnitAgentsProxy) (*[]platformclientv2.Managementunit, *platformclientv2.APIResponse, error) {
	listing, resp, err := p.workforceManagementApi.GetWorkforcemanagementManagementunits(100, 1, "", "", "")
	if err != nil {
		return nil, resp, err
	}

	return listing.Entities, resp, nil
}
