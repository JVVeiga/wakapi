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

func (s *MCPServer) memberSummaryTool() (mcpgo.Tool, mcpserver.ToolHandlerFunc) {
	tool := mcpgo.NewTool("get_member_summary",
		mcpgo.WithDescription("Retorna o resumo detalhado de coding de um membro do time: tempo total, projetos, linguagens, editors, por período."),
		mcpgo.WithString("team_id", mcpgo.Required(), mcpgo.Description("ID do time")),
		mcpgo.WithString("user_id", mcpgo.Required(), mcpgo.Description("ID do membro")),
		mcpgo.WithString("project", mcpgo.Description("Filtrar por projeto")),
		mcpgo.WithString("language", mcpgo.Description("Filtrar por linguagem")),
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

		filters := &models.Filters{}
		if p := request.GetString("project", ""); p != "" {
			filters.Project = models.OrFilter([]string{p})
		}
		if l := request.GetString("language", ""); l != "" {
			filters.Language = models.OrFilter([]string{l})
		}

		summary, err := s.summarySrvc.Aliased(from, to, targetUser, s.summarySrvc.Retrieve, filters, nil, false)
		if err != nil {
			return toolError("Erro ao buscar summary do usuário"), nil
		}

		total := summary.TotalTime()

		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("Summary de %s (%s)\n", userID, fmtDateRange(from, to)))
		sb.WriteString(fmt.Sprintf("Tempo total: %s\n\n", fmtDuration(total)))

		if len(summary.Projects) > 0 {
			sb.WriteString("Projetos:\n")
			sb.WriteString(fmtItems(summary.Projects, total, 10))
			sb.WriteString("\n")
		}

		if len(summary.Languages) > 0 {
			sb.WriteString("Linguagens:\n")
			sb.WriteString(fmtItems(summary.Languages, total, 10))
			sb.WriteString("\n")
		}

		if len(summary.Editors) > 0 {
			sb.WriteString("Editors:\n")
			sb.WriteString(fmtItems(summary.Editors, total, 5))
			sb.WriteString("\n")
		}

		if len(summary.OperatingSystems) > 0 {
			sb.WriteString("Sistemas Operacionais:\n")
			sb.WriteString(fmtItems(summary.OperatingSystems, total, 5))
			sb.WriteString("\n")
		}

		if len(summary.Machines) > 0 {
			sb.WriteString("Máquinas:\n")
			sb.WriteString(fmtItems(summary.Machines, total, 5))
		}

		if total == 0 {
			sb.Reset()
			sb.WriteString(fmt.Sprintf("Nenhuma atividade registrada para %s no período %s.", userID, fmtDateRange(from, to)))
		}

		return toolResult(sb.String()), nil
	}

	return tool, handler
}

// fetchMemberSummary is a shared helper for fetching a team member's summary.
func (s *MCPServer) fetchMemberSummary(userID string, from, to time.Time, filters *models.Filters) (*models.Summary, error) {
	user, err := s.userSrvc.GetUserById(userID)
	if err != nil {
		return nil, err
	}
	return s.summarySrvc.Aliased(from, to, user, s.summarySrvc.Retrieve, filters, nil, to.After(time.Now()))
}
