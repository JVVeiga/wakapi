package models

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
)

var uuidV4Pattern = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

func TestNewTeam_AssignsUniqueUUIDv4(t *testing.T) {
	a := NewTeam("Team A", "desc", "owner")
	b := NewTeam("Team B", "desc", "owner")

	assert.Regexp(t, uuidV4Pattern, a.ID)
	assert.Regexp(t, uuidV4Pattern, b.ID)
	assert.NotEqual(t, a.ID, b.ID)
	assert.Equal(t, "Team A", a.Name)
	assert.Equal(t, "owner", a.OwnerID)
}
