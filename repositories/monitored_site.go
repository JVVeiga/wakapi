package repositories

import (
	"github.com/muety/wakapi/models"
	"gorm.io/gorm"
)

type MonitoredSiteRepository struct {
	*BaseRepository
}

func NewMonitoredSiteRepository(db *gorm.DB) *MonitoredSiteRepository {
	return &MonitoredSiteRepository{
		BaseRepository: &BaseRepository{
			db: db,
		},
	}
}

func (r *MonitoredSiteRepository) GetAll() ([]*models.MonitoredSite, error) {
	var sites []*models.MonitoredSite
	if err := r.db.
		Preload("User").
		Order("created_at DESC").
		Find(&sites).Error; err != nil {
		return nil, err
	}
	return sites, nil
}

func (r *MonitoredSiteRepository) GetByID(id uint) (*models.MonitoredSite, error) {
	site := &models.MonitoredSite{}
	if err := r.db.
		Preload("User").
		Where("id = ?", id).
		First(site).Error; err != nil {
		return nil, err
	}
	return site, nil
}

func (r *MonitoredSiteRepository) Insert(site *models.MonitoredSite) (*models.MonitoredSite, error) {
	if err := r.db.Create(site).Error; err != nil {
		return nil, err
	}
	return site, nil
}

func (r *MonitoredSiteRepository) Update(site *models.MonitoredSite) (*models.MonitoredSite, error) {
	if err := r.db.Save(site).Error; err != nil {
		return nil, err
	}
	return site, nil
}

func (r *MonitoredSiteRepository) Delete(id uint) error {
	return r.db.Delete(&models.MonitoredSite{}, id).Error
}
