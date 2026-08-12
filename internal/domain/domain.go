package domain

type InstanceCapabilities struct {
	Alias             string   `json:"alias"`
	BaseURL           string   `json:"base_url"`
	APIVersion        string   `json:"api_version"`
	ForgejoVersion    string   `json:"forgejo_version"`
	ForgejoMajor      int      `json:"forgejo_major"`
	Compatibility     string   `json:"compatibility_status"`
	Capabilities      []string `json:"capabilities"`
	MaxResponseItems  int      `json:"max_response_items,omitempty"`
	DefaultPageSize   int      `json:"default_page_size,omitempty"`
	OpenAPIAvailable  bool     `json:"openapi_available"`
	AuthenticatedUser string   `json:"authenticated_user,omitempty"`
}

type Repository struct {
	ID            int64  `json:"id"`
	Owner         string `json:"owner"`
	Name          string `json:"name"`
	FullName      string `json:"full_name"`
	Description   string `json:"description,omitempty"`
	Private       bool   `json:"private"`
	DefaultBranch string `json:"default_branch,omitempty"`
	HTMLURL       string `json:"html_url,omitempty"`
	CloneURL      string `json:"clone_url,omitempty"`
	SSHURL        string `json:"ssh_url,omitempty"`
	Archived      bool   `json:"archived"`
}

type ListOptions struct {
	Page  int
	Limit int
}

type RepositoryPage struct {
	Items []Repository
	Page  int
	Limit int
	Total int64
	Next  string
}

type Page[T any] struct {
	Items []T
	Page  int
	Limit int
	Total int64
	Next  string
}

type User struct {
	ID       int64  `json:"id"`
	Login    string `json:"login"`
	FullName string `json:"full_name,omitempty"`
	HTMLURL  string `json:"html_url,omitempty"`
}

type Commit struct {
	ID        string `json:"id"`
	Message   string `json:"message,omitempty"`
	Timestamp string `json:"timestamp,omitempty"`
	URL       string `json:"url,omitempty"`
}

type Branch struct {
	Name              string `json:"name"`
	Commit            Commit `json:"commit"`
	Protected         bool   `json:"protected"`
	RequiredApprovals int    `json:"required_approvals,omitempty"`
	UserCanMerge      bool   `json:"user_can_merge"`
	UserCanPush       bool   `json:"user_can_push"`
}

type Label struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Color       string `json:"color,omitempty"`
	Description string `json:"description,omitempty"`
	Exclusive   bool   `json:"exclusive"`
	Archived    bool   `json:"is_archived"`
	URL         string `json:"url,omitempty"`
}

type Issue struct {
	ID        int64   `json:"id"`
	Number    int64   `json:"number"`
	Title     string  `json:"title"`
	Body      string  `json:"body,omitempty"`
	State     string  `json:"state"`
	HTMLURL   string  `json:"html_url,omitempty"`
	User      *User   `json:"user,omitempty"`
	Labels    []Label `json:"labels"`
	Comments  int     `json:"comments"`
	CreatedAt string  `json:"created_at,omitempty"`
	UpdatedAt string  `json:"updated_at,omitempty"`
	ClosedAt  string  `json:"closed_at,omitempty"`
	IsPull    bool    `json:"is_pull_request"`
}

type Comment struct {
	ID        int64  `json:"id"`
	Body      string `json:"body,omitempty"`
	HTMLURL   string `json:"html_url,omitempty"`
	User      *User  `json:"user,omitempty"`
	CreatedAt string `json:"created_at,omitempty"`
	UpdatedAt string `json:"updated_at,omitempty"`
}

type PullBranch struct {
	Label string `json:"label,omitempty"`
	Ref   string `json:"ref,omitempty"`
	SHA   string `json:"sha,omitempty"`
}

type PullRequest struct {
	ID             int64      `json:"id"`
	Number         int64      `json:"number"`
	Title          string     `json:"title"`
	Body           string     `json:"body,omitempty"`
	State          string     `json:"state"`
	Draft          bool       `json:"draft"`
	Merged         bool       `json:"merged"`
	Mergeable      bool       `json:"mergeable"`
	MergeCommitSHA string     `json:"merge_commit_sha,omitempty"`
	HTMLURL        string     `json:"html_url,omitempty"`
	User           *User      `json:"user,omitempty"`
	Base           PullBranch `json:"base"`
	Head           PullBranch `json:"head"`
	Labels         []Label    `json:"labels"`
	Comments       int        `json:"comments"`
	ReviewComments int        `json:"review_comments"`
	CreatedAt      string     `json:"created_at,omitempty"`
	UpdatedAt      string     `json:"updated_at,omitempty"`
	MergedAt       string     `json:"merged_at,omitempty"`
	ClosedAt       string     `json:"closed_at,omitempty"`
}

type Review struct {
	ID            int64  `json:"id"`
	State         string `json:"state"`
	Body          string `json:"body,omitempty"`
	CommitID      string `json:"commit_id,omitempty"`
	User          *User  `json:"user,omitempty"`
	Official      bool   `json:"official"`
	Dismissed     bool   `json:"dismissed"`
	Stale         bool   `json:"stale"`
	CommentsCount int    `json:"comments_count"`
	SubmittedAt   string `json:"submitted_at,omitempty"`
	UpdatedAt     string `json:"updated_at,omitempty"`
	HTMLURL       string `json:"html_url,omitempty"`
}

type Release struct {
	ID              int64  `json:"id"`
	TagName         string `json:"tag_name"`
	TargetCommitish string `json:"target_commitish,omitempty"`
	Name            string `json:"name,omitempty"`
	Body            string `json:"body,omitempty"`
	Draft           bool   `json:"draft"`
	Prerelease      bool   `json:"prerelease"`
	HTMLURL         string `json:"html_url,omitempty"`
	Author          *User  `json:"author,omitempty"`
	CreatedAt       string `json:"created_at,omitempty"`
	PublishedAt     string `json:"published_at,omitempty"`
}

type ActionRun struct {
	ID           int64  `json:"id"`
	Index        int64  `json:"index_in_repo"`
	Title        string `json:"title,omitempty"`
	Status       string `json:"status"`
	Event        string `json:"event,omitempty"`
	TriggerEvent string `json:"trigger_event,omitempty"`
	WorkflowID   string `json:"workflow_id,omitempty"`
	CommitSHA    string `json:"commit_sha,omitempty"`
	Ref          string `json:"prettyref,omitempty"`
	HTMLURL      string `json:"html_url,omitempty"`
	NeedApproval bool   `json:"need_approval"`
	TriggerUser  *User  `json:"trigger_user,omitempty"`
	CreatedAt    string `json:"created,omitempty"`
	StartedAt    string `json:"started,omitempty"`
	StoppedAt    string `json:"stopped,omitempty"`
	UpdatedAt    string `json:"updated,omitempty"`
}

type ActionJob struct {
	ID      int64    `json:"id"`
	RunID   int64    `json:"run_id"`
	Name    string   `json:"name"`
	Status  string   `json:"status"`
	Attempt int      `json:"attempt"`
	RunsOn  []string `json:"runs_on"`
	Needs   []string `json:"needs"`
}
