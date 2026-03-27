package mcp

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	mcpgo "github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
	"github.com/muety/wakapi/models"
)

func (s *MCPServer) teamOverviewTool() (mcpgo.Tool, mcpserver.ToolHandlerFunc) {
	tool := mcpgo.NewTool("get_team_overview",
		mcpgo.WithDescription("Visão agregada do time: tempo total, ranking de membros, top projetos e linguagens."),
		mcpgo.WithString("team_id", mcpgo.Required(), mcpgo.Description("ID do time")),
	)
	tool = addIntervalParams(tool)

	handler := func(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
		teamID, _ := request.RequireString("team_id")

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

		type memberTotal struct {
			UserID string
			Total  time.Duration
		}

		members = limitMembers(members)
		memberSummaries := make([]*models.Summary, 0, len(members))
		memberTotals := make([]memberTotal, 0, len(members))
		activeCount := 0

		for _, member := range members {
			summary, err := s.fetchMemberSummary(member.UserID, from, to, &models.Filters{})
			if err != nil {
				continue
			}
			total := summary.TotalTime()
			memberSummaries = append(memberSummaries, summary)
			memberTotals = append(memberTotals, memberTotal{UserID: member.UserID, Total: total})
			if total > 0 {
				activeCount++
			}
		}

		sort.Slice(memberTotals, func(i, j int) bool {
			return memberTotals[i].Total > memberTotals[j].Total
		})

		aggregated, err := s.summarySrvc.MergeSummariesAcrossUsers(memberSummaries)
		if err != nil {
			aggregated = models.NewEmptySummary()
		}
		aggregated = aggregated.Sorted()

		totalTime := aggregated.TotalTime()

		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("Time: %s (%s)\n", teamName, fmtDateRange(from, to)))
		sb.WriteString(fmt.Sprintf("Tempo total do time: %s | Membros ativos: %d/%d\n\n", fmtDuration(totalTime), activeCount, len(members)))

		sb.WriteString("Ranking:\n")
		for i, mt := range memberTotals {
			sb.WriteString(fmt.Sprintf("  %d. %-20s %s\n", i+1, mt.UserID, fmtDuration(mt.Total)))
		}
		sb.WriteString("\n")

		if len(aggregated.Projects) > 0 {
			sb.WriteString("Top Projetos:     ")
			sb.WriteString(fmtTopItems(aggregated.Projects, 5))
			sb.WriteString("\n")
		}

		if len(aggregated.Languages) > 0 {
			sb.WriteString("Top Linguagens:   ")
			sb.WriteString(fmtTopItems(aggregated.Languages, 5))
			sb.WriteString("\n")
		}

		return toolResult(sb.String()), nil
	}

	return tool, handler
}
