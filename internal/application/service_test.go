package application

import (
	"context"
	"testing"

	"github.com/quasarea/forgejo-bridge/internal/contracts"
	"github.com/quasarea/forgejo-bridge/internal/domain"
)

type fakeClient struct {
	owner string
	name  string
}

func (f *fakeClient) Probe(context.Context) (domain.InstanceCapabilities, *contracts.BridgeError) {
	return domain.InstanceCapabilities{}, nil
}

func (f *fakeClient) ListRepositories(context.Context, domain.ListOptions) (domain.RepositoryPage, *contracts.BridgeError) {
	return domain.RepositoryPage{}, nil
}

func (f *fakeClient) GetRepository(_ context.Context, owner, name string) (domain.Repository, *contracts.BridgeError) {
	f.owner, f.name = owner, name
	return domain.Repository{FullName: owner + "/" + name}, nil
}

func (f *fakeClient) ListBranches(context.Context, string, string, domain.ListOptions) (domain.Page[domain.Branch], *contracts.BridgeError) {
	return domain.Page[domain.Branch]{}, nil
}
func (f *fakeClient) GetBranch(context.Context, string, string, string) (domain.Branch, *contracts.BridgeError) {
	return domain.Branch{}, nil
}
func (f *fakeClient) ListIssues(context.Context, string, string, string, domain.ListOptions) (domain.Page[domain.Issue], *contracts.BridgeError) {
	return domain.Page[domain.Issue]{}, nil
}
func (f *fakeClient) GetIssue(context.Context, string, string, int64) (domain.Issue, *contracts.BridgeError) {
	return domain.Issue{}, nil
}
func (f *fakeClient) ListIssueComments(context.Context, string, string, int64, domain.ListOptions) (domain.Page[domain.Comment], *contracts.BridgeError) {
	return domain.Page[domain.Comment]{}, nil
}
func (f *fakeClient) ListPullRequests(context.Context, string, string, string, domain.ListOptions) (domain.Page[domain.PullRequest], *contracts.BridgeError) {
	return domain.Page[domain.PullRequest]{}, nil
}
func (f *fakeClient) GetPullRequest(context.Context, string, string, int64) (domain.PullRequest, *contracts.BridgeError) {
	return domain.PullRequest{}, nil
}
func (f *fakeClient) ListPullReviews(context.Context, string, string, int64, domain.ListOptions) (domain.Page[domain.Review], *contracts.BridgeError) {
	return domain.Page[domain.Review]{}, nil
}
func (f *fakeClient) ListLabels(context.Context, string, string, domain.ListOptions) (domain.Page[domain.Label], *contracts.BridgeError) {
	return domain.Page[domain.Label]{}, nil
}
func (f *fakeClient) ListReleases(context.Context, string, string, domain.ListOptions) (domain.Page[domain.Release], *contracts.BridgeError) {
	return domain.Page[domain.Release]{}, nil
}
func (f *fakeClient) GetRelease(context.Context, string, string, int64) (domain.Release, *contracts.BridgeError) {
	return domain.Release{}, nil
}
func (f *fakeClient) ListActionRuns(context.Context, string, string, domain.ListOptions) (domain.Page[domain.ActionRun], *contracts.BridgeError) {
	return domain.Page[domain.ActionRun]{}, nil
}
func (f *fakeClient) GetActionRun(context.Context, string, string, int64) (domain.ActionRun, *contracts.BridgeError) {
	return domain.ActionRun{}, nil
}
func (f *fakeClient) ListActionJobs(context.Context, string, string, int64) ([]domain.ActionJob, *contracts.BridgeError) {
	return nil, nil
}

func TestGetRepositoryValidatesFullName(t *testing.T) {
	fake := &fakeClient{}
	service := New(fake)
	result, err := service.GetRepository(context.Background(), "team/service")
	if err != nil {
		t.Fatal(err)
	}
	if result.FullName != "team/service" || fake.owner != "team" || fake.name != "service" {
		t.Fatalf("unexpected result: %#v", result)
	}
	if _, err := service.GetRepository(context.Background(), "not-valid"); err == nil || err.Code != "invalid_input" {
		t.Fatalf("expected invalid_input, got %#v", err)
	}
}
