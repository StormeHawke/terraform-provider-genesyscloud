package workforcemanagement_managementunits_agents

import (
	"fmt"
	"sort"
	"strings"

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

// checkMoveResults returns an error describing every user the move did not place in the requested management unit.
// The api answers a move with 202 and delivers the per-user results through the WfmMoveAgentsComplete event, so the response normally has no
// results and this only reports what the response does contain. Moves that fail are detected by waiting for the membership to change.
func checkMoveResults(response *platformclientv2.Moveagentsresponse) error {
	var failures []string
	for _, result := range *response.Results {
		userId := *result.User.Id
		moveResult := *result.Result
		if !acceptedMoveResults[moveResult] {
			failures = append(failures, fmt.Sprintf("%s: %s", userId, moveResult))
		}
	}

	if len(failures) > 0 {
		return fmt.Errorf("unable to move %d agent(s): %s", len(failures), strings.Join(failures, ", "))
	}
	return nil
}

func toLookup(values []string) map[string]bool {
	lookup := make(map[string]bool, len(values))
	for _, value := range values {
		lookup[value] = true
	}
	return lookup
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
func GenerateManagementUnitAgentsResource(resourceLabel string, managementUnitId string, userIds string) string {
	return fmt.Sprintf(`resource "%s" "%s" {
	management_unit_id = %s
	user_ids           = %s
}
`, ResourceType, resourceLabel, managementUnitId, userIds)
}
