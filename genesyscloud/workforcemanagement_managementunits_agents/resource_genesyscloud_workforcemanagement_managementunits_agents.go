package workforcemanagement_managementunits_agents

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"
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
	NOTE: This resource's Id is the id of the management unit, because the resource represents the membership of the whole management unit.

	Moving agents in and out of a management unit is asynchronous (the api answers 202), so after each move the members are
	polled until they match. The schedulable flag can only be written for users that are already members, so it is applied last.
*/

/*
The resource_genesyscloud_workforcemanagement_managementunits_agents.go contains all of the methods that perform the core logic for a resource.
*/

// membershipTimeout is how long to wait for an asynchronous move to be reflected in the management unit
var membershipTimeout = 5 * time.Minute

// getAllManagementUnitAgents retrieves the management units that have agents via Terraform in the Genesys Cloud and is used for the exporter
func getAllManagementUnitAgents(ctx context.Context, clientConfig *platformclientv2.Configuration) (resourceExporter.ResourceIDMetaMap, diag.Diagnostics) {
	proxy := getManagementUnitAgentsProxy(clientConfig)
	resources := make(resourceExporter.ResourceIDMetaMap)

	managementUnits, resp, err := proxy.getAllManagementUnits(ctx)
	if err != nil {
		return nil, util.BuildAPIDiagnosticError(ResourceType, fmt.Sprintf("Failed to get management units: %v", err), resp)
	}

	for _, managementUnit := range *managementUnits {
		members, resp, err := proxy.getMembers(ctx, *managementUnit.Id)
		if err != nil {
			return nil, util.BuildAPIDiagnosticError(ResourceType, fmt.Sprintf("Failed to get agents of management unit %s: %v", *managementUnit.Id, err), resp)
		}
		if len(members) == 0 {
			continue
		}
		resources[*managementUnit.Id] = &resourceExporter.ResourceMeta{BlockLabel: *managementUnit.Name + "_agents"}
	}

	return resources, nil
}

// createManagementUnitAgents is used by the workforcemanagement_managementunits_agents resource to put agents in a management unit
func createManagementUnitAgents(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	sdkConfig := meta.(*provider.ProviderMeta).ClientConfig
	proxy := getManagementUnitAgentsProxy(sdkConfig)
	managementUnitId := d.Get("management_unit_id").(string)

	schedulable := setToSlice(d.Get("schedulable_user_ids").(*schema.Set))
	unschedulable := setToSlice(d.Get("unschedulable_user_ids").(*schema.Set))

	log.Printf("Adding agents to management unit %s", managementUnitId)
	if diagErr := applyAgents(ctx, proxy, managementUnitId, schedulable, unschedulable, nil, nil); diagErr != nil {
		return diagErr
	}

	d.SetId(managementUnitId)
	log.Printf("Added agents to management unit %s", managementUnitId)
	return readManagementUnitAgents(ctx, d, meta)
}

// readManagementUnitAgents is used by the workforcemanagement_managementunits_agents resource to read the agents of a management unit from Genesys Cloud
func readManagementUnitAgents(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	sdkConfig := meta.(*provider.ProviderMeta).ClientConfig
	proxy := getManagementUnitAgentsProxy(sdkConfig)
	cc := consistency_checker.NewConsistencyCheck(ctx, d, meta, ResourceManagementUnitAgents(), constants.ConsistencyChecks(), ResourceType)

	log.Printf("Reading agents of management unit %s", d.Id())

	return util.WithRetriesForRead(ctx, d, func() *retry.RetryError {
		members, resp, getErr := proxy.getMembers(ctx, d.Id())
		if getErr != nil {
			// A 404 is not retried: the management unit is readable as soon as it is created, so a 404 means it is gone and WithRetriesForRead removes it from state
			return retry.NonRetryableError(util.BuildWithRetriesApiDiagnosticError(ResourceType, fmt.Sprintf("Failed to read agents of management unit %s: %s", d.Id(), getErr), resp))
		}

		schedulable, unschedulable, partitionErr := partitionBySchedulable(ctx, proxy, d.Id(), members)
		if partitionErr != nil {
			return retry.NonRetryableError(partitionErr)
		}

		_ = d.Set("management_unit_id", d.Id())
		_ = d.Set("schedulable_user_ids", sliceToSet(schedulable))
		_ = d.Set("unschedulable_user_ids", sliceToSet(unschedulable))

		log.Printf("Read agents of management unit %s", d.Id())
		return cc.CheckState(d)
	})
}

// updateManagementUnitAgents is used by the workforcemanagement_managementunits_agents resource to update the agents of a management unit in Genesys Cloud
func updateManagementUnitAgents(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	sdkConfig := meta.(*provider.ProviderMeta).ClientConfig
	proxy := getManagementUnitAgentsProxy(sdkConfig)

	oldSchedulable, newSchedulable := d.GetChange("schedulable_user_ids")
	oldUnschedulable, newUnschedulable := d.GetChange("unschedulable_user_ids")

	log.Printf("Updating agents of management unit %s", d.Id())
	if diagErr := applyAgents(ctx, proxy, d.Id(),
		setToSlice(newSchedulable.(*schema.Set)), setToSlice(newUnschedulable.(*schema.Set)),
		setToSlice(oldSchedulable.(*schema.Set)), setToSlice(oldUnschedulable.(*schema.Set))); diagErr != nil {
		return diagErr
	}

	log.Printf("Updated agents of management unit %s", d.Id())
	return readManagementUnitAgents(ctx, d, meta)
}

// deleteManagementUnitAgents is used by the workforcemanagement_managementunits_agents resource to remove the agents it manages from the management unit.
// Only the users in the state are removed, not every member, so agents the resource never managed are left alone.
func deleteManagementUnitAgents(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	sdkConfig := meta.(*provider.ProviderMeta).ClientConfig
	proxy := getManagementUnitAgentsProxy(sdkConfig)

	managed := desiredMembers(setToSlice(d.Get("schedulable_user_ids").(*schema.Set)), setToSlice(d.Get("unschedulable_user_ids").(*schema.Set)))

	log.Printf("Removing agents from management unit %s", d.Id())
	members, resp, err := proxy.getMembers(ctx, d.Id())
	if err != nil {
		if util.IsStatus404(resp) {
			log.Printf("Management unit %s no longer exists", d.Id())
			return nil
		}
		return util.BuildAPIDiagnosticError(ResourceType, fmt.Sprintf("Failed to read agents of management unit %s before removing them: %s", d.Id(), err), resp)
	}

	stillMembers := overlap(managed, members)
	if len(stillMembers) == 0 {
		return nil
	}

	if diagErr := moveAgents(ctx, proxy, stillMembers, nil); diagErr != nil {
		return diagErr
	}
	if diagErr := waitForAgentsRemoved(ctx, proxy, d.Id(), stillMembers); diagErr != nil {
		return diagErr
	}

	log.Printf("Removed agents from management unit %s", d.Id())
	return nil
}

// applyAgents makes the members of the management unit match the desired schedulable and unschedulable users.
// The old users are the previously applied ones, and only users that changed set have their schedulable flag written.
func applyAgents(ctx context.Context, proxy *managementUnitAgentsProxy, managementUnitId string, schedulable []string, unschedulable []string, oldSchedulable []string, oldUnschedulable []string) diag.Diagnostics {
	if shared := overlap(schedulable, unschedulable); len(shared) > 0 {
		errorMsg := fmt.Sprintf("Users cannot be both schedulable and unschedulable in management unit %s: %s", managementUnitId, strings.Join(shared, ", "))
		return util.BuildDiagnosticError(ResourceType, errorMsg, fmt.Errorf("%s", errorMsg))
	}

	desired := desiredMembers(schedulable, unschedulable)
	current, resp, err := proxy.getMembers(ctx, managementUnitId)
	if err != nil {
		return util.BuildAPIDiagnosticError(ResourceType, fmt.Sprintf("Failed to read agents of management unit %s: %s", managementUnitId, err), resp)
	}

	toAdd, toRemove := diffMembers(desired, current)
	if len(toAdd) > 0 {
		if diagErr := moveAgents(ctx, proxy, toAdd, &managementUnitId); diagErr != nil {
			return diagErr
		}
	}
	if len(toRemove) > 0 {
		if diagErr := moveAgents(ctx, proxy, toRemove, nil); diagErr != nil {
			return diagErr
		}
	}
	if len(toAdd) > 0 || len(toRemove) > 0 {
		if diagErr := waitForMembers(ctx, proxy, managementUnitId, desired); diagErr != nil {
			return diagErr
		}
	}

	makeSchedulable, makeUnschedulable := usersNeedingScheduleUpdate(oldSchedulable, oldUnschedulable, schedulable, unschedulable)
	return updateSchedulable(ctx, proxy, managementUnitId, buildScheduleUpdates(makeSchedulable, makeUnschedulable))
}

// moveAgents moves the users into the destination management unit, or out of their management unit when the destination is nil, in as few requests as the move limit allows
func moveAgents(ctx context.Context, proxy *managementUnitAgentsProxy, userIds []string, destinationManagementUnitId *string) diag.Diagnostics {
	for _, batch := range chunkIds(userIds, moveBatchSize) {
		response, resp, err := proxy.moveAgents(ctx, batch, destinationManagementUnitId)
		if err != nil {
			return util.BuildAPIDiagnosticError(ResourceType, fmt.Sprintf("Failed to move agents: %s", err), resp)
		}
		if resultErr := checkMoveResults(response); resultErr != nil {
			return util.BuildDiagnosticError(ResourceType, resultErr.Error(), resultErr)
		}
	}
	return nil
}

func isTransientStatus(resp *platformclientv2.APIResponse) bool {
	return util.IsStatus429(resp) || (resp != nil && resp.StatusCode >= http.StatusInternalServerError)
}

// updateSchedulable writes the schedulable flag in batches. The api rejects users that are not yet members, so a membership
// that has not settled is retried. Rate limiting and server errors are transient and are retried too.
func updateSchedulable(ctx context.Context, proxy *managementUnitAgentsProxy, managementUnitId string, updates []platformclientv2.Updatemuagentrequest) diag.Diagnostics {
	for _, batch := range chunkUpdates(updates, scheduleUpdateBatchSize) {
		batch := batch
		diagErr := util.WithRetries(ctx, 60*time.Second, func() *retry.RetryError {
			resp, err := proxy.updateAgents(ctx, managementUnitId, batch)
			if err == nil {
				return nil
			}
			builtDiagErr := util.BuildWithRetriesApiDiagnosticError(ResourceType, fmt.Sprintf("Failed to update agents of management unit %s: %s", managementUnitId, err), resp)
			if isTransientStatus(resp) || (resp != nil && strings.Contains(strings.ToLower(resp.ErrorMessage), "not found")) {
				return retry.RetryableError(builtDiagErr)
			}
			return retry.NonRetryableError(builtDiagErr)
		})
		if diagErr != nil {
			return diagErr
		}
	}
	return nil
}

// waitForMembers waits until the management unit has exactly the desired members.
// The move api reports each user's result only through the WfmMoveAgentsComplete event, so a move that failed permanently (for example
// because an agent limit was exceeded) can only be noticed here, as users that never join.
func waitForMembers(ctx context.Context, proxy *managementUnitAgentsProxy, managementUnitId string, desired []string) diag.Diagnostics {
	return util.WithRetries(ctx, membershipTimeout, func() *retry.RetryError {
		members, resp, err := proxy.getMembers(ctx, managementUnitId)
		if err != nil {
			return retry.NonRetryableError(util.BuildWithRetriesApiDiagnosticError(ResourceType, fmt.Sprintf("Failed to read agents of management unit %s while waiting for the move: %s", managementUnitId, err), resp))
		}
		if sameMembers(members, desired) {
			return nil
		}

		missing, unexpected := diffMembers(desired, members)
		return retry.RetryableError(util.BuildWithRetriesApiDiagnosticError(ResourceType, fmt.Sprintf(
			"management unit %s does not have the expected agents yet (not yet members: %v, still members: %v). If this persists, the move may have failed: "+
				"the agent limit of the management unit or business unit may be exceeded, an agent may already be moving to a different management unit, "+
				"or the source management unit may not be authorized", managementUnitId, missing, unexpected), resp))
	})
}

// waitForAgentsRemoved waits until none of the removed users are members of the management unit any more
func waitForAgentsRemoved(ctx context.Context, proxy *managementUnitAgentsProxy, managementUnitId string, removed []string) diag.Diagnostics {
	return util.WithRetries(ctx, membershipTimeout, func() *retry.RetryError {
		members, resp, err := proxy.getMembers(ctx, managementUnitId)
		if err != nil {
			if util.IsStatus404(resp) {
				return nil
			}
			return retry.NonRetryableError(util.BuildWithRetriesApiDiagnosticError(ResourceType, fmt.Sprintf("Failed to read agents of management unit %s while waiting for the removal: %s", managementUnitId, err), resp))
		}
		if len(overlap(removed, members)) == 0 {
			return nil
		}
		return retry.RetryableError(util.BuildWithRetriesApiDiagnosticError(ResourceType, fmt.Sprintf("management unit %s still has agents that should have been removed", managementUnitId), resp))
	})
}

func chunkIds(ids []string, size int) [][]string {
	var chunks [][]string
	for i := 0; i < len(ids); i += size {
		end := i + size
		if end > len(ids) {
			end = len(ids)
		}
		chunks = append(chunks, ids[i:end])
	}
	return chunks
}
