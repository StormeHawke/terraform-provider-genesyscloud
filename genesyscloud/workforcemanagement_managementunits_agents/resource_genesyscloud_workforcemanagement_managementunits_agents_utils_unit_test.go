package workforcemanagement_managementunits_agents

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/mypurecloud/platform-client-sdk-go/v200/platformclientv2"
)

func TestUnitDiffMembers(t *testing.T) {
	toAdd, toRemove := diffMembers([]string{"a", "c", "d"}, []string{"a", "b"})
	if !reflect.DeepEqual(toAdd, []string{"c", "d"}) || !reflect.DeepEqual(toRemove, []string{"b"}) {
		t.Errorf("unexpected diff: add %v remove %v", toAdd, toRemove)
	}
}

func TestUnitUsersNeedingScheduleUpdate(t *testing.T) {
	makeSchedulable, makeUnschedulable := usersNeedingScheduleUpdate(
		[]string{"a", "b"}, []string{"c"}, // previously schedulable a,b and unschedulable c
		[]string{"a", "c", "d"}, []string{"b"}, // now a and new user d schedulable, c flipped to schedulable, b flipped to unschedulable
	)
	if !reflect.DeepEqual(makeSchedulable, []string{"c", "d"}) {
		t.Errorf("unexpected users to make schedulable: %v", makeSchedulable)
	}
	if !reflect.DeepEqual(makeUnschedulable, []string{"b"}) {
		t.Errorf("unexpected users to make unschedulable: %v", makeUnschedulable)
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
	mutex         sync.Mutex
	members       []string
	movedIn       [][]string
	movedOut      [][]string
	updates       [][]platformclientv2.Updatemuagentrequest
	moveResults   map[string]string
	schedulableBy map[string]*bool
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
		updateAgentsAttr: func(ctx context.Context, p *managementUnitAgentsProxy, id string, agents []platformclientv2.Updatemuagentrequest) (*platformclientv2.APIResponse, error) {
			f.mutex.Lock()
			defer f.mutex.Unlock()
			f.updates = append(f.updates, agents)
			return nil, nil
		},
		getAgentSchedulableAttr: func(ctx context.Context, p *managementUnitAgentsProxy, id string, userId string) (*bool, *platformclientv2.APIResponse, error) {
			if flag, ok := f.schedulableBy[userId]; ok {
				return flag, nil, nil
			}
			return nil, &platformclientv2.APIResponse{StatusCode: http.StatusNotFound}, fmt.Errorf("not found")
		},
	}
}

func TestUnitApplyAgentsMovesAndOnlyUpdatesChangedSchedulableFlags(t *testing.T) {
	api := &fakeApi{members: []string{"a", "b"}}
	proxy := api.proxy("mu-1")

	diagErr := applyAgents(context.Background(), proxy, "mu-1",
		[]string{"a"}, []string{"c"}, // desired: a schedulable, new member c unschedulable, b removed
		[]string{"a", "b"}, nil, // previously: a and b schedulable
	)
	if diagErr != nil {
		t.Fatalf("unexpected error: %v", diagErr)
	}

	if !reflect.DeepEqual(api.movedIn, [][]string{{"c"}}) {
		t.Errorf("expected c moved in, got %v", api.movedIn)
	}
	if !reflect.DeepEqual(api.movedOut, [][]string{{"b"}}) {
		t.Errorf("expected b moved out, got %v", api.movedOut)
	}
	if len(api.updates) != 1 || len(api.updates[0]) != 1 ||
		*api.updates[0][0].UserId != "c" || *api.updates[0][0].Schedulable {
		t.Errorf("expected a single update making c unschedulable, got %+v", api.updates)
	}
}

func TestUnitApplyAgentsSkipsMovesWhenMembersAlreadyMatch(t *testing.T) {
	api := &fakeApi{members: []string{"a", "b"}}

	diagErr := applyAgents(context.Background(), api.proxy("mu-1"), "mu-1", []string{"a", "b"}, nil, []string{"a", "b"}, nil)
	if diagErr != nil {
		t.Fatalf("unexpected error: %v", diagErr)
	}
	if len(api.movedIn) != 0 || len(api.movedOut) != 0 || len(api.updates) != 0 {
		t.Errorf("expected no api writes, got in %v out %v updates %v", api.movedIn, api.movedOut, api.updates)
	}
}

func TestUnitApplyAgentsReportsFailedMoves(t *testing.T) {
	api := &fakeApi{members: nil, moveResults: map[string]string{"b": "DestinationBusinessUnitAgentLimitExceeded"}}

	diagErr := applyAgents(context.Background(), api.proxy("mu-1"), "mu-1", []string{"a", "b"}, nil, nil, nil)
	if diagErr == nil || !diagErr.HasError() {
		t.Fatal("expected an error for the failed move")
	}
	if detail := diagErr[0].Detail + diagErr[0].Summary; !strings.Contains(detail, "DestinationBusinessUnitAgentLimitExceeded") {
		t.Errorf("expected the move result in the error, got %q", detail)
	}
	if len(api.updates) != 0 {
		t.Error("schedulable must not be written after a failed move")
	}
}

func TestUnitApplyAgentsRejectsUserInBothSets(t *testing.T) {
	api := &fakeApi{}
	diagErr := applyAgents(context.Background(), api.proxy("mu-1"), "mu-1", []string{"a"}, []string{"a"}, nil, nil)
	if diagErr == nil || !diagErr.HasError() {
		t.Fatal("expected an error for a user in both sets")
	}
	if len(api.movedIn) != 0 {
		t.Error("nothing may be moved when the configuration is invalid")
	}
}

func TestUnitPartitionBySchedulable(t *testing.T) {
	notSchedulable := false
	schedulable := true
	api := &fakeApi{schedulableBy: map[string]*bool{"a": &schedulable, "b": &notSchedulable, "c": nil}}

	gotSchedulable, gotUnschedulable, err := partitionBySchedulable(context.Background(), api.proxy("mu-1"), "mu-1", []string{"a", "b", "c", "gone"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !reflect.DeepEqual(gotSchedulable, []string{"a"}) {
		t.Errorf("expected only a, got %v", gotSchedulable)
	}
	if !sameMembers(gotUnschedulable, []string{"b", "c"}) {
		t.Errorf("expected b and c (a missing flag counts as unschedulable), got %v", gotUnschedulable)
	}
}

func TestUnitApplyAgentsTimeoutNamesUsersThatNeverJoined(t *testing.T) {
	previousTimeout := membershipTimeout
	membershipTimeout = 2 * time.Second
	defer func() { membershipTimeout = previousTimeout }()

	// The move is accepted but the user never becomes a member, as happens when the move fails permanently after the 202
	api := &fakeApi{members: nil}
	proxy := api.proxy("mu-1")
	proxy.moveAgentsAttr = func(ctx context.Context, p *managementUnitAgentsProxy, userIds []string, destination *string) (*platformclientv2.Moveagentsresponse, *platformclientv2.APIResponse, error) {
		return &platformclientv2.Moveagentsresponse{Results: &[]platformclientv2.Moveagentresponse{}}, nil, nil
	}

	diagErr := applyAgents(context.Background(), proxy, "mu-1", []string{"stuck-user"}, nil, nil, nil)
	if diagErr == nil || !diagErr.HasError() {
		t.Fatal("expected an error when the agent never joins")
	}
	detail := diagErr[0].Summary + diagErr[0].Detail
	if !strings.Contains(detail, "stuck-user") || !strings.Contains(detail, "agent limit") {
		t.Errorf("expected the error to name the user and the likely reasons, got %q", detail)
	}
	if len(api.updates) != 0 {
		t.Error("schedulable must not be written when the agents never joined")
	}
}

func TestUnitApplyAgentsSendsLargeMovesAsOneRequest(t *testing.T) {
	users := make([]string, 250)
	for i := range users {
		users[i] = fmt.Sprintf("user-%03d", i)
	}
	api := &fakeApi{members: nil}

	diagErr := applyAgents(context.Background(), api.proxy("mu-1"), "mu-1", users, nil, nil, nil)
	if diagErr != nil {
		t.Fatalf("unexpected error: %v", diagErr)
	}
	if len(api.movedIn) != 1 || len(api.movedIn[0]) != 250 {
		t.Errorf("expected a single move request with all 250 users, got %d request(s)", len(api.movedIn))
	}
	if len(api.updates) != 3 {
		t.Errorf("expected the 250 schedulable updates in 3 batches of at most 100, got %d", len(api.updates))
	}
}

func TestUnitApplyAgentsSplitsMovesAboveTheMoveLimit(t *testing.T) {
	users := make([]string, 3200)
	for i := range users {
		users[i] = fmt.Sprintf("user-%04d", i)
	}
	api := &fakeApi{members: nil}

	diagErr := applyAgents(context.Background(), api.proxy("mu-1"), "mu-1", users, nil, nil, nil)
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
