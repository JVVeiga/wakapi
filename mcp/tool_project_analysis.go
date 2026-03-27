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

func (s *MCPServer) projectAnalysisTool() (mcpgo.Tool, mcpserver.ToolHandlerFunc) {
	tool := mcpgo.NewTool("get_project_analysis",
		mcpgo.WithDescription("Mostra quem trabalha em cada projeto do time, com tempo e linguagens. Se 'project' for omitido, lista todos os projetos com contribuidores."),
		mcpgo.WithString("team_id", mcpgo.Required(), mcpgo.Description("ID do time")),
		mcpgo.WithString("project", mcpgo.Description("Nome do projeto (opcional — sem = mostra todos)")),
	)
	tool = addIntervalParams(tool)

	handler := func(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
		teamID, _ := request.RequireString("team_id")
		project := request.GetString("project", "")

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

		if project != "" {
			return s.projectDetailAnalysis(members, sanitizeInput(project), teamName, from, to)
		}
		return s.projectOverviewAnalysis(members, teamName, from, to)
	}

	return tool, handler
}

func (s *MCPServer) projectDetailAnalysis(members []*models.TeamMember, project, teamName string, from, to time.Time) (*mcpgo.CallToolResult, error) {
	type contributor struct {
		UserID    string
		Total     time.Duration
		Languages string
	}

	contributors := make([]contributor, 0)
	var projectTotal time.Duration

	for _, member := range members {
		filters := &models.Filters{Project: models.OrFilter([]string{project})}
		summary, err := s.fetchMemberSummary(member.UserID, from, to, filters)
		if err != nil {
			continue
		}

		total := summary.TotalTime()
		if total == 0 {
			continue
		}

		// Build language breakdown
		langParts := make([]string, 0)
		for i, lang := range summary.Languages {
			if i >= 3 {
				break
			}
			langParts = append(langParts, fmt.Sprintf("%s %s", lang.Key, fmtPercent(lang.Total*time.Second, total)))
		}

		contributors = append(contributors, contributor{
			UserID:    member.UserID,
			Total:     total,
			Languages: strings.Join(langParts, ", "),
		})
		projectTotal += total
	}

	sort.Slice(contributors, func(i, j int) bool {
		return contributors[i].Total > contributors[j].Total
	})

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Projeto: %s (%s) — %s\n\n", project, fmtDateRange(from, to), teamName))

	if len(contributors) == 0 {
		sb.WriteString("Nenhum contribuidor encontrado neste período.")
		return toolResult(sb.String()), nil
	}

	sb.WriteString("Contribuidores:\n")
	for _, c := range contributors {
		sb.WriteString(fmt.Sprintf("  %-20s %10s  (%s)\n", c.UserID, fmtDuration(c.Total), c.Languages))
	}
	sb.WriteString(fmt.Sprintf("\nTotal: %s\n", fmtDuration(projectTotal)))

	return toolResult(sb.String()), nil
}

func (s *MCPServer) projectOverviewAnalysis(members []*models.TeamMember, teamName string, from, to time.Time) (*mcpgo.CallToolResult, error) {
	type projectInfo struct {
		Name         string
		Total        time.Duration
		Contributors []string
	}

	projectMap := make(map[string]*projectInfo)

	for _, member := range members {
		summary, err := s.fetchMemberSummary(member.UserID, from, to, &models.Filters{})
		if err != nil {
			continue
		}

		for _, p := range summary.Projects {
			dur := p.Total * time.Second
			if dur == 0 {
				continue
			}
			if _, ok := projectMap[p.Key]; !ok {
				projectMap[p.Key] = &projectInfo{Name: p.Key}
			}
			projectMap[p.Key].Total += dur
			projectMap[p.Key].Contributors = append(projectMap[p.Key].Contributors, member.UserID)
		}
	}

	projects := make([]*projectInfo, 0, len(projectMap))
	for _, p := range projectMap {
		projects = append(projects, p)
	}
	sort.Slice(projects, func(i, j int) bool {
		return projects[i].Total > projects[j].Total
	})

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Distribuição de Projetos — %s (%s)\n\n", teamName, fmtDateRange(from, to)))

	// Limit output to top 20 projects
	if len(projects) > 20 {
		projects = projects[:20]
	}

	headers := []string{"Projeto", "Tempo Total", "Contribuidores"}
	rows := make([][]string, 0, len(projects))
	for _, p := range projects {
		rows = append(rows, []string{p.Name, fmtDuration(p.Total), strings.Join(p.Contributors, ", ")})
	}

	sb.WriteString(fmtTable(headers, rows))

	return toolResult(sb.String()), nil
}
