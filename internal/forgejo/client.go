package forgejo

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/quasarea/forgejo-bridge/internal/config"
	"github.com/quasarea/forgejo-bridge/internal/contracts"
	"github.com/quasarea/forgejo-bridge/internal/domain"
)

const maxJSONResponse = 8 << 20

type Client struct {
	alias      string
	baseURL    string
	apiURL     string
	token      string
	httpClient *http.Client
	allowed    func(string) bool
	allowlist  []string
}

func New(instance config.NamedInstance) (*Client, error) {
	token, err := instance.ResolveCredential()
	if err != nil {
		return nil, contracts.WrapError("authentication_required", "could not resolve instance credential", err)
	}

	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.TLSClientConfig = &tls.Config{MinVersion: tls.VersionTLS12}
	if instance.TLSCAFile != "" {
		pem, err := os.ReadFile(instance.TLSCAFile)
		if err != nil {
			return nil, fmt.Errorf("read TLS CA file: %w", err)
		}
		pool, err := x509.SystemCertPool()
		if err != nil || pool == nil {
			pool = x509.NewCertPool()
		}
		if !pool.AppendCertsFromPEM(pem) {
			return nil, errors.New("TLS CA file contains no certificates")
		}
		transport.TLSClientConfig.RootCAs = pool
	}

	return &Client{
		alias:   instance.Alias,
		baseURL: instance.BaseURL,
		apiURL:  instance.APIURL,
		token:   token,
		httpClient: &http.Client{
			Timeout:   30 * time.Second,
			Transport: transport,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) > 0 && !sameOrigin(req.URL, via[0].URL) {
					return errors.New("cross-origin redirect rejected")
				}
				if len(via) >= 5 {
					return errors.New("too many redirects")
				}
				return nil
			},
		},
		allowed:   instance.RepositoryAllowed,
		allowlist: append([]string(nil), instance.AllowedRepositories...),
	}, nil
}

func (c *Client) Probe(ctx context.Context) (domain.InstanceCapabilities, *contracts.BridgeError) {
	var version struct {
		Version string `json:"version"`
	}
	if _, err := c.getJSON(ctx, c.apiURL+"/version", &version); err != nil {
		return domain.InstanceCapabilities{}, err
	}
	major := parseMajor(version.Version)
	compatibility := "unsupported"
	if major == 15 || major == 16 {
		compatibility = "certified"
	}

	result := domain.InstanceCapabilities{
		Alias:          c.alias,
		BaseURL:        c.baseURL,
		APIVersion:     "v1",
		ForgejoVersion: version.Version,
		ForgejoMajor:   major,
		Compatibility:  compatibility,
		Capabilities: []string{
			"repository.read", "branch.read", "issue.read", "issue.comment.read",
			"pull_request.read", "review.read", "release.read", "label.read",
		},
	}
	var settings struct {
		MaxResponseItems int `json:"max_response_items"`
		DefaultPagingNum int `json:"default_paging_num"`
	}
	if _, err := c.getJSON(ctx, c.apiURL+"/settings/api", &settings); err == nil {
		result.MaxResponseItems = settings.MaxResponseItems
		result.DefaultPageSize = settings.DefaultPagingNum
	}

	var user struct {
		Login string `json:"login"`
	}
	if c.token != "" {
		if _, err := c.getJSON(ctx, c.apiURL+"/user", &user); err == nil {
			result.AuthenticatedUser = user.Login
		}
	}
	if major >= 16 && c.actionsAvailable(ctx) {
		result.Capabilities = append(result.Capabilities,
			"actions.run.read", "actions.job.read")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/swagger.v1.json", nil)
	if err == nil {
		response, requestErr := c.do(req)
		if requestErr == nil {
			result.OpenAPIAvailable = response.StatusCode >= 200 && response.StatusCode < 300
			io.Copy(io.Discard, io.LimitReader(response.Body, 4096))
			response.Body.Close()
		}
	}
	return result, nil
}

func (c *Client) actionsAvailable(ctx context.Context) bool {
	if c.token == "" || len(c.allowlist) == 0 {
		return false
	}
	owner, name, ok := strings.Cut(c.allowlist[0], "/")
	if !ok || owner == "" || name == "" || strings.Contains(name, "/") {
		return false
	}
	var response struct {
		Runs []json.RawMessage `json:"workflow_runs"`
	}
	_, err := c.getJSON(ctx, c.apiURL+"/repos/"+url.PathEscape(owner)+"/"+url.PathEscape(name)+"/actions/runs?limit=1", &response)
	return err == nil
}

func (c *Client) ListRepositories(ctx context.Context, options domain.ListOptions) (domain.RepositoryPage, *contracts.BridgeError) {
	if c.token == "" {
		return domain.RepositoryPage{}, contracts.NewError("authentication_required", "repository listing requires an authenticated credential")
	}
	if options.Page < 1 {
		options.Page = 1
	}
	if options.Limit < 1 {
		options.Limit = 30
	}
	endpoint := fmt.Sprintf("%s/user/repos?page=%d&limit=%d", c.apiURL, options.Page, options.Limit)
	var raw []repositoryDTO
	response, err := c.getJSON(ctx, endpoint, &raw)
	if err != nil {
		return domain.RepositoryPage{}, err
	}
	items := make([]domain.Repository, 0, len(raw))
	for _, repository := range raw {
		mapped := repository.mapDomain()
		if c.allowed(mapped.FullName) {
			items = append(items, mapped)
		}
	}
	total, _ := strconv.ParseInt(response.Header.Get("X-Total-Count"), 10, 64)
	return domain.RepositoryPage{
		Items: items,
		Page:  options.Page,
		Limit: options.Limit,
		Total: total,
		Next:  nextLink(response.Header.Get("Link")),
	}, nil
}

func (c *Client) GetRepository(ctx context.Context, owner, name string) (domain.Repository, *contracts.BridgeError) {
	fullName := owner + "/" + name
	if !c.allowed(fullName) {
		return domain.Repository{}, contracts.NewError("permission_denied", "repository is outside the configured allowlist")
	}
	var raw repositoryDTO
	_, err := c.getJSON(ctx, c.apiURL+"/repos/"+url.PathEscape(owner)+"/"+url.PathEscape(name), &raw)
	if err != nil {
		return domain.Repository{}, err
	}
	return raw.mapDomain(), nil
}

func (c *Client) getJSON(ctx context.Context, endpoint string, target any) (*http.Response, *contracts.BridgeError) {
	var last *contracts.BridgeError
	for attempt := 0; attempt < 3; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
		if err != nil {
			return nil, contracts.WrapError("invalid_input", "invalid request URL", err)
		}
		response, bridgeErr := c.do(req)
		if bridgeErr != nil {
			last = bridgeErr
			if !bridgeErr.Retryable || attempt == 2 {
				return nil, bridgeErr
			}
			timer := time.NewTimer(time.Duration(100*(1<<attempt)) * time.Millisecond)
			select {
			case <-ctx.Done():
				timer.Stop()
				return nil, contracts.WrapError("cancelled", "request was cancelled", ctx.Err())
			case <-timer.C:
			}
			continue
		}
		defer response.Body.Close()
		decoder := json.NewDecoder(io.LimitReader(response.Body, maxJSONResponse))
		if err := decoder.Decode(target); err != nil {
			return nil, contracts.WrapError("response_invalid", "Forgejo returned invalid JSON", err)
		}
		return response, nil
	}
	return nil, last
}

func (c *Client) do(req *http.Request) (*http.Response, *contracts.BridgeError) {
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "forgejo-bridge/0.1")
	if c.token != "" {
		req.Header.Set("Authorization", "token "+c.token)
	}
	response, err := c.httpClient.Do(req)
	if err != nil {
		return nil, &contracts.BridgeError{Code: "transport_error", Message: "could not reach Forgejo", Retryable: true, Cause: err}
	}
	if response.StatusCode >= 200 && response.StatusCode < 300 {
		return response, nil
	}
	defer response.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(response.Body, 64<<10))
	message := strings.TrimSpace(string(body))
	if message == "" {
		message = response.Status
	}
	code := "upstream_error"
	retryable := response.StatusCode == http.StatusTooManyRequests || response.StatusCode >= 500
	switch response.StatusCode {
	case http.StatusBadRequest, http.StatusUnprocessableEntity:
		code = "invalid_input"
	case http.StatusUnauthorized:
		code = "authentication_failed"
	case http.StatusForbidden:
		code = "permission_denied"
	case http.StatusNotFound:
		code = "resource_not_found"
	case http.StatusConflict:
		code = "conflict"
	case http.StatusPreconditionFailed:
		code = "precondition_failed"
	case http.StatusTooManyRequests:
		code = "rate_limited"
	}
	return nil, &contracts.BridgeError{
		Code:       code,
		Message:    message,
		Retryable:  retryable,
		HTTPStatus: response.StatusCode,
	}
}

type repositoryDTO struct {
	ID            int64  `json:"id"`
	Name          string `json:"name"`
	FullName      string `json:"full_name"`
	Description   string `json:"description"`
	Private       bool   `json:"private"`
	DefaultBranch string `json:"default_branch"`
	HTMLURL       string `json:"html_url"`
	CloneURL      string `json:"clone_url"`
	SSHURL        string `json:"ssh_url"`
	Archived      bool   `json:"archived"`
	Owner         struct {
		Login string `json:"login"`
	} `json:"owner"`
}

func (r repositoryDTO) mapDomain() domain.Repository {
	return domain.Repository{
		ID: r.ID, Owner: r.Owner.Login, Name: r.Name, FullName: r.FullName,
		Description: r.Description, Private: r.Private, DefaultBranch: r.DefaultBranch,
		HTMLURL: r.HTMLURL, CloneURL: r.CloneURL, SSHURL: r.SSHURL, Archived: r.Archived,
	}
}

func parseMajor(version string) int {
	major, _ := strconv.Atoi(strings.SplitN(version, ".", 2)[0])
	return major
}

func nextLink(header string) string {
	for _, part := range strings.Split(header, ",") {
		sections := strings.Split(part, ";")
		if len(sections) < 2 || !strings.Contains(sections[1], `rel="next"`) {
			continue
		}
		return strings.Trim(strings.TrimSpace(sections[0]), "<>")
	}
	return ""
}

func sameOrigin(left, right *url.URL) bool {
	return strings.EqualFold(left.Scheme, right.Scheme) && strings.EqualFold(left.Host, right.Host)
}
