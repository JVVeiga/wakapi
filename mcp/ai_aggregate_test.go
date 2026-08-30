package mcp

import (
	"testing"
	"time"

	"github.com/muety/wakapi/models"
	"github.com/stretchr/testify/assert"
)

// summary item totals are stored in seconds, hence the raw integers here
func categorySummary(pairs ...any) *models.Summary {
	items := make([]*models.SummaryItem, 0, len(pairs)/2)
	for i := 0; i+1 < len(pairs); i += 2 {
		items = append(items, &models.SummaryItem{
			Type:  models.SummaryCategory,
			Key:   pairs[i].(string),
			Total: time.Duration(pairs[i+1].(int)),
		})
	}
	return &models.Summary{Categories: items}
}

func TestAIStatsFromSummary_WithAI(t *testing.T) {
	s := categorySummary("coding", 3600, "ai coding", 1200, "browsing", 600)
	stats := aiStatsFromSummary(s)

	assert.True(t, stats.Instrumented)
	assert.Equal(t, 20*time.Minute, stats.AITime)
	assert.Equal(t, 80*time.Minute, stats.CodingTime) // 3600s + 1200s
	assert.InDelta(t, 0.25, stats.Ratio, 0.0001)
}

func TestAIStatsFromSummary_NoAICategory(t *testing.T) {
	s := categorySummary("coding", 3600, "browsing", 600)
	stats := aiStatsFromSummary(s)

	assert.False(t, stats.Instrumented, "sem categoria 'ai coding' o membro não está instrumentado")
	assert.Equal(t, time.Duration(0), stats.AITime)
	assert.Equal(t, time.Hour, stats.CodingTime)
	assert.Equal(t, float64(0), stats.Ratio)
}

func TestAIStatsFromSummary_NoCategoriesAtAll(t *testing.T) {
	stats := aiStatsFromSummary(&models.Summary{})

	assert.False(t, stats.Instrumented)
	assert.Equal(t, time.Duration(0), stats.CodingTime)
	assert.Equal(t, float64(0), stats.Ratio, "sem denominador o ratio é 0, não NaN")
}

func TestAIStatsFromSummary_Nil(t *testing.T) {
	stats := aiStatsFromSummary(nil)
	assert.False(t, stats.Instrumented)
	assert.Equal(t, float64(0), stats.Ratio)
}

func TestAIStatsFromSummary_OnlyAICoding(t *testing.T) {
	s := categorySummary("ai coding", 1800)
	stats := aiStatsFromSummary(s)

	assert.True(t, stats.Instrumented)
	assert.Equal(t, 30*time.Minute, stats.AITime)
	assert.Equal(t, 30*time.Minute, stats.CodingTime)
	assert.InDelta(t, 1.0, stats.Ratio, 0.0001)
}

func TestAIStatsFromSummary_CaseAndSpacing(t *testing.T) {
	s := categorySummary(" AI Coding ", 600, "CODING", 1800)
	stats := aiStatsFromSummary(s)

	assert.True(t, stats.Instrumented)
	assert.Equal(t, 10*time.Minute, stats.AITime)
	assert.Equal(t, 40*time.Minute, stats.CodingTime)
}

func TestAIStats_Add(t *testing.T) {
	a := AIStats{AITime: time.Hour, CodingTime: 4 * time.Hour, Ratio: 0.25, Instrumented: true}
	b := AIStats{AITime: 0, CodingTime: 4 * time.Hour, Ratio: 0}

	merged := a.add(b)
	assert.Equal(t, time.Hour, merged.AITime)
	assert.Equal(t, 8*time.Hour, merged.CodingTime)
	assert.InDelta(t, 0.125, merged.Ratio, 0.0001, "o ratio é recalculado sobre os totais, não somado")
	assert.True(t, merged.Instrumented)
}

func TestAggregateAIStats(t *testing.T) {
	stats := map[string]AIStats{
		"alice": {AITime: time.Hour, CodingTime: 2 * time.Hour, Instrumented: true},
		"bob":   {AITime: 0, CodingTime: 2 * time.Hour},
	}
	total := aggregateAIStats(stats)

	assert.Equal(t, time.Hour, total.AITime)
	assert.Equal(t, 4*time.Hour, total.CodingTime)
	assert.InDelta(t, 0.25, total.Ratio, 0.0001)
	assert.True(t, total.Instrumented, "basta um membro instrumentado para o time contar como instrumentado")
}

func TestAggregateAIStats_NoneInstrumented(t *testing.T) {
	stats := map[string]AIStats{
		"alice": {CodingTime: 2 * time.Hour},
		"bob":   {CodingTime: 2 * time.Hour},
	}
	assert.False(t, aggregateAIStats(stats).Instrumented)
}

func TestPreviousPeriod(t *testing.T) {
	from := time.Date(2026, 8, 8, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 8, 15, 0, 0, 0, 0, time.UTC)

	prevFrom, prevTo := previousPeriod(from, to)
	assert.Equal(t, time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC), prevFrom)
	assert.Equal(t, from, prevTo)
	assert.Equal(t, to.Sub(from), prevTo.Sub(prevFrom))
}

func TestAIStatsByProject(t *testing.T) {
	durations := models.Durations{
		&models.Duration{Project: "wakapi", Category: "coding", Duration: time.Hour},
		&models.Duration{Project: "wakapi", Category: "ai coding", Duration: time.Hour},
		&models.Duration{Project: "api", Category: "coding", Duration: 2 * time.Hour},
		&models.Duration{Project: "api", Category: "browsing", Duration: 3 * time.Hour},
		&models.Duration{Project: "", Category: "ai coding", Duration: 30 * time.Minute},
	}

	byProject := aiStatsByProject(durations)

	assert.InDelta(t, 0.5, byProject["wakapi"].Ratio, 0.0001)
	assert.Equal(t, 2*time.Hour, byProject["wakapi"].CodingTime)
	assert.True(t, byProject["wakapi"].Instrumented)

	assert.Equal(t, 2*time.Hour, byProject["api"].CodingTime, "'browsing' não entra no denominador")
	assert.False(t, byProject["api"].Instrumented)

	assert.Equal(t, 30*time.Minute, byProject[models.UnknownSummaryKey].AITime, "projeto vazio cai em 'unknown'")
}

func TestAIStatsByProject_Empty(t *testing.T) {
	assert.Empty(t, aiStatsByProject(models.Durations{}))
}
