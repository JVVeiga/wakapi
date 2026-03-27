package mcp

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"testing"

	mcpgo "github.com/mark3labs/mcp-go/mcp"
	"github.com/muety/wakapi/mocks"
	"github.com/muety/wakapi/models"
	"github.com/stretchr/testify/assert"
)

func TestGetPrincipal_WithUser(t *testing.T) {
	user := &models.User{ID: "alice"}
	ctx := context.WithValue(context.Background(), principalKey, user)
	result := getPrincipal(ctx)
	assert.NotNil(t, result)
	assert.Equal(t, "alice", result.ID)
}

func TestGetPrincipal_WithoutUser(t *testing.T) {
	result := getPrincipal(context.Background())
	assert.Nil(t, result)
}

func TestGetPrincipal_WrongType(t *testing.T) {
	ctx := context.WithValue(context.Background(), principalKey, "not a user")
	result := getPrincipal(ctx)
	assert.Nil(t, result)
}

func TestExtractAuthContext_ValidKey(t *testing.T) {
	apiKey := "test-api-key"
	user := &models.User{ID: "alice", ApiKey: apiKey}

	userSrvc := new(mocks.UserServiceMock)
	userSrvc.On("GetUserByKey", apiKey, false).Return(user, nil)

	srv := &MCPServer{userSrvc: userSrvc}
	extractFn := srv.extractAuthContext()

	req, _ := http.NewRequest("GET", "/mcp/sse", nil)
	req.Header.Set("Authorization", fmt.Sprintf("Basic %s", base64.StdEncoding.EncodeToString([]byte(apiKey))))

	ctx := extractFn(context.Background(), req)
	principal := getPrincipal(ctx)
	assert.NotNil(t, principal)
	assert.Equal(t, "alice", principal.ID)
}

func TestExtractAuthContext_InvalidKey(t *testing.T) {
	userSrvc := new(mocks.UserServiceMock)
	userSrvc.On("GetUserByKey", "bad-key", false).Return(nil, fmt.Errorf("not found"))

	srv := &MCPServer{userSrvc: userSrvc}
	extractFn := srv.extractAuthContext()

	req, _ := http.NewRequest("GET", "/mcp/sse", nil)
	req.Header.Set("Authorization", fmt.Sprintf("Basic %s", base64.StdEncoding.EncodeToString([]byte("bad-key"))))

	ctx := extractFn(context.Background(), req)
	assert.Nil(t, getPrincipal(ctx))
}

func TestExtractAuthContext_NoHeader(t *testing.T) {
	userSrvc := new(mocks.UserServiceMock)
	srv := &MCPServer{userSrvc: userSrvc}
	extractFn := srv.extractAuthContext()

	req, _ := http.NewRequest("GET", "/mcp/sse", nil)
	ctx := extractFn(context.Background(), req)
	assert.Nil(t, getPrincipal(ctx))
}

func TestAuthMiddleware_Authenticated(t *testing.T) {
	user := &models.User{ID: "alice"}
	srv := &MCPServer{}
	mw := srv.authMiddleware()

	called := false
	handler := mw(func(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
		called = true
		return toolResult("ok"), nil
	})

	ctx := context.WithValue(context.Background(), principalKey, user)
	result, err := handler(ctx, mcpgo.CallToolRequest{})
	assert.Nil(t, err)
	assert.True(t, called)
	assert.False(t, result.IsError)
}

func TestAuthMiddleware_Unauthenticated(t *testing.T) {
	srv := &MCPServer{}
	mw := srv.authMiddleware()

	called := false
	handler := mw(func(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
		called = true
		return toolResult("ok"), nil
	})

	result, err := handler(context.Background(), mcpgo.CallToolRequest{})
	assert.Nil(t, err)
	assert.False(t, called) // handler should not be called
	assert.True(t, result.IsError)
}
