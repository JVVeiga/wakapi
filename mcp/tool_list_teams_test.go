package mcp

import (
	"testing"

	mcpgo "github.com/mark3labs/mcp-go/mcp"
	"github.com/muety/wakapi/models"
	"github.com/stretchr/testify/assert"
)

// extractText extracts the text from the first content item in a CallToolResult
func extractText(result *mcpgo.CallToolResult) string {
	if len(result.Content) == 0 {
		return ""
	}
	if tc, ok := result.Content[0].(mcpgo.TextContent); ok {
		return tc.Text
	}
	return ""
}

func TestListTeams_Success(t *testing.T) {
	srv, _, teamSrvc, _, _, _ := newMCPServerWithMocks()

	teams := []*models.Team{
		{ID: "team1", Name: "Backend", OwnerID: "alice"},
		{ID: "team2", Name: "Frontend", OwnerID: "bob"},
	}

	teamSrvc.On("GetByUser", "alice").Return(teams, nil)
	teamSrvc.On("IsTeamOwnerOrCoOwner", "team1", "alice").Return(true, nil)
	teamSrvc.On("IsTeamOwnerOrCoOwner", "team2", "alice").Return(true, nil)
	teamSrvc.On("GetMembers", "team1").Return([]*models.TeamMember{
		{UserID: "alice"}, {UserID: "bob"},
	}, nil)
	teamSrvc.On("GetMembers", "team2").Return([]*models.TeamMember{
		{UserID: "carol"},
	}, nil)

	_, handler := srv.listTeamsTool()
	ctx := ctxWithUser(&models.User{ID: "alice"})

	result, err := handler(ctx, makeRequest(nil))
	assert.Nil(t, err)
	assert.False(t, result.IsError)
	text := extractText(result)
	assert.Contains(t, text, "Backend")
	assert.Contains(t, text, "Frontend")
	assert.Contains(t, text, "alice, bob")
	assert.Contains(t, text, "owner")
}

func TestListTeams_OnlyOwnedTeams(t *testing.T) {
	srv, _, teamSrvc, _, _, _ := newMCPServerWithMocks()

	teams := []*models.Team{
		{ID: "team1", Name: "Backend", OwnerID: "alice"},
		{ID: "team2", Name: "Other", OwnerID: "someone"},
	}

	teamSrvc.On("GetByUser", "alice").Return(teams, nil)
	teamSrvc.On("IsTeamOwnerOrCoOwner", "team1", "alice").Return(true, nil)
	teamSrvc.On("IsTeamOwnerOrCoOwner", "team2", "alice").Return(false, nil)
	teamSrvc.On("GetMembers", "team1").Return([]*models.TeamMember{{UserID: "alice"}}, nil)

	_, handler := srv.listTeamsTool()
	ctx := ctxWithUser(&models.User{ID: "alice"})

	result, err := handler(ctx, makeRequest(nil))
	assert.Nil(t, err)
	text := extractText(result)
	assert.Contains(t, text, "Backend")
	assert.NotContains(t, text, "Other")
}

func TestListTeams_NoTeams(t *testing.T) {
	srv, _, teamSrvc, _, _, _ := newMCPServerWithMocks()
	teamSrvc.On("GetByUser", "alice").Return([]*models.Team{}, nil)

	_, handler := srv.listTeamsTool()
	ctx := ctxWithUser(&models.User{ID: "alice"})

	result, err := handler(ctx, makeRequest(nil))
	assert.Nil(t, err)
	text := extractText(result)
	assert.Contains(t, text, "Nenhum time")
}

func TestListTeams_Unauthenticated(t *testing.T) {
	srv, _, _, _, _, _ := newMCPServerWithMocks()
	_, handler := srv.listTeamsTool()

	result, err := handler(ctxWithUser(nil), makeRequest(nil))
	assert.Nil(t, err)
	assert.True(t, result.IsError)
}
