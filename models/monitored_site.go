package models

type MonitoredSite struct {
	ID        uint       `gorm:"primary_key" hash:"ignore"`
	URL       string     `gorm:"not null;size:255;index:idx_monitored_site_url" hash:"ignore"`
	Label     string     `gorm:"not null;size:255" hash:"ignore"`
	UserID    string     `gorm:"not null;index:idx_monitored_site_user;size:255" hash:"ignore"`
	User      *User      `gorm:"constraint:OnDelete:CASCADE" hash:"ignore" json:"-"`
	CreatedAt CustomTime `gorm:"type:timestamp;default:CURRENT_TIMESTAMP" swaggertype:"string" format:"date" example:"2006-01-02 15:04:05.000" hash:"ignore"`
	UpdatedAt CustomTime `gorm:"type:timestamp;default:CURRENT_TIMESTAMP" swaggertype:"string" format:"date" example:"2006-01-02 15:04:05.000" hash:"ignore"`
}

func (s *MonitoredSite) IsValid() bool {
	return s.URL != "" &&
		s.Label != "" &&
		s.UserID != "" &&
		len(s.URL) <= 255 &&
		len(s.Label) <= 255
}
