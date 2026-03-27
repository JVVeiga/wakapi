package mcp

import (
	"context"
	"fmt"
	"strings"
	"time"

	mcpgo "github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
	"github.com/muety/wakapi/models"
)

func (s *MCPServer) compareMembersTool() (mcpgo.Tool, mcpserver.ToolHandlerFunc) {
	tool := mcpgo.NewTool("compare_members",
		mcpgo.WithDescription("Compara métricas de coding lado a lado entre membros do time."),
		mcpgo.WithString("team_id", mcpgo.Required(), mcpgo.Description("ID do time")),
		mcpgo.WithArray("user_ids", mcpgo.Required(), mcpgo.Description("Lista de IDs de membros para comparar (2-10)")),
	)
	tool = addIntervalParams(tool)

	handler := func(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
		teamID, _ := request.RequireString("team_id")

		requester, errResult := s.checkTeamAccess(ctx, teamID)
		if errResult != nil {
			return errResult, nil
		}

		userIDs, err := request.RequireStringSlice("user_ids")
		if err != nil || len(userIDs) < 2 {
			return toolError("Forneça pelo menos 2 user_ids para comparação"), nil
		}
		if len(userIDs) > 10 {
			userIDs = userIDs[:10]
		}

		from, to, errResult := resolveInterval(request, requester.TZ())
		if errResult != nil {
			return errResult, nil
		}

		team, _ := s.teamSrvc.GetByID(teamID)
		teamName := teamID
		if team != nil {
			teamName = team.Name
		}

		type memberRow struct {
			UserID      string
			Total       time.Duration
			TopProject  string
			TopLanguage string
			NumProjects int
		}

		rows := make([]memberRow, 0, len(userIDs))
		for _, uid := range userIDs {
			uid = sanitizeInput(uid)
			if !requester.IsAdmin {
				isMember, err := s.teamSrvc.IsTeamMember(teamID, uid)
				if err != nil || !isMember {
					continue
				}
			}

			summary, err := s.fetchMemberSummary(uid, from, to, &models.Filters{})
			if err != nil {
				continue
			}

			row := memberRow{
				UserID:      uid,
				Total:       summary.TotalTime(),
				NumProjects: len(summary.Projects),
			}
			if p := maxItem(summary.Projects); p != nil {
				row.TopProject = p.Key
			}
			if l := maxItem(summary.Languages); l != nil {
				row.TopLanguage = l.Key
			}
			rows = append(rows, row)
		}

		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("Comparação (%s) — %s\n\n", fmtDateRange(from, to), teamName))

		headers := []string{"Membro", "Tempo Total", "Top Projeto", "Top Linguagem", "Projetos"}
		tableRows := make([][]string, 0, len(rows))
		for _, r := range rows {
			tableRows = append(tableRows, []string{
				r.UserID,
				fmtDuration(r.Total),
				r.TopProject,
				r.TopLanguage,
				fmt.Sprintf("%d", r.NumProjects),
			})
		}

		sb.WriteString(fmtTable(headers, tableRows))

		return toolResult(sb.String()), nil
	}

	return tool, handler
}
