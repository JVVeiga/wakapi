package view

import (
	"time"

	"github.com/muety/wakapi/models"
)

type TeamsViewModel struct {
	SharedLoggedInViewModel
	Teams            []*models.Team
	TeamMemberCounts map[string]int64
}

type MemberSummaryItem struct {
	UserID    string
	TotalTime time.Duration
}

type TeamDetailViewModel struct {
	SharedLoggedInViewModel
	Team            *models.Team
	Members         []*models.TeamMember
	Summary         *models.Summary
	MemberSummaries []*MemberSummaryItem
	From            time.Time
	To              time.Time
	IntervalLabel   string
	IsOwner         bool
}
