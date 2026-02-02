package services

import (
	"time"

	"github.com/muety/wakapi/models"
	"github.com/muety/wakapi/repositories"
	cache "github.com/patrickmn/go-cache"
)

const (
	monitoredSiteCacheTTL      = 1 * time.Hour
	monitoredSiteCacheKey      = "monitored_sites_all"
	monitoredSiteCountCacheKey = "monitored_sites_count"
)

type MonitoredSiteService struct {
	repository repositories.IMonitoredSiteRepository
	cache      *cache.Cache
}

func NewMonitoredSiteService(repo repositories.IMonitoredSiteRepository) *MonitoredSiteService {
	return &MonitoredSiteService{
		repository: repo,
		cache:      cache.New(monitoredSiteCacheTTL, monitoredSiteCacheTTL*2),
	}
}

func (s *MonitoredSiteService) GetAll() ([]*models.MonitoredSite, error) {
	if cached, found := s.cache.Get(monitoredSiteCacheKey); found {
		return cached.([]*models.MonitoredSite), nil
	}

	sites, err := s.repository.GetAll()
	if err != nil {
		return nil, err
	}

	s.cache.Set(monitoredSiteCacheKey, sites, cache.DefaultExpiration)
	return sites, nil
}

func (s *MonitoredSiteService) GetByID(id uint) (*models.MonitoredSite, error) {
	return s.repository.GetByID(id)
}

func (s *MonitoredSiteService) Create(site *models.MonitoredSite) (*models.MonitoredSite, error) {
	created, err := s.repository.Insert(site)
	if err != nil {
		return nil, err
	}
	s.invalidateCache()
	return created, nil
}

func (s *MonitoredSiteService) Update(site *models.MonitoredSite) (*models.MonitoredSite, error) {
	updated, err := s.repository.Update(site)
	if err != nil {
		return nil, err
	}
	s.invalidateCache()
	return updated, nil
}

func (s *MonitoredSiteService) Delete(id uint) error {
	if err := s.repository.Delete(id); err != nil {
		return err
	}
	s.invalidateCache()
	return nil
}

func (s *MonitoredSiteService) Count() (int, error) {
	if cached, found := s.cache.Get(monitoredSiteCountCacheKey); found {
		return cached.(int), nil
	}

	sites, err := s.GetAll()
	if err != nil {
		return 0, err
	}

	count := len(sites)
	s.cache.Set(monitoredSiteCountCacheKey, count, cache.DefaultExpiration)
	return count, nil
}

func (s *MonitoredSiteService) invalidateCache() {
	s.cache.Delete(monitoredSiteCacheKey)
	s.cache.Delete(monitoredSiteCountCacheKey)
}
