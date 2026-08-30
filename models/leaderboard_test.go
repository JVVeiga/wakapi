package models

import (
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/muety/wakapi/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// TeamLeaderboardItem keeps TopLanguages in a JSON-encoded column, populated by its BeforeCreate
// and AfterFind hooks. Gorm only calls those when they take a *gorm.DB — with the parameter missing
// it logs a warning and skips them silently, so the column round-trips as empty. These tests pin the
// hooks down by going through a real database rather than calling them directly.

func newTeamLeaderboardTestDb(t *testing.T) *gorm.DB {
	t.Helper()

	config.Set(config.Empty())
	config.Get().Db.Dialect = config.SQLDialectSqlite

	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared&_pragma=foreign_keys(0)"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&TeamLeaderboardItem{}))

	t.Cleanup(func() {
		if sqlDb, err := db.DB(); err == nil {
			sqlDb.Close()
		}
	})
	return db
}

func TestTeamLeaderboardItem_TopLanguagesRoundTrip(t *testing.T) {
	db := newTeamLeaderboardTestDb(t)

	item := &TeamLeaderboardItem{
		TeamID:       "team-1",
		TeamName:     "Team One",
		Interval:     "7_days",
		MemberCount:  3,
		Total:        90 * time.Minute,
		TopLanguages: []string{"Go", "TypeScript", "SQL"},
		CreatedAt:    CustomTime(time.Now()),
	}

	require.NoError(t, db.Create(item).Error)

	// BeforeCreate must have serialized the slice into the column
	var stored string
	require.NoError(t, db.Model(&TeamLeaderboardItem{}).
		Where("team_id = ?", "team-1").
		Pluck("top_languages", &stored).Error)
	assert.Equal(t, `["Go","TypeScript","SQL"]`, stored)

	// AfterFind must have deserialized it back
	var found TeamLeaderboardItem
	require.NoError(t, db.Where("team_id = ?", "team-1").First(&found).Error)
	assert.Equal(t, []string{"Go", "TypeScript", "SQL"}, found.TopLanguages)
	assert.Equal(t, 3, found.MemberCount)
	assert.Equal(t, 90*time.Minute, found.Total)
}

func TestTeamLeaderboardItem_NoTopLanguages(t *testing.T) {
	db := newTeamLeaderboardTestDb(t)

	item := &TeamLeaderboardItem{
		TeamID:      "team-empty",
		TeamName:    "Team Empty",
		Interval:    "7_days",
		MemberCount: 1,
		Total:       time.Minute,
		CreatedAt:   CustomTime(time.Now()),
	}
	require.NoError(t, db.Create(item).Error)

	var found TeamLeaderboardItem
	require.NoError(t, db.Where("team_id = ?", "team-empty").First(&found).Error)
	assert.Empty(t, found.TopLanguages)
}

func TestTeamLeaderboardItem_HooksSatisfyGormInterfaces(t *testing.T) {
	// Compile-time guard: if either signature loses its *gorm.DB parameter, gorm stops calling the
	// hook and only logs a warning, which no other test would catch.
	var item any = &TeamLeaderboardItem{}

	_, isBeforeCreate := item.(interface{ BeforeCreate(*gorm.DB) error })
	_, isAfterFind := item.(interface{ AfterFind(*gorm.DB) error })

	assert.True(t, isBeforeCreate, "TeamLeaderboardItem must implement BeforeCreate(*gorm.DB) error")
	assert.True(t, isAfterFind, "TeamLeaderboardItem must implement AfterFind(*gorm.DB) error")
}
