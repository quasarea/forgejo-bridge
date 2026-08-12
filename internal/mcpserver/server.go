package mcpserver

import (
	"context"
	"errors"
	"flag"
	"os"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/quasarea/forgejo-bridge/internal/application"
	"github.com/quasarea/forgejo-bridge/internal/cli"
	"github.com/quasarea/forgejo-bridge/internal/config"
	"github.com/quasarea/forgejo-bridge/internal/contracts"
	"github.com/quasarea/forgejo-bridge/internal/domain"
	"github.com/quasarea/forgejo-bridge/internal/forgejo"
	"github.com/quasarea/forgejo-bridge/internal/requestid"
)

type server struct {
	configPath string
}

type EmptyInput struct{}

type InstanceInput struct {
	Instance string `json:"instance,omitempty" jsonschema:"configured Forgejo instance alias; omit only when a default is unambiguous"`
}

type RepositoryListInput struct {
	Instance string `json:"instance,omitempty" jsonschema:"configured Forgejo instance alias"`
	Page     int    `json:"page,omitempty" jsonschema:"one-based page number"`
	Limit    int    `json:"limit,omitempty" jsonschema:"maximum repositories to return"`
}

type RepositoryGetInput struct {
	Instance   string `json:"instance,omitempty" jsonschema:"configured Forgejo instance alias"`
	Repository string `json:"repository" jsonschema:"repository in owner/name form"`
}

func RunStdio(ctx context.Context, args []string) error {
	set := flag.NewFlagSet("mcp stdio", flag.ContinueOnError)
	set.SetOutput(os.Stderr)
	configPath := set.String("config", "", "configuration path")
	if err := set.Parse(args); err != nil {
		return err
	}

	return newMCPServer(*configPath).Run(ctx, &mcp.StdioTransport{})
}

func newMCPServer(configPath string) *mcp.Server {
	adapter := &server{configPath: configPath}
	mcpServer := mcp.NewServer(&mcp.Implementation{Name: "forgejo-bridge", Version: cli.Version}, nil)

	mcp.AddTool(mcpServer, &mcp.Tool{
		Name:        "forgejo_instance_list",
		Description: "List configured Forgejo instance aliases without returning credentials.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true, OpenWorldHint: boolPtr(false)},
	}, adapter.listInstances)
	mcp.AddTool(mcpServer, &mcp.Tool{
		Name:        "forgejo_instance_probe",
		Description: "Discover the selected Forgejo version, API limits, identity, and supported bridge capabilities.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true, OpenWorldHint: boolPtr(true)},
	}, adapter.probeInstance)
	mcp.AddTool(mcpServer, &mcp.Tool{
		Name:        "forgejo_repo_list",
		Description: "List repositories visible to the configured Forgejo credential, filtered by the instance allowlist.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true, OpenWorldHint: boolPtr(true)},
	}, adapter.listRepositories)
	mcp.AddTool(mcpServer, &mcp.Tool{
		Name:        "forgejo_repo_get",
		Description: "Get normalized metadata for one explicitly named Forgejo repository.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true, OpenWorldHint: boolPtr(true)},
	}, adapter.getRepository)
	adapter.registerResourceTools(mcpServer)

	return mcpServer
}

func (s *server) listInstances(_ context.Context, _ *mcp.CallToolRequest, _ EmptyInput) (*mcp.CallToolResult, contracts.Envelope, error) {
	cfg, bridgeErr := s.loadConfig()
	if bridgeErr != nil {
		return toolFailure("instance.list", "", bridgeErr)
	}
	type item struct {
		Alias    string `json:"alias"`
		BaseURL  string `json:"base_url"`
		ReadOnly bool   `json:"read_only"`
		Default  bool   `json:"default"`
	}
	items := make([]item, 0, len(cfg.Instances))
	for _, alias := range cfg.Aliases() {
		instance := cfg.Instances[alias]
		items = append(items, item{Alias: alias, BaseURL: instance.BaseURL, ReadOnly: instance.ReadOnly, Default: alias == cfg.DefaultInstance})
	}
	return nil, contracts.Success("instance.list", requestid.New(), "", items), nil
}

func (s *server) probeInstance(ctx context.Context, _ *mcp.CallToolRequest, input InstanceInput) (*mcp.CallToolResult, contracts.Envelope, error) {
	service, alias, bridgeErr := s.service(input.Instance)
	if bridgeErr != nil {
		return toolFailure("instance.probe", input.Instance, bridgeErr)
	}
	result, bridgeErr := service.Probe(ctx)
	if bridgeErr != nil {
		return toolFailure("instance.probe", alias, bridgeErr)
	}
	envelope := contracts.Success("instance.probe", requestid.New(), alias, result)
	envelope.Meta.ForgejoVersion = result.ForgejoVersion
	envelope.Meta.Capabilities = result.Capabilities
	return nil, envelope, nil
}

func (s *server) listRepositories(ctx context.Context, _ *mcp.CallToolRequest, input RepositoryListInput) (*mcp.CallToolResult, contracts.Envelope, error) {
	service, alias, bridgeErr := s.service(input.Instance)
	if bridgeErr != nil {
		return toolFailure("repo.list", input.Instance, bridgeErr)
	}
	result, bridgeErr := service.ListRepositories(ctx, domain.ListOptions{Page: input.Page, Limit: input.Limit})
	if bridgeErr != nil {
		return toolFailure("repo.list", alias, bridgeErr)
	}
	envelope := contracts.Success("repo.list", requestid.New(), alias, result.Items)
	envelope.Page = &contracts.Page{Number: result.Page, Limit: result.Limit, Total: result.Total, Next: result.Next}
	return nil, envelope, nil
}

func (s *server) getRepository(ctx context.Context, _ *mcp.CallToolRequest, input RepositoryGetInput) (*mcp.CallToolResult, contracts.Envelope, error) {
	service, alias, bridgeErr := s.service(input.Instance)
	if bridgeErr != nil {
		return toolFailure("repo.get", input.Instance, bridgeErr)
	}
	result, bridgeErr := service.GetRepository(ctx, input.Repository)
	if bridgeErr != nil {
		return toolFailure("repo.get", alias, bridgeErr)
	}
	return nil, contracts.Success("repo.get", requestid.New(), alias, result), nil
}

func (s *server) loadConfig() (*config.Config, *contracts.BridgeError) {
	cfg, err := config.Load(s.configPath)
	if err != nil {
		return nil, contracts.WrapError("invalid_input", "could not load configuration", err)
	}
	return cfg, nil
}

func (s *server) service(alias string) (*application.Service, string, *contracts.BridgeError) {
	cfg, bridgeErr := s.loadConfig()
	if bridgeErr != nil {
		return nil, alias, bridgeErr
	}
	selected, err := cfg.Select(alias)
	if err != nil {
		code := "instance_not_found"
		if alias == "" {
			code = "instance_ambiguous"
		}
		return nil, alias, contracts.WrapError(code, "could not select instance", err)
	}
	client, err := forgejo.New(selected)
	if err != nil {
		var bridgeErr *contracts.BridgeError
		if errors.As(err, &bridgeErr) {
			return nil, selected.Alias, bridgeErr
		}
		return nil, selected.Alias, contracts.WrapError("invalid_input", "could not initialize Forgejo client", err)
	}
	return application.New(client), selected.Alias, nil
}

func toolFailure(operation, instance string, bridgeErr *contracts.BridgeError) (*mcp.CallToolResult, contracts.Envelope, error) {
	return &mcp.CallToolResult{IsError: true}, contracts.Failure(operation, requestid.New(), instance, bridgeErr), nil
}

func boolPtr(value bool) *bool { return &value }
