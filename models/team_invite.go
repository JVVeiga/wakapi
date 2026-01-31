package models

import "time"

type TeamInvite struct {
	ID        uint        `json:"id" gorm:"primary_key"`
	Code      string      `json:"code" gorm:"uniqueIndex; size:36; not null"`
	TeamID    string      `json:"team_id" gorm:"not null; index; size:36"`
	Team      *Team       `json:"-" gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	CreatedBy string      `json:"created_by" gorm:"not null; size:255"`
	Creator   *User       `json:"-" gorm:"foreignKey:CreatedBy; constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	UsedBy    *string     `json:"used_by" gorm:"size:255"`
	UsedAt    *CustomTime `json:"used_at"`
	ExpiresAt CustomTime  `json:"expires_at" gorm:"not null"`
	CreatedAt CustomTime  `json:"created_at" gorm:"not null"`
}

func (i *TeamInvite) IsExpired() bool {
	return time.Time(i.ExpiresAt).Before(time.Now())
}

func (i *TeamInvite) IsUsed() bool {
	return i.UsedBy != nil
}

func (i *TeamInvite) UsedByName() string {
	if i.UsedBy != nil {
		return *i.UsedBy
	}
	return ""
}

func (i *TeamInvite) Status() string {
	if i.IsUsed() {
		return "used"
	}
	if i.IsExpired() {
		return "expired"
	}
	return "active"
}
