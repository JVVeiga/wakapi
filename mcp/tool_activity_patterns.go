package mcp

import (
	"context"
	"fmt"
	"strings"
	"time"

	mcpgo "github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
)

func (s *MCPServer) activityPatternsTool() (mcpgo.Tool, mcpserver.ToolHandlerFunc) {
	tool := mcpgo.NewTool("get_activity_patterns",
		mcpgo.WithDescription("Mostra padrões de atividade de um membro: distribuição horária e estatísticas de sessão."),
		mcpgo.WithString("team_id", mcpgo.Required(), mcpgo.Description("ID do time")),
		mcpgo.WithString("user_id", mcpgo.Required(), mcpgo.Description("ID do membro")),
	)
	tool = addIntervalParams(tool)

	handler := func(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
		teamID, _ := request.RequireString("team_id")
		userID, _ := request.RequireString("user_id")

		_, targetUser, errResult := s.checkMemberAccess(ctx, teamID, userID)
		if errResult != nil {
			return errResult, nil
		}

		from, to, errResult := resolveInterval(request, targetUser.TZ())
		if errResult != nil {
			return errResult, nil
		}

		durations, err := s.durationSrvc.Get(from, to, targetUser, nil, nil, false)
		if err != nil {
			return toolError("Erro ao buscar dados de atividade"), nil
		}

		if len(durations) == 0 {
			return toolResult(fmt.Sprintf("Nenhuma atividade registrada para %s no período %s.", userID, fmtDateRange(from, to))), nil
		}

		// Hourly distribution
		hourlyDuration := make([]time.Duration, 24)
		var maxHourly time.Duration
		activeDays := make(map[string]bool)
		var totalDuration time.Duration
		var longestSession time.Duration

		for _, d := range durations {
			t := d.Time.T().In(targetUser.TZ())
			hour := t.Hour()
			hourlyDuration[hour] += d.Duration
			if hourlyDuration[hour] > maxHourly {
				maxHourly = hourlyDuration[hour]
			}
			activeDays[t.Format("2006-01-02")] = true
			totalDuration += d.Duration
			if d.Duration > longestSession {
				longestSession = d.Duration
			}
		}

		totalDays := int(to.Sub(from).Hours()/24) + 1
		avgSession := totalDuration / time.Duration(len(durations))

		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("Padrões de %s (%s)\n\n", userID, fmtDateRange(from, to)))

		sb.WriteString("Distribuição horária:\n")
		for hour := 0; hour < 24; hour++ {
			if hourlyDuration[hour] > 0 {
				sb.WriteString(fmt.Sprintf("  %02d:00  %s %s\n",
					hour,
					fmtBar(hourlyDuration[hour], maxHourly, 10),
					fmtDuration(hourlyDuration[hour]),
				))
			}
		}
		sb.WriteString("\n")

		sb.WriteString(fmt.Sprintf("Sessões: %d total | Média: %s | Mais longa: %s\n",
			len(durations), fmtDuration(avgSession), fmtDuration(longestSession)))
		sb.WriteString(fmt.Sprintf("Dias ativos: %d/%d", len(activeDays), totalDays))

		// Last activity
		if hb, err := s.heartbeatSrvc.GetLatestByUser(targetUser); err == nil && hb != nil {
			sb.WriteString(fmt.Sprintf(" | Última atividade: %s", hb.Time.T().In(targetUser.TZ()).Format("02/01 15:04")))
		}
		sb.WriteString("\n")

		return toolResult(sb.String()), nil
	}

	return tool, handler
}
