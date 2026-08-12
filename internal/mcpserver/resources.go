package mcpserver

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/quasarea/forgejo-bridge/internal/contracts"
	"github.com/quasarea/forgejo-bridge/internal/domain"
	"github.com/quasarea/forgejo-bridge/internal/requestid"
)

type RepositoryPageInput struct {
	Instance   string `json:"instance,omitempty" jsonschema:"configured Forgejo instance alias"`
	Repository string `json:"repository" jsonschema:"repository in owner/name form"`
	Page       int    `json:"page,omitempty" jsonschema:"one-based page number"`
	Limit      int    `json:"limit,omitempty" jsonschema:"maximum items to return"`
}

type RepositoryStatePageInput struct {
	Instance   string `json:"instance,omitempty" jsonschema:"configured Forgejo instance alias"`
	Repository string `json:"repository" jsonschema:"repository in owner/name form"`
	State      string `json:"state,omitempty" jsonschema:"open, closed, or all"`
	Page       int    `json:"page,omitempty" jsonschema:"one-based page number"`
	Limit      int    `json:"limit,omitempty" jsonschema:"maximum items to return"`
}

type RepositoryNameInput struct {
	Instance   string `json:"instance,omitempty" jsonschema:"configured Forgejo instance alias"`
	Repository string `json:"repository" jsonschema:"repository in owner/name form"`
	Name       string `json:"name" jsonschema:"resource name"`
}

type RepositoryNumberInput struct {
	Instance   string `json:"instance,omitempty" jsonschema:"configured Forgejo instance alias"`
	Repository string `json:"repository" jsonschema:"repository in owner/name form"`
	Number     int64  `json:"number" jsonschema:"positive issue or pull request number"`
	Page       int    `json:"page,omitempty" jsonschema:"one-based page number for child resources"`
	Limit      int    `json:"limit,omitempty" jsonschema:"maximum child resources to return"`
}

type RepositoryIDInput struct {
	Instance   string `json:"instance,omitempty" jsonschema:"configured Forgejo instance alias"`
	Repository string `json:"repository" jsonschema:"repository in owner/name form"`
	ID         int64  `json:"id" jsonschema:"positive Forgejo resource id"`
	Page       int    `json:"page,omitempty" jsonschema:"one-based page number"`
	Limit      int    `json:"limit,omitempty" jsonschema:"maximum items to return"`
}

func readTool(name, description string) *mcp.Tool {
	return &mcp.Tool{Name: name, Description: description, Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true, OpenWorldHint: boolPtr(true)}}
}

func (s *server) registerResourceTools(target *mcp.Server) {
	mcp.AddTool(target, readTool("forgejo_branch_list", "List branches in an allowed Forgejo repository."), s.listBranches)
	mcp.AddTool(target, readTool("forgejo_branch_get", "Get one branch and its head commit."), s.getBranch)
	mcp.AddTool(target, readTool("forgejo_issue_list", "List issues with normalized state and pagination."), s.listIssues)
	mcp.AddTool(target, readTool("forgejo_issue_get", "Get one issue by repository number."), s.getIssue)
	mcp.AddTool(target, readTool("forgejo_issue_comment_list", "List comments on an issue or pull request."), s.listIssueComments)
	mcp.AddTool(target, readTool("forgejo_pull_request_list", "List pull requests with normalized branch and merge state."), s.listPullRequests)
	mcp.AddTool(target, readTool("forgejo_pull_request_get", "Get one pull request by repository number."), s.getPullRequest)
	mcp.AddTool(target, readTool("forgejo_pull_review_list", "List reviews submitted on a pull request."), s.listPullReviews)
	mcp.AddTool(target, readTool("forgejo_label_list", "List repository labels."), s.listLabels)
	mcp.AddTool(target, readTool("forgejo_release_list", "List repository releases."), s.listReleases)
	mcp.AddTool(target, readTool("forgejo_release_get", "Get one release by Forgejo release id."), s.getRelease)
	mcp.AddTool(target, readTool("forgejo_actions_run_list", "List Forgejo Actions workflow runs; requires Forgejo 16 or newer."), s.listActionRuns)
	mcp.AddTool(target, readTool("forgejo_actions_run_get", "Get one Forgejo Actions workflow run; requires Forgejo 16 or newer."), s.getActionRun)
	mcp.AddTool(target, readTool("forgejo_actions_job_list", "List jobs for a Forgejo Actions run; requires Forgejo 16 or newer."), s.listActionJobs)
}

func pageEnvelope[T any](operation, alias string, page domain.Page[T]) contracts.Envelope {
	envelope := contracts.Success(operation, requestid.New(), alias, page.Items)
	envelope.Page = &contracts.Page{Number: page.Page, Limit: page.Limit, Total: page.Total, Next: page.Next}
	return envelope
}

func (s *server) listBranches(ctx context.Context, _ *mcp.CallToolRequest, input RepositoryPageInput) (*mcp.CallToolResult, contracts.Envelope, error) {
	service, alias, err := s.service(input.Instance)
	if err != nil {
		return toolFailure("branch.list", input.Instance, err)
	}
	result, err := service.ListBranches(ctx, input.Repository, domain.ListOptions{Page: input.Page, Limit: input.Limit})
	if err != nil {
		return toolFailure("branch.list", alias, err)
	}
	return nil, pageEnvelope("branch.list", alias, result), nil
}

func (s *server) getBranch(ctx context.Context, _ *mcp.CallToolRequest, input RepositoryNameInput) (*mcp.CallToolResult, contracts.Envelope, error) {
	service, alias, err := s.service(input.Instance)
	if err != nil {
		return toolFailure("branch.get", input.Instance, err)
	}
	result, err := service.GetBranch(ctx, input.Repository, input.Name)
	if err != nil {
		return toolFailure("branch.get", alias, err)
	}
	return nil, contracts.Success("branch.get", requestid.New(), alias, result), nil
}

func (s *server) listIssues(ctx context.Context, _ *mcp.CallToolRequest, input RepositoryStatePageInput) (*mcp.CallToolResult, contracts.Envelope, error) {
	service, alias, err := s.service(input.Instance)
	if err != nil {
		return toolFailure("issue.list", input.Instance, err)
	}
	result, err := service.ListIssues(ctx, input.Repository, input.State, domain.ListOptions{Page: input.Page, Limit: input.Limit})
	if err != nil {
		return toolFailure("issue.list", alias, err)
	}
	return nil, pageEnvelope("issue.list", alias, result), nil
}

func (s *server) getIssue(ctx context.Context, _ *mcp.CallToolRequest, input RepositoryNumberInput) (*mcp.CallToolResult, contracts.Envelope, error) {
	service, alias, err := s.service(input.Instance)
	if err != nil {
		return toolFailure("issue.get", input.Instance, err)
	}
	result, err := service.GetIssue(ctx, input.Repository, input.Number)
	if err != nil {
		return toolFailure("issue.get", alias, err)
	}
	return nil, contracts.Success("issue.get", requestid.New(), alias, result), nil
}

func (s *server) listIssueComments(ctx context.Context, _ *mcp.CallToolRequest, input RepositoryNumberInput) (*mcp.CallToolResult, contracts.Envelope, error) {
	service, alias, err := s.service(input.Instance)
	if err != nil {
		return toolFailure("issue.comments", input.Instance, err)
	}
	result, err := service.ListIssueComments(ctx, input.Repository, input.Number, domain.ListOptions{Page: input.Page, Limit: input.Limit})
	if err != nil {
		return toolFailure("issue.comments", alias, err)
	}
	return nil, pageEnvelope("issue.comments", alias, result), nil
}

func (s *server) listPullRequests(ctx context.Context, _ *mcp.CallToolRequest, input RepositoryStatePageInput) (*mcp.CallToolResult, contracts.Envelope, error) {
	service, alias, err := s.service(input.Instance)
	if err != nil {
		return toolFailure("pr.list", input.Instance, err)
	}
	result, err := service.ListPullRequests(ctx, input.Repository, input.State, domain.ListOptions{Page: input.Page, Limit: input.Limit})
	if err != nil {
		return toolFailure("pr.list", alias, err)
	}
	return nil, pageEnvelope("pr.list", alias, result), nil
}

func (s *server) getPullRequest(ctx context.Context, _ *mcp.CallToolRequest, input RepositoryNumberInput) (*mcp.CallToolResult, contracts.Envelope, error) {
	service, alias, err := s.service(input.Instance)
	if err != nil {
		return toolFailure("pr.get", input.Instance, err)
	}
	result, err := service.GetPullRequest(ctx, input.Repository, input.Number)
	if err != nil {
		return toolFailure("pr.get", alias, err)
	}
	return nil, contracts.Success("pr.get", requestid.New(), alias, result), nil
}

func (s *server) listPullReviews(ctx context.Context, _ *mcp.CallToolRequest, input RepositoryNumberInput) (*mcp.CallToolResult, contracts.Envelope, error) {
	service, alias, err := s.service(input.Instance)
	if err != nil {
		return toolFailure("pr.reviews", input.Instance, err)
	}
	result, err := service.ListPullReviews(ctx, input.Repository, input.Number, domain.ListOptions{Page: input.Page, Limit: input.Limit})
	if err != nil {
		return toolFailure("pr.reviews", alias, err)
	}
	return nil, pageEnvelope("pr.reviews", alias, result), nil
}

func (s *server) listLabels(ctx context.Context, _ *mcp.CallToolRequest, input RepositoryPageInput) (*mcp.CallToolResult, contracts.Envelope, error) {
	service, alias, err := s.service(input.Instance)
	if err != nil {
		return toolFailure("label.list", input.Instance, err)
	}
	result, err := service.ListLabels(ctx, input.Repository, domain.ListOptions{Page: input.Page, Limit: input.Limit})
	if err != nil {
		return toolFailure("label.list", alias, err)
	}
	return nil, pageEnvelope("label.list", alias, result), nil
}

func (s *server) listReleases(ctx context.Context, _ *mcp.CallToolRequest, input RepositoryPageInput) (*mcp.CallToolResult, contracts.Envelope, error) {
	service, alias, err := s.service(input.Instance)
	if err != nil {
		return toolFailure("release.list", input.Instance, err)
	}
	result, err := service.ListReleases(ctx, input.Repository, domain.ListOptions{Page: input.Page, Limit: input.Limit})
	if err != nil {
		return toolFailure("release.list", alias, err)
	}
	return nil, pageEnvelope("release.list", alias, result), nil
}

func (s *server) getRelease(ctx context.Context, _ *mcp.CallToolRequest, input RepositoryIDInput) (*mcp.CallToolResult, contracts.Envelope, error) {
	service, alias, err := s.service(input.Instance)
	if err != nil {
		return toolFailure("release.get", input.Instance, err)
	}
	result, err := service.GetRelease(ctx, input.Repository, input.ID)
	if err != nil {
		return toolFailure("release.get", alias, err)
	}
	return nil, contracts.Success("release.get", requestid.New(), alias, result), nil
}

func (s *server) listActionRuns(ctx context.Context, _ *mcp.CallToolRequest, input RepositoryPageInput) (*mcp.CallToolResult, contracts.Envelope, error) {
	service, alias, err := s.service(input.Instance)
	if err != nil {
		return toolFailure("actions.run.list", input.Instance, err)
	}
	result, err := service.ListActionRuns(ctx, input.Repository, domain.ListOptions{Page: input.Page, Limit: input.Limit})
	if err != nil {
		return toolFailure("actions.run.list", alias, err)
	}
	return nil, pageEnvelope("actions.run.list", alias, result), nil
}

func (s *server) getActionRun(ctx context.Context, _ *mcp.CallToolRequest, input RepositoryIDInput) (*mcp.CallToolResult, contracts.Envelope, error) {
	service, alias, err := s.service(input.Instance)
	if err != nil {
		return toolFailure("actions.run.get", input.Instance, err)
	}
	result, err := service.GetActionRun(ctx, input.Repository, input.ID)
	if err != nil {
		return toolFailure("actions.run.get", alias, err)
	}
	return nil, contracts.Success("actions.run.get", requestid.New(), alias, result), nil
}

func (s *server) listActionJobs(ctx context.Context, _ *mcp.CallToolRequest, input RepositoryIDInput) (*mcp.CallToolResult, contracts.Envelope, error) {
	service, alias, err := s.service(input.Instance)
	if err != nil {
		return toolFailure("actions.run.jobs", input.Instance, err)
	}
	result, err := service.ListActionJobs(ctx, input.Repository, input.ID)
	if err != nil {
		return toolFailure("actions.run.jobs", alias, err)
	}
	return nil, contracts.Success("actions.run.jobs", requestid.New(), alias, result), nil
}
