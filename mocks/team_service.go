package mocks

import (
	"github.com/stretchr/testify/mock"

	"github.com/muety/wakapi/models"
)

type TeamServiceMock struct {
	mock.Mock
}

func (m *TeamServiceMock) CreateTeam(name, description, ownerID string) (*models.Team, error) {
	args := m.Called(name, description, ownerID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Team), args.Error(1)
}

func (m *TeamServiceMock) DeleteTeam(teamID string) error {
	args := m.Called(teamID)
	return args.Error(0)
}

func (m *TeamServiceMock) UpdateTeam(team *models.Team) (*models.Team, error) {
	args := m.Called(team)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Team), args.Error(1)
}

func (m *TeamServiceMock) GetByID(teamID string) (*models.Team, error) {
	args := m.Called(teamID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Team), args.Error(1)
}

func (m *TeamServiceMock) GetByUser(userID string) ([]*models.Team, error) {
	args := m.Called(userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Team), args.Error(1)
}

func (m *TeamServiceMock) GetAll() ([]*models.Team, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Team), args.Error(1)
}

func (m *TeamServiceMock) AddMember(teamID, userID, role string) (*models.TeamMember, error) {
	args := m.Called(teamID, userID, role)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.TeamMember), args.Error(1)
}

func (m *TeamServiceMock) RemoveMember(teamID, userID string) error {
	args := m.Called(teamID, userID)
	return args.Error(0)
}

func (m *TeamServiceMock) GetMembers(teamID string) ([]*models.TeamMember, error) {
	args := m.Called(teamID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.TeamMember), args.Error(1)
}

func (m *TeamServiceMock) CountMembers(teamID string) (int64, error) {
	args := m.Called(teamID)
	return args.Get(0).(int64), args.Error(1)
}

func (m *TeamServiceMock) IsTeamOwner(teamID, userID string) (bool, error) {
	args := m.Called(teamID, userID)
	return args.Bool(0), args.Error(1)
}

func (m *TeamServiceMock) IsTeamMember(teamID, userID string) (bool, error) {
	args := m.Called(teamID, userID)
	return args.Bool(0), args.Error(1)
}

func (m *TeamServiceMock) TransferOwnership(teamID, userID string) error {
	args := m.Called(teamID, userID)
	return args.Error(0)
}

func (m *TeamServiceMock) GenerateInvite(teamID, role string) (*models.TeamInvite, error) {
	args := m.Called(teamID, role)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.TeamInvite), args.Error(1)
}

func (m *TeamServiceMock) AcceptInvite(code, userID string) (*models.Team, error) {
	args := m.Called(code, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Team), args.Error(1)
}

func (m *TeamServiceMock) GetInvites(teamID string, limit int) ([]*models.TeamInvite, int64, error) {
	args := m.Called(teamID, limit)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]*models.TeamInvite), args.Get(1).(int64), args.Error(2)
}

func (m *TeamServiceMock) GetInviteByCode(code string) (*models.TeamInvite, error) {
	args := m.Called(code)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.TeamInvite), args.Error(1)
}

func (m *TeamServiceMock) IsTeamOwnerOrCoOwner(teamID, userID string) (bool, error) {
	args := m.Called(teamID, userID)
	return args.Bool(0), args.Error(1)
}

func (m *TeamServiceMock) CanManageInvites(teamID, userID string) (bool, error) {
	args := m.Called(teamID, userID)
	return args.Bool(0), args.Error(1)
}

func (m *TeamServiceMock) CanViewMemberDashboards(teamID, userID string) (bool, error) {
	args := m.Called(teamID, userID)
	return args.Bool(0), args.Error(1)
}

func (m *TeamServiceMock) CanRemoveMembers(teamID, userID string) (bool, error) {
	args := m.Called(teamID, userID)
	return args.Bool(0), args.Error(1)
}

func (m *TeamServiceMock) CanPromoteMembers(teamID, userID string) (bool, error) {
	args := m.Called(teamID, userID)
	return args.Bool(0), args.Error(1)
}

func (m *TeamServiceMock) UpdateMemberRole(teamID, userID, role string) error {
	args := m.Called(teamID, userID, role)
	return args.Error(0)
}

func (m *TeamServiceMock) GetUserPermissions(teamID, userID string) (*models.TeamPermissions, error) {
	args := m.Called(teamID, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.TeamPermissions), args.Error(1)
}
