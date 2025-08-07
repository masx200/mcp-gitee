package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"gitee.com/masx200/mcp-gitee/operations/issues"
	"gitee.com/masx200/mcp-gitee/operations/notifications"
	"gitee.com/masx200/mcp-gitee/operations/pulls"
	"gitee.com/masx200/mcp-gitee/operations/repository"
	"gitee.com/masx200/mcp-gitee/operations/users"
	"gitee.com/masx200/mcp-gitee/utils"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

var (
	Version              = utils.Version
	disabledToolsetsFlag string
	enabledToolsetsFlag  string
)

func newMCPServer() *server.MCPServer {
	hooks := &server.Hooks{}

	hooks.OnBeforeCallTool = append(hooks.OnBeforeCallTool, func(ctx context.Context, id any, message *mcp.CallToolRequest) {
		log.Printf("[tool call] ToolName: %s, Params: %v", message.Params.Name, message.Params.Arguments)
	})

	hooks.OnAfterCallTool = append(hooks.OnAfterCallTool, func(ctx context.Context, id any, message *mcp.CallToolRequest, result *mcp.CallToolResult) {
		if result != nil && result.IsError {
			log.Printf("[tool call error] ToolName: %s, Params: %v, error msg: %v", message.Params.Name, message.Params.Arguments, result.Content)
		}
	})

	return server.NewMCPServer(
		"mcp-gitee",
		Version,
		server.WithToolCapabilities(true),
		server.WithLogging(),
		server.WithHooks(hooks),
	)
}

func addTool(s *server.MCPServer, tool mcp.Tool, handleFunc server.ToolHandlerFunc) {
	enabledToolsets := getEnabledToolsets()
	if len(enabledToolsets) == 0 {
		s.AddTool(tool, handleFunc)
		return
	}

	for i := range enabledToolsets {
		enabledToolsets[i] = strings.TrimSpace(enabledToolsets[i])
	}

	for _, keepTool := range enabledToolsets {
		if tool.Name == keepTool {
			s.AddTool(tool, handleFunc)
			return
		}
	}
}

func disableTools(s *server.MCPServer) {
	if enabledToolsetsFlag != "" {
		enabledToolsetsFlag = os.Getenv("ENABLED_TOOLSETS")
	}

	if enabledToolsetsFlag != "" {
		return
	}

	if disabledTools := getDisabledToolsets(); len(disabledTools) > 0 {
		s.DeleteTools(disabledTools...)
	}
}

func addTools(s *server.MCPServer) {
	// Repository Tools
	addTool(s, repository.ListUserReposTool, repository.ListUserReposHandler)
	addTool(s, repository.GetFileContentTool, repository.GetFileContentHandler)
	addTool(s, repository.NewCreateRepoTool(repository.CreateUserRepo), repository.CreateRepoHandleFunc(repository.CreateUserRepo))
	addTool(s, repository.NewCreateRepoTool(repository.CreateOrgRepo), repository.CreateRepoHandleFunc(repository.CreateOrgRepo))
	addTool(s, repository.NewCreateRepoTool(repository.CreateEnterRepo), repository.CreateRepoHandleFunc(repository.CreateEnterRepo))
	addTool(s, repository.CreateReleaseTool, repository.CreateReleaseHandleFunc)
	addTool(s, repository.ListReleasesTool, repository.ListReleasesHandleFunc)
	addTool(s, repository.SearchReposTool, repository.SearchOpenSourceReposHandler)
	addTool(s, repository.ForkRepositoryTool, repository.ForkRepositoryHandler)

	// Pulls Tools
	addTool(s, pulls.NewListPullsTool(pulls.ListRepoPullsToolName), pulls.ListPullsHandleFunc(pulls.ListRepoPullsToolName))
	addTool(s, pulls.CreatePullTool, pulls.CreatePullHandleFunc)
	addTool(s, pulls.UpdatePullTool, pulls.UpdatePullHandleFunc)
	addTool(s, pulls.GetPullDetailTool, pulls.GetPullDetailHandleFunc)
	addTool(s, pulls.CommentPullTool, pulls.CommentPullHandleFunc)
	addTool(s, pulls.MergePullTool, pulls.MergePullHandleFunc)
	addTool(s, pulls.ListPullCommentsTool, pulls.ListPullCommentsHandleFunc)
	addTool(s, pulls.GetDiffFilesTool, pulls.GetDiffFilesHandleFunc)

	// Issues Tools
	addTool(s, issues.CreateIssueTool, issues.CreateIssueHandleFunc)
	addTool(s, issues.UpdateIssueTool, issues.UpdateIssueHandleFunc)
	addTool(s, issues.NewGetIssueDetailTool(issues.GetRepoIssueDetailToolName), issues.GetIssueDetailHandleFunc(issues.GetRepoIssueDetailToolName))
	addTool(s, issues.NewListIssuesTool(issues.ListRepoIssuesToolName), issues.ListIssuesHandleFunc(issues.ListRepoIssuesToolName))
	addTool(s, issues.CommentIssueTool, issues.CommentIssueHandleFunc)
	addTool(s, issues.ListIssueCommentsTool, issues.ListIssueCommentsHandleFunc)

	// Notifications Tools
	addTool(s, notifications.ListUserNotificationsTool, notifications.ListUserNotificationsHandler)

	// Users Tools
	addTool(s, users.GetUserInfoTool, users.GetUserInfoHandleFunc())
	addTool(s, users.SearchUsersTool, users.SearchUsersHandler)
}

func getDisabledToolsets() []string {
	if disabledToolsetsFlag == "" {
		disabledToolsetsFlag = os.Getenv("DISABLED_TOOLSETS")
	}

	if disabledToolsetsFlag == "" {
		return nil
	}

	tools := strings.Split(disabledToolsetsFlag, ",")
	for i := range tools {
		tools[i] = strings.TrimSpace(tools[i])
	}

	return tools
}

func getEnabledToolsets() []string {
	if enabledToolsetsFlag == "" {
		enabledToolsetsFlag = os.Getenv("ENABLED_TOOLSETS")
	}
	if enabledToolsetsFlag == "" {
		return nil
	}
	tools := strings.Split(enabledToolsetsFlag, ",")
	for i := range tools {
		tools[i] = strings.TrimSpace(tools[i])
	}
	return tools
}

func run(transport, addr string) error {
	s := newMCPServer()
	addTools(s)
	disableTools(s)

	switch transport {
	case "stdio":
		if err := server.ServeStdio(s); err != nil {
			if err == context.Canceled {
				return nil
			}
			return err
		}
	case "sse":
		srv := server.NewSSEServer(s, server.WithBaseURL(addr),
			/* 为SSE服务器添加Bearer token认证支持

			   在SSE服务器中添加上下文处理函数，从请求头中提取Bearer token并存入上下文 */
			server.WithSSEContextFunc(func(ctx context.Context, r *http.Request) context.Context {
				auth := r.Header.Get("Authorization")
				if len(auth) > 7 && auth[:7] == "Bearer " {
					token := auth[7:]
					ctx = context.WithValue(ctx, "access_token", token)
				}
				return ctx
			}),
		)
		log.Printf("SSE server listening on %s", addr)
		if err := srv.Start(addr); err != nil {
			if err == context.Canceled {
				return nil
			}
			return fmt.Errorf("server error: %v", err)
		}
	case "http":
		httpServer := server.NewStreamableHTTPServer(s,
			server.WithStateLess(true),
			server.WithHTTPContextFunc(func(ctx context.Context, r *http.Request) context.Context {
				auth := r.Header.Get("Authorization")
				if len(auth) > 7 && auth[:7] == "Bearer " {
					token := auth[7:]
					ctx = context.WithValue(ctx, "access_token", token)
				}
				return ctx
			}),
		)
		log.Printf("HTTP server listening on %s", addr)
		if err := httpServer.Start(addr); err != nil {
			if err == context.Canceled {
				return nil
			}
			return fmt.Errorf("server error: %v", err)
		}
	default:
		return fmt.Errorf(
			"invalid transport type: %s. Must be 'stdio'、'sse' or 'http'",
			transport,
		)
	}
	return nil
}

func main() {
	accessToken := flag.String("token", "", "Gitee access token")
	apiBase := flag.String("api-base", "", "Gitee API base URL (default: https://gitee.com/api/v5)")
	showVersion := flag.Bool("version", false, "Show version information")
	transport := flag.String("transport", "stdio", "Transport type (stdio or sse)")
	addr := flag.String("address", "localhost:8000", "The host and port to start the sse/http server on")
	flag.StringVar(&disabledToolsetsFlag, "disabled-toolsets", "", "Comma-separated list of tools to disable")
	flag.StringVar(&enabledToolsetsFlag, "enabled-toolsets", "", "Comma-separated list of tools to enable (if specified, only these tools will be available)")
	flag.Parse()

	if *showVersion {
		fmt.Printf("Gitee MCP Server\n")
		fmt.Printf("Version: %s\n", Version)
		os.Exit(0)
	}

	if *accessToken != "" {
		utils.SetGiteeAccessToken(*accessToken)
	}

	if *apiBase != "" {
		utils.SetApiBase(*apiBase)
	}

	if err := run(*transport, *addr); err != nil {
		panic(err)
	}

}
