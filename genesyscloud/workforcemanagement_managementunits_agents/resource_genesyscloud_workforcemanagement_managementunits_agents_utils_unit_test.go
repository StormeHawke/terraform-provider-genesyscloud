package workforcemanagement_managementunits_agents

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/mypurecloud/platform-client-sdk-go/v200/platformclientv2"
	"github.com/mypurecloud/terraform-provider-genesyscloud/genesyscloud/util"
)

// fastPolling makes the waits for an asynchronous move short, so tests do not spend seconds waiting for a stubbed api
func fastPolling(t *testing.T) {
	previous := util.AsyncPollSchedule
	util.AsyncPollSchedule = []time.Duration{time.Millisecond}
	t.Cleanup(func() { util.AsyncPollSchedule = previous })
}

func TestUnitDiffMembers(t *testing.T) {
	toAdd, toRemove := diffMembers([]string{"a", "c", "d"}, []string{"a", "b"})
	if !reflect.DeepEqual(toAdd, []string{"c", "d"}) || !reflect.DeepEqual(toRemove, []string{"b"}) {
		t.Errorf("unexpected diff: add %v remove %v", toAdd, toRemove)
	}
}

func TestUnitOverlap(t *testing.T) {
	if got := overlap([]string{"a", "b"}, []string{"b", "c"}); !reflect.DeepEqual(got, []string{"b"}) {
		t.Errorf("expected [b], got %v", got)
	}
}

func moveResponse(results map[string]string) *platformclientv2.Moveagentsresponse {
	entries := make([]platformclientv2.Moveagentresponse, 0, len(results))
	for userId, result := range results {
		entries = append(entries, platformclientv2.Moveagentresponse{
			User:   &platformclientv2.Userreference{Id: platformclientv2.String(userId)},
			Result: platformclientv2.String(result),
		})
	}
	return &platformclientv2.Moveagentsresponse{Results: &entries}
}

func TestUnitCheckMoveResults(t *testing.T) {
	if err := checkMoveResults(moveResponse(map[string]string{"a": "MoveSuccessful", "b": "AlreadyMoved", "c": "NothingToDo"})); err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	err := checkMoveResults(moveResponse(map[string]string{"a": "MoveSuccessful", "b": "DestinationManagementUnitAgentLimitExceeded"}))
	if err == nil || !strings.Contains(err.Error(), "b: DestinationManagementUnitAgentLimitExceeded") || strings.Contains(err.Error(), "a:") {
		t.Errorf("expected only the failed user in the error, got %v", err)
	}
}

// fakeApi records the calls made through a stubbed proxy and simulates asynchronous membership
type fakeApi struct {
	mutex       sync.Mutex
	members     []string
	movedIn     [][]string
	movedOut    [][]string
	moveResults map[string]string
}

func (f *fakeApi) proxy(managementUnitId string) *managementUnitAgentsProxy {
	return &managementUnitAgentsProxy{
		getMembersAttr: func(ctx context.Context, p *managementUnitAgentsProxy, id string) ([]string, *platformclientv2.APIResponse, error) {
			f.mutex.Lock()
			defer f.mutex.Unlock()
			return append([]string(nil), f.members...), nil, nil
		},
		moveAgentsAttr: func(ctx context.Context, p *managementUnitAgentsProxy, userIds []string, destination *string) (*platformclientv2.Moveagentsresponse, *platformclientv2.APIResponse, error) {
			f.mutex.Lock()
			defer f.mutex.Unlock()
			results := map[string]string{}
			for _, id := range userIds {
				results[id] = "MoveSuccessful"
				if override, ok := f.moveResults[id]; ok {
					results[id] = override
				}
			}
			if destination != nil && *destination == managementUnitId {
				f.movedIn = append(f.movedIn, userIds)
				for _, id := range userIds {
					if results[id] == "MoveSuccessful" {
						f.members = append(f.members, id)
					}
				}
			} else {
				f.movedOut = append(f.movedOut, userIds)
				removed := toLookup(userIds)
				kept := f.members[:0:0]
				for _, id := range f.members {
					if !removed[id] {
						kept = append(kept, id)
					}
				}
				f.members = kept
			}
			return moveResponse(results), nil, nil
		},
	}
}

func TestUnitApplyAgentsMovesUsersInAndOut(t *testing.T) {
	fastPolling(t)
	api := &fakeApi{members: []string{"a", "b"}}
	proxy := api.proxy("mu-1")

	diagErr := applyAgents(context.Background(), proxy, "mu-1", []string{"a", "c"}) // c is new, b is no longer wanted
	if diagErr != nil {
		t.Fatalf("unexpected error: %v", diagErr)
	}

	if !reflect.DeepEqual(api.movedIn, [][]string{{"c"}}) {
		t.Errorf("expected c moved in, got %v", api.movedIn)
	}
	if !reflect.DeepEqual(api.movedOut, [][]string{{"b"}}) {
		t.Errorf("expected b moved out, got %v", api.movedOut)
	}
}

func TestUnitApplyAgentsSkipsMovesWhenMembersAlreadyMatch(t *testing.T) {
	fastPolling(t)
	api := &fakeApi{members: []string{"a", "b"}}

	diagErr := applyAgents(context.Background(), api.proxy("mu-1"), "mu-1", []string{"a", "b"})
	if diagErr != nil {
		t.Fatalf("unexpected error: %v", diagErr)
	}
	if len(api.movedIn) != 0 || len(api.movedOut) != 0 {
		t.Errorf("expected no api writes, got in %v out %v", api.movedIn, api.movedOut)
	}
}

func TestUnitApplyAgentsReportsFailedMoves(t *testing.T) {
	fastPolling(t)
	api := &fakeApi{members: nil, moveResults: map[string]string{"b": "DestinationBusinessUnitAgentLimitExceeded"}}

	diagErr := applyAgents(context.Background(), api.proxy("mu-1"), "mu-1", []string{"a", "b"})
	if diagErr == nil || !diagErr.HasError() {
		t.Fatal("expected an error for the failed move")
	}
	if detail := diagErr[0].Detail + diagErr[0].Summary; !strings.Contains(detail, "DestinationBusinessUnitAgentLimitExceeded") {
		t.Errorf("expected the move result in the error, got %q", detail)
	}
}

func TestUnitManagementUnitIdChangeDoesNotRecreateTheResource(t *testing.T) {
	if ResourceManagementUnitAgents().Schema["management_unit_id"].ForceNew {
		t.Error("changing management_unit_id must move the agents instead of removing and adding them")
	}
}

func TestUnitApplyAgentsTimeoutNamesUsersThatNeverJoined(t *testing.T) {
	fastPolling(t)
	previousTimeout, previousSchedule := membershipTimeout, util.AsyncPollSchedule
	membershipTimeout, util.AsyncPollSchedule = 200*time.Millisecond, []time.Duration{50 * time.Millisecond}
	defer func() { membershipTimeout, util.AsyncPollSchedule = previousTimeout, previousSchedule }()

	// The move is accepted but the user never becomes a member, as happens when the move fails permanently after the 202
	api := &fakeApi{members: nil}
	proxy := api.proxy("mu-1")
	proxy.moveAgentsAttr = func(ctx context.Context, p *managementUnitAgentsProxy, userIds []string, destination *string) (*platformclientv2.Moveagentsresponse, *platformclientv2.APIResponse, error) {
		return &platformclientv2.Moveagentsresponse{Results: &[]platformclientv2.Moveagentresponse{}}, nil, nil
	}

	diagErr := applyAgents(context.Background(), proxy, "mu-1", []string{"stuck-user"})
	if diagErr == nil || !diagErr.HasError() {
		t.Fatal("expected an error when the agent never joins")
	}
	detail := diagErr[0].Summary + diagErr[0].Detail
	if !strings.Contains(detail, "stuck-user") || !strings.Contains(detail, "agent limit") {
		t.Errorf("expected the error to name the user and the likely reasons, got %q", detail)
	}
}

func TestUnitApplyAgentsSendsLargeMovesAsOneRequest(t *testing.T) {
	fastPolling(t)
	users := make([]string, 250)
	for i := range users {
		users[i] = fmt.Sprintf("user-%03d", i)
	}
	api := &fakeApi{members: nil}

	diagErr := applyAgents(context.Background(), api.proxy("mu-1"), "mu-1", users)
	if diagErr != nil {
		t.Fatalf("unexpected error: %v", diagErr)
	}
	if len(api.movedIn) != 1 || len(api.movedIn[0]) != 250 {
		t.Errorf("expected a single move request with all 250 users, got %d request(s)", len(api.movedIn))
	}
}

func TestUnitApplyAgentsSplitsMovesAboveTheMoveLimit(t *testing.T) {
	fastPolling(t)
	users := make([]string, 3200)
	for i := range users {
		users[i] = fmt.Sprintf("user-%04d", i)
	}
	api := &fakeApi{members: nil}

	diagErr := applyAgents(context.Background(), api.proxy("mu-1"), "mu-1", users)
	if diagErr != nil {
		t.Fatalf("unexpected error: %v", diagErr)
	}
	if len(api.movedIn) != 3 || len(api.movedIn[0]) != 1500 || len(api.movedIn[1]) != 1500 || len(api.movedIn[2]) != 200 {
		sizes := make([]int, len(api.movedIn))
		for i, batch := range api.movedIn {
			sizes[i] = len(batch)
		}
		t.Errorf("expected move requests of 1500, 1500 and 200 users, got %v", sizes)
	}
}
