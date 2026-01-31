package view

import (
	"github.com/muety/wakapi/models"
	"github.com/muety/wakapi/utils"
	"time"
)

type LeaderboardViewModel struct {
	SharedLoggedInViewModel
	By                   string
	Key                  string
	Tab                  string
	Items                []*models.LeaderboardItemRanked
	TeamItems            []*models.TeamLeaderboardItemRanked
	UserTeamIDs          map[string]bool
	MemberDashboardLinks map[string]string // userID -> "/teams/{teamID}/members/{userID}"
	TopKeys              []string
	UserLanguages        map[string][]string
	IntervalLabel        string
	PageParams           *utils.PageParams
}

func (s *LeaderboardViewModel) WithSuccess(m string) *LeaderboardViewModel {
	s.SetSuccess(m)
	return s
}

func (s *LeaderboardViewModel) WithError(m string) *LeaderboardViewModel {
	s.SetError(m)
	return s
}

func (s *LeaderboardViewModel) ColorModifier(item *models.LeaderboardItemRanked, principal *models.User) string {
	if principal != nil && item.UserID == principal.ID {
		return "self"
	}
	if item.Rank == 1 {
		return "gold"
	}
	if item.Rank == 2 {
		return "silver"
	}
	if item.Rank == 3 {
		return "bronze"
	}
	return "default"
}

func (s *LeaderboardViewModel) ColorModifierTeam(item *models.TeamLeaderboardItemRanked) string {
	if item.Rank == 1 {
		return "gold"
	}
	if item.Rank == 2 {
		return "silver"
	}
	if item.Rank == 3 {
		return "bronze"
	}
	return "default"
}

func (s *LeaderboardViewModel) MemberDashboardLink(userID string) string {
	return s.MemberDashboardLinks[userID]
}

func (s *LeaderboardViewModel) CanAccessTeam(teamID string) bool {
	if s.User != nil && s.User.IsAdmin {
		return true
	}
	return s.UserTeamIDs[teamID]
}

func (s *LeaderboardViewModel) LangIcon(lang string) string {
	return GetLanguageIcon(lang)
}

func (s *LeaderboardViewModel) LastUpdate() time.Time {
	tz := time.Local
	if s.User != nil {
		tz = s.User.TZ()
	}
	return models.Leaderboard(s.Items).LastUpdate().In(tz)
}
