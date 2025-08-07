package issues

import (
	"context"
	"fmt"

	"gitee.com/masx200/mcp-gitee/operations/types"
	"gitee.com/masx200/mcp-gitee/utils"
	"github.com/mark3labs/mcp-go/mcp"
)

const (
	UpdateIssueToolName = "update_issue"
)

// UpdateIssueConfig structure for updating issues
type UpdateIssueConfig struct {
	UrlTemplate string
	PathParams  []string
}

// Configuration mapping for issue updates
var updateIssueConfigs = map[string]UpdateIssueConfig{
	UpdateIssueToolName: {
		UrlTemplate: "/repos/%s/issues/%s",
		PathParams:  []string{"owner", "number"},
	},
}

var UpdateIssueTool = func() mcp.Tool {
	options := utils.CombineOptions(
		[]mcp.ToolOption{
			mcp.WithDescription("Update an issue"),
			mcp.WithString(
				"number",
				mcp.Description("The number of the issue"),
				mcp.Required(),
			),
			mcp.WithString(
				"state",
				mcp.Description("The state of the issue"),
				mcp.Enum("open", "progressing", "closed"),
			),
		},
		BasicOptions,
		BasicIssueOptions,
	)
	return mcp.NewTool(UpdateIssueToolName, options...)
}()

// UpdateIssueHandleFunc handles requests to update repository issues
func UpdateIssueHandleFunc(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return UpdateIssueHandleFuncCommon(UpdateIssueToolName)(ctx, request)
}

// UpdateIssueHandleFuncCommon is a common handler function for processing issue update requests
func UpdateIssueHandleFuncCommon(updateType string) func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		config, ok := updateIssueConfigs[updateType]
		if !ok {
			errMsg := fmt.Sprintf("Unsupported issue update type: %s", updateType)
			return mcp.NewToolResultError(errMsg), fmt.Errorf(errMsg)
		}

		apiUrl := config.UrlTemplate
		if len(config.PathParams) > 0 {
			args, _ := utils.ConvertArgumentsToMap(request.Params.Arguments)
			apiUrlArgs := make([]interface{}, 0, len(config.PathParams))
			for _, param := range config.PathParams {
				value, ok := args[param].(string)
				if !ok {
					errMsg := fmt.Sprintf("Missing required path parameter: %s", param)
					return mcp.NewToolResultError(errMsg), fmt.Errorf(errMsg)
				}
				apiUrlArgs = append(apiUrlArgs, value)
			}
			apiUrl = fmt.Sprintf(apiUrl, apiUrlArgs...)
		}

		giteeClient := utils.NewGiteeClient("PATCH", apiUrl, utils.WithContext(ctx), utils.WithPayload(request.Params.Arguments))
		issue := &types.BasicIssue{}

		return giteeClient.HandleMCPResult(issue)
	}
}
