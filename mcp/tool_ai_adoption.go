package mcp

import (
	"context"
	"fmt"
	"sort"
	"strings"

	mcpgo "github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
	"github.com/muety/wakapi/models"
)

func (s *MCPServer) aiAdoptionTool() (mcpgo.Tool, mcpserver.ToolHandlerFunc) {
	tool := mcpgo.NewTool("get_ai_adoption",
		mcpgo.WithDescription("Mede a adoção de IA no time: percentual do tempo de coding feito com assistência de IA, ranking por membro e variação vs o período anterior. Distingue 'não usou IA' de 'sem telemetria de IA'."),
		mcpgo.WithString("team_id", mcpgo.Required(), mcpgo.Description("ID do time")),
		mcpgo.WithBoolean("compare", mcpgo.Description("Comparar com o período anterior de mesmo tamanho (padrão: true)")),
	)
	tool = addIntervalParams(tool)

	handler := func(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
		teamID, _ := request.RequireString("team_id")
		compare := request.GetBool("compare", true)

		requester, errResult := s.checkTeamAccess(ctx, teamID)
		if errResult != nil {
			return errResult, nil
		}

		from, to, errResult := resolveInterval(request, requester.TZ())
		if errResult != nil {
			return errResult, nil
		}

		members, err := s.teamSrvc.GetMembers(teamID)
		if err != nil {
			return toolError("Erro ao buscar membros do time"), nil
		}

		team, _ := s.teamSrvc.GetByID(teamID)
		teamName := teamID
		if team != nil {
			teamName = team.Name
		}

		members = limitMembers(members)

		current := s.collectAIStats(members, from, to)
		teamStats := aggregateAIStats(current)

		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("Adoção de IA — %s (%s)\n\n", teamName, fmtDateRange(from, to)))

		if !teamStats.Instrumented {
			sb.WriteString("Nenhuma atividade com categoria 'ai coding' registrada neste período.\n")
			sb.WriteString("Isso significa ausência de telemetria de IA, e não necessariamente ausência de uso de IA:\n")
			sb.WriteString("os plugins do time podem não estar reportando a categoria. Não interprete como 0% de adoção.\n")
			return toolResult(sb.String()), nil
		}

		var previous map[string]AIStats
		if compare {
			prevFrom, prevTo := previousPeriod(from, to)
			previous = s.collectAIStats(members, prevFrom, prevTo)
			prevTeam := aggregateAIStats(previous)
			sb.WriteString(fmt.Sprintf("Time: %s de %s com IA (%s)  %s\n",
				fmtDuration(teamStats.AITime),
				fmtDuration(teamStats.CodingTime),
				fmtRatio(teamStats.Ratio),
				fmtRatioBar(teamStats.Ratio, 20),
			))
			sb.WriteString(fmt.Sprintf("Período anterior (%s): %s | Variação do tempo com IA: %s\n\n",
				fmtDateRange(prevFrom, prevTo),
				fmtRatio(prevTeam.Ratio),
				fmtChange(teamStats.AITime, prevTeam.AITime),
			))
		} else {
			sb.WriteString(fmt.Sprintf("Time: %s de %s com IA (%s)  %s\n\n",
				fmtDuration(teamStats.AITime),
				fmtDuration(teamStats.CodingTime),
				fmtRatio(teamStats.Ratio),
				fmtRatioBar(teamStats.Ratio, 20),
			))
		}

		type memberRow struct {
			UserID string
			Stats  AIStats
		}

		rowsData := make([]memberRow, 0, len(current))
		for _, member := range members {
			st, ok := current[member.UserID]
			if !ok {
				continue
			}
			rowsData = append(rowsData, memberRow{UserID: member.UserID, Stats: st})
		}

		// Highest AI ratio first; ties broken by absolute AI time so that a member with
		// 100% of five minutes doesn't outrank one with 60% of a full week.
		sort.Slice(rowsData, func(i, j int) bool {
			if rowsData[i].Stats.Ratio != rowsData[j].Stats.Ratio {
				return rowsData[i].Stats.Ratio > rowsData[j].Stats.Ratio
			}
			return rowsData[i].Stats.AITime > rowsData[j].Stats.AITime
		})

		headers := []string{"Membro", "Tempo IA", "Coding total", "IA %", ""}
		if compare {
			headers = append(headers, "Δ IA")
		}

		rows := make([][]string, 0, len(rowsData))
		for _, r := range rowsData {
			// A member with no AI telemetry gets "--" rather than "0.0%": printing a
			// percentage would assert they didn't use AI, which is not what the data says.
			aiTime, ratio, bar := "--", "--", ""
			if r.Stats.Instrumented {
				aiTime = fmtDuration(r.Stats.AITime)
				ratio = fmtRatio(r.Stats.Ratio)
				bar = fmtRatioBar(r.Stats.Ratio, 10)
			}

			row := []string{r.UserID, aiTime, fmtDuration(r.Stats.CodingTime), ratio, bar}
			if compare {
				delta := "--"
				if r.Stats.Instrumented {
					delta = fmtChange(r.Stats.AITime, previous[r.UserID].AITime)
				}
				row = append(row, delta)
			}
			rows = append(rows, row)
		}

		sb.WriteString(fmtTable(headers, rows))

		if uninstrumented := uninstrumentedMembers(members, current); len(uninstrumented) > 0 {
			sb.WriteString(fmt.Sprintf("\nSem telemetria de IA no período (%d): %s\n",
				len(uninstrumented), strings.Join(uninstrumented, ", ")))
			sb.WriteString("Aparecem como \"--\" na tabela: o dado pode simplesmente não estar sendo reportado pelo plugin.\n")
		}

		return toolResult(sb.String()), nil
	}

	return tool, handler
}

// uninstrumentedMembers lists members with no AI-coding activity recorded at all,
// preserving the team's member order.
func uninstrumentedMembers(members []*models.TeamMember, stats map[string]AIStats) []string {
	out := make([]string, 0)
	for _, member := range members {
		if st, ok := stats[member.UserID]; ok && !st.Instrumented {
			out = append(out, member.UserID)
		}
	}
	return out
}
