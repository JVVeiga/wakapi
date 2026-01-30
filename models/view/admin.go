package view

import (
	"github.com/muety/wakapi/models"
)

type AdminDashboardViewModel struct {
	SharedLoggedInViewModel
	TotalUsers      int64
	ActiveUsers     int
	OnlineUsers     int
	TotalHeartbeats int64
	Users           []*models.User
	Page            int
	PageSize        int
	TotalPages      int
}

type AdminUserDetailViewModel struct {
	SharedLoggedInViewModel
	TargetUser    *models.User
	ApiKeys       []*models.ApiKey
	MaskedMainKey string
	TotalTime     string
	LastActivity  string
	UserTeams     []*models.Team
}

type AdminTeamsViewModel struct {
	SharedLoggedInViewModel
	Teams            []*models.Team
	TeamMemberCounts map[string]int64
	Users            []*models.User
}

type AdminTeamDetailViewModel struct {
	SharedLoggedInViewModel
	Team     *models.Team
	Members  []*models.TeamMember
	AllUsers []*models.User
}
