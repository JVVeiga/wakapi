package mcp

import (
	"testing"
	"time"

	"github.com/muety/wakapi/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestTrendAnalysis_SingleMember(t *testing.T) {
	srv, userSrvc, teamSrvc, summarySrvc, _, _ := newMCPServerWithMocks()

	teamSrvc.On("IsTeamOwnerOrCoOwner", "team1", "alice").Return(true, nil)
	teamSrvc.On("IsTeamMember", "team1", "bob").Return(true, nil)
	userSrvc.On("GetUserById", "bob").Return(&models.User{ID: "bob"}, nil)

	currentSummary := &models.Summary{
		Languages: []*models.SummaryItem{{Key: "Go", Total: time.Duration(7200)}},
	}
	previousSummary := &models.Summary{
		Languages: []*models.SummaryItem{{Key: "Go", Total: time.Duration(3600)}},
	}

	// The handler calls fetchMemberSummary twice (current and previous periods)
	summarySrvc.On("Aliased", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(currentSummary, nil).Once()
	summarySrvc.On("Aliased", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(previousSummary, nil).Once()
	summarySrvc.On("Retrieve", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(currentSummary, nil)

	_, handler := srv.trendAnalysisTool()
	ctx := ctxWithUser(&models.User{ID: "alice"})

	result, err := handler(ctx, makeRequest(map[string]any{
		"team_id":           "team1",
		"user_id":           "bob",
		"current_interval":  "week",
		"previous_interval": "last_week",
	}))

	assert.Nil(t, err)
	assert.False(t, result.IsError)
	text := extractText(result)
	assert.Contains(t, text, "bob")
	assert.Contains(t, text, "Tempo Total")
	assert.Contains(t, text, "Go")
}

func TestTrendAnalysis_TeamWide(t *testing.T) {
	srv, userSrvc, teamSrvc, summarySrvc, _, _ := newMCPServerWithMocks()

	teamSrvc.On("IsTeamOwnerOrCoOwner", "team1", "alice").Return(true, nil)
	teamSrvc.On("GetMembers", "team1").Return([]*models.TeamMember{
		{UserID: "bob"},
	}, nil)
	teamSrvc.On("GetByID", "team1").Return(&models.Team{ID: "team1", Name: "Backend"}, nil)

	userSrvc.On("GetUserById", "bob").Return(&models.User{ID: "bob"}, nil)

	summary := &models.Summary{
		Languages: []*models.SummaryItem{{Key: "Go", Total: time.Duration(3600)}},
	}
	merged := &models.Summary{
		Languages: []*models.SummaryItem{{Key: "Go", Total: time.Duration(3600)}},
	}

	summarySrvc.On("Aliased", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(summary, nil)
	summarySrvc.On("Retrieve", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(summary, nil)
	summarySrvc.On("MergeSummariesAcrossUsers", mock.Anything).Return(merged, nil)

	_, handler := srv.trendAnalysisTool()
	ctx := ctxWithUser(&models.User{ID: "alice"})

	result, err := handler(ctx, makeRequest(map[string]any{
		"team_id": "team1",
	}))

	assert.Nil(t, err)
	assert.False(t, result.IsError)
	text := extractText(result)
	assert.Contains(t, text, "Backend")
}

func TestTrendAnalysis_CustomDates(t *testing.T) {
	srv, userSrvc, teamSrvc, summarySrvc, _, _ := newMCPServerWithMocks()

	teamSrvc.On("IsTeamOwnerOrCoOwner", "team1", "alice").Return(true, nil)
	teamSrvc.On("IsTeamMember", "team1", "bob").Return(true, nil)
	userSrvc.On("GetUserById", "bob").Return(&models.User{ID: "bob"}, nil)

	summary := &models.Summary{
		Languages: []*models.SummaryItem{{Key: "Go", Total: time.Duration(3600)}},
	}
	summarySrvc.On("Aliased", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(summary, nil)
	summarySrvc.On("Retrieve", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(summary, nil)

	_, handler := srv.trendAnalysisTool()
	ctx := ctxWithUser(&models.User{ID: "alice"})

	result, err := handler(ctx, makeRequest(map[string]any{
		"team_id":       "team1",
		"user_id":       "bob",
		"current_from":  "2024-03-20",
		"current_to":    "2024-03-27",
		"previous_from": "2024-03-13",
		"previous_to":   "2024-03-19",
	}))

	assert.Nil(t, err)
	assert.False(t, result.IsError)
}

func TestTrendAnalysis_AccessDenied(t *testing.T) {
	srv, _, teamSrvc, _, _, _ := newMCPServerWithMocks()
	teamSrvc.On("IsTeamOwnerOrCoOwner", "team1", "bob").Return(false, nil)

	_, handler := srv.trendAnalysisTool()
	ctx := ctxWithUser(&models.User{ID: "bob"})

	result, err := handler(ctx, makeRequest(map[string]any{
		"team_id": "team1",
	}))

	assert.Nil(t, err)
	assert.True(t, result.IsError)
}

func TestResolveTrendInterval_WithInterval(t *testing.T) {
	from, to, errResult := resolveTrendInterval("week", "", "", time.UTC)
	assert.Nil(t, errResult)
	assert.False(t, from.IsZero())
	assert.True(t, to.After(from))
}

func TestResolveTrendInterval_WithDates(t *testing.T) {
	from, to, errResult := resolveTrendInterval("", "2024-03-20", "2024-03-27", time.UTC)
	assert.Nil(t, errResult)
	assert.Equal(t, 20, from.Day())
	assert.True(t, to.After(from))
}

func TestResolveTrendInterval_InvalidInterval(t *testing.T) {
	_, _, errResult := resolveTrendInterval("invalid", "", "", time.UTC)
	assert.NotNil(t, errResult)
	assert.True(t, errResult.IsError)
}

func TestResolveTrendInterval_NoInput(t *testing.T) {
	_, _, errResult := resolveTrendInterval("", "", "", time.UTC)
	assert.NotNil(t, errResult)
	assert.True(t, errResult.IsError)
}
