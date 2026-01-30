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
}

type AdminUserDetailViewModel struct {
	SharedLoggedInViewModel
	TargetUser    *models.User
	ApiKeys       []*models.ApiKey
	MaskedMainKey string
	TotalTime     string
	LastActivity  string
}
