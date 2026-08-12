package forgejo

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/quasarea/forgejo-bridge/internal/config"
	"github.com/quasarea/forgejo-bridge/internal/domain"
)

func TestProbeAndRepositoryMapping(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/version" && r.Header.Get("Authorization") != "token test-token" {
			http.Error(w, "missing token", http.StatusUnauthorized)
			return
		}
		switch r.URL.Path {
		case "/api/v1/version":
			fmt.Fprint(w, `{"version":"16.0.2"}`)
		case "/api/v1/settings/api":
			fmt.Fprint(w, `{"max_response_items":50,"default_paging_num":30}`)
		case "/api/v1/user":
			fmt.Fprint(w, `{"login":"alice"}`)
		case "/swagger.v1.json":
			fmt.Fprint(w, `{}`)
		case "/api/v1/user/repos":
			w.Header().Set("X-Total-Count", "1")
			fmt.Fprint(w, `[{"id":42,"name":"service","full_name":"team/service","private":true,"default_branch":"main","owner":{"login":"team"}}]`)
		case "/api/v1/repos/team/service":
			fmt.Fprint(w, `{"id":42,"name":"service","full_name":"team/service","private":true,"default_branch":"main","owner":{"login":"team"}}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	t.Setenv("TEST_FORGEJO_TOKEN", "test-token")
	client, err := New(config.NamedInstance{Alias: "test", Instance: config.Instance{
		BaseURL: server.URL, APIURL: server.URL + "/api/v1", Credential: "env:TEST_FORGEJO_TOKEN",
		AllowedRepositories: []string{"team/service"},
	}})
	if err != nil {
		t.Fatal(err)
	}
	probe, bridgeErr := client.Probe(context.Background())
	if bridgeErr != nil {
		t.Fatal(bridgeErr)
	}
	if probe.ForgejoMajor != 16 || probe.Compatibility != "certified" || !probe.OpenAPIAvailable || probe.AuthenticatedUser != "alice" {
		t.Fatalf("unexpected probe: %#v", probe)
	}
	page, bridgeErr := client.ListRepositories(context.Background(), domain.ListOptions{})
	if bridgeErr != nil {
		t.Fatal(bridgeErr)
	}
	if len(page.Items) != 1 || page.Items[0].ID != 42 || page.Total != 1 {
		t.Fatalf("unexpected repository page: %#v", page)
	}
	repository, bridgeErr := client.GetRepository(context.Background(), "team", "service")
	if bridgeErr != nil || repository.FullName != "team/service" {
		t.Fatalf("unexpected repository: %#v err=%v", repository, bridgeErr)
	}
}

func TestRepositoryAllowlistBlocksBeforeRequest(t *testing.T) {
	client := &Client{allowed: func(string) bool { return false }}
	_, err := client.GetRepository(context.Background(), "other", "repo")
	if err == nil || err.Code != "permission_denied" {
		t.Fatalf("expected permission_denied, got %#v", err)
	}
}

func TestResourceReadsAndPagination(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/version" && r.Header.Get("Authorization") != "token test-token" {
			http.Error(w, "missing token", http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v1/version":
			fmt.Fprint(w, `{"version":"16.0.2"}`)
		case "/api/v1/repos/team/service/branches":
			w.Header().Set("X-Total-Count", "1")
			w.Header().Set("Link", `<https://forge.example/api/v1/repos/team/service/branches?page=2>; rel="next"`)
			fmt.Fprint(w, `[{"name":"main","commit":{"id":"abc"},"protected":true}]`)
		case "/api/v1/repos/team/service/branches/main":
			fmt.Fprint(w, `{"name":"main","commit":{"id":"abc"}}`)
		case "/api/v1/repos/team/service/issues":
			if r.URL.Query().Get("state") != "all" || r.URL.Query().Get("page") != "2" {
				http.Error(w, "missing filters", http.StatusBadRequest)
				return
			}
			fmt.Fprint(w, `[{"id":1,"number":7,"title":"bug","state":"open","labels":[],"pull_request":{"url":"x"}}]`)
		case "/api/v1/repos/team/service/issues/7":
			fmt.Fprint(w, `{"id":1,"number":7,"title":"bug","state":"open","labels":[]}`)
		case "/api/v1/repos/team/service/issues/7/comments":
			fmt.Fprint(w, `[{"id":8,"body":"note","user":{"id":2,"login":"alice"}}]`)
		case "/api/v1/repos/team/service/pulls":
			fmt.Fprint(w, `[{"id":3,"number":9,"title":"change","state":"open","base":{"ref":"main"},"head":{"ref":"topic"},"labels":[]}]`)
		case "/api/v1/repos/team/service/pulls/9":
			fmt.Fprint(w, `{"id":3,"number":9,"title":"change","state":"open","base":{"ref":"main"},"head":{"ref":"topic"},"labels":[]}`)
		case "/api/v1/repos/team/service/pulls/9/reviews":
			fmt.Fprint(w, `[{"id":4,"state":"APPROVED","user":{"login":"alice"}}]`)
		case "/api/v1/repos/team/service/labels":
			fmt.Fprint(w, `[{"id":5,"name":"bug","color":"ff0000"}]`)
		case "/api/v1/repos/team/service/releases":
			fmt.Fprint(w, `[{"id":6,"tag_name":"v1.0.0","name":"one"}]`)
		case "/api/v1/repos/team/service/releases/6":
			fmt.Fprint(w, `{"id":6,"tag_name":"v1.0.0","name":"one"}`)
		case "/api/v1/repos/team/service/actions/runs":
			fmt.Fprint(w, `{"total_count":1,"workflow_runs":[{"id":10,"index_in_repo":2,"status":"success","workflow_id":"test.yml"}]}`)
		case "/api/v1/repos/team/service/actions/runs/10":
			fmt.Fprint(w, `{"id":10,"index_in_repo":2,"status":"success"}`)
		case "/api/v1/repos/team/service/actions/runs/10/jobs":
			fmt.Fprint(w, `[{"id":11,"run_id":10,"name":"test","status":"success","runs_on":["docker"]}]`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	t.Setenv("TEST_FORGEJO_TOKEN", "test-token")
	client, err := New(config.NamedInstance{Alias: "test", Instance: config.Instance{
		BaseURL: server.URL, APIURL: server.URL + "/api/v1", Credential: "env:TEST_FORGEJO_TOKEN",
		AllowedRepositories: []string{"team/service"},
	}})
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	branches, bridgeErr := client.ListBranches(ctx, "team", "service", domain.ListOptions{Limit: 10})
	if bridgeErr != nil || len(branches.Items) != 1 || branches.Total != 1 || branches.Next == "" {
		t.Fatalf("branches = %#v err=%v", branches, bridgeErr)
	}
	if branch, bridgeErr := client.GetBranch(ctx, "team", "service", "main"); bridgeErr != nil || branch.Commit.ID != "abc" {
		t.Fatalf("branch = %#v err=%v", branch, bridgeErr)
	}
	issues, bridgeErr := client.ListIssues(ctx, "team", "service", "all", domain.ListOptions{Page: 2})
	if bridgeErr != nil || len(issues.Items) != 1 || !issues.Items[0].IsPull {
		t.Fatalf("issues = %#v err=%v", issues, bridgeErr)
	}
	issue, bridgeErr := client.GetIssue(ctx, "team", "service", 7)
	if bridgeErr != nil || issue.Number != 7 || issue.IsPull {
		t.Fatalf("issue = %#v err=%v", issue, bridgeErr)
	}
	comments, bridgeErr := client.ListIssueComments(ctx, "team", "service", 7, domain.ListOptions{})
	if bridgeErr != nil || len(comments.Items) != 1 || comments.Items[0].User.Login != "alice" {
		t.Fatalf("comments = %#v err=%v", comments, bridgeErr)
	}
	pulls, bridgeErr := client.ListPullRequests(ctx, "team", "service", "open", domain.ListOptions{})
	if bridgeErr != nil || len(pulls.Items) != 1 || pulls.Items[0].Head.Ref != "topic" {
		t.Fatalf("pulls = %#v err=%v", pulls, bridgeErr)
	}
	if pull, err := client.GetPullRequest(ctx, "team", "service", 9); err != nil || pull.Base.Ref != "main" {
		t.Fatalf("pull = %#v err=%v", pull, err)
	}
	if reviews, err := client.ListPullReviews(ctx, "team", "service", 9, domain.ListOptions{}); err != nil || len(reviews.Items) != 1 {
		t.Fatalf("reviews = %#v err=%v", reviews, err)
	}
	if labels, err := client.ListLabels(ctx, "team", "service", domain.ListOptions{}); err != nil || labels.Items[0].Name != "bug" {
		t.Fatalf("labels = %#v err=%v", labels, err)
	}
	if releases, err := client.ListReleases(ctx, "team", "service", domain.ListOptions{}); err != nil || releases.Items[0].TagName != "v1.0.0" {
		t.Fatalf("releases = %#v err=%v", releases, err)
	}
	if release, err := client.GetRelease(ctx, "team", "service", 6); err != nil || release.ID != 6 {
		t.Fatalf("release = %#v err=%v", release, err)
	}
	runs, bridgeErr := client.ListActionRuns(ctx, "team", "service", domain.ListOptions{})
	if bridgeErr != nil || len(runs.Items) != 1 || runs.Total != 1 {
		t.Fatalf("runs = %#v err=%v", runs, bridgeErr)
	}
	if run, err := client.GetActionRun(ctx, "team", "service", 10); err != nil || run.ID != 10 {
		t.Fatalf("run = %#v err=%v", run, err)
	}
	if jobs, err := client.ListActionJobs(ctx, "team", "service", 10); err != nil || len(jobs) != 1 || jobs[0].RunsOn[0] != "docker" {
		t.Fatalf("jobs = %#v err=%v", jobs, err)
	}
}

func TestActionsRequireForgejo16(t *testing.T) {
	requestedActions := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/version" {
			fmt.Fprint(w, `{"version":"15.0.6"}`)
			return
		}
		requestedActions = true
		http.NotFound(w, r)
	}))
	defer server.Close()
	client := &Client{apiURL: server.URL + "/api/v1", httpClient: server.Client(), allowed: func(string) bool { return true }}
	_, bridgeErr := client.ListActionRuns(context.Background(), "team", "service", domain.ListOptions{})
	if bridgeErr == nil || bridgeErr.Code != "capability_unsupported" || requestedActions {
		t.Fatalf("error = %#v requestedActions=%v", bridgeErr, requestedActions)
	}
}

func TestDisabledActionsAreNotAdvertisedAndReturnCapabilityError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/version":
			fmt.Fprint(w, `{"version":"16.0.2"}`)
		case "/api/v1/user":
			fmt.Fprint(w, `{"login":"bridge"}`)
		case "/api/v1/repos/team/service/actions/runs":
			http.NotFound(w, r)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	t.Setenv("DISABLED_ACTIONS_TOKEN", "read-token")
	client, err := New(config.NamedInstance{Alias: "test", Instance: config.Instance{
		BaseURL: server.URL, APIURL: server.URL + "/api/v1", Credential: "env:DISABLED_ACTIONS_TOKEN",
		AllowedRepositories: []string{"team/service"},
	}})
	if err != nil {
		t.Fatal(err)
	}
	probe, bridgeErr := client.Probe(context.Background())
	if bridgeErr != nil {
		t.Fatal(bridgeErr)
	}
	for _, capability := range probe.Capabilities {
		if capability == "actions.run.read" || capability == "actions.job.read" {
			t.Fatalf("disabled Actions capability was advertised: %#v", probe.Capabilities)
		}
	}
	_, bridgeErr = client.ListActionRuns(context.Background(), "team", "service", domain.ListOptions{})
	if bridgeErr == nil || bridgeErr.Code != "capability_unsupported" {
		t.Fatalf("expected capability_unsupported, got %#v", bridgeErr)
	}
}
