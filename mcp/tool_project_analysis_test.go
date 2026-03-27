package mcp

import (
	"testing"
	"time"

	"github.com/muety/wakapi/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestProjectAnalysis_WithProject(t *testing.T) {
	srv, userSrvc, teamSrvc, summarySrvc, _, _ := newMCPServerWithMocks()

	teamSrvc.On("IsTeamOwnerOrCoOwner", "team1", "alice").Return(true, nil)
	teamSrvc.On("GetMembers", "team1").Return([]*models.TeamMember{
		{UserID: "bob"}, {UserID: "carol"},
	}, nil)
	teamSrvc.On("GetByID", "team1").Return(&models.Team{ID: "team1", Name: "Backend"}, nil)

	userSrvc.On("GetUserById", "bob").Return(&models.User{ID: "bob"}, nil)
	userSrvc.On("GetUserById", "carol").Return(&models.User{ID: "carol"}, nil)

	summaryBob := &models.Summary{
		Projects:  []*models.SummaryItem{{Key: "wakapi", Total: time.Duration(3600)}},
		Languages: []*models.SummaryItem{{Key: "Go", Total: time.Duration(3600)}},
	}
	summaryCarol := &models.Summary{
		Projects:  []*models.SummaryItem{{Key: "wakapi", Total: time.Duration(1800)}},
		Languages: []*models.SummaryItem{{Key: "Go", Total: time.Duration(1800)}},
	}

	summarySrvc.On("Aliased", mock.Anything, mock.Anything, mock.MatchedBy(func(u *models.User) bool { return u.ID == "bob" }), mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(summaryBob, nil)
	summarySrvc.On("Aliased", mock.Anything, mock.Anything, mock.MatchedBy(func(u *models.User) bool { return u.ID == "carol" }), mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(summaryCarol, nil)
	summarySrvc.On("Retrieve", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(summaryBob, nil)

	_, handler := srv.projectAnalysisTool()
	ctx := ctxWithUser(&models.User{ID: "alice"})

	result, err := handler(ctx, makeRequest(map[string]any{
		"team_id": "team1",
		"project": "wakapi",
	}))

	assert.Nil(t, err)
	assert.False(t, result.IsError)
	text := extractText(result)
	assert.Contains(t, text, "wakapi")
	assert.Contains(t, text, "bob")
	assert.Contains(t, text, "carol")
	assert.Contains(t, text, "Contribuidores")
}

func TestProjectAnalysis_Overview(t *testing.T) {
	srv, userSrvc, teamSrvc, summarySrvc, _, _ := newMCPServerWithMocks()

	teamSrvc.On("IsTeamOwnerOrCoOwner", "team1", "alice").Return(true, nil)
	teamSrvc.On("GetMembers", "team1").Return([]*models.TeamMember{
		{UserID: "bob"},
	}, nil)
	teamSrvc.On("GetByID", "team1").Return(&models.Team{ID: "team1", Name: "Backend"}, nil)

	userSrvc.On("GetUserById", "bob").Return(&models.User{ID: "bob"}, nil)

	summary := &models.Summary{
		Projects:  []*models.SummaryItem{{Key: "wakapi", Total: time.Duration(3600)}, {Key: "api", Total: time.Duration(1800)}},
		Languages: []*models.SummaryItem{{Key: "Go", Total: time.Duration(5400)}},
	}

	summarySrvc.On("Aliased", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(summary, nil)
	summarySrvc.On("Retrieve", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(summary, nil)

	_, handler := srv.projectAnalysisTool()
	ctx := ctxWithUser(&models.User{ID: "alice"})

	result, err := handler(ctx, makeRequest(map[string]any{
		"team_id": "team1",
	}))

	assert.Nil(t, err)
	assert.False(t, result.IsError)
	text := extractText(result)
	assert.Contains(t, text, "wakapi")
	assert.Contains(t, text, "api")
	assert.Contains(t, text, "bob")
}

func TestProjectAnalysis_AccessDenied(t *testing.T) {
	srv, _, teamSrvc, _, _, _ := newMCPServerWithMocks()
	teamSrvc.On("IsTeamOwnerOrCoOwner", "team1", "bob").Return(false, nil)

	_, handler := srv.projectAnalysisTool()
	ctx := ctxWithUser(&models.User{ID: "bob"})

	result, err := handler(ctx, makeRequest(map[string]any{
		"team_id": "team1",
	}))

	assert.Nil(t, err)
	assert.True(t, result.IsError)
}
