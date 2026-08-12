package forgejo

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/quasarea/forgejo-bridge/internal/contracts"
	"github.com/quasarea/forgejo-bridge/internal/domain"
)

func normalizeList(options domain.ListOptions) domain.ListOptions {
	if options.Page < 1 {
		options.Page = 1
	}
	if options.Limit < 1 {
		options.Limit = 30
	}
	if options.Limit > 100 {
		options.Limit = 100
	}
	return options
}

func (c *Client) repoEndpoint(owner, name, suffix string) (string, *contracts.BridgeError) {
	fullName := owner + "/" + name
	if !c.allowed(fullName) {
		return "", contracts.NewError("permission_denied", "repository is outside the configured allowlist")
	}
	return c.apiURL + "/repos/" + url.PathEscape(owner) + "/" + url.PathEscape(name) + suffix, nil
}

func getPage[T any](ctx context.Context, c *Client, endpoint string, options domain.ListOptions) (domain.Page[T], *contracts.BridgeError) {
	options = normalizeList(options)
	parsed, err := url.Parse(endpoint)
	if err != nil {
		return domain.Page[T]{}, contracts.WrapError("invalid_input", "invalid request URL", err)
	}
	query := parsed.Query()
	query.Set("page", strconv.Itoa(options.Page))
	query.Set("limit", strconv.Itoa(options.Limit))
	parsed.RawQuery = query.Encode()
	var items []T
	response, bridgeErr := c.getJSON(ctx, parsed.String(), &items)
	if bridgeErr != nil {
		return domain.Page[T]{}, bridgeErr
	}
	total, _ := strconv.ParseInt(response.Header.Get("X-Total-Count"), 10, 64)
	return domain.Page[T]{Items: items, Page: options.Page, Limit: options.Limit, Total: total, Next: nextLink(response.Header.Get("Link"))}, nil
}

func withState(endpoint, state string) (string, *contracts.BridgeError) {
	state = strings.ToLower(strings.TrimSpace(state))
	if state == "" {
		return endpoint, nil
	}
	if state != "open" && state != "closed" && state != "all" {
		return "", contracts.NewError("invalid_input", "state must be open, closed, or all")
	}
	parsed, err := url.Parse(endpoint)
	if err != nil {
		return "", contracts.WrapError("invalid_input", "invalid request URL", err)
	}
	query := parsed.Query()
	query.Set("state", state)
	parsed.RawQuery = query.Encode()
	return parsed.String(), nil
}

func (c *Client) ListBranches(ctx context.Context, owner, name string, options domain.ListOptions) (domain.Page[domain.Branch], *contracts.BridgeError) {
	endpoint, err := c.repoEndpoint(owner, name, "/branches")
	if err != nil {
		return domain.Page[domain.Branch]{}, err
	}
	return getPage[domain.Branch](ctx, c, endpoint, options)
}

func (c *Client) GetBranch(ctx context.Context, owner, name, branch string) (domain.Branch, *contracts.BridgeError) {
	endpoint, err := c.repoEndpoint(owner, name, "/branches/"+url.PathEscape(branch))
	if err != nil {
		return domain.Branch{}, err
	}
	var result domain.Branch
	_, bridgeErr := c.getJSON(ctx, endpoint, &result)
	return result, bridgeErr
}

type issueDTO struct {
	domain.Issue
	PullRequest any `json:"pull_request"`
}

func (i issueDTO) normalized() domain.Issue {
	result := i.Issue
	result.IsPull = i.PullRequest != nil
	return result
}

func (c *Client) ListIssues(ctx context.Context, owner, name, state string, options domain.ListOptions) (domain.Page[domain.Issue], *contracts.BridgeError) {
	endpoint, err := c.repoEndpoint(owner, name, "/issues")
	if err != nil {
		return domain.Page[domain.Issue]{}, err
	}
	endpoint, err = withState(endpoint, state)
	if err != nil {
		return domain.Page[domain.Issue]{}, err
	}
	raw, bridgeErr := getPage[issueDTO](ctx, c, endpoint, options)
	if bridgeErr != nil {
		return domain.Page[domain.Issue]{}, bridgeErr
	}
	items := make([]domain.Issue, 0, len(raw.Items))
	for _, item := range raw.Items {
		items = append(items, item.normalized())
	}
	return domain.Page[domain.Issue]{Items: items, Page: raw.Page, Limit: raw.Limit, Total: raw.Total, Next: raw.Next}, nil
}

func (c *Client) GetIssue(ctx context.Context, owner, name string, number int64) (domain.Issue, *contracts.BridgeError) {
	endpoint, err := c.repoEndpoint(owner, name, "/issues/"+strconv.FormatInt(number, 10))
	if err != nil {
		return domain.Issue{}, err
	}
	var raw issueDTO
	_, bridgeErr := c.getJSON(ctx, endpoint, &raw)
	return raw.normalized(), bridgeErr
}

func (c *Client) ListIssueComments(ctx context.Context, owner, name string, number int64, options domain.ListOptions) (domain.Page[domain.Comment], *contracts.BridgeError) {
	endpoint, err := c.repoEndpoint(owner, name, "/issues/"+strconv.FormatInt(number, 10)+"/comments")
	if err != nil {
		return domain.Page[domain.Comment]{}, err
	}
	return getPage[domain.Comment](ctx, c, endpoint, options)
}

func (c *Client) ListPullRequests(ctx context.Context, owner, name, state string, options domain.ListOptions) (domain.Page[domain.PullRequest], *contracts.BridgeError) {
	endpoint, err := c.repoEndpoint(owner, name, "/pulls")
	if err != nil {
		return domain.Page[domain.PullRequest]{}, err
	}
	endpoint, err = withState(endpoint, state)
	if err != nil {
		return domain.Page[domain.PullRequest]{}, err
	}
	return getPage[domain.PullRequest](ctx, c, endpoint, options)
}

func (c *Client) GetPullRequest(ctx context.Context, owner, name string, number int64) (domain.PullRequest, *contracts.BridgeError) {
	endpoint, err := c.repoEndpoint(owner, name, "/pulls/"+strconv.FormatInt(number, 10))
	if err != nil {
		return domain.PullRequest{}, err
	}
	var result domain.PullRequest
	_, bridgeErr := c.getJSON(ctx, endpoint, &result)
	return result, bridgeErr
}

func (c *Client) ListPullReviews(ctx context.Context, owner, name string, number int64, options domain.ListOptions) (domain.Page[domain.Review], *contracts.BridgeError) {
	endpoint, err := c.repoEndpoint(owner, name, "/pulls/"+strconv.FormatInt(number, 10)+"/reviews")
	if err != nil {
		return domain.Page[domain.Review]{}, err
	}
	return getPage[domain.Review](ctx, c, endpoint, options)
}

func (c *Client) ListLabels(ctx context.Context, owner, name string, options domain.ListOptions) (domain.Page[domain.Label], *contracts.BridgeError) {
	endpoint, err := c.repoEndpoint(owner, name, "/labels")
	if err != nil {
		return domain.Page[domain.Label]{}, err
	}
	return getPage[domain.Label](ctx, c, endpoint, options)
}

func (c *Client) ListReleases(ctx context.Context, owner, name string, options domain.ListOptions) (domain.Page[domain.Release], *contracts.BridgeError) {
	endpoint, err := c.repoEndpoint(owner, name, "/releases")
	if err != nil {
		return domain.Page[domain.Release]{}, err
	}
	return getPage[domain.Release](ctx, c, endpoint, options)
}

func (c *Client) GetRelease(ctx context.Context, owner, name string, id int64) (domain.Release, *contracts.BridgeError) {
	endpoint, err := c.repoEndpoint(owner, name, "/releases/"+strconv.FormatInt(id, 10))
	if err != nil {
		return domain.Release{}, err
	}
	var result domain.Release
	_, bridgeErr := c.getJSON(ctx, endpoint, &result)
	return result, bridgeErr
}

func (c *Client) requireMajor(ctx context.Context, minimum int, capability string) *contracts.BridgeError {
	var version struct {
		Version string `json:"version"`
	}
	if _, err := c.getJSON(ctx, c.apiURL+"/version", &version); err != nil {
		return err
	}
	if parseMajor(version.Version) < minimum {
		return &contracts.BridgeError{
			Code:    "capability_unsupported",
			Message: fmt.Sprintf("%s requires Forgejo %d or newer", capability, minimum),
			Details: map[string]any{"capability": capability, "forgejo_version": version.Version},
		}
	}
	return nil
}

func actionUnavailable(err *contracts.BridgeError, capability string) *contracts.BridgeError {
	if err == nil || err.HTTPStatus != 404 {
		return err
	}
	return &contracts.BridgeError{
		Code:    "capability_unsupported",
		Message: capability + " is unavailable; Forgejo Actions may be disabled",
		Details: map[string]any{"capability": capability},
	}
}

func (c *Client) ListActionRuns(ctx context.Context, owner, name string, options domain.ListOptions) (domain.Page[domain.ActionRun], *contracts.BridgeError) {
	endpoint, err := c.repoEndpoint(owner, name, "/actions/runs")
	if err != nil {
		return domain.Page[domain.ActionRun]{}, err
	}
	if err := c.requireMajor(ctx, 16, "actions.run.read"); err != nil {
		return domain.Page[domain.ActionRun]{}, err
	}
	options = normalizeList(options)
	parsed, _ := url.Parse(endpoint)
	query := parsed.Query()
	query.Set("page", strconv.Itoa(options.Page))
	query.Set("limit", strconv.Itoa(options.Limit))
	parsed.RawQuery = query.Encode()
	var raw struct {
		Total int64              `json:"total_count"`
		Runs  []domain.ActionRun `json:"workflow_runs"`
	}
	response, bridgeErr := c.getJSON(ctx, parsed.String(), &raw)
	if bridgeErr != nil {
		return domain.Page[domain.ActionRun]{}, actionUnavailable(bridgeErr, "actions.run.read")
	}
	return domain.Page[domain.ActionRun]{Items: raw.Runs, Page: options.Page, Limit: options.Limit, Total: raw.Total, Next: nextLink(response.Header.Get("Link"))}, nil
}

func (c *Client) GetActionRun(ctx context.Context, owner, name string, id int64) (domain.ActionRun, *contracts.BridgeError) {
	endpoint, err := c.repoEndpoint(owner, name, "/actions/runs/"+strconv.FormatInt(id, 10))
	if err != nil {
		return domain.ActionRun{}, err
	}
	if err := c.requireMajor(ctx, 16, "actions.run.read"); err != nil {
		return domain.ActionRun{}, err
	}
	var result domain.ActionRun
	_, bridgeErr := c.getJSON(ctx, endpoint, &result)
	return result, actionUnavailable(bridgeErr, "actions.run.read")
}

func (c *Client) ListActionJobs(ctx context.Context, owner, name string, id int64) ([]domain.ActionJob, *contracts.BridgeError) {
	endpoint, err := c.repoEndpoint(owner, name, "/actions/runs/"+strconv.FormatInt(id, 10)+"/jobs")
	if err != nil {
		return nil, err
	}
	if err := c.requireMajor(ctx, 16, "actions.job.read"); err != nil {
		return nil, err
	}
	var result []domain.ActionJob
	_, bridgeErr := c.getJSON(ctx, endpoint, &result)
	return result, actionUnavailable(bridgeErr, "actions.job.read")
}
