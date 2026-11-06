package workforcemanagement_managementunits_agents

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/mypurecloud/platform-client-sdk-go/v200/platformclientv2"
	"github.com/mypurecloud/terraform-provider-genesyscloud/genesyscloud/util/lists"
)

const (
	// moveBatchSize is the maximum number of users sent per move request, which the service prefers over many small requests.
	// The service rejects a move with more users than the organization's agents.per.management.unit.max limit, whose only allowed
	// values are 1500 (the default) and 2000. 1500 is therefore valid for every organization, and an organization that raised its limit
	// only needs one extra request for a larger move, so the limit is not looked up.
	moveBatchSize = 1500
	// scheduleUpdateBatchSize is the number of agents sent per update of the schedulable flag. The update has no documented limit,
	// so it stays conservative.
	scheduleUpdateBatchSize = 100
	// schedulableReadConcurrency bounds the parallel per-agent reads. The api has no public route that lists agent configuration, and
	// reading one agent is rate limited to 60 requests per minute per user per organization, so reading every agent of a large
	// management unit takes minutes regardless of the concurrency and relies on the client retrying rate limited requests.
	schedulableReadConcurrency = 10
)

// acceptedMoveResults are the move results that mean the user is, or already was, in the requested management unit
var acceptedMoveResults = map[string]bool{
	"MoveSuccessful": true,
	"AlreadyMoved":   true,
	"NothingToDo":    true,
}

func setToSlice(set *schema.Set) []string {
	if set == nil {
		return nil
	}
	values := lists.SetToStringList(set)
	if values == nil {
		return nil
	}
	return *values
}

func sliceToSet(values []string) *schema.Set {
	return lists.StringListToSet(values)
}

// desiredMembers returns the union of the schedulable and unschedulable user ids
func desiredMembers(schedulable []string, unschedulable []string) []string {
	members := make(map[string]bool, len(schedulable)+len(unschedulable))
	for _, id := range schedulable {
		members[id] = true
	}
	for _, id := range unschedulable {
		members[id] = true
	}
	return sortedKeys(members)
}

// diffMembers returns the users to move in and the users to move out so the current members match the desired members
func diffMembers(desired []string, current []string) (toAdd []string, toRemove []string) {
	desiredSet := toLookup(desired)
	currentSet := toLookup(current)
	for _, id := range desired {
		if !currentSet[id] {
			toAdd = append(toAdd, id)
		}
	}
	for _, id := range current {
		if !desiredSet[id] {
			toRemove = append(toRemove, id)
		}
	}
	sort.Strings(toAdd)
	sort.Strings(toRemove)
	return toAdd, toRemove
}

// overlap returns the users that appear in both lists
func overlap(a []string, b []string) []string {
	bSet := toLookup(b)
	var shared []string
	for _, id := range a {
		if bSet[id] {
			shared = append(shared, id)
		}
	}
	sort.Strings(shared)
	return shared
}

// usersNeedingScheduleUpdate returns the users whose schedulable flag must be written: users that are newly listed in a set,
// either because they are new members or because they moved from the other set. Users that stay in the same set are left alone.
func usersNeedingScheduleUpdate(oldSchedulable []string, oldUnschedulable []string, newSchedulable []string, newUnschedulable []string) (makeSchedulable []string, makeUnschedulable []string) {
	oldSchedulableSet := toLookup(oldSchedulable)
	oldUnschedulableSet := toLookup(oldUnschedulable)
	for _, id := range newSchedulable {
		if !oldSchedulableSet[id] {
			makeSchedulable = append(makeSchedulable, id)
		}
	}
	for _, id := range newUnschedulable {
		if !oldUnschedulableSet[id] {
			makeUnschedulable = append(makeUnschedulable, id)
		}
	}
	sort.Strings(makeSchedulable)
	sort.Strings(makeUnschedulable)
	return makeSchedulable, makeUnschedulable
}

// checkMoveResults returns an error describing every user the move did not place in the requested management unit.
// The api answers a move with 202 and delivers the per-user results through the WfmMoveAgentsComplete event, so the response normally has no
// results and this only reports what the response does contain. Moves that fail are detected by waiting for the membership to change.
func checkMoveResults(response *platformclientv2.Moveagentsresponse) error {
	if response == nil || response.Results == nil {
		return nil
	}

	var failures []string
	for _, result := range *response.Results {
		userId := "unknown user"
		if result.User != nil && result.User.Id != nil {
			userId = *result.User.Id
		}
		moveResult := "no result"
		if result.Result != nil {
			moveResult = *result.Result
		}
		if !acceptedMoveResults[moveResult] {
			failures = append(failures, fmt.Sprintf("%s: %s", userId, moveResult))
		}
	}

	if len(failures) > 0 {
		return fmt.Errorf("unable to move %d agent(s): %s", len(failures), strings.Join(failures, ", "))
	}
	return nil
}

// validateDistinctUserSets rejects a configuration that lists the same user as both schedulable and unschedulable
func validateDistinctUserSets(_ context.Context, diff *schema.ResourceDiff, _ interface{}) error {
	if !diff.NewValueKnown("schedulable_user_ids") || !diff.NewValueKnown("unschedulable_user_ids") {
		return nil
	}

	shared := overlap(setToSlice(diff.Get("schedulable_user_ids").(*schema.Set)), setToSlice(diff.Get("unschedulable_user_ids").(*schema.Set)))
	if len(shared) > 0 {
		return fmt.Errorf("users cannot be both schedulable and unschedulable, found in both schedulable_user_ids and unschedulable_user_ids: %s", strings.Join(shared, ", "))
	}
	return nil
}

// partitionBySchedulable reads the schedulable flag of each member and splits the members into schedulable and unschedulable users.
// A member without a schedulable flag is treated as unschedulable: the service only treats an agent as schedulable when the flag is explicitly true.
func partitionBySchedulable(ctx context.Context, proxy *managementUnitAgentsProxy, managementUnitId string, members []string) (schedulable []string, unschedulable []string, apiErr error) {
	type outcome struct {
		userId      string
		schedulable bool
		skipped     bool
		err         error
	}

	outcomes := make([]outcome, len(members))
	semaphore := make(chan struct{}, schedulableReadConcurrency)
	var wg sync.WaitGroup
	for i, userId := range members {
		wg.Add(1)
		go func(i int, userId string) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			flag, resp, err := proxy.getAgentSchedulable(ctx, managementUnitId, userId)
			if err != nil {
				// An agent that left the management unit between listing and reading is no longer a member
				if resp != nil && resp.StatusCode == 404 {
					outcomes[i] = outcome{userId: userId, skipped: true}
					return
				}
				outcomes[i] = outcome{userId: userId, err: fmt.Errorf("failed to read agent %s of management unit %s: %w", userId, managementUnitId, err)}
				return
			}
			outcomes[i] = outcome{userId: userId, schedulable: flag != nil && *flag}
		}(i, userId)
	}
	wg.Wait()

	for _, result := range outcomes {
		switch {
		case result.err != nil:
			return nil, nil, result.err
		case result.skipped:
			continue
		case result.schedulable:
			schedulable = append(schedulable, result.userId)
		default:
			unschedulable = append(unschedulable, result.userId)
		}
	}
	return schedulable, unschedulable, nil
}

func buildScheduleUpdates(makeSchedulable []string, makeUnschedulable []string) []platformclientv2.Updatemuagentrequest {
	updates := make([]platformclientv2.Updatemuagentrequest, 0, len(makeSchedulable)+len(makeUnschedulable))
	for _, id := range makeSchedulable {
		updates = append(updates, platformclientv2.Updatemuagentrequest{UserId: platformclientv2.String(id), Schedulable: platformclientv2.Bool(true)})
	}
	for _, id := range makeUnschedulable {
		updates = append(updates, platformclientv2.Updatemuagentrequest{UserId: platformclientv2.String(id), Schedulable: platformclientv2.Bool(false)})
	}
	return updates
}

func chunkUpdates(updates []platformclientv2.Updatemuagentrequest, size int) [][]platformclientv2.Updatemuagentrequest {
	var chunks [][]platformclientv2.Updatemuagentrequest
	for i := 0; i < len(updates); i += size {
		end := i + size
		if end > len(updates) {
			end = len(updates)
		}
		chunks = append(chunks, updates[i:end])
	}
	return chunks
}

func toLookup(values []string) map[string]bool {
	lookup := make(map[string]bool, len(values))
	for _, value := range values {
		lookup[value] = true
	}
	return lookup
}

func sortedKeys(set map[string]bool) []string {
	keys := make([]string, 0, len(set))
	for key := range set {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

// sameMembers reports whether two lists contain the same users, ignoring order
func sameMembers(a []string, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	toAdd, toRemove := diffMembers(a, b)
	return len(toAdd) == 0 && len(toRemove) == 0
}

// GenerateManagementUnitAgentsResource returns terraform config for a management unit agents resource, used in tests
func GenerateManagementUnitAgentsResource(resourceLabel string, managementUnitId string, schedulableUserIds string, unschedulableUserIds string) string {
	return fmt.Sprintf(`resource "%s" "%s" {
	management_unit_id     = %s
	schedulable_user_ids   = %s
	unschedulable_user_ids = %s
}
`, ResourceType, resourceLabel, managementUnitId, schedulableUserIds, unschedulableUserIds)
}
