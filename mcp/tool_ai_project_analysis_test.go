package mcp

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/muety/wakapi/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func aiProjectMocks(t *testing.T, aliceDurations, bobDurations models.Durations) *MCPServer {
	t.Helper()

	srv, userSrvc, teamSrvc, _, _, durationSrvc := newMCPServerWithMocks()

	teamSrvc.On("IsTeamOwnerOrCoOwner", "team1", "alice").Return(true, nil)
	teamSrvc.On("GetMembers", "team1").Return([]*models.TeamMember{
		{UserID: "alice"}, {UserID: "bob"},
	}, nil)
	teamSrvc.On("GetByID", "team1").Return(&models.Team{ID: "team1", Name: "Backend"}, nil)

	userSrvc.On("GetUserById", "alice").Return(&models.User{ID: "alice"}, nil)
	userSrvc.On("GetUserById", "bob").Return(&models.User{ID: "bob"}, nil)

	durationSrvc.On("Get", mock.Anything, mock.Anything, mock.MatchedBy(func(u *models.User) bool { return u.ID == "alice" }), mock.Anything, mock.Anything, mock.Anything).Return(aliceDurations, nil)
	durationSrvc.On("Get", mock.Anything, mock.Anything, mock.MatchedBy(func(u *models.User) bool { return u.ID == "bob" }), mock.Anything, mock.Anything, mock.Anything).Return(bobDurations, nil)

	return srv
}

func TestAIProjectAnalysis_Overview(t *testing.T) {
	srv := aiProjectMocks(t,
		models.Durations{
			{Project: "wakapi", Category: "coding", Duration: time.Hour},
			{Project: "wakapi", Category: "ai coding", Duration: time.Hour},
			{Project: "api", Category: "coding", Duration: 30 * time.Minute},
		},
		models.Durations{
			{Project: "wakapi", Category: "coding", Duration: 2 * time.Hour},
		},
	)

	_, handler := srv.aiProjectAnalysisTool()
	result, err := handler(ctxWithUser(&models.User{ID: "alice"}), makeRequest(map[string]any{
		"team_id": "team1",
	}))

	assert.Nil(t, err)
	assert.False(t, result.IsError)

	text := extractText(result)
	assert.Contains(t, text, "wakapi")
	assert.Contains(t, text, "api")
	// wakapi: 1h de IA sobre 4h de coding
	assert.Contains(t, text, "25.0%")
	assert.Less(t, strings.Index(text, "wakapi"), strings.Index(text, "\napi"), "projeto com mais coding vem primeiro")
}

func TestAIProjectAnalysis_Detail(t *testing.T) {
	srv := aiProjectMocks(t,
		models.Durations{
			{Project: "wakapi", Category: "coding", Duration: time.Hour},
			{Project: "wakapi", Category: "ai coding", Duration: time.Hour},
		},
		models.Durations{
			{Project: "wakapi", Category: "coding", Duration: 2 * time.Hour},
			{Project: "wakapi", Category: "ai coding", Duration: 30 * time.Minute},
		},
	)

	_, handler := srv.aiProjectAnalysisTool()
	result, err := handler(ctxWithUser(&models.User{ID: "alice"}), makeRequest(map[string]any{
		"team_id": "team1",
		"project": "wakapi",
	}))

	assert.Nil(t, err)
	assert.False(t, result.IsError)

	text := extractText(result)
	assert.Contains(t, text, "alice")
	assert.Contains(t, text, "bob")
	assert.Contains(t, text, "50.0%") // alice
	assert.Contains(t, text, "20.0%") // bob
	assert.Less(t, strings.Index(text, "alice"), strings.Index(text, "bob"), "maior ratio primeiro")
}

func TestAIProjectAnalysis_DetailCaseInsensitive(t *testing.T) {
	srv := aiProjectMocks(t,
		models.Durations{
			{Project: "Wakapi", Category: "coding", Duration: time.Hour},
			{Project: "Wakapi", Category: "ai coding", Duration: time.Hour},
		},
		models.Durations{},
	)

	_, handler := srv.aiProjectAnalysisTool()
	result, _ := handler(ctxWithUser(&models.User{ID: "alice"}), makeRequest(map[string]any{
		"team_id": "team1",
		"project": "wakapi",
	}))

	assert.Contains(t, extractText(result), "alice")
}

func TestAIProjectAnalysis_DetailUnknownProject(t *testing.T) {
	srv := aiProjectMocks(t,
		models.Durations{
			{Project: "wakapi", Category: "ai coding", Duration: time.Hour},
		},
		models.Durations{},
	)

	_, handler := srv.aiProjectAnalysisTool()
	result, _ := handler(ctxWithUser(&models.User{ID: "alice"}), makeRequest(map[string]any{
		"team_id": "team1",
		"project": "inexistente",
	}))

	assert.Contains(t, extractText(result), "Nenhum contribuidor")
}

func TestAIProjectAnalysis_NotInstrumented(t *testing.T) {
	srv := aiProjectMocks(t,
		models.Durations{
			{Project: "wakapi", Category: "coding", Duration: time.Hour},
		},
		models.Durations{
			{Project: "api", Category: "browsing", Duration: time.Hour},
		},
	)

	_, handler := srv.aiProjectAnalysisTool()
	result, err := handler(ctxWithUser(&models.User{ID: "alice"}), makeRequest(map[string]any{
		"team_id": "team1",
	}))

	assert.Nil(t, err)
	assert.False(t, result.IsError)

	text := extractText(result)
	assert.Contains(t, text, "Ausência de telemetria")
	assert.NotContains(t, text, "0.0%")
}

func TestAIProjectAnalysis_AccessDenied(t *testing.T) {
	srv, _, teamSrvc, _, _, _ := newMCPServerWithMocks()
	teamSrvc.On("IsTeamOwnerOrCoOwner", "team1", "mallory").Return(false, nil)

	_, handler := srv.aiProjectAnalysisTool()
	result, err := handler(ctxWithUser(&models.User{ID: "mallory"}), makeRequest(map[string]any{
		"team_id": "team1",
	}))

	assert.Nil(t, err)
	assert.True(t, result.IsError)
}

func TestAIProjectAnalysis_Unauthenticated(t *testing.T) {
	srv, _, _, _, _, _ := newMCPServerWithMocks()

	_, handler := srv.aiProjectAnalysisTool()
	result, err := handler(context.Background(), makeRequest(map[string]any{"team_id": "team1"}))

	assert.Nil(t, err)
	assert.True(t, result.IsError)
}

func TestLookupProject_CaseInsensitive(t *testing.T) {
	byProject := map[string]AIStats{"Wakapi": {CodingTime: time.Hour}}

	st, ok := lookupProject(byProject, "Wakapi")
	assert.True(t, ok)
	assert.Equal(t, time.Hour, st.CodingTime)

	st, ok = lookupProject(byProject, "wakapi")
	assert.True(t, ok)
	assert.Equal(t, time.Hour, st.CodingTime)

	_, ok = lookupProject(byProject, "outro")
	assert.False(t, ok)
}
