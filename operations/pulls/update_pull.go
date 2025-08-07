package pulls

import (
	"context"
	"fmt"

	"gitee.com/masx200/mcp-gitee/operations/types"
	"gitee.com/masx200/mcp-gitee/utils"
	"github.com/mark3labs/mcp-go/mcp"
)

const (
	// UpdatePullToolName is the name of the tool
	UpdatePullToolName = "update_pull"
)

var UpdatePullTool = func() mcp.Tool {
	options := utils.CombineOptions(
		BasicOptions,
		BasicPullOptions,
		[]mcp.ToolOption{
			mcp.WithDescription("Update a pull request"),
			mcp.WithNumber(
				"number",
				mcp.Description("The number of the pull request"),
				mcp.Required(),
			),
			mcp.WithString(
				"title",
				mcp.Description("The title of the pull request"),
			),
			mcp.WithString(
				"state",
				mcp.Description("The state of the pull request"),
				mcp.Enum("closed", "open"),
			),
		},
	)
	return mcp.NewTool(UpdatePullToolName, options...)
}()

func UpdatePullHandleFunc(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, _ := utils.ConvertArgumentsToMap(request.Params.Arguments)
	owner := args["owner"].(string)
	repo := args["repo"].(string)
	numberArg, exists := args["number"]
	if !exists {
		return mcp.NewToolResultError("Missing required parameter: number"),
			utils.NewParamError("number", "parameter is required")
	}
	number, err := utils.SafelyConvertToInt(numberArg)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), err
	}
	apiUrl := fmt.Sprintf("/repos/%s/%s/pulls/%d", owner, repo, number)
	giteeClient := utils.NewGiteeClient("PATCH", apiUrl, utils.WithContext(ctx), utils.WithPayload(args))
	pull := &types.BasicPull{}
	return giteeClient.HandleMCPResult(pull)
}
