package mcp

import (
	"context"
	"fmt"
	"strings"
	"time"

	mcpgo "github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
	conf "github.com/muety/wakapi/config"
	"github.com/muety/wakapi/helpers"
	"github.com/muety/wakapi/models"
)

func (s *MCPServer) trendAnalysisTool() (mcpgo.Tool, mcpserver.ToolHandlerFunc) {
	tool := mcpgo.NewTool("get_trend_analysis",
		mcpgo.WithDescription("Compara dois períodos para identificar tendências. Sem user_id = tendência do time todo."),
		mcpgo.WithString("team_id", mcpgo.Required(), mcpgo.Description("ID do time")),
		mcpgo.WithString("user_id", mcpgo.Description("ID do membro (opcional — sem = time todo)")),
		mcpgo.WithString("current_interval", mcpgo.Description("Intervalo atual: week, month, 7_days, 30_days (default: week)")),
		mcpgo.WithString("previous_interval", mcpgo.Description("Intervalo anterior: last_week, last_month (default: last_week)")),
		mcpgo.WithString("current_from", mcpgo.Description("Data início do período atual (YYYY-MM-DD)")),
		mcpgo.WithString("current_to", mcpgo.Description("Data fim do período atual (YYYY-MM-DD)")),
		mcpgo.WithString("previous_from", mcpgo.Description("Data início do período anterior (YYYY-MM-DD)")),
		mcpgo.WithString("previous_to", mcpgo.Description("Data fim do período anterior (YYYY-MM-DD)")),
	)

	handler := func(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
		teamID, _ := request.RequireString("team_id")
		userID := request.GetString("user_id", "")

		requester, errResult := s.checkTeamAccess(ctx, teamID)
		if errResult != nil {
			return errResult, nil
		}

		tz := requester.TZ()

		// Resolve current period
		curFrom, curTo, errResult := resolveTrendInterval(
			request.GetString("current_interval", "week"),
			request.GetString("current_from", ""),
			request.GetString("current_to", ""),
			tz,
		)
		if errResult != nil {
			return errResult, nil
		}

		// Resolve previous period
		prevFrom, prevTo, errResult := resolveTrendInterval(
			request.GetString("previous_interval", "last_week"),
			request.GetString("previous_from", ""),
			request.GetString("previous_to", ""),
			tz,
		)
		if errResult != nil {
			return errResult, nil
		}

		var curSummary, prevSummary *models.Summary
		var label string

		if userID != "" {
			// Single member trend
			_, _, errResult := s.checkMemberAccess(ctx, teamID, userID)
			if errResult != nil {
				return errResult, nil
			}

			curSummary, _ = s.fetchMemberSummary(userID, curFrom, curTo, &models.Filters{})
			prevSummary, _ = s.fetchMemberSummary(userID, prevFrom, prevTo, &models.Filters{})
			label = userID
		} else {
			// Team-wide trend
			members, err := s.teamSrvc.GetMembers(teamID)
			if err != nil {
				return toolError("Erro ao buscar membros"), nil
			}

			members = limitMembers(members)
			curSummaries := make([]*models.Summary, 0)
			prevSummaries := make([]*models.Summary, 0)
			for _, m := range members {
				if cs, err := s.fetchMemberSummary(m.UserID, curFrom, curTo, &models.Filters{}); err == nil {
					curSummaries = append(curSummaries, cs)
				}
				if ps, err := s.fetchMemberSummary(m.UserID, prevFrom, prevTo, &models.Filters{}); err == nil {
					prevSummaries = append(prevSummaries, ps)
				}
			}

			curSummary, _ = s.summarySrvc.MergeSummariesAcrossUsers(curSummaries)
			prevSummary, _ = s.summarySrvc.MergeSummariesAcrossUsers(prevSummaries)

			team, _ := s.teamSrvc.GetByID(teamID)
			if team != nil {
				label = team.Name
			} else {
				label = "Time"
			}
		}

		if curSummary == nil {
			curSummary = models.NewEmptySummary()
		}
		if prevSummary == nil {
			prevSummary = models.NewEmptySummary()
		}

		curTotal := curSummary.TotalTime()
		prevTotal := prevSummary.TotalTime()

		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("Tendência: %s\n", label))
		sb.WriteString(fmt.Sprintf("Atual: %s | Anterior: %s\n\n", fmtDateRange(curFrom, curTo), fmtDateRange(prevFrom, prevTo)))

		headers := []string{"", "Atual", "Anterior", "Variação"}
		rows := [][]string{
			{"Tempo Total", fmtDuration(curTotal), fmtDuration(prevTotal), fmtChange(curTotal, prevTotal)},
		}

		// Top languages comparison
		langMap := make(map[string][2]time.Duration)
		for _, l := range curSummary.Languages {
			langMap[l.Key] = [2]time.Duration{l.Total * time.Second, 0}
		}
		for _, l := range prevSummary.Languages {
			if v, ok := langMap[l.Key]; ok {
				v[1] = l.Total * time.Second
				langMap[l.Key] = v
			} else {
				langMap[l.Key] = [2]time.Duration{0, l.Total * time.Second}
			}
		}
		for lang, durations := range langMap {
			if durations[0] > 0 || durations[1] > 0 {
				rows = append(rows, []string{
					lang,
					fmtDuration(durations[0]),
					fmtDuration(durations[1]),
					fmtChange(durations[0], durations[1]),
				})
			}
		}

		sb.WriteString(fmtTable(headers, rows))

		// Partial period warning
		now := time.Now().In(tz)
		if now.Before(curTo) {
			totalDays := int(curTo.Sub(curFrom).Hours()/24) + 1
			elapsedDays := int(now.Sub(curFrom).Hours()/24) + 1
			if elapsedDays < totalDays {
				sb.WriteString(fmt.Sprintf("\n⚠ Período atual parcial (%d de %d dias).\n", elapsedDays, totalDays))
			}
		}

		return toolResult(sb.String()), nil
	}

	return tool, handler
}

func resolveTrendInterval(interval, fromStr, toStr string, tz *time.Location) (time.Time, time.Time, *mcpgo.CallToolResult) {
	if fromStr != "" && toStr != "" {
		from, err := time.ParseInLocation(conf.SimpleDateFormat, sanitizeInput(fromStr), tz)
		if err != nil {
			return time.Time{}, time.Time{}, toolError("Data inválida (use YYYY-MM-DD)")
		}
		to, err := time.ParseInLocation(conf.SimpleDateFormat, sanitizeInput(toStr), tz)
		if err != nil {
			return time.Time{}, time.Time{}, toolError("Data inválida (use YYYY-MM-DD)")
		}
		to = to.Add(24*time.Hour - time.Second)
		return from, to, nil
	}

	if interval != "" {
		err, from, to := helpers.ResolveIntervalRawTZ(sanitizeInput(interval), tz, time.Monday)
		if err != nil {
			return time.Time{}, time.Time{}, toolError("Intervalo inválido")
		}
		return from, to, nil
	}

	return time.Time{}, time.Time{}, toolError("Forneça 'interval' ou 'from'/'to'")
}
