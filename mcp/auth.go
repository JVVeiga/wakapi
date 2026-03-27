package mcp

import (
	"context"
	"net/http"

	mcpgo "github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
	"github.com/muety/wakapi/models"
	"github.com/muety/wakapi/utils"
)

type contextKey string

const principalKey contextKey = "mcp_principal"

func (s *MCPServer) extractHTTPAuthContext() func(ctx context.Context, r *http.Request) context.Context {
	return s.extractAuthContext()
}

func (s *MCPServer) extractAuthContext() func(ctx context.Context, r *http.Request) context.Context {
	return func(ctx context.Context, r *http.Request) context.Context {
		key, err := utils.ExtractBearerAuth(r)
		if err != nil {
			return ctx
		}

		user, err := s.userSrvc.GetUserByKey(key, false)
		if err != nil {
			return ctx
		}

		return context.WithValue(ctx, principalKey, user)
	}
}

func (s *MCPServer) authMiddleware() mcpserver.ToolHandlerMiddleware {
	return func(next mcpserver.ToolHandlerFunc) mcpserver.ToolHandlerFunc {
		return func(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
			user := getPrincipal(ctx)
			if user == nil {
				return toolError("Unauthorized: invalid or missing API key"), nil
			}
			return next(ctx, request)
		}
	}
}

func getPrincipal(ctx context.Context) *models.User {
	if user, ok := ctx.Value(principalKey).(*models.User); ok {
		return user
	}
	return nil
}
