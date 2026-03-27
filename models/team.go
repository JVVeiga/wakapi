package models

import "github.com/gofrs/uuid/v5"

const (
	TeamRoleOwner   = "owner"
	TeamRoleCoOwner = "co-owner"
	TeamRoleMember  = "member"
)

type Team struct {
	ID          string     `json:"id" gorm:"primary_key; size:36"`
	Name        string     `json:"name" gorm:"not null; size:255"`
	Description string     `json:"description" gorm:"type:text"`
	Owner       *User      `json:"-" gorm:"not null; constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	OwnerID     string     `json:"owner_id" gorm:"not null; index:idx_team_owner; size:255"`
	CreatedAt   CustomTime `json:"created_at" swaggertype:"string" format:"date" example:"2006-01-02 15:04:05.000"`
}

type TeamMember struct {
	ID       uint       `json:"id" gorm:"primary_key"`
	Team     *Team      `json:"-" gorm:"not null; constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	TeamID   string     `json:"team_id" gorm:"not null; size:36; uniqueIndex:idx_team_member_composite"`
	User     *User      `json:"-" gorm:"not null; constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	UserID   string     `json:"user_id" gorm:"not null; size:255; uniqueIndex:idx_team_member_composite"`
	Role     string     `json:"role" gorm:"not null; size:32"`
	JoinedAt CustomTime `json:"joined_at" swaggertype:"string" format:"date" example:"2006-01-02 15:04:05.000"`
}

func (t *Team) IsValid() bool {
	return t.ID != "" && t.Name != "" && t.OwnerID != ""
}

func (tm *TeamMember) IsValid() bool {
	return tm.TeamID != "" && tm.UserID != "" &&
		(tm.Role == TeamRoleOwner || tm.Role == TeamRoleCoOwner || tm.Role == TeamRoleMember)
}

// TeamPermissions represents all permissions a user has for a specific team
type TeamPermissions struct {
	IsOwner           bool // True if user is the team owner
	IsCoOwner         bool // True if user is owner OR co-owner
	IsMember          bool // True if user is any kind of member
	CanRemove         bool // Can remove team members (owner only)
	CanPromote        bool // Can promote/demote members (owner only)
	CanManageInvites  bool // Can create and manage invites (owner or co-owner)
	CanViewDashboards bool // Can view member dashboards (owner or co-owner)
}

func NewTeam(name, description, ownerID string) *Team {
	return &Team{
		ID:          uuid.Must(uuid.NewV4()).String(),
		Name:        name,
		Description: description,
		OwnerID:     ownerID,
	}
}
