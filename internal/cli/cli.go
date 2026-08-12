package cli

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"strconv"

	"github.com/quasarea/forgejo-bridge/internal/application"
	"github.com/quasarea/forgejo-bridge/internal/config"
	"github.com/quasarea/forgejo-bridge/internal/contracts"
	"github.com/quasarea/forgejo-bridge/internal/domain"
	"github.com/quasarea/forgejo-bridge/internal/forgejo"
	"github.com/quasarea/forgejo-bridge/internal/requestid"
)

var (
	Version   = "0.2.0"
	Commit    = "unknown"
	BuildDate = "unknown"
)

type Runner struct {
	Stdout io.Writer
	Stderr io.Writer
}

func (r Runner) Run(ctx context.Context, args []string) int {
	if len(args) == 0 {
		return r.writeFailure("command", "", contracts.NewError("invalid_input", "a command is required"))
	}
	switch args[0] {
	case "version":
		return r.writeSuccess("version", "", map[string]string{
			"version":    Version,
			"commit":     Commit,
			"build_date": BuildDate,
		})
	case "instance":
		return r.runInstance(ctx, args[1:])
	case "repo":
		return r.runRepo(ctx, args[1:])
	case "branch":
		return r.runBranch(ctx, args[1:])
	case "issue":
		return r.runIssue(ctx, args[1:])
	case "pr":
		return r.runPullRequest(ctx, args[1:])
	case "label":
		return r.runLabel(ctx, args[1:])
	case "release":
		return r.runRelease(ctx, args[1:])
	case "actions":
		return r.runActions(ctx, args[1:])
	default:
		return r.writeFailure("command", "", contracts.NewError("invalid_input", "unknown command: "+args[0]))
	}
}

func (r Runner) runInstance(ctx context.Context, args []string) int {
	if len(args) == 0 {
		return r.writeFailure("instance", "", contracts.NewError("invalid_input", "an instance command is required"))
	}
	switch args[0] {
	case "list":
		set := flag.NewFlagSet("instance list", flag.ContinueOnError)
		set.SetOutput(r.Stderr)
		path := set.String("config", "", "configuration path")
		if err := set.Parse(args[1:]); err != nil {
			return r.writeFailure("instance.list", "", contracts.WrapError("invalid_input", "invalid flags", err))
		}
		cfg, bridgeErr := loadConfig(*path)
		if bridgeErr != nil {
			return r.writeFailure("instance.list", "", bridgeErr)
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
		return r.writeSuccess("instance.list", "", items)
	case "probe":
		set := flag.NewFlagSet("instance probe", flag.ContinueOnError)
		set.SetOutput(r.Stderr)
		path := set.String("config", "", "configuration path")
		alias := set.String("instance", "", "instance alias")
		if err := set.Parse(args[1:]); err != nil {
			return r.writeFailure("instance.probe", *alias, contracts.WrapError("invalid_input", "invalid flags", err))
		}
		service, selected, bridgeErr := serviceFor(*path, *alias)
		if bridgeErr != nil {
			return r.writeFailure("instance.probe", *alias, bridgeErr)
		}
		result, bridgeErr := service.Probe(ctx)
		if bridgeErr != nil {
			return r.writeFailure("instance.probe", selected, bridgeErr)
		}
		return r.writeSuccess("instance.probe", selected, result)
	default:
		return r.writeFailure("instance", "", contracts.NewError("invalid_input", "unknown instance command: "+args[0]))
	}
}

func (r Runner) runRepo(ctx context.Context, args []string) int {
	if len(args) == 0 {
		return r.writeFailure("repo", "", contracts.NewError("invalid_input", "a repository command is required"))
	}
	switch args[0] {
	case "list":
		set := flag.NewFlagSet("repo list", flag.ContinueOnError)
		set.SetOutput(r.Stderr)
		path := set.String("config", "", "configuration path")
		alias := set.String("instance", "", "instance alias")
		page := set.Int("page", 1, "page number")
		limit := set.Int("limit", 30, "page size")
		if err := set.Parse(args[1:]); err != nil {
			return r.writeFailure("repo.list", *alias, contracts.WrapError("invalid_input", "invalid flags", err))
		}
		service, selected, bridgeErr := serviceFor(*path, *alias)
		if bridgeErr != nil {
			return r.writeFailure("repo.list", *alias, bridgeErr)
		}
		result, bridgeErr := service.ListRepositories(ctx, domain.ListOptions{Page: *page, Limit: *limit})
		if bridgeErr != nil {
			return r.writeFailure("repo.list", selected, bridgeErr)
		}
		envelope := contracts.Success("repo.list", requestid.New(), selected, result.Items)
		envelope.Page = &contracts.Page{Number: result.Page, Limit: result.Limit, Total: result.Total, Next: result.Next}
		return r.write(envelope, 0)
	case "get":
		set := flag.NewFlagSet("repo get", flag.ContinueOnError)
		set.SetOutput(r.Stderr)
		path := set.String("config", "", "configuration path")
		alias := set.String("instance", "", "instance alias")
		if err := set.Parse(args[1:]); err != nil {
			return r.writeFailure("repo.get", *alias, contracts.WrapError("invalid_input", "invalid flags", err))
		}
		if set.NArg() != 1 {
			return r.writeFailure("repo.get", *alias, contracts.NewError("invalid_input", "repo get requires owner/name"))
		}
		service, selected, bridgeErr := serviceFor(*path, *alias)
		if bridgeErr != nil {
			return r.writeFailure("repo.get", *alias, bridgeErr)
		}
		result, bridgeErr := service.GetRepository(ctx, set.Arg(0))
		if bridgeErr != nil {
			return r.writeFailure("repo.get", selected, bridgeErr)
		}
		return r.writeSuccess("repo.get", selected, result)
	default:
		return r.writeFailure("repo", "", contracts.NewError("invalid_input", "unknown repository command: "+args[0]))
	}
}

func loadConfig(path string) (*config.Config, *contracts.BridgeError) {
	cfg, err := config.Load(path)
	if err != nil {
		return nil, contracts.WrapError("invalid_input", "could not load configuration", err)
	}
	return cfg, nil
}

func serviceFor(path, alias string) (*application.Service, string, *contracts.BridgeError) {
	cfg, bridgeErr := loadConfig(path)
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
		if bridgeErr, ok := err.(*contracts.BridgeError); ok {
			return nil, selected.Alias, bridgeErr
		}
		return nil, selected.Alias, contracts.WrapError("invalid_input", "could not initialize Forgejo client", err)
	}
	return application.New(client), selected.Alias, nil
}

func (r Runner) writeSuccess(operation, instance string, data any) int {
	return r.write(contracts.Success(operation, requestid.New(), instance, data), 0)
}

func (r Runner) writeFailure(operation, instance string, bridgeErr *contracts.BridgeError) int {
	return r.write(contracts.Failure(operation, requestid.New(), instance, bridgeErr), exitCode(bridgeErr.Code))
}

func (r Runner) write(envelope contracts.Envelope, code int) int {
	encoder := json.NewEncoder(r.Stdout)
	encoder.SetIndent("", "  ")
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(envelope); err != nil {
		fmt.Fprintln(r.Stderr, "encode output:", err)
		return 8
	}
	return code
}

func exitCode(errorCode string) int {
	switch errorCode {
	case "invalid_input", "instance_not_found", "instance_ambiguous":
		return 2
	case "authentication_required", "authentication_failed":
		return 3
	case "permission_denied":
		return 4
	case "resource_not_found":
		return 5
	case "conflict", "precondition_failed":
		return 6
	case "rate_limited":
		return 7
	case "capability_unsupported", "unsupported_instance":
		return 9
	case "confirmation_required":
		return 10
	case "operation_indeterminate":
		return 11
	default:
		if value, err := strconv.Atoi(errorCode); err == nil {
			return value
		}
		return 8
	}
}
