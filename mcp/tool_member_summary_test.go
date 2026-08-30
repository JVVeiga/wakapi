package mcp

import (
	"testing"
	"time"

	"github.com/muety/wakapi/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestMemberSummary_Success(t *testing.T) {
	srv, userSrvc, teamSrvc, summarySrvc, _, _ := newMCPServerWithMocks()

	teamSrvc.On("IsTeamOwnerOrCoOwner", "team1", "alice").Return(true, nil)
	teamSrvc.On("IsTeamMember", "team1", "bob").Return(true, nil)
	userSrvc.On("GetUserById", "bob").Return(&models.User{ID: "bob"}, nil)

	summary := &models.Summary{
		Projects:  []*models.SummaryItem{{Key: "wakapi", Total: time.Duration(7200)}},
		Languages: []*models.SummaryItem{{Key: "Go", Total: time.Duration(7200)}},
	}
	summarySrvc.On("Aliased", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(summary, nil)
	summarySrvc.On("Retrieve", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(summary, nil)

	_, handler := srv.memberSummaryTool()
	ctx := ctxWithUser(&models.User{ID: "alice"})

	result, err := handler(ctx, makeRequest(map[string]any{
		"team_id": "team1",
		"user_id": "bob",
	}))

	assert.Nil(t, err)
	assert.False(t, result.IsError)
	text := extractText(result)
	assert.Contains(t, text, "bob")
	assert.Contains(t, text, "wakapi")
	assert.Contains(t, text, "Go")
}

func TestMemberSummary_NotOwner(t *testing.T) {
	srv, _, teamSrvc, _, _, _ := newMCPServerWithMocks()
	teamSrvc.On("IsTeamOwnerOrCoOwner", "team1", "bob").Return(false, nil)

	_, handler := srv.memberSummaryTool()
	ctx := ctxWithUser(&models.User{ID: "bob"})

	result, err := handler(ctx, makeRequest(map[string]any{
		"team_id": "team1",
		"user_id": "alice",
	}))

	assert.Nil(t, err)
	assert.True(t, result.IsError)
}

func TestMemberSummary_WithFilters(t *testing.T) {
	srv, userSrvc, teamSrvc, summarySrvc, _, _ := newMCPServerWithMocks()

	teamSrvc.On("IsTeamOwnerOrCoOwner", "team1", "alice").Return(true, nil)
	teamSrvc.On("IsTeamMember", "team1", "bob").Return(true, nil)
	userSrvc.On("GetUserById", "bob").Return(&models.User{ID: "bob"}, nil)

	summary := &models.Summary{
		Projects:  []*models.SummaryItem{{Key: "wakapi", Total: time.Duration(3600)}},
		Languages: []*models.SummaryItem{{Key: "Go", Total: time.Duration(3600)}},
	}
	summarySrvc.On("Aliased", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(summary, nil)
	summarySrvc.On("Retrieve", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(summary, nil)

	_, handler := srv.memberSummaryTool()
	ctx := ctxWithUser(&models.User{ID: "alice"})

	result, err := handler(ctx, makeRequest(map[string]any{
		"team_id":  "team1",
		"user_id":  "bob",
		"project":  "wakapi",
		"language": "Go",
	}))

	assert.Nil(t, err)
	assert.False(t, result.IsError)
}

func TestMemberSummary_EmptyResult(t *testing.T) {
	srv, userSrvc, teamSrvc, summarySrvc, _, _ := newMCPServerWithMocks()

	teamSrvc.On("IsTeamOwnerOrCoOwner", "team1", "alice").Return(true, nil)
	teamSrvc.On("IsTeamMember", "team1", "bob").Return(true, nil)
	userSrvc.On("GetUserById", "bob").Return(&models.User{ID: "bob"}, nil)

	summary := &models.Summary{}
	summarySrvc.On("Aliased", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(summary, nil)
	summarySrvc.On("Retrieve", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(summary, nil)

	_, handler := srv.memberSummaryTool()
	ctx := ctxWithUser(&models.User{ID: "alice"})

	result, err := handler(ctx, makeRequest(map[string]any{
		"team_id": "team1",
		"user_id": "bob",
	}))

	assert.Nil(t, err)
	assert.False(t, result.IsError)
	text := extractText(result)
	assert.Contains(t, text, "Nenhuma atividade")
}

func TestMemberSummary_AILinePresentWhenInstrumented(t *testing.T) {
	srv, userSrvc, teamSrvc, summarySrvc, _, _ := newMCPServerWithMocks()

	teamSrvc.On("IsTeamOwnerOrCoOwner", "team1", "alice").Return(true, nil)
	teamSrvc.On("IsTeamMember", "team1", "bob").Return(true, nil)
	userSrvc.On("GetUserById", "bob").Return(&models.User{ID: "bob"}, nil)

	summary := &models.Summary{
		Projects: []*models.SummaryItem{{Key: "wakapi", Total: time.Duration(7200)}},
		Categories: []*models.SummaryItem{
			{Type: models.SummaryCategory, Key: "coding", Total: time.Duration(5400)},
			{Type: models.SummaryCategory, Key: "ai coding", Total: time.Duration(1800)},
		},
	}
	summarySrvc.On("Aliased", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(summary, nil)
	summarySrvc.On("Retrieve", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(summary, nil)

	_, handler := srv.memberSummaryTool()
	result, err := handler(ctxWithUser(&models.User{ID: "alice"}), makeRequest(map[string]any{
		"team_id": "team1",
		"user_id": "bob",
	}))

	assert.Nil(t, err)
	assert.False(t, result.IsError)
	text := extractText(result)
	assert.Contains(t, text, "Coding com IA")
	assert.Contains(t, text, "25.0%") // 1800s de 7200s
}

func TestMemberSummary_AILineAbsentWhenNotInstrumented(t *testing.T) {
	srv, userSrvc, teamSrvc, summarySrvc, _, _ := newMCPServerWithMocks()

	teamSrvc.On("IsTeamOwnerOrCoOwner", "team1", "alice").Return(true, nil)
	teamSrvc.On("IsTeamMember", "team1", "bob").Return(true, nil)
	userSrvc.On("GetUserById", "bob").Return(&models.User{ID: "bob"}, nil)

	summary := &models.Summary{
		Projects: []*models.SummaryItem{{Key: "wakapi", Total: time.Duration(7200)}},
		Categories: []*models.SummaryItem{
			{Type: models.SummaryCategory, Key: "coding", Total: time.Duration(7200)},
		},
	}
	summarySrvc.On("Aliased", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(summary, nil)
	summarySrvc.On("Retrieve", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(summary, nil)

	_, handler := srv.memberSummaryTool()
	result, _ := handler(ctxWithUser(&models.User{ID: "alice"}), makeRequest(map[string]any{
		"team_id": "team1",
		"user_id": "bob",
	}))

	text := extractText(result)
	assert.NotContains(t, text, "Coding com IA", "sem telemetria a linha de IA não aparece")
	assert.Contains(t, text, "wakapi", "o resto do summary segue intacto")
}
