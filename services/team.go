package services

import (
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/gofrs/uuid/v5"
	"github.com/patrickmn/go-cache"

	"github.com/muety/wakapi/config"
	"github.com/muety/wakapi/models"
	"github.com/muety/wakapi/repositories"
)

type TeamService struct {
	config     *config.Config
	cache      *cache.Cache
	repository repositories.ITeamRepository
}

func NewTeamService(teamRepository repositories.ITeamRepository) *TeamService {
	return &TeamService{
		config:     config.Get(),
		repository: teamRepository,
		cache:      cache.New(1*time.Hour, 1*time.Hour),
	}
}

func (srv *TeamService) CreateTeam(name, description, ownerID string) (*models.Team, error) {
	team := models.NewTeam(name, description, ownerID)

	member := &models.TeamMember{
		TeamID: team.ID,
		UserID: ownerID,
		Role:   models.TeamRoleOwner,
	}

	createdTeam, err := srv.repository.InsertWithOwner(team, member)
	if err != nil {
		return nil, err
	}

	srv.invalidateTeam(team.ID)
	srv.invalidateUserTeams(ownerID)
	return createdTeam, nil
}

func (srv *TeamService) DeleteTeam(teamID string) error {
	// get members before deleting so we can invalidate their caches
	members, _ := srv.repository.GetMembersByTeam(teamID)

	err := srv.repository.Delete(teamID)
	if err == nil {
		srv.invalidateTeam(teamID)
		for _, m := range members {
			srv.invalidateUserTeams(m.UserID)
		}
	}
	return err
}

func (srv *TeamService) UpdateTeam(team *models.Team) (*models.Team, error) {
	updated, err := srv.repository.Update(team)
	if err != nil {
		return nil, err
	}
	srv.invalidateTeam(team.ID)
	return updated, nil
}

func (srv *TeamService) GetByID(teamID string) (*models.Team, error) {
	cacheKey := "team_" + teamID
	if cached, found := srv.cache.Get(cacheKey); found {
		return cached.(*models.Team), nil
	}

	team, err := srv.repository.GetByID(teamID)
	if err != nil {
		return nil, err
	}

	srv.cache.Set(cacheKey, team, cache.DefaultExpiration)
	return team, nil
}

func (srv *TeamService) GetByUser(userID string) ([]*models.Team, error) {
	cacheKey := "user_teams_" + userID
	if cached, found := srv.cache.Get(cacheKey); found {
		return cached.([]*models.Team), nil
	}

	teams, err := srv.repository.GetByUser(userID)
	if err != nil {
		return nil, err
	}

	srv.cache.Set(cacheKey, teams, cache.DefaultExpiration)
	return teams, nil
}

func (srv *TeamService) GetAll() ([]*models.Team, error) {
	return srv.repository.GetAll()
}

func (srv *TeamService) AddMember(teamID, userID, role string) (*models.TeamMember, error) {
	if role == models.TeamRoleOwner {
		return nil, errors.New("cannot add a second owner; owner is set at team creation")
	}
	if role != models.TeamRoleMember {
		return nil, errors.New("invalid role")
	}

	existing, _ := srv.repository.GetMemberByTeamAndUser(teamID, userID)
	if existing != nil {
		return nil, errors.New("user is already a team member")
	}

	member := &models.TeamMember{
		TeamID: teamID,
		UserID: userID,
		Role:   role,
	}

	added, err := srv.repository.AddMember(member)
	if err != nil {
		return nil, err
	}

	srv.invalidateTeam(teamID)
	srv.invalidateUserTeams(userID)
	return added, nil
}

func (srv *TeamService) RemoveMember(teamID, userID string) error {
	member, err := srv.repository.GetMemberByTeamAndUser(teamID, userID)
	if err != nil {
		return errors.New("member not found")
	}

	if member.Role == models.TeamRoleOwner {
		return errors.New("cannot remove team owner")
	}

	if err := srv.repository.RemoveMember(teamID, userID); err != nil {
		return err
	}

	srv.invalidateTeam(teamID)
	srv.invalidateUserTeams(userID)
	return nil
}

func (srv *TeamService) TransferOwnership(teamID, newOwnerID string) error {
	_, err := srv.repository.GetMemberByTeamAndUser(teamID, newOwnerID)
	if err != nil {
		return errors.New("new owner must be a member of the team")
	}

	if err := srv.repository.TransferOwnership(teamID, newOwnerID); err != nil {
		return err
	}

	srv.invalidateTeam(teamID)
	return nil
}

func (srv *TeamService) GetMembers(teamID string) ([]*models.TeamMember, error) {
	cacheKey := "team_members_" + teamID
	if cached, found := srv.cache.Get(cacheKey); found {
		return cached.([]*models.TeamMember), nil
	}

	members, err := srv.repository.GetMembersByTeam(teamID)
	if err != nil {
		return nil, err
	}

	srv.cache.Set(cacheKey, members, cache.DefaultExpiration)
	return members, nil
}

func (srv *TeamService) CountMembers(teamID string) (int64, error) {
	return srv.repository.CountByTeam(teamID)
}

func (srv *TeamService) IsTeamOwner(teamID, userID string) (bool, error) {
	member, err := srv.repository.GetMemberByTeamAndUser(teamID, userID)
	if err != nil {
		return false, nil
	}
	return member.Role == models.TeamRoleOwner, nil
}

func (srv *TeamService) IsTeamMember(teamID, userID string) (bool, error) {
	_, err := srv.repository.GetMemberByTeamAndUser(teamID, userID)
	return err == nil, nil
}

// invalidateTeam removes cached data for a specific team
func (srv *TeamService) invalidateTeam(teamID string) {
	srv.cache.Delete(fmt.Sprintf("team_%s", teamID))
	srv.cache.Delete(fmt.Sprintf("team_members_%s", teamID))
}

// invalidateUserTeams removes cached team list for a specific user
func (srv *TeamService) invalidateUserTeams(userID string) {
	srv.cache.Delete(fmt.Sprintf("user_teams_%s", userID))
}

// invalidateByPrefix removes all cache entries with the given prefix (unused but available for future use)
func (srv *TeamService) invalidateByPrefix(prefix string) {
	for k := range srv.cache.Items() {
		if strings.HasPrefix(k, prefix) {
			srv.cache.Delete(k)
		}
	}
}

const invitePageSize = 5

func (srv *TeamService) GenerateInvite(teamID, creatorID string) (*models.TeamInvite, error) {
	code := uuid.Must(uuid.NewV4()).String()
	now := models.CustomTime(time.Now())
	expiresAt := models.CustomTime(time.Now().Add(2 * time.Hour))

	invite := &models.TeamInvite{
		Code:      code,
		TeamID:    teamID,
		CreatedBy: creatorID,
		ExpiresAt: expiresAt,
		CreatedAt: now,
	}

	return srv.repository.CreateInvite(invite)
}

func (srv *TeamService) AcceptInvite(code, userID string) (*models.Team, error) {
	invite, err := srv.repository.GetInviteByCode(code)
	if err != nil {
		return nil, errors.New("invite not found")
	}

	if invite.IsUsed() {
		return nil, errors.New("invite already used")
	}

	if invite.IsExpired() {
		return nil, errors.New("invite expired")
	}

	// Check if already a member
	isMember, _ := srv.IsTeamMember(invite.TeamID, userID)
	if isMember {
		return invite.Team, errors.New("already a member")
	}

	// Add as member
	member := &models.TeamMember{
		TeamID: invite.TeamID,
		UserID: userID,
		Role:   models.TeamRoleMember,
	}
	if _, err := srv.repository.AddMember(member); err != nil {
		return nil, err
	}

	// Mark invite as used
	if err := srv.repository.MarkInviteUsed(code, userID); err != nil {
		return nil, err
	}

	srv.invalidateTeam(invite.TeamID)
	srv.invalidateUserTeams(userID)

	return invite.Team, nil
}

func (srv *TeamService) GetInvites(teamID string, page int) ([]*models.TeamInvite, int64, error) {
	invites, total, err := srv.repository.GetInvitesByTeam(teamID, page, invitePageSize)
	if err != nil {
		return nil, 0, err
	}
	return invites, int64(math.Ceil(float64(total) / float64(invitePageSize))), nil
}

func (srv *TeamService) GetInviteByCode(code string) (*models.TeamInvite, error) {
	return srv.repository.GetInviteByCode(code)
}
