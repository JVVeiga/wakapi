package mcp

import (
	"context"
	"fmt"
	"time"

	mcpgo "github.com/mark3labs/mcp-go/mcp"
	conf "github.com/muety/wakapi/config"
	"github.com/muety/wakapi/helpers"
	"github.com/muety/wakapi/models"
)

const (
	maxMembersPerQuery = 50  // max team members to process in a single tool call
	maxDateRangeDays   = 365 // max date range in days
	maxInputLen        = 255 // max length for user-provided string inputs
)

// sanitizeInput truncates user input to prevent log injection and unbounded output.
func sanitizeInput(s string) string {
	if len(s) > maxInputLen {
		return s[:maxInputLen]
	}
	return s
}

// checkTeamAccess verifies the authenticated user is owner/co-owner of the given team.
func (s *MCPServer) checkTeamAccess(ctx context.Context, teamID string) (*models.User, *mcpgo.CallToolResult) {
	user := getPrincipal(ctx)
	if user == nil {
		return nil, toolError("Unauthorized: invalid or missing API key")
	}

	if teamID == "" {
		return nil, toolError("team_id é obrigatório")
	}

	isOwnerOrCoOwner, err := s.teamSrvc.IsTeamOwnerOrCoOwner(sanitizeInput(teamID), user.ID)
	if err != nil || !isOwnerOrCoOwner {
		return nil, toolError("Acesso negado: você não é owner/co-owner deste time")
	}

	return user, nil
}

// checkMemberAccess verifies team access and that the target user is a member of the team.
func (s *MCPServer) checkMemberAccess(ctx context.Context, teamID, userID string) (*models.User, *models.User, *mcpgo.CallToolResult) {
	requester, errResult := s.checkTeamAccess(ctx, teamID)
	if errResult != nil {
		return nil, nil, errResult
	}

	if userID == "" {
		return nil, nil, toolError("user_id é obrigatório")
	}

	isMember, err := s.teamSrvc.IsTeamMember(teamID, sanitizeInput(userID))
	if err != nil || !isMember {
		return nil, nil, toolError("Usuário não é membro deste time")
	}

	targetUser, err := s.userSrvc.GetUserById(userID)
	if err != nil {
		return nil, nil, toolError("Usuário não encontrado")
	}

	return requester, targetUser, nil
}

// resolveInterval resolves the date range from request arguments.
func resolveInterval(request mcpgo.CallToolRequest, tz *time.Location) (time.Time, time.Time, *mcpgo.CallToolResult) {
	interval := request.GetString("interval", "")
	fromStr := request.GetString("from", "")
	toStr := request.GetString("to", "")

	if interval != "" {
		err, from, to := helpers.ResolveIntervalRawTZ(sanitizeInput(interval), tz, time.Monday)
		if err != nil {
			return time.Time{}, time.Time{}, toolError("Intervalo inválido")
		}
		return from, to, nil
	}

	if fromStr != "" && toStr != "" {
		from, err := time.ParseInLocation(conf.SimpleDateFormat, sanitizeInput(fromStr), tz)
		if err != nil {
			return time.Time{}, time.Time{}, toolError("Data 'from' inválida (use YYYY-MM-DD)")
		}
		to, err := time.ParseInLocation(conf.SimpleDateFormat, sanitizeInput(toStr), tz)
		if err != nil {
			return time.Time{}, time.Time{}, toolError("Data 'to' inválida (use YYYY-MM-DD)")
		}
		// Make 'to' inclusive (end of day)
		to = to.Add(24*time.Hour - time.Second)

		// Enforce max date range
		if to.Sub(from).Hours()/24 > float64(maxDateRangeDays) {
			return time.Time{}, time.Time{}, toolError(fmt.Sprintf("Intervalo máximo permitido: %d dias", maxDateRangeDays))
		}

		if !to.After(from) {
			return time.Time{}, time.Time{}, toolError("Data 'to' deve ser posterior a 'from'")
		}

		return from, to, nil
	}

	// Default to last 7 days
	now := time.Now().In(tz)
	return now.AddDate(0, 0, -7), now, nil
}

// limitMembers caps the number of team members to process, returning at most maxMembersPerQuery.
func limitMembers(members []*models.TeamMember) []*models.TeamMember {
	if len(members) > maxMembersPerQuery {
		return members[:maxMembersPerQuery]
	}
	return members
}
