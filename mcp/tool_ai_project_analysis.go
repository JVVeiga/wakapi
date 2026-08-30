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

const maxAIProjectsListed = 20

func (s *MCPServer) aiProjectAnalysisTool() (mcpgo.Tool, mcpserver.ToolHandlerFunc) {
	tool := mcpgo.NewTool("get_ai_project_analysis",
		mcpgo.WithDescription("Mostra o percentual de coding assistido por IA em cada projeto do time. Se 'project' for informado, detalha os contribuidores desse projeto e o quanto cada um usou IA nele."),
		mcpgo.WithString("team_id", mcpgo.Required(), mcpgo.Description("ID do time")),
		mcpgo.WithString("project", mcpgo.Description("Nome do projeto (opcional — sem = mostra todos)")),
	)
	tool = addIntervalParams(tool)

	handler := func(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
		teamID, _ := request.RequireString("team_id")
		project := sanitizeInput(request.GetString("project", ""))

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

		// One durations query per member yields the full project × category matrix,
		// which avoids the members × projects fan-out a summary-based approach would need.
		perMember := make(map[string]map[string]AIStats, len(members))
		anyInstrumented := false

		for _, member := range members {
			durations, err := s.fetchMemberDurations(member.UserID, from, to, &models.Filters{})
			if err != nil {
				continue
			}
			byProject := aiStatsByProject(durations)
			perMember[member.UserID] = byProject
			for _, st := range byProject {
				if st.Instrumented {
					anyInstrumented = true
					break
				}
			}
		}

		var sb strings.Builder

		if project != "" {
			sb.WriteString(fmt.Sprintf("Uso de IA no projeto %s — %s (%s)\n\n", project, teamName, fmtDateRange(from, to)))
		} else {
			sb.WriteString(fmt.Sprintf("Uso de IA por projeto — %s (%s)\n\n", teamName, fmtDateRange(from, to)))
		}

		if !anyInstrumented {
			sb.WriteString("Nenhuma atividade com categoria 'ai coding' registrada neste período.\n")
			sb.WriteString("Ausência de telemetria de IA não é o mesmo que ausência de uso de IA — não leia como 0% de adoção.\n")
			return toolResult(sb.String()), nil
		}

		if project != "" {
			return aiProjectDetail(&sb, members, perMember, project), nil
		}
		return aiProjectOverview(&sb, perMember), nil
	}

	return tool, handler
}

// aiProjectOverview ranks the team's projects by total coding time, showing the AI ratio of each.
func aiProjectOverview(sb *strings.Builder, perMember map[string]map[string]AIStats) *mcpgo.CallToolResult {
	type projectRow struct {
		Name         string
		Stats        AIStats
		Contributors int
	}

	totals := make(map[string]*projectRow)
	for _, byProject := range perMember {
		for name, st := range byProject {
			if _, ok := totals[name]; !ok {
				totals[name] = &projectRow{Name: name}
			}
			totals[name].Stats = totals[name].Stats.add(st)
			totals[name].Contributors++
		}
	}

	projects := make([]*projectRow, 0, len(totals))
	for _, p := range totals {
		if p.Stats.CodingTime == 0 {
			continue
		}
		projects = append(projects, p)
	}

	// Ranked by coding volume, not by ratio: a project with 2h of work and 100% AI
	// is less informative to a team lead than the project everyone actually works in.
	sort.Slice(projects, func(i, j int) bool {
		return projects[i].Stats.CodingTime > projects[j].Stats.CodingTime
	})

	if len(projects) > maxAIProjectsListed {
		projects = projects[:maxAIProjectsListed]
	}

	headers := []string{"Projeto", "Tempo IA", "Coding total", "IA %", "", "Contribuidores"}
	rows := make([][]string, 0, len(projects))
	for _, p := range projects {
		// "--" rather than "0.0%" for projects without AI telemetry — see aiAdoptionTool.
		aiTime, ratio, bar := "--", "--", ""
		if p.Stats.Instrumented {
			aiTime = fmtDuration(p.Stats.AITime)
			ratio = fmtRatio(p.Stats.Ratio)
			bar = fmtRatioBar(p.Stats.Ratio, 10)
		}
		rows = append(rows, []string{
			p.Name, aiTime, fmtDuration(p.Stats.CodingTime), ratio, bar,
			fmt.Sprintf("%d", p.Contributors),
		})
	}

	sb.WriteString(fmtTable(headers, rows))
	return toolResult(sb.String())
}

// aiProjectDetail breaks a single project down by contributor.
func aiProjectDetail(sb *strings.Builder, members []*models.TeamMember, perMember map[string]map[string]AIStats, project string) *mcpgo.CallToolResult {
	type contributorRow struct {
		UserID string
		Stats  AIStats
	}

	contributors := make([]contributorRow, 0, len(members))
	var projectTotal AIStats

	for _, member := range members {
		byProject, ok := perMember[member.UserID]
		if !ok {
			continue
		}
		st, ok := lookupProject(byProject, project)
		if !ok || st.CodingTime == 0 {
			continue
		}
		contributors = append(contributors, contributorRow{UserID: member.UserID, Stats: st})
		projectTotal = projectTotal.add(st)
	}

	if len(contributors) == 0 {
		sb.WriteString("Nenhum contribuidor com tempo de coding neste projeto no período.")
		return toolResult(sb.String())
	}

	sb.WriteString(fmt.Sprintf("Projeto: %s de %s com IA (%s)  %s\n\n",
		fmtDuration(projectTotal.AITime),
		fmtDuration(projectTotal.CodingTime),
		fmtRatio(projectTotal.Ratio),
		fmtRatioBar(projectTotal.Ratio, 20),
	))

	sort.Slice(contributors, func(i, j int) bool {
		if contributors[i].Stats.Ratio != contributors[j].Stats.Ratio {
			return contributors[i].Stats.Ratio > contributors[j].Stats.Ratio
		}
		return contributors[i].Stats.AITime > contributors[j].Stats.AITime
	})

	headers := []string{"Membro", "Tempo IA", "Coding no projeto", "IA %", ""}
	rows := make([][]string, 0, len(contributors))
	for _, c := range contributors {
		// "--" rather than "0.0%" for contributors without AI telemetry — see aiAdoptionTool.
		aiTime, ratio, bar := "--", "--", ""
		if c.Stats.Instrumented {
			aiTime = fmtDuration(c.Stats.AITime)
			ratio = fmtRatio(c.Stats.Ratio)
			bar = fmtRatioBar(c.Stats.Ratio, 10)
		}
		rows = append(rows, []string{c.UserID, aiTime, fmtDuration(c.Stats.CodingTime), ratio, bar})
	}

	sb.WriteString(fmtTable(headers, rows))
	return toolResult(sb.String())
}

// lookupProject resolves a project name case-insensitively, since the name reaches
// the tool as free text from the client model rather than from a picker.
func lookupProject(byProject map[string]AIStats, project string) (AIStats, bool) {
	if st, ok := byProject[project]; ok {
		return st, true
	}
	target := strings.ToLower(project)
	for name, st := range byProject {
		if strings.ToLower(name) == target {
			return st, true
		}
	}
	return AIStats{}, false
}
