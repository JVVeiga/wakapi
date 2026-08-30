package mcp

import (
	"context"
	"strings"
	"testing"

	"github.com/muety/wakapi/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func aiAdoptionMocks(t *testing.T, aliceCats, bobCats *models.Summary) *MCPServer {
	t.Helper()

	srv, userSrvc, teamSrvc, summarySrvc, _, _ := newMCPServerWithMocks()

	teamSrvc.On("IsTeamOwnerOrCoOwner", "team1", "alice").Return(true, nil)
	teamSrvc.On("GetMembers", "team1").Return([]*models.TeamMember{
		{UserID: "alice"}, {UserID: "bob"},
	}, nil)
	teamSrvc.On("GetByID", "team1").Return(&models.Team{ID: "team1", Name: "Backend"}, nil)

	userSrvc.On("GetUserById", "alice").Return(&models.User{ID: "alice"}, nil)
	userSrvc.On("GetUserById", "bob").Return(&models.User{ID: "bob"}, nil)

	summarySrvc.On("Aliased", mock.Anything, mock.Anything, mock.MatchedBy(func(u *models.User) bool { return u.ID == "alice" }), mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(aliceCats, nil)
	summarySrvc.On("Aliased", mock.Anything, mock.Anything, mock.MatchedBy(func(u *models.User) bool { return u.ID == "bob" }), mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(bobCats, nil)

	return srv
}

func TestAIAdoption_Success(t *testing.T) {
	srv := aiAdoptionMocks(t,
		categorySummary("coding", 3600, "ai coding", 3600), // alice: 50%
		categorySummary("coding", 7200, "ai coding", 1800), // bob: 20%
	)

	_, handler := srv.aiAdoptionTool()
	ctx := ctxWithUser(&models.User{ID: "alice"})

	result, err := handler(ctx, makeRequest(map[string]any{
		"team_id": "team1",
		"compare": false,
	}))

	assert.Nil(t, err)
	assert.False(t, result.IsError)

	text := extractText(result)
	assert.Contains(t, text, "Backend")
	assert.Contains(t, text, "alice")
	assert.Contains(t, text, "bob")
	assert.Contains(t, text, "50.0%")
	assert.Contains(t, text, "20.0%")
	// team: 5400s AI over 16200s coding
	assert.Contains(t, text, "33.3%")
	assert.NotContains(t, text, "Δ IA")
}

func TestAIAdoption_RankedByRatio(t *testing.T) {
	srv := aiAdoptionMocks(t,
		categorySummary("coding", 7200, "ai coding", 1800), // alice: 20%
		categorySummary("coding", 3600, "ai coding", 3600), // bob: 50%
	)

	_, handler := srv.aiAdoptionTool()
	result, _ := handler(ctxWithUser(&models.User{ID: "alice"}), makeRequest(map[string]any{
		"team_id": "team1",
		"compare": false,
	}))

	text := extractText(result)
	assert.Less(t, strings.Index(text, "bob"), strings.Index(text, "alice"), "maior ratio de IA vem primeiro")
}

func TestAIAdoption_NotInstrumented(t *testing.T) {
	srv := aiAdoptionMocks(t,
		categorySummary("coding", 3600, "browsing", 600),
		categorySummary("coding", 7200),
	)

	_, handler := srv.aiAdoptionTool()
	result, err := handler(ctxWithUser(&models.User{ID: "alice"}), makeRequest(map[string]any{
		"team_id": "team1",
	}))

	assert.Nil(t, err)
	assert.False(t, result.IsError)

	text := extractText(result)
	assert.Contains(t, text, "ausência de telemetria")
	assert.NotContains(t, text, "0.0%", "sem telemetria não se reporta 0%")
}

func TestAIAdoption_PartiallyInstrumented(t *testing.T) {
	srv := aiAdoptionMocks(t,
		categorySummary("coding", 3600, "ai coding", 900), // alice instrumentada
		categorySummary("coding", 7200),                   // bob sem telemetria
	)

	_, handler := srv.aiAdoptionTool()
	result, _ := handler(ctxWithUser(&models.User{ID: "alice"}), makeRequest(map[string]any{
		"team_id": "team1",
		"compare": false,
	}))

	text := extractText(result)
	assert.Contains(t, text, "Sem telemetria de IA no período (1)")
	assert.Contains(t, text, "bob")
	assert.Contains(t, text, "20.0%", "alice, instrumentada, mantém o percentual")

	// bob has no AI telemetry: his row must carry no percentage at all, since any
	// number there would assert something the data does not support
	bobRow := lineStartingWith(text, "bob")
	assert.NotEmpty(t, bobRow, "linha do bob não encontrada em:\n"+text)
	assert.Contains(t, bobRow, "--")
	assert.NotContains(t, bobRow, "%")
}

func TestAIAdoption_WithCompare(t *testing.T) {
	srv := aiAdoptionMocks(t,
		categorySummary("coding", 3600, "ai coding", 3600),
		categorySummary("coding", 7200, "ai coding", 1800),
	)

	_, handler := srv.aiAdoptionTool()
	result, _ := handler(ctxWithUser(&models.User{ID: "alice"}), makeRequest(map[string]any{
		"team_id": "team1",
		"compare": true,
	}))

	text := extractText(result)
	assert.Contains(t, text, "Período anterior")
	assert.Contains(t, text, "Δ IA")
}

func TestAIAdoption_AccessDenied(t *testing.T) {
	srv, _, teamSrvc, _, _, _ := newMCPServerWithMocks()
	teamSrvc.On("IsTeamOwnerOrCoOwner", "team1", "mallory").Return(false, nil)

	_, handler := srv.aiAdoptionTool()
	result, err := handler(ctxWithUser(&models.User{ID: "mallory"}), makeRequest(map[string]any{
		"team_id": "team1",
	}))

	assert.Nil(t, err)
	assert.True(t, result.IsError)
}

func TestAIAdoption_Unauthenticated(t *testing.T) {
	srv, _, _, _, _, _ := newMCPServerWithMocks()

	_, handler := srv.aiAdoptionTool()
	result, err := handler(context.Background(), makeRequest(map[string]any{"team_id": "team1"}))

	assert.Nil(t, err)
	assert.True(t, result.IsError)
}

func TestAIAdoption_InvalidInterval(t *testing.T) {
	srv, _, teamSrvc, _, _, _ := newMCPServerWithMocks()
	teamSrvc.On("IsTeamOwnerOrCoOwner", "team1", "alice").Return(true, nil)

	_, handler := srv.aiAdoptionTool()
	result, err := handler(ctxWithUser(&models.User{ID: "alice"}), makeRequest(map[string]any{
		"team_id":  "team1",
		"interval": "nao_existe",
	}))

	assert.Nil(t, err)
	assert.True(t, result.IsError)
}
