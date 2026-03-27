package mcp

import (
	"context"
	"fmt"
	"strings"

	mcpgo "github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
	"github.com/muety/wakapi/models"
)

func (s *MCPServer) listTeamsTool() (mcpgo.Tool, mcpserver.ToolHandlerFunc) {
	tool := mcpgo.NewTool("list_teams",
		mcpgo.WithDescription("Lista os times que você lidera (owner ou co-owner). Admins veem todos os times. Use como ponto de partida para descobrir team_id e user_id dos membros."),
	)

	handler := func(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
		user := getPrincipal(ctx)
		if user == nil {
			return toolError("Unauthorized"), nil
		}

		var teams []*models.Team
		var err error

		if user.IsAdmin {
			teams, err = s.teamSrvc.GetAll()
		} else {
			teams, err = s.teamSrvc.GetByUser(user.ID)
		}
		if err != nil {
			return toolError("Erro ao buscar times"), nil
		}

		var sb strings.Builder
		if user.IsAdmin {
			sb.WriteString("Todos os times (admin):\n\n")
		} else {
			sb.WriteString("Times que você lidera:\n\n")
		}

		count := 0
		for _, team := range teams {
			if !user.IsAdmin {
				isOwnerOrCoOwner, _ := s.teamSrvc.IsTeamOwnerOrCoOwner(team.ID, user.ID)
				if !isOwnerOrCoOwner {
					continue
				}
			}
			count++

			role := "admin"
			if !user.IsAdmin {
				role = "co-owner"
				if team.OwnerID == user.ID {
					role = "owner"
				}
			}

			members, _ := s.teamSrvc.GetMembers(team.ID)
			memberNames := make([]string, 0, len(members))
			for _, m := range members {
				memberNames = append(memberNames, m.UserID)
			}

			sb.WriteString(fmt.Sprintf("%d. %s (ID: %s) — %s\n", count, team.Name, team.ID, role))
			sb.WriteString(fmt.Sprintf("   Membros: %s (%d)\n\n", strings.Join(memberNames, ", "), len(members)))
		}

		if count == 0 {
			sb.Reset()
			sb.WriteString("Nenhum time encontrado.")
		}

		return toolResult(sb.String()), nil
	}

	return tool, handler
}
