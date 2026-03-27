package mcp

import (
	"log/slog"

	"github.com/go-chi/chi/v5"
	mcpgo "github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
	conf "github.com/muety/wakapi/config"
	"github.com/muety/wakapi/services"
)

type MCPServer struct {
	config          *conf.Config
	userSrvc        services.IUserService
	teamSrvc        services.ITeamService
	summarySrvc     services.ISummaryService
	heartbeatSrvc   services.IHeartbeatService
	durationSrvc    services.IDurationService
	leaderboardSrvc services.ILeaderboardService
	mcpSrv          *mcpserver.MCPServer
}

func NewMCPServer(
	userService services.IUserService,
	teamService services.ITeamService,
	summaryService services.ISummaryService,
	heartbeatService services.IHeartbeatService,
	durationService services.IDurationService,
	leaderboardService services.ILeaderboardService,
) *MCPServer {
	srv := &MCPServer{
		config:          conf.Get(),
		userSrvc:        userService,
		teamSrvc:        teamService,
		summarySrvc:     summaryService,
		heartbeatSrvc:   heartbeatService,
		durationSrvc:    durationService,
		leaderboardSrvc: leaderboardService,
	}

	srv.mcpSrv = mcpserver.NewMCPServer(
		"Wakapi Team Insights",
		srv.config.Version,
		mcpserver.WithToolHandlerMiddleware(srv.authMiddleware()),
	)

	srv.registerTools()

	return srv
}

func (s *MCPServer) RegisterRoutes(router chi.Router) {
	path := s.config.MCP.Path

	httpServer := mcpserver.NewStreamableHTTPServer(s.mcpSrv,
		mcpserver.WithEndpointPath(path),
		mcpserver.WithHTTPContextFunc(s.extractHTTPAuthContext()),
	)

	router.Mount(path, httpServer)

	slog.Info("MCP server registered", "path", path)
}

func (s *MCPServer) registerTools() {
	s.mcpSrv.AddTool(s.listTeamsTool())
	s.mcpSrv.AddTool(s.memberSummaryTool())
	s.mcpSrv.AddTool(s.teamOverviewTool())
	s.mcpSrv.AddTool(s.compareMembersTool())
	s.mcpSrv.AddTool(s.activityPatternsTool())
	s.mcpSrv.AddTool(s.projectAnalysisTool())
	s.mcpSrv.AddTool(s.trendAnalysisTool())
}

func toolError(msg string) *mcpgo.CallToolResult {
	return &mcpgo.CallToolResult{
		Content: []mcpgo.Content{mcpgo.TextContent{Type: "text", Text: msg}},
		IsError: true,
	}
}

func toolResult(text string) *mcpgo.CallToolResult {
	return &mcpgo.CallToolResult{
		Content: []mcpgo.Content{mcpgo.TextContent{Type: "text", Text: text}},
	}
}

func addIntervalParams(tool mcpgo.Tool) mcpgo.Tool {
	// Add common interval/date range parameters
	opts := []mcpgo.ToolOption{
		mcpgo.WithString("interval",
			mcpgo.Description("Preset interval: today, yesterday, week, last_week, month, last_month, 7_days, 14_days, 30_days, 6_months, 12_months, all_time"),
		),
		mcpgo.WithString("from", mcpgo.Description("Start date (YYYY-MM-DD). Used if interval not set")),
		mcpgo.WithString("to", mcpgo.Description("End date (YYYY-MM-DD). Used if interval not set")),
	}
	for _, opt := range opts {
		opt(&tool)
	}
	return tool
}

