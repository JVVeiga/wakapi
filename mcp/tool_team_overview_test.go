package mcp

import (
	"testing"
	"time"

	"github.com/muety/wakapi/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestTeamOverview_Success(t *testing.T) {
	srv, userSrvc, teamSrvc, summarySrvc, _, _ := newMCPServerWithMocks()

	teamSrvc.On("IsTeamOwnerOrCoOwner", "team1", "alice").Return(true, nil)
	teamSrvc.On("GetMembers", "team1").Return([]*models.TeamMember{
		{UserID: "alice"}, {UserID: "bob"},
	}, nil)
	teamSrvc.On("GetByID", "team1").Return(&models.Team{ID: "team1", Name: "Backend"}, nil)

	userSrvc.On("GetUserById", "alice").Return(&models.User{ID: "alice"}, nil)
	userSrvc.On("GetUserById", "bob").Return(&models.User{ID: "bob"}, nil)

	summaryAlice := &models.Summary{
		Projects:  []*models.SummaryItem{{Key: "wakapi", Total: time.Duration(7200)}},
		Languages: []*models.SummaryItem{{Key: "Go", Total: time.Duration(7200)}},
	}
	summaryBob := &models.Summary{
		Projects:  []*models.SummaryItem{{Key: "api", Total: time.Duration(3600)}},
		Languages: []*models.SummaryItem{{Key: "Python", Total: time.Duration(3600)}},
	}
	merged := &models.Summary{
		Projects:  []*models.SummaryItem{{Key: "wakapi", Total: time.Duration(7200)}, {Key: "api", Total: time.Duration(3600)}},
		Languages: []*models.SummaryItem{{Key: "Go", Total: time.Duration(7200)}, {Key: "Python", Total: time.Duration(3600)}},
	}

	summarySrvc.On("Aliased", mock.Anything, mock.Anything, mock.MatchedBy(func(u *models.User) bool { return u.ID == "alice" }), mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(summaryAlice, nil)
	summarySrvc.On("Aliased", mock.Anything, mock.Anything, mock.MatchedBy(func(u *models.User) bool { return u.ID == "bob" }), mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(summaryBob, nil)
	summarySrvc.On("Retrieve", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(summaryAlice, nil)
	summarySrvc.On("MergeSummariesAcrossUsers", mock.Anything).Return(merged, nil)

	_, handler := srv.teamOverviewTool()
	ctx := ctxWithUser(&models.User{ID: "alice"})

	result, err := handler(ctx, makeRequest(map[string]any{
		"team_id": "team1",
	}))

	assert.Nil(t, err)
	assert.False(t, result.IsError)
	text := extractText(result)
	assert.Contains(t, text, "Backend")
	assert.Contains(t, text, "alice")
	assert.Contains(t, text, "bob")
	assert.Contains(t, text, "Ranking")
}

func TestTeamOverview_AccessDenied(t *testing.T) {
	srv, _, teamSrvc, _, _, _ := newMCPServerWithMocks()
	teamSrvc.On("IsTeamOwnerOrCoOwner", "team1", "bob").Return(false, nil)

	_, handler := srv.teamOverviewTool()
	ctx := ctxWithUser(&models.User{ID: "bob"})

	result, err := handler(ctx, makeRequest(map[string]any{
		"team_id": "team1",
	}))

	assert.Nil(t, err)
	assert.True(t, result.IsError)
}

func TestTeamOverview_AIRatioShownWhenInstrumented(t *testing.T) {
	srv, userSrvc, teamSrvc, summarySrvc, _, _ := newMCPServerWithMocks()

	teamSrvc.On("IsTeamOwnerOrCoOwner", "team1", "alice").Return(true, nil)
	teamSrvc.On("GetMembers", "team1").Return([]*models.TeamMember{
		{UserID: "alice"}, {UserID: "bob"},
	}, nil)
	teamSrvc.On("GetByID", "team1").Return(&models.Team{ID: "team1", Name: "Backend"}, nil)

	userSrvc.On("GetUserById", "alice").Return(&models.User{ID: "alice"}, nil)
	userSrvc.On("GetUserById", "bob").Return(&models.User{ID: "bob"}, nil)

	// alice has AI telemetry, bob does not
	summaryAlice := &models.Summary{
		Projects: []*models.SummaryItem{{Key: "wakapi", Total: time.Duration(7200)}},
		Categories: []*models.SummaryItem{
			{Type: models.SummaryCategory, Key: "coding", Total: time.Duration(5400)},
			{Type: models.SummaryCategory, Key: "ai coding", Total: time.Duration(1800)},
		},
	}
	summaryBob := &models.Summary{
		Projects: []*models.SummaryItem{{Key: "api", Total: time.Duration(3600)}},
		Categories: []*models.SummaryItem{
			{Type: models.SummaryCategory, Key: "coding", Total: time.Duration(3600)},
		},
	}
	merged := &models.Summary{
		Projects: []*models.SummaryItem{{Key: "wakapi", Total: time.Duration(7200)}, {Key: "api", Total: time.Duration(3600)}},
		Categories: []*models.SummaryItem{
			{Type: models.SummaryCategory, Key: "coding", Total: time.Duration(9000)},
			{Type: models.SummaryCategory, Key: "ai coding", Total: time.Duration(1800)},
		},
	}

	summarySrvc.On("Aliased", mock.Anything, mock.Anything, mock.MatchedBy(func(u *models.User) bool { return u.ID == "alice" }), mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(summaryAlice, nil)
	summarySrvc.On("Aliased", mock.Anything, mock.Anything, mock.MatchedBy(func(u *models.User) bool { return u.ID == "bob" }), mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(summaryBob, nil)
	summarySrvc.On("Retrieve", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(summaryAlice, nil)
	summarySrvc.On("MergeSummariesAcrossUsers", mock.Anything).Return(merged, nil)

	_, handler := srv.teamOverviewTool()
	result, err := handler(ctxWithUser(&models.User{ID: "alice"}), makeRequest(map[string]any{
		"team_id": "team1",
	}))

	assert.Nil(t, err)
	assert.False(t, result.IsError)

	text := extractText(result)
	assert.Contains(t, text, "Coding com IA")
	assert.Contains(t, text, "16.7%")    // team: 1800s de 10800s
	assert.Contains(t, text, "IA 25.0%") // alice: 1800s de 7200s
	assert.NotContains(t, text, "IA 0.0%", "bob não tem telemetria, então não ganha coluna de IA")
}

func TestTeamOverview_NoAISectionWhenNotInstrumented(t *testing.T) {
	srv, userSrvc, teamSrvc, summarySrvc, _, _ := newMCPServerWithMocks()

	teamSrvc.On("IsTeamOwnerOrCoOwner", "team1", "alice").Return(true, nil)
	teamSrvc.On("GetMembers", "team1").Return([]*models.TeamMember{{UserID: "alice"}}, nil)
	teamSrvc.On("GetByID", "team1").Return(&models.Team{ID: "team1", Name: "Backend"}, nil)
	userSrvc.On("GetUserById", "alice").Return(&models.User{ID: "alice"}, nil)

	summary := &models.Summary{
		Projects: []*models.SummaryItem{{Key: "wakapi", Total: time.Duration(7200)}},
		Categories: []*models.SummaryItem{
			{Type: models.SummaryCategory, Key: "coding", Total: time.Duration(7200)},
		},
	}
	summarySrvc.On("Aliased", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(summary, nil)
	summarySrvc.On("Retrieve", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(summary, nil)
	summarySrvc.On("MergeSummariesAcrossUsers", mock.Anything).Return(summary, nil)

	_, handler := srv.teamOverviewTool()
	result, _ := handler(ctxWithUser(&models.User{ID: "alice"}), makeRequest(map[string]any{
		"team_id": "team1",
	}))

	text := extractText(result)
	assert.NotContains(t, text, "Coding com IA")
	assert.NotContains(t, text, "IA ")
	assert.Contains(t, text, "Ranking", "o overview normal segue funcionando")
}
