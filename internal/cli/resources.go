package cli

import (
	"context"
	"flag"
	"strconv"

	"github.com/quasarea/forgejo-bridge/internal/contracts"
	"github.com/quasarea/forgejo-bridge/internal/domain"
	"github.com/quasarea/forgejo-bridge/internal/requestid"
)

type resourceFlags struct {
	config   string
	instance string
	page     int
	limit    int
	state    string
	args     []string
}

func (r Runner) parseResourceFlags(command string, args []string, stateful bool) (resourceFlags, *contracts.BridgeError) {
	set := flag.NewFlagSet(command, flag.ContinueOnError)
	set.SetOutput(r.Stderr)
	path := set.String("config", "", "configuration path")
	instance := set.String("instance", "", "instance alias")
	page := set.Int("page", 1, "page number")
	limit := set.Int("limit", 30, "page size")
	var state *string
	if stateful {
		state = set.String("state", "open", "open, closed, or all")
	}
	if err := set.Parse(args); err != nil {
		return resourceFlags{}, contracts.WrapError("invalid_input", "invalid flags", err)
	}
	result := resourceFlags{config: *path, instance: *instance, page: *page, limit: *limit, args: set.Args()}
	if state != nil {
		result.state = *state
	}
	return result, nil
}

func parsePositive(value, kind string) (int64, *contracts.BridgeError) {
	id, err := strconv.ParseInt(value, 10, 64)
	if err != nil || id < 1 {
		return 0, contracts.NewError("invalid_input", kind+" must be a positive integer")
	}
	return id, nil
}

func writePage[T any](r Runner, operation, instance string, result domain.Page[T]) int {
	envelope := contracts.Success(operation, requestid.New(), instance, result.Items)
	envelope.Page = &contracts.Page{Number: result.Page, Limit: result.Limit, Total: result.Total, Next: result.Next}
	return r.write(envelope, 0)
}

func (r Runner) runBranch(ctx context.Context, args []string) int {
	if len(args) == 0 {
		return r.writeFailure("branch", "", contracts.NewError("invalid_input", "a branch command is required"))
	}
	flags, err := r.parseResourceFlags("branch "+args[0], args[1:], false)
	if err != nil {
		return r.writeFailure("branch."+args[0], flags.instance, err)
	}
	service, selected, bridgeErr := serviceFor(flags.config, flags.instance)
	if bridgeErr != nil {
		return r.writeFailure("branch."+args[0], flags.instance, bridgeErr)
	}
	switch args[0] {
	case "list":
		if len(flags.args) != 1 {
			return r.writeFailure("branch.list", selected, contracts.NewError("invalid_input", "branch list requires owner/repo"))
		}
		result, bridgeErr := service.ListBranches(ctx, flags.args[0], domain.ListOptions{Page: flags.page, Limit: flags.limit})
		if bridgeErr != nil {
			return r.writeFailure("branch.list", selected, bridgeErr)
		}
		return writePage(r, "branch.list", selected, result)
	case "get":
		if len(flags.args) != 2 {
			return r.writeFailure("branch.get", selected, contracts.NewError("invalid_input", "branch get requires owner/repo and branch"))
		}
		result, bridgeErr := service.GetBranch(ctx, flags.args[0], flags.args[1])
		if bridgeErr != nil {
			return r.writeFailure("branch.get", selected, bridgeErr)
		}
		return r.writeSuccess("branch.get", selected, result)
	default:
		return r.writeFailure("branch", selected, contracts.NewError("invalid_input", "unknown branch command: "+args[0]))
	}
}

func (r Runner) runIssue(ctx context.Context, args []string) int {
	if len(args) == 0 {
		return r.writeFailure("issue", "", contracts.NewError("invalid_input", "an issue command is required"))
	}
	flags, err := r.parseResourceFlags("issue "+args[0], args[1:], args[0] == "list")
	if err != nil {
		return r.writeFailure("issue."+args[0], flags.instance, err)
	}
	service, selected, bridgeErr := serviceFor(flags.config, flags.instance)
	if bridgeErr != nil {
		return r.writeFailure("issue."+args[0], flags.instance, bridgeErr)
	}
	switch args[0] {
	case "list":
		if len(flags.args) != 1 {
			return r.writeFailure("issue.list", selected, contracts.NewError("invalid_input", "issue list requires owner/repo"))
		}
		result, bridgeErr := service.ListIssues(ctx, flags.args[0], flags.state, domain.ListOptions{Page: flags.page, Limit: flags.limit})
		if bridgeErr != nil {
			return r.writeFailure("issue.list", selected, bridgeErr)
		}
		return writePage(r, "issue.list", selected, result)
	case "get", "comments":
		if len(flags.args) != 2 {
			return r.writeFailure("issue."+args[0], selected, contracts.NewError("invalid_input", "issue "+args[0]+" requires owner/repo and issue number"))
		}
		number, inputErr := parsePositive(flags.args[1], "issue number")
		if inputErr != nil {
			return r.writeFailure("issue."+args[0], selected, inputErr)
		}
		if args[0] == "get" {
			result, bridgeErr := service.GetIssue(ctx, flags.args[0], number)
			if bridgeErr != nil {
				return r.writeFailure("issue.get", selected, bridgeErr)
			}
			return r.writeSuccess("issue.get", selected, result)
		}
		result, bridgeErr := service.ListIssueComments(ctx, flags.args[0], number, domain.ListOptions{Page: flags.page, Limit: flags.limit})
		if bridgeErr != nil {
			return r.writeFailure("issue.comments", selected, bridgeErr)
		}
		return writePage(r, "issue.comments", selected, result)
	default:
		return r.writeFailure("issue", selected, contracts.NewError("invalid_input", "unknown issue command: "+args[0]))
	}
}

func (r Runner) runPullRequest(ctx context.Context, args []string) int {
	if len(args) == 0 {
		return r.writeFailure("pr", "", contracts.NewError("invalid_input", "a pull request command is required"))
	}
	flags, err := r.parseResourceFlags("pr "+args[0], args[1:], args[0] == "list")
	if err != nil {
		return r.writeFailure("pr."+args[0], flags.instance, err)
	}
	service, selected, bridgeErr := serviceFor(flags.config, flags.instance)
	if bridgeErr != nil {
		return r.writeFailure("pr."+args[0], flags.instance, bridgeErr)
	}
	switch args[0] {
	case "list":
		if len(flags.args) != 1 {
			return r.writeFailure("pr.list", selected, contracts.NewError("invalid_input", "pr list requires owner/repo"))
		}
		result, bridgeErr := service.ListPullRequests(ctx, flags.args[0], flags.state, domain.ListOptions{Page: flags.page, Limit: flags.limit})
		if bridgeErr != nil {
			return r.writeFailure("pr.list", selected, bridgeErr)
		}
		return writePage(r, "pr.list", selected, result)
	case "get", "reviews":
		if len(flags.args) != 2 {
			return r.writeFailure("pr."+args[0], selected, contracts.NewError("invalid_input", "pr "+args[0]+" requires owner/repo and pull request number"))
		}
		number, inputErr := parsePositive(flags.args[1], "pull request number")
		if inputErr != nil {
			return r.writeFailure("pr."+args[0], selected, inputErr)
		}
		if args[0] == "get" {
			result, bridgeErr := service.GetPullRequest(ctx, flags.args[0], number)
			if bridgeErr != nil {
				return r.writeFailure("pr.get", selected, bridgeErr)
			}
			return r.writeSuccess("pr.get", selected, result)
		}
		result, bridgeErr := service.ListPullReviews(ctx, flags.args[0], number, domain.ListOptions{Page: flags.page, Limit: flags.limit})
		if bridgeErr != nil {
			return r.writeFailure("pr.reviews", selected, bridgeErr)
		}
		return writePage(r, "pr.reviews", selected, result)
	default:
		return r.writeFailure("pr", selected, contracts.NewError("invalid_input", "unknown pull request command: "+args[0]))
	}
}

func (r Runner) runLabel(ctx context.Context, args []string) int {
	if len(args) == 0 || args[0] != "list" {
		return r.writeFailure("label", "", contracts.NewError("invalid_input", "supported label command: list"))
	}
	flags, err := r.parseResourceFlags("label list", args[1:], false)
	if err != nil {
		return r.writeFailure("label.list", flags.instance, err)
	}
	if len(flags.args) != 1 {
		return r.writeFailure("label.list", flags.instance, contracts.NewError("invalid_input", "label list requires owner/repo"))
	}
	service, selected, bridgeErr := serviceFor(flags.config, flags.instance)
	if bridgeErr != nil {
		return r.writeFailure("label.list", flags.instance, bridgeErr)
	}
	result, bridgeErr := service.ListLabels(ctx, flags.args[0], domain.ListOptions{Page: flags.page, Limit: flags.limit})
	if bridgeErr != nil {
		return r.writeFailure("label.list", selected, bridgeErr)
	}
	return writePage(r, "label.list", selected, result)
}

func (r Runner) runRelease(ctx context.Context, args []string) int {
	if len(args) == 0 {
		return r.writeFailure("release", "", contracts.NewError("invalid_input", "a release command is required"))
	}
	flags, err := r.parseResourceFlags("release "+args[0], args[1:], false)
	if err != nil {
		return r.writeFailure("release."+args[0], flags.instance, err)
	}
	service, selected, bridgeErr := serviceFor(flags.config, flags.instance)
	if bridgeErr != nil {
		return r.writeFailure("release."+args[0], flags.instance, bridgeErr)
	}
	if args[0] == "list" {
		if len(flags.args) != 1 {
			return r.writeFailure("release.list", selected, contracts.NewError("invalid_input", "release list requires owner/repo"))
		}
		result, bridgeErr := service.ListReleases(ctx, flags.args[0], domain.ListOptions{Page: flags.page, Limit: flags.limit})
		if bridgeErr != nil {
			return r.writeFailure("release.list", selected, bridgeErr)
		}
		return writePage(r, "release.list", selected, result)
	}
	if args[0] == "get" {
		if len(flags.args) != 2 {
			return r.writeFailure("release.get", selected, contracts.NewError("invalid_input", "release get requires owner/repo and release id"))
		}
		id, inputErr := parsePositive(flags.args[1], "release id")
		if inputErr != nil {
			return r.writeFailure("release.get", selected, inputErr)
		}
		result, bridgeErr := service.GetRelease(ctx, flags.args[0], id)
		if bridgeErr != nil {
			return r.writeFailure("release.get", selected, bridgeErr)
		}
		return r.writeSuccess("release.get", selected, result)
	}
	return r.writeFailure("release", selected, contracts.NewError("invalid_input", "unknown release command: "+args[0]))
}

func (r Runner) runActions(ctx context.Context, args []string) int {
	if len(args) < 2 || args[0] != "run" {
		return r.writeFailure("actions", "", contracts.NewError("invalid_input", "supported actions commands: run list, run get, run jobs"))
	}
	command := args[1]
	flags, err := r.parseResourceFlags("actions run "+command, args[2:], false)
	if err != nil {
		return r.writeFailure("actions.run."+command, flags.instance, err)
	}
	service, selected, bridgeErr := serviceFor(flags.config, flags.instance)
	if bridgeErr != nil {
		return r.writeFailure("actions.run."+command, flags.instance, bridgeErr)
	}
	if command == "list" {
		if len(flags.args) != 1 {
			return r.writeFailure("actions.run.list", selected, contracts.NewError("invalid_input", "actions run list requires owner/repo"))
		}
		result, bridgeErr := service.ListActionRuns(ctx, flags.args[0], domain.ListOptions{Page: flags.page, Limit: flags.limit})
		if bridgeErr != nil {
			return r.writeFailure("actions.run.list", selected, bridgeErr)
		}
		return writePage(r, "actions.run.list", selected, result)
	}
	if command == "get" || command == "jobs" {
		if len(flags.args) != 2 {
			return r.writeFailure("actions.run."+command, selected, contracts.NewError("invalid_input", "actions run "+command+" requires owner/repo and run id"))
		}
		id, inputErr := parsePositive(flags.args[1], "action run id")
		if inputErr != nil {
			return r.writeFailure("actions.run."+command, selected, inputErr)
		}
		if command == "get" {
			result, bridgeErr := service.GetActionRun(ctx, flags.args[0], id)
			if bridgeErr != nil {
				return r.writeFailure("actions.run.get", selected, bridgeErr)
			}
			return r.writeSuccess("actions.run.get", selected, result)
		}
		result, bridgeErr := service.ListActionJobs(ctx, flags.args[0], id)
		if bridgeErr != nil {
			return r.writeFailure("actions.run.jobs", selected, bridgeErr)
		}
		return r.writeSuccess("actions.run.jobs", selected, result)
	}
	return r.writeFailure("actions.run", selected, contracts.NewError("invalid_input", "unknown actions run command: "+command))
}
