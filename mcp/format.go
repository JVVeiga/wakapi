package mcp

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/muety/wakapi/models"
)

func fmtDuration(d time.Duration) string {
	d = d.Round(time.Minute)
	h := d / time.Hour
	d -= h * time.Hour
	m := d / time.Minute
	if h > 0 {
		return fmt.Sprintf("%dh %02dmin", h, m)
	}
	return fmt.Sprintf("%dmin", m)
}

func fmtPercent(part, total time.Duration) string {
	if total == 0 {
		return "0.0%"
	}
	return fmt.Sprintf("%.1f%%", float64(part)/float64(total)*100)
}

func fmtDateRange(from, to time.Time) string {
	return fmt.Sprintf("%s a %s", from.Format("02/01/2006"), to.Format("02/01/2006"))
}

func fmtItems(items []*models.SummaryItem, total time.Duration, limit int) string {
	var sb strings.Builder
	for i, item := range items {
		if limit > 0 && i >= limit {
			break
		}
		dur := item.Total * time.Second
		sb.WriteString(fmt.Sprintf("  %-20s %10s (%s)\n", item.Key, fmtDuration(dur), fmtPercent(dur, total)))
	}
	return sb.String()
}

func fmtTopItems(items []*models.SummaryItem, limit int) string {
	parts := make([]string, 0, limit)
	for i, item := range items {
		if i >= limit {
			break
		}
		parts = append(parts, fmt.Sprintf("%s (%s)", item.Key, fmtDuration(item.Total*time.Second)))
	}
	return strings.Join(parts, ", ")
}

func fmtTable(headers []string, rows [][]string) string {
	widths := make([]int, len(headers))
	for i, h := range headers {
		widths[i] = len(h)
	}
	for _, row := range rows {
		for i, cell := range row {
			if i < len(widths) && len(cell) > widths[i] {
				widths[i] = len(cell)
			}
		}
	}

	var sb strings.Builder

	// header
	for i, h := range headers {
		if i > 0 {
			sb.WriteString(" | ")
		}
		sb.WriteString(fmt.Sprintf("%-*s", widths[i], h))
	}
	sb.WriteString("\n")

	// separator
	for i, w := range widths {
		if i > 0 {
			sb.WriteString("-+-")
		}
		sb.WriteString(strings.Repeat("-", w))
	}
	sb.WriteString("\n")

	// rows
	for _, row := range rows {
		for i, cell := range row {
			if i > 0 {
				sb.WriteString(" | ")
			}
			if i < len(widths) {
				sb.WriteString(fmt.Sprintf("%-*s", widths[i], cell))
			}
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

func fmtChange(current, previous time.Duration) string {
	if previous == 0 {
		if current > 0 {
			return "+100%"
		}
		return "--"
	}
	change := float64(current-previous) / float64(previous) * 100
	if math.Abs(change) < 0.1 {
		return "0.0%"
	}
	return fmt.Sprintf("%+.1f%%", change)
}

func fmtBar(value, max time.Duration, width int) string {
	if max == 0 {
		return strings.Repeat("░", width)
	}
	filled := int(float64(value) / float64(max) * float64(width))
	if filled > width {
		filled = width
	}
	return strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
}

func maxItem(items []*models.SummaryItem) *models.SummaryItem {
	if len(items) == 0 {
		return nil
	}
	max := items[0]
	for _, item := range items[1:] {
		if item.Total > max.Total {
			max = item
		}
	}
	return max
}
