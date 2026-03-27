package mcp

import (
	"testing"
	"time"

	"github.com/muety/wakapi/models"
	"github.com/stretchr/testify/assert"
)

func TestFmtDuration(t *testing.T) {
	assert.Equal(t, "0min", fmtDuration(0))
	assert.Equal(t, "5min", fmtDuration(5*time.Minute))
	assert.Equal(t, "1h 00min", fmtDuration(1*time.Hour))
	assert.Equal(t, "2h 30min", fmtDuration(2*time.Hour+30*time.Minute))
	assert.Equal(t, "10h 05min", fmtDuration(10*time.Hour+5*time.Minute))
	assert.Equal(t, "0min", fmtDuration(29*time.Second)) // rounds down
}

func TestFmtPercent(t *testing.T) {
	assert.Equal(t, "50.0%", fmtPercent(50*time.Minute, 100*time.Minute))
	assert.Equal(t, "100.0%", fmtPercent(100*time.Minute, 100*time.Minute))
	assert.Equal(t, "0.0%", fmtPercent(0, 100*time.Minute))
	assert.Equal(t, "0.0%", fmtPercent(0, 0))
	assert.Equal(t, "33.3%", fmtPercent(1*time.Hour, 3*time.Hour))
}

func TestFmtDateRange(t *testing.T) {
	from := time.Date(2024, 3, 20, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 3, 27, 0, 0, 0, 0, time.UTC)
	assert.Equal(t, "20/03/2024 a 27/03/2024", fmtDateRange(from, to))
}

func TestFmtItems(t *testing.T) {
	items := []*models.SummaryItem{
		{Key: "Go", Total: time.Duration(3600)},       // 3600 seconds = 1h
		{Key: "Python", Total: time.Duration(1800)},    // 1800 seconds = 30min
		{Key: "Rust", Total: time.Duration(900)},       // 900 seconds = 15min
	}
	total := 6300 * time.Second // 1h45min

	result := fmtItems(items, total, 2)
	assert.Contains(t, result, "Go")
	assert.Contains(t, result, "Python")
	assert.NotContains(t, result, "Rust") // limit=2

	result = fmtItems(items, total, 0)
	assert.NotEmpty(t, result) // limit=0 means no limit, shows all
}

func TestFmtTopItems(t *testing.T) {
	items := []*models.SummaryItem{
		{Key: "Go", Total: time.Duration(3600)},
		{Key: "Python", Total: time.Duration(1800)},
		{Key: "Rust", Total: time.Duration(900)},
	}

	result := fmtTopItems(items, 2)
	assert.Contains(t, result, "Go")
	assert.Contains(t, result, "Python")
	assert.NotContains(t, result, "Rust")
}

func TestFmtTable(t *testing.T) {
	headers := []string{"Name", "Value"}
	rows := [][]string{
		{"Alice", "100"},
		{"Bob", "200"},
	}

	result := fmtTable(headers, rows)
	assert.Contains(t, result, "Name")
	assert.Contains(t, result, "Value")
	assert.Contains(t, result, "Alice")
	assert.Contains(t, result, "Bob")
	assert.Contains(t, result, "---") // separator
}

func TestFmtChange(t *testing.T) {
	assert.Equal(t, "+100.0%", fmtChange(200*time.Minute, 100*time.Minute))
	assert.Equal(t, "-50.0%", fmtChange(50*time.Minute, 100*time.Minute))
	assert.Equal(t, "0.0%", fmtChange(100*time.Minute, 100*time.Minute))
	assert.Equal(t, "--", fmtChange(0, 0))
	assert.Equal(t, "+100%", fmtChange(100*time.Minute, 0))
}

func TestFmtBar(t *testing.T) {
	bar := fmtBar(50*time.Minute, 100*time.Minute, 10)
	assert.Equal(t, 10, len([]rune(bar)))
	assert.Contains(t, bar, "█")

	bar = fmtBar(0, 100*time.Minute, 10)
	assert.Equal(t, "░░░░░░░░░░", bar)

	bar = fmtBar(100*time.Minute, 100*time.Minute, 10)
	assert.Equal(t, "██████████", bar)

	bar = fmtBar(50*time.Minute, 0, 10)
	assert.Equal(t, "░░░░░░░░░░", bar) // max=0
}

func TestMaxItem(t *testing.T) {
	items := []*models.SummaryItem{
		{Key: "Go", Total: time.Duration(100)},
		{Key: "Python", Total: time.Duration(300)},
		{Key: "Rust", Total: time.Duration(200)},
	}

	max := maxItem(items)
	assert.Equal(t, "Python", max.Key)

	assert.Nil(t, maxItem(nil))
	assert.Nil(t, maxItem([]*models.SummaryItem{}))
}
