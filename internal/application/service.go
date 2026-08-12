package application

import (
	"context"
	"strings"

	"github.com/quasarea/forgejo-bridge/internal/contracts"
	"github.com/quasarea/forgejo-bridge/internal/domain"
)

type ForgejoClient interface {
	Probe(context.Context) (domain.InstanceCapabilities, *contracts.BridgeError)
	ListRepositories(context.Context, domain.ListOptions) (domain.RepositoryPage, *contracts.BridgeError)
	GetRepository(context.Context, string, string) (domain.Repository, *contracts.BridgeError)
	ListBranches(context.Context, string, string, domain.ListOptions) (domain.Page[domain.Branch], *contracts.BridgeError)
	GetBranch(context.Context, string, string, string) (domain.Branch, *contracts.BridgeError)
	ListIssues(context.Context, string, string, string, domain.ListOptions) (domain.Page[domain.Issue], *contracts.BridgeError)
	GetIssue(context.Context, string, string, int64) (domain.Issue, *contracts.BridgeError)
	ListIssueComments(context.Context, string, string, int64, domain.ListOptions) (domain.Page[domain.Comment], *contracts.BridgeError)
	ListPullRequests(context.Context, string, string, string, domain.ListOptions) (domain.Page[domain.PullRequest], *contracts.BridgeError)
	GetPullRequest(context.Context, string, string, int64) (domain.PullRequest, *contracts.BridgeError)
	ListPullReviews(context.Context, string, string, int64, domain.ListOptions) (domain.Page[domain.Review], *contracts.BridgeError)
	ListLabels(context.Context, string, string, domain.ListOptions) (domain.Page[domain.Label], *contracts.BridgeError)
	ListReleases(context.Context, string, string, domain.ListOptions) (domain.Page[domain.Release], *contracts.BridgeError)
	GetRelease(context.Context, string, string, int64) (domain.Release, *contracts.BridgeError)
	ListActionRuns(context.Context, string, string, domain.ListOptions) (domain.Page[domain.ActionRun], *contracts.BridgeError)
	GetActionRun(context.Context, string, string, int64) (domain.ActionRun, *contracts.BridgeError)
	ListActionJobs(context.Context, string, string, int64) ([]domain.ActionJob, *contracts.BridgeError)
}

func splitRepository(fullName string) (string, string, *contracts.BridgeError) {
	owner, name, ok := strings.Cut(fullName, "/")
	if !ok || owner == "" || name == "" || strings.Contains(name, "/") {
		return "", "", contracts.NewError("invalid_input", "repository must use owner/name format")
	}
	return owner, name, nil
}

type Service struct {
	client ForgejoClient
}

func New(client ForgejoClient) *Service { return &Service{client: client} }

func (s *Service) Probe(ctx context.Context) (domain.InstanceCapabilities, *contracts.BridgeError) {
	return s.client.Probe(ctx)
}

func (s *Service) ListRepositories(ctx context.Context, options domain.ListOptions) (domain.RepositoryPage, *contracts.BridgeError) {
	return s.client.ListRepositories(ctx, options)
}

func (s *Service) GetRepository(ctx context.Context, fullName string) (domain.Repository, *contracts.BridgeError) {
	owner, name, err := splitRepository(fullName)
	if err != nil {
		return domain.Repository{}, err
	}
	return s.client.GetRepository(ctx, owner, name)
}

func (s *Service) ListBranches(ctx context.Context, repo string, options domain.ListOptions) (domain.Page[domain.Branch], *contracts.BridgeError) {
	owner, name, err := splitRepository(repo)
	if err != nil {
		return domain.Page[domain.Branch]{}, err
	}
	return s.client.ListBranches(ctx, owner, name, options)
}

func (s *Service) GetBranch(ctx context.Context, repo, branch string) (domain.Branch, *contracts.BridgeError) {
	owner, name, err := splitRepository(repo)
	if err != nil {
		return domain.Branch{}, err
	}
	if strings.TrimSpace(branch) == "" {
		return domain.Branch{}, contracts.NewError("invalid_input", "branch is required")
	}
	return s.client.GetBranch(ctx, owner, name, branch)
}

func (s *Service) ListIssues(ctx context.Context, repo, state string, options domain.ListOptions) (domain.Page[domain.Issue], *contracts.BridgeError) {
	owner, name, err := splitRepository(repo)
	if err != nil {
		return domain.Page[domain.Issue]{}, err
	}
	return s.client.ListIssues(ctx, owner, name, state, options)
}

func (s *Service) GetIssue(ctx context.Context, repo string, number int64) (domain.Issue, *contracts.BridgeError) {
	owner, name, err := splitRepository(repo)
	if err != nil {
		return domain.Issue{}, err
	}
	if number < 1 {
		return domain.Issue{}, contracts.NewError("invalid_input", "issue number must be positive")
	}
	return s.client.GetIssue(ctx, owner, name, number)
}

func (s *Service) ListIssueComments(ctx context.Context, repo string, number int64, options domain.ListOptions) (domain.Page[domain.Comment], *contracts.BridgeError) {
	owner, name, err := splitRepository(repo)
	if err != nil {
		return domain.Page[domain.Comment]{}, err
	}
	if number < 1 {
		return domain.Page[domain.Comment]{}, contracts.NewError("invalid_input", "issue number must be positive")
	}
	return s.client.ListIssueComments(ctx, owner, name, number, options)
}

func (s *Service) ListPullRequests(ctx context.Context, repo, state string, options domain.ListOptions) (domain.Page[domain.PullRequest], *contracts.BridgeError) {
	owner, name, err := splitRepository(repo)
	if err != nil {
		return domain.Page[domain.PullRequest]{}, err
	}
	return s.client.ListPullRequests(ctx, owner, name, state, options)
}

func (s *Service) GetPullRequest(ctx context.Context, repo string, number int64) (domain.PullRequest, *contracts.BridgeError) {
	owner, name, err := splitRepository(repo)
	if err != nil {
		return domain.PullRequest{}, err
	}
	if number < 1 {
		return domain.PullRequest{}, contracts.NewError("invalid_input", "pull request number must be positive")
	}
	return s.client.GetPullRequest(ctx, owner, name, number)
}

func (s *Service) ListPullReviews(ctx context.Context, repo string, number int64, options domain.ListOptions) (domain.Page[domain.Review], *contracts.BridgeError) {
	owner, name, err := splitRepository(repo)
	if err != nil {
		return domain.Page[domain.Review]{}, err
	}
	if number < 1 {
		return domain.Page[domain.Review]{}, contracts.NewError("invalid_input", "pull request number must be positive")
	}
	return s.client.ListPullReviews(ctx, owner, name, number, options)
}

func (s *Service) ListLabels(ctx context.Context, repo string, options domain.ListOptions) (domain.Page[domain.Label], *contracts.BridgeError) {
	owner, name, err := splitRepository(repo)
	if err != nil {
		return domain.Page[domain.Label]{}, err
	}
	return s.client.ListLabels(ctx, owner, name, options)
}

func (s *Service) ListReleases(ctx context.Context, repo string, options domain.ListOptions) (domain.Page[domain.Release], *contracts.BridgeError) {
	owner, name, err := splitRepository(repo)
	if err != nil {
		return domain.Page[domain.Release]{}, err
	}
	return s.client.ListReleases(ctx, owner, name, options)
}

func (s *Service) GetRelease(ctx context.Context, repo string, id int64) (domain.Release, *contracts.BridgeError) {
	owner, name, err := splitRepository(repo)
	if err != nil {
		return domain.Release{}, err
	}
	if id < 1 {
		return domain.Release{}, contracts.NewError("invalid_input", "release id must be positive")
	}
	return s.client.GetRelease(ctx, owner, name, id)
}

func (s *Service) ListActionRuns(ctx context.Context, repo string, options domain.ListOptions) (domain.Page[domain.ActionRun], *contracts.BridgeError) {
	owner, name, err := splitRepository(repo)
	if err != nil {
		return domain.Page[domain.ActionRun]{}, err
	}
	return s.client.ListActionRuns(ctx, owner, name, options)
}

func (s *Service) GetActionRun(ctx context.Context, repo string, id int64) (domain.ActionRun, *contracts.BridgeError) {
	owner, name, err := splitRepository(repo)
	if err != nil {
		return domain.ActionRun{}, err
	}
	if id < 1 {
		return domain.ActionRun{}, contracts.NewError("invalid_input", "action run id must be positive")
	}
	return s.client.GetActionRun(ctx, owner, name, id)
}

func (s *Service) ListActionJobs(ctx context.Context, repo string, id int64) ([]domain.ActionJob, *contracts.BridgeError) {
	owner, name, err := splitRepository(repo)
	if err != nil {
		return nil, err
	}
	if id < 1 {
		return nil, contracts.NewError("invalid_input", "action run id must be positive")
	}
	return s.client.ListActionJobs(ctx, owner, name, id)
}
