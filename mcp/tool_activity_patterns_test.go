package mcp

import (
	"testing"
	"time"

	"github.com/muety/wakapi/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestActivityPatterns_Success(t *testing.T) {
	srv, userSrvc, teamSrvc, _, heartbeatSrvc, durationSrvc := newMCPServerWithMocks()

	teamSrvc.On("IsTeamOwnerOrCoOwner", "team1", "alice").Return(true, nil)
	teamSrvc.On("IsTeamMember", "team1", "bob").Return(true, nil)
	userSrvc.On("GetUserById", "bob").Return(&models.User{ID: "bob"}, nil)

	now := time.Now()
	durations := models.Durations{
		&models.Duration{
			Time:     models.CustomTime(now.Add(-2 * time.Hour)),
			Duration: 45 * time.Minute,
		},
		&models.Duration{
			Time:     models.CustomTime(now.Add(-1 * time.Hour)),
			Duration: 30 * time.Minute,
		},
	}
	durationSrvc.On("Get", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(durations, nil)
	heartbeatSrvc.On("GetLatestByUser", mock.Anything, false).Return(&models.Heartbeat{
		Time: models.CustomTime(now),
	}, nil)

	_, handler := srv.activityPatternsTool()
	ctx := ctxWithUser(&models.User{ID: "alice"})

	result, err := handler(ctx, makeRequest(map[string]any{
		"team_id": "team1",
		"user_id": "bob",
	}))

	assert.Nil(t, err)
	assert.False(t, result.IsError)
	text := extractText(result)
	assert.Contains(t, text, "bob")
	assert.Contains(t, text, "Sessões:")
	assert.Contains(t, text, "Dias ativos:")
	assert.Contains(t, text, "Última atividade:")
}

// The heartbeat repository returns a zero-value heartbeat (not nil) when a user has none;
// it must not be rendered as a bogus "last activity" date.
func TestActivityPatterns_ZeroLatestHeartbeat(t *testing.T) {
	srv, userSrvc, teamSrvc, _, heartbeatSrvc, durationSrvc := newMCPServerWithMocks()

	teamSrvc.On("IsTeamOwnerOrCoOwner", "team1", "alice").Return(true, nil)
	teamSrvc.On("IsTeamMember", "team1", "bob").Return(true, nil)
	userSrvc.On("GetUserById", "bob").Return(&models.User{ID: "bob"}, nil)

	durations := models.Durations{
		&models.Duration{Time: models.CustomTime(time.Now().Add(-time.Hour)), Duration: 30 * time.Minute},
	}
	durationSrvc.On("Get", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(durations, nil)
	heartbeatSrvc.On("GetLatestByUser", mock.Anything, false).Return(&models.Heartbeat{}, nil)

	_, handler := srv.activityPatternsTool()
	result, err := handler(ctxWithUser(&models.User{ID: "alice"}), makeRequest(map[string]any{
		"team_id": "team1",
		"user_id": "bob",
	}))

	assert.Nil(t, err)
	assert.False(t, result.IsError)
	assert.NotContains(t, extractText(result), "Última atividade:")
}

func TestActivityPatterns_NoDurations(t *testing.T) {
	srv, userSrvc, teamSrvc, _, _, durationSrvc := newMCPServerWithMocks()

	teamSrvc.On("IsTeamOwnerOrCoOwner", "team1", "alice").Return(true, nil)
	teamSrvc.On("IsTeamMember", "team1", "bob").Return(true, nil)
	userSrvc.On("GetUserById", "bob").Return(&models.User{ID: "bob"}, nil)
	durationSrvc.On("Get", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(models.Durations{}, nil)

	_, handler := srv.activityPatternsTool()
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

func TestActivityPatterns_AccessDenied(t *testing.T) {
	srv, _, teamSrvc, _, _, _ := newMCPServerWithMocks()
	teamSrvc.On("IsTeamOwnerOrCoOwner", "team1", "bob").Return(false, nil)

	_, handler := srv.activityPatternsTool()
	ctx := ctxWithUser(&models.User{ID: "bob"})

	result, err := handler(ctx, makeRequest(map[string]any{
		"team_id": "team1",
		"user_id": "alice",
	}))

	assert.Nil(t, err)
	assert.True(t, result.IsError)
}
