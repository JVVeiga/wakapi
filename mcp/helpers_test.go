package mcp

import (
	"context"
	"fmt"
	"testing"
	"time"

	mcpgo "github.com/mark3labs/mcp-go/mcp"
	"github.com/muety/wakapi/mocks"
	"github.com/muety/wakapi/models"
	"github.com/stretchr/testify/assert"
)

func newMCPServerWithMocks() (*MCPServer, *mocks.UserServiceMock, *mocks.TeamServiceMock, *mocks.SummaryServiceMock, *mocks.HeartbeatServiceMock, *mocks.DurationServiceMock) {
	userSrvc := new(mocks.UserServiceMock)
	teamSrvc := new(mocks.TeamServiceMock)
	summarySrvc := new(mocks.SummaryServiceMock)
	heartbeatSrvc := new(mocks.HeartbeatServiceMock)
	durationSrvc := new(mocks.DurationServiceMock)

	srv := &MCPServer{
		userSrvc:      userSrvc,
		teamSrvc:      teamSrvc,
		summarySrvc:   summarySrvc,
		heartbeatSrvc: heartbeatSrvc,
		durationSrvc:  durationSrvc,
	}
	return srv, userSrvc, teamSrvc, summarySrvc, heartbeatSrvc, durationSrvc
}

func ctxWithUser(user *models.User) context.Context {
	return context.WithValue(context.Background(), principalKey, user)
}

func makeRequest(args map[string]any) mcpgo.CallToolRequest {
	return mcpgo.CallToolRequest{
		Params: mcpgo.CallToolParams{
			Arguments: args,
		},
	}
}

func TestCheckTeamAccess_OwnerAllowed(t *testing.T) {
	srv, _, teamSrvc, _, _, _ := newMCPServerWithMocks()
	teamSrvc.On("IsTeamOwnerOrCoOwner", "team1", "alice").Return(true, nil)

	ctx := ctxWithUser(&models.User{ID: "alice"})
	user, errResult := srv.checkTeamAccess(ctx, "team1")
	assert.Nil(t, errResult)
	assert.Equal(t, "alice", user.ID)
}

func TestCheckTeamAccess_MemberDenied(t *testing.T) {
	srv, _, teamSrvc, _, _, _ := newMCPServerWithMocks()
	teamSrvc.On("IsTeamOwnerOrCoOwner", "team1", "bob").Return(false, nil)

	ctx := ctxWithUser(&models.User{ID: "bob"})
	user, errResult := srv.checkTeamAccess(ctx, "team1")
	assert.Nil(t, user)
	assert.NotNil(t, errResult)
	assert.True(t, errResult.IsError)
}

func TestCheckTeamAccess_Unauthenticated(t *testing.T) {
	srv, _, _, _, _, _ := newMCPServerWithMocks()
	user, errResult := srv.checkTeamAccess(context.Background(), "team1")
	assert.Nil(t, user)
	assert.NotNil(t, errResult)
	assert.True(t, errResult.IsError)
}

func TestCheckMemberAccess_Valid(t *testing.T) {
	srv, userSrvc, teamSrvc, _, _, _ := newMCPServerWithMocks()
	teamSrvc.On("IsTeamOwnerOrCoOwner", "team1", "alice").Return(true, nil)
	teamSrvc.On("IsTeamMember", "team1", "bob").Return(true, nil)
	userSrvc.On("GetUserById", "bob").Return(&models.User{ID: "bob"}, nil)

	ctx := ctxWithUser(&models.User{ID: "alice"})
	requester, target, errResult := srv.checkMemberAccess(ctx, "team1", "bob")
	assert.Nil(t, errResult)
	assert.Equal(t, "alice", requester.ID)
	assert.Equal(t, "bob", target.ID)
}

func TestCheckMemberAccess_NotMember(t *testing.T) {
	srv, _, teamSrvc, _, _, _ := newMCPServerWithMocks()
	teamSrvc.On("IsTeamOwnerOrCoOwner", "team1", "alice").Return(true, nil)
	teamSrvc.On("IsTeamMember", "team1", "stranger").Return(false, nil)

	ctx := ctxWithUser(&models.User{ID: "alice"})
	_, _, errResult := srv.checkMemberAccess(ctx, "team1", "stranger")
	assert.NotNil(t, errResult)
	assert.True(t, errResult.IsError)
}

func TestCheckMemberAccess_UserNotFound(t *testing.T) {
	srv, userSrvc, teamSrvc, _, _, _ := newMCPServerWithMocks()
	teamSrvc.On("IsTeamOwnerOrCoOwner", "team1", "alice").Return(true, nil)
	teamSrvc.On("IsTeamMember", "team1", "ghost").Return(true, nil)
	userSrvc.On("GetUserById", "ghost").Return(nil, fmt.Errorf("not found"))

	ctx := ctxWithUser(&models.User{ID: "alice"})
	_, _, errResult := srv.checkMemberAccess(ctx, "team1", "ghost")
	assert.NotNil(t, errResult)
	assert.True(t, errResult.IsError)
}

func TestResolveInterval_WithInterval(t *testing.T) {
	tz := time.UTC
	req := makeRequest(map[string]any{"interval": "today"})
	from, to, errResult := resolveInterval(req, tz)
	assert.Nil(t, errResult)
	assert.False(t, from.IsZero())
	assert.False(t, to.IsZero())
	assert.True(t, to.After(from))
}

func TestResolveInterval_WithDates(t *testing.T) {
	tz := time.UTC
	req := makeRequest(map[string]any{"from": "2024-03-20", "to": "2024-03-27"})
	from, to, errResult := resolveInterval(req, tz)
	assert.Nil(t, errResult)
	assert.Equal(t, 2024, from.Year())
	assert.Equal(t, time.March, from.Month())
	assert.Equal(t, 20, from.Day())
	assert.True(t, to.After(from))
}

func TestResolveInterval_InvalidInterval(t *testing.T) {
	tz := time.UTC
	req := makeRequest(map[string]any{"interval": "invalid_interval"})
	_, _, errResult := resolveInterval(req, tz)
	assert.NotNil(t, errResult)
	assert.True(t, errResult.IsError)
}

func TestResolveInterval_InvalidDate(t *testing.T) {
	tz := time.UTC
	req := makeRequest(map[string]any{"from": "not-a-date", "to": "2024-03-27"})
	_, _, errResult := resolveInterval(req, tz)
	assert.NotNil(t, errResult)
	assert.True(t, errResult.IsError)
}

func TestResolveInterval_Default(t *testing.T) {
	tz := time.UTC
	req := makeRequest(map[string]any{})
	from, to, errResult := resolveInterval(req, tz)
	assert.Nil(t, errResult)
	// Should default to last 7 days
	assert.InDelta(t, 7, to.Sub(from).Hours()/24, 1)
}

func TestResolveInterval_ExceedsMaxRange(t *testing.T) {
	tz := time.UTC
	req := makeRequest(map[string]any{"from": "2020-01-01", "to": "2026-01-01"})
	_, _, errResult := resolveInterval(req, tz)
	assert.NotNil(t, errResult)
	assert.True(t, errResult.IsError)
}

func TestResolveInterval_ToBeforeFrom(t *testing.T) {
	tz := time.UTC
	req := makeRequest(map[string]any{"from": "2024-03-27", "to": "2024-03-20"})
	_, _, errResult := resolveInterval(req, tz)
	assert.NotNil(t, errResult)
	assert.True(t, errResult.IsError)
}

func TestSanitizeInput(t *testing.T) {
	assert.Equal(t, "hello", sanitizeInput("hello"))
	assert.Equal(t, 255, len(sanitizeInput(string(make([]byte, 1000)))))
}

func TestCheckTeamAccess_EmptyTeamID(t *testing.T) {
	srv, _, _, _, _, _ := newMCPServerWithMocks()
	ctx := ctxWithUser(&models.User{ID: "alice"})
	_, errResult := srv.checkTeamAccess(ctx, "")
	assert.NotNil(t, errResult)
	assert.True(t, errResult.IsError)
}

func TestCheckMemberAccess_EmptyUserID(t *testing.T) {
	srv, _, teamSrvc, _, _, _ := newMCPServerWithMocks()
	teamSrvc.On("IsTeamOwnerOrCoOwner", "team1", "alice").Return(true, nil)
	ctx := ctxWithUser(&models.User{ID: "alice"})
	_, _, errResult := srv.checkMemberAccess(ctx, "team1", "")
	assert.NotNil(t, errResult)
	assert.True(t, errResult.IsError)
}

func TestLimitMembers(t *testing.T) {
	members := make([]*models.TeamMember, 100)
	for i := range members {
		members[i] = &models.TeamMember{UserID: fmt.Sprintf("user%d", i)}
	}
	limited := limitMembers(members)
	assert.Equal(t, maxMembersPerQuery, len(limited))

	small := []*models.TeamMember{{UserID: "a"}, {UserID: "b"}}
	assert.Equal(t, 2, len(limitMembers(small)))
}
