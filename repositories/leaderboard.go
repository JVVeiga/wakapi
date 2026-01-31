package repositories

import (
	"time"

	"github.com/muety/wakapi/models"
	"github.com/muety/wakapi/utils"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type LeaderboardRepository struct {
	BaseRepository
}

func NewLeaderboardRepository(db *gorm.DB) *LeaderboardRepository {
	return &LeaderboardRepository{BaseRepository: NewBaseRepository(db)}
}

func (r *LeaderboardRepository) InsertBatch(items []*models.LeaderboardItem) error {
	if err := r.db.
		Clauses(clause.OnConflict{DoNothing: true}).
		Create(&items).Error; err != nil {
		return err
	}
	return nil
}

func (r *LeaderboardRepository) CountAllByUser(userId string) (int64, error) {
	var count int64
	err := r.db.
		Table("leaderboard_items").
		Where("user_id = ?", userId).
		Count(&count).Error
	return count, err
}

func (r *LeaderboardRepository) CountUsers(excludeZero bool) (int64, error) {
	var count int64
	q := r.db.Table("leaderboard_items").Distinct("user_id")
	if excludeZero {
		q = q.Where("total > 0")
	}
	err := q.Count(&count).Error
	return count, err
}

func (r *LeaderboardRepository) GetAll() ([]*models.LeaderboardItem, error) {
	var items []*models.LeaderboardItem
	if err := r.db.
		Where(&models.LeaderboardItem{}).
		Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

func (r *LeaderboardRepository) GetAllAggregatedByInterval(key *models.IntervalKey, by *uint8, limit, skip int) ([]*models.LeaderboardItemRanked, error) {
	// TODO: distinct by (user, key) to filter out potential duplicates ?

	var items []*models.LeaderboardItemRanked
	subq := r.db.
		Table("leaderboard_items").
		Select("*, rank() over (partition by \"key\" order by total desc) as \"rank\"").
		Where("\"interval\" in ?", *key)
	subq = utils.WhereNullable(subq, "\"by\"", by)

	q := r.db.Table("(?) as ranked", subq)
	q = r.withPaging(q, limit, skip)

	if err := q.Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

func (r *LeaderboardRepository) GetAggregatedByUserAndInterval(userId string, key *models.IntervalKey, by *uint8, limit, skip int) ([]*models.LeaderboardItemRanked, error) {
	var items []*models.LeaderboardItemRanked
	subq := r.db.
		Table("leaderboard_items").
		Select("*, rank() over (partition by \"key\" order by total desc) as \"rank\"").
		Where("\"interval\" in ?", *key)
	subq = utils.WhereNullable(subq, "\"by\"", by)

	q := r.db.Table("(?) as ranked", subq).Where("user_id = ?", userId)
	q = r.withPaging(q, limit, skip)

	if err := q.Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

func (r *LeaderboardRepository) DeleteByUser(userId string) error {
	if err := r.db.
		Where("user_id = ?", userId).
		Delete(models.LeaderboardItem{}).Error; err != nil {
		return err
	}
	return nil
}

func (r *LeaderboardRepository) DeleteByUserAndInterval(userId string, key *models.IntervalKey) error {
	if err := r.db.
		Where("user_id = ?", userId).
		Where("\"interval\" in ?", *key).
		Delete(models.LeaderboardItem{}).Error; err != nil {
		return err
	}
	return nil
}

func (r *LeaderboardRepository) SumByUsersAndInterval(userIDs []string, key *models.IntervalKey) (time.Duration, error) {
	var total int64
	if err := r.db.
		Table("leaderboard_items").
		Select("COALESCE(SUM(total), 0)").
		Where("user_id IN ?", userIDs).
		Where("\"interval\" IN ?", *key).
		Where("\"by\" IS NULL").
		Scan(&total).Error; err != nil {
		return 0, err
	}
	return time.Duration(total), nil
}

func (r *LeaderboardRepository) TopKeysByUsersAndInterval(userIDs []string, key *models.IntervalKey, by uint8, limit int) ([]string, error) {
	var results []struct {
		Key      string
		SumTotal int64
	}
	if err := r.db.
		Table("leaderboard_items").
		Select("\"key\", SUM(total) as sum_total").
		Where("user_id IN ?", userIDs).
		Where("\"interval\" IN ?", *key).
		Where("\"by\" = ?", by).
		Where("\"key\" IS NOT NULL").
		Group("\"key\"").
		Order("sum_total DESC").
		Limit(limit).
		Scan(&results).Error; err != nil {
		return nil, err
	}
	keys := make([]string, len(results))
	for i, r := range results {
		keys[i] = r.Key
	}
	return keys, nil
}

func (r *LeaderboardRepository) InsertTeamBatch(items []*models.TeamLeaderboardItem) error {
	if err := r.db.
		Clauses(clause.OnConflict{DoNothing: true}).
		Create(&items).Error; err != nil {
		return err
	}
	return nil
}

func (r *LeaderboardRepository) GetTeamLeaderboardByInterval(key *models.IntervalKey) ([]*models.TeamLeaderboardItem, error) {
	var items []*models.TeamLeaderboardItem
	if err := r.db.
		Where("\"interval\" IN ?", *key).
		Order("total DESC").
		Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

func (r *LeaderboardRepository) DeleteTeamByInterval(key *models.IntervalKey) error {
	if err := r.db.
		Where("\"interval\" IN ?", *key).
		Delete(models.TeamLeaderboardItem{}).Error; err != nil {
		return err
	}
	return nil
}

func (r *LeaderboardRepository) withPaging(q *gorm.DB, limit, skip int) *gorm.DB {
	if limit > 0 {
		q = q.Where("\"rank\" <= ?", skip+limit)
	}
	if skip > 0 {
		q = q.Where("\"rank\" > ?", skip)
	}
	return q
}
