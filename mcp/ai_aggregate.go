package mcp

import (
	"strings"
	"time"

	"github.com/muety/wakapi/models"
)

const (
	categoryAICoding = "ai coding"
	categoryCoding   = "coding"
)

// AIStats holds category-derived AI usage metrics for a single entity (member, project or team).
type AIStats struct {
	AITime     time.Duration
	CodingTime time.Duration // "coding" + "ai coding", i.e. the denominator of Ratio
	Ratio      float64

	// Instrumented reports whether any AI-coding activity was recorded at all.
	// A zero ratio on an instrumented entity means "did not use AI"; on a
	// non-instrumented one it means "we have no data" — the two must not be conflated,
	// otherwise a client model reads missing telemetry as "the team doesn't use AI".
	Instrumented bool
}

// aiStatsFromSummary derives AI usage from a summary's category items, mirroring the
// ratio shown in the web UI: CategoryRatio("ai coding", "ai coding", "coding").
// Summary item totals are stored in seconds, hence the conversion.
func aiStatsFromSummary(s *models.Summary) AIStats {
	var stats AIStats
	if s == nil {
		return stats
	}

	for _, item := range s.Categories {
		dur := item.Total * time.Second
		switch strings.ToLower(strings.TrimSpace(item.Key)) {
		case categoryAICoding:
			stats.AITime += dur
			stats.CodingTime += dur
			stats.Instrumented = true
		case categoryCoding:
			stats.CodingTime += dur
		}
	}

	stats.Ratio = ratioOf(stats.AITime, stats.CodingTime)
	return stats
}

// add merges another AIStats into this one, recomputing the ratio over the merged totals.
func (a AIStats) add(other AIStats) AIStats {
	a.AITime += other.AITime
	a.CodingTime += other.CodingTime
	a.Instrumented = a.Instrumented || other.Instrumented
	a.Ratio = ratioOf(a.AITime, a.CodingTime)
	return a
}

func ratioOf(part, total time.Duration) float64 {
	if total <= 0 {
		return 0
	}
	return float64(part) / float64(total)
}

// previousPeriod returns the window of the same length immediately preceding [from, to).
func previousPeriod(from, to time.Time) (time.Time, time.Time) {
	length := to.Sub(from)
	return from.Add(-length), from
}

// collectAIStats fetches per-member AI stats for a window, keyed by user id.
// Members whose summary cannot be fetched are skipped, matching the behaviour of
// the other team-wide tools.
func (s *MCPServer) collectAIStats(members []*models.TeamMember, from, to time.Time) map[string]AIStats {
	stats := make(map[string]AIStats, len(members))
	for _, member := range members {
		summary, err := s.fetchMemberSummary(member.UserID, from, to, &models.Filters{})
		if err != nil {
			continue
		}
		stats[member.UserID] = aiStatsFromSummary(summary)
	}
	return stats
}

// aggregateAIStats folds a per-member map into a single team-level AIStats.
func aggregateAIStats(stats map[string]AIStats) AIStats {
	var total AIStats
	for _, st := range stats {
		total = total.add(st)
	}
	return total
}

// fetchMemberDurations loads a member's durations for a window. Unlike summaries,
// durations keep the project and category on the same record, which lets a single
// query per member yield the whole project × category matrix.
func (s *MCPServer) fetchMemberDurations(userID string, from, to time.Time, filters *models.Filters) (models.Durations, error) {
	user, err := s.userSrvc.GetUserById(userID)
	if err != nil {
		return nil, err
	}
	if filters == nil {
		filters = &models.Filters{}
	}
	return s.durationSrvc.Get(from, to, user, filters, nil, false)
}

// aiStatsByProject folds durations into per-project AI stats.
// Duration.Duration is a real duration (unlike SummaryItem.Total, which holds seconds),
// so no unit conversion is applied here.
func aiStatsByProject(durations models.Durations) map[string]AIStats {
	byProject := make(map[string]AIStats)

	for _, d := range durations {
		project := d.Project
		if project == "" {
			project = models.UnknownSummaryKey
		}

		stats := byProject[project]
		switch strings.ToLower(strings.TrimSpace(d.Category)) {
		case categoryAICoding:
			stats.AITime += d.Duration
			stats.CodingTime += d.Duration
			stats.Instrumented = true
		case categoryCoding:
			stats.CodingTime += d.Duration
		default:
			continue
		}
		stats.Ratio = ratioOf(stats.AITime, stats.CodingTime)
		byProject[project] = stats
	}

	return byProject
}
