package repositories

import (
	"errors"
	"time"

	"github.com/muety/wakapi/config"
	"github.com/muety/wakapi/models"
	"gorm.io/gorm"
)

type TeamRepository struct {
	BaseRepository
	config *config.Config
}

func NewTeamRepository(db *gorm.DB) *TeamRepository {
	return &TeamRepository{
		BaseRepository: NewBaseRepository(db),
		config:         config.Get(),
	}
}

func (r *TeamRepository) GetAll() ([]*models.Team, error) {
	var teams []*models.Team
	if err := r.db.Preload("Owner").Find(&teams).Error; err != nil {
		return nil, err
	}
	return teams, nil
}

func (r *TeamRepository) GetByID(teamID string) (*models.Team, error) {
	var team models.Team
	if err := r.db.Preload("Owner").Where("id = ?", teamID).First(&team).Error; err != nil {
		return nil, err
	}
	return &team, nil
}

func (r *TeamRepository) GetByUser(userID string) ([]*models.Team, error) {
	var teams []*models.Team
	if err := r.db.
		Joins("JOIN team_members ON teams.id = team_members.team_id").
		Where("team_members.user_id = ?", userID).
		Preload("Owner").
		Find(&teams).Error; err != nil {
		return nil, err
	}
	return teams, nil
}

func (r *TeamRepository) GetByOwner(ownerID string) ([]*models.Team, error) {
	var teams []*models.Team
	if err := r.db.
		Where("owner_id = ?", ownerID).
		Preload("Owner").
		Find(&teams).Error; err != nil {
		return nil, err
	}
	return teams, nil
}

func (r *TeamRepository) Insert(team *models.Team) (*models.Team, error) {
	if !team.IsValid() {
		return nil, errors.New("invalid team")
	}
	if err := r.db.Create(team).Error; err != nil {
		return nil, err
	}
	return team, nil
}

func (r *TeamRepository) InsertWithOwner(team *models.Team, owner *models.TeamMember) (*models.Team, error) {
	if !team.IsValid() {
		return nil, errors.New("invalid team")
	}
	if !owner.IsValid() {
		return nil, errors.New("invalid team member")
	}
	err := r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(team).Error; err != nil {
			return err
		}
		if err := tx.Create(owner).Error; err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return team, nil
}

func (r *TeamRepository) Update(team *models.Team) (*models.Team, error) {
	if !team.IsValid() {
		return nil, errors.New("invalid team")
	}
	if err := r.db.Save(team).Error; err != nil {
		return nil, err
	}
	return team, nil
}

func (r *TeamRepository) Delete(teamID string) error {
	return r.db.Where("id = ?", teamID).Delete(&models.Team{}).Error
}

func (r *TeamRepository) GetMembersByTeam(teamID string) ([]*models.TeamMember, error) {
	var members []*models.TeamMember
	if err := r.db.
		Where("team_id = ?", teamID).
		Preload("User").
		Find(&members).Error; err != nil {
		return nil, err
	}
	return members, nil
}

func (r *TeamRepository) GetMemberByTeamAndUser(teamID, userID string) (*models.TeamMember, error) {
	var member models.TeamMember
	if err := r.db.
		Where("team_id = ? AND user_id = ?", teamID, userID).
		Preload("User").
		First(&member).Error; err != nil {
		return nil, err
	}
	return &member, nil
}

func (r *TeamRepository) AddMember(member *models.TeamMember) (*models.TeamMember, error) {
	if !member.IsValid() {
		return nil, errors.New("invalid team member")
	}
	if err := r.db.Create(member).Error; err != nil {
		return nil, err
	}
	return member, nil
}

func (r *TeamRepository) RemoveMember(teamID, userID string) error {
	return r.db.
		Where("team_id = ? AND user_id = ?", teamID, userID).
		Delete(&models.TeamMember{}).Error
}

func (r *TeamRepository) TransferOwnership(teamID, newOwnerID string) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// Update old owner's role to member
		if err := tx.Model(&models.TeamMember{}).
			Where("team_id = ? AND role = ?", teamID, models.TeamRoleOwner).
			Update("role", models.TeamRoleMember).Error; err != nil {
			return err
		}
		// Update new owner's role to owner
		if err := tx.Model(&models.TeamMember{}).
			Where("team_id = ? AND user_id = ?", teamID, newOwnerID).
			Update("role", models.TeamRoleOwner).Error; err != nil {
			return err
		}
		// Update team's owner_id
		if err := tx.Model(&models.Team{}).
			Where("id = ?", teamID).
			Update("owner_id", newOwnerID).Error; err != nil {
			return err
		}
		return nil
	})
}

func (r *TeamRepository) CountByTeam(teamID string) (int64, error) {
	var count int64
	err := r.db.
		Model(&models.TeamMember{}).
		Where("team_id = ?", teamID).
		Count(&count).Error
	return count, err
}

func (r *TeamRepository) CreateInvite(invite *models.TeamInvite) (*models.TeamInvite, error) {
	if err := r.db.Create(invite).Error; err != nil {
		return nil, err
	}
	return invite, nil
}

func (r *TeamRepository) GetInviteByCode(code string) (*models.TeamInvite, error) {
	var invite models.TeamInvite
	if err := r.db.
		Preload("Team").
		Preload("Creator").
		Where("code = ?", code).
		First(&invite).Error; err != nil {
		return nil, err
	}
	return &invite, nil
}

func (r *TeamRepository) GetInvitesByTeam(teamID string, page, pageSize int) ([]*models.TeamInvite, int64, error) {
	var invites []*models.TeamInvite
	var total int64

	r.db.Model(&models.TeamInvite{}).Where("team_id = ?", teamID).Count(&total)

	if err := r.db.
		Where("team_id = ?", teamID).
		Preload("Creator").
		Order("created_at DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&invites).Error; err != nil {
		return nil, 0, err
	}
	return invites, total, nil
}

func (r *TeamRepository) MarkInviteUsed(code string, userID string) error {
	now := models.CustomTime(time.Now())
	return r.db.
		Model(&models.TeamInvite{}).
		Where("code = ?", code).
		Updates(map[string]interface{}{
			"used_by": userID,
			"used_at": now,
		}).Error
}
