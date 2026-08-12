# forgejo-bridge

`forgejo-bridge` is a small Forgejo client exposed through one deterministic CLI
and an MCP server. Both adapters use the same typed application service, policy,
capability discovery, and error contracts.

The certified compatibility target is Forgejo 15 and Forgejo 16. Operations
that are unavailable on an instance fail with `capability_unsupported`; the
bridge does not emulate GitHub APIs.

Current release: `v0.2.0`. Release binaries report their public source commit
and UTC build time through the deterministic `version` envelope.

This is the public, environment-neutral bridge. Deployment wrappers, private
instance configuration, credentials, and consumer-specific installations must
live outside this repository.

## Install

Download a release archive for your platform from GitHub Releases and verify
its SHA-256 checksum before placing `forgejo-bridge` on `PATH`. Alternatively,
build from source with Go 1.25 or newer:

```text
go install github.com/quasarea/forgejo-bridge/cmd/forgejo-bridge@latest
```

## Development

The host does not need Go installed when Docker is available:

```powershell
docker run --rm -v "${PWD}:/src" -w /src golang:1.25 `
  sh -lc "go test ./..."
```

The repository is intentionally separate from application checkouts and
environment-specific coordination repositories.

## CLI

```text
forgejo-bridge version
forgejo-bridge instance list --config <path>
forgejo-bridge instance probe --instance <alias> --config <path>
forgejo-bridge repo list --instance <alias> --config <path>
forgejo-bridge repo get <owner>/<repo> --instance <alias> --config <path>
forgejo-bridge branch list [flags] <owner>/<repo>
forgejo-bridge branch get [flags] <owner>/<repo> <branch>
forgejo-bridge issue list [--state open|closed|all] [flags] <owner>/<repo>
forgejo-bridge issue get [flags] <owner>/<repo> <number>
forgejo-bridge issue comments [flags] <owner>/<repo> <number>
forgejo-bridge pr list [--state open|closed|all] [flags] <owner>/<repo>
forgejo-bridge pr get [flags] <owner>/<repo> <number>
forgejo-bridge pr reviews [flags] <owner>/<repo> <number>
forgejo-bridge label list [flags] <owner>/<repo>
forgejo-bridge release list [flags] <owner>/<repo>
forgejo-bridge release get [flags] <owner>/<repo> <release-id>
forgejo-bridge actions run list [flags] <owner>/<repo>
forgejo-bridge actions run get [flags] <owner>/<repo> <run-id>
forgejo-bridge actions run jobs [flags] <owner>/<repo> <run-id>
forgejo-bridge mcp stdio --config <path>
```

Resource flags must precede positional arguments. List commands accept `--page`
and `--limit`; the bridge caps requested page size at 100 and preserves Forgejo's
`Link` and `X-Total-Count` pagination metadata in the shared envelope.

The MCP server exposes the same read surface as 18 deterministic tools. Forgejo
Actions run and job reads require Forgejo 16 or newer and otherwise return
`capability_unsupported`. Binary workflow logs and artifact downloads remain
out of scope until the bounded-download contract is implemented.

JSON is the default and only machine contract in the MVP. Human diagnostics are
written to stderr. Secrets are referenced from environment variables or files
and are never stored in configuration.

## Configuration

```toml
default_instance = "work"

[instances.work]
base_url = "https://forge.example.net"
credential = "env:FORGEJO_TOKEN"
allowed_repositories = ["team/service"]
read_only = true
```

The credential schemes currently implemented are `env:NAME` and `file:/path`.
OS keyring and OIDC providers remain next-phase work.

## Deployment boundary

Run the bridge as an on-demand CLI or MCP stdio process. Keep credentials and
configuration in the deployment environment; do not package them with clients.
Consumers should depend only on the versioned JSON/MCP contracts documented in
[`docs/contracts.md`](docs/contracts.md), not on internal Go types.

## Security

See [`SECURITY.md`](SECURITY.md) for the supported reporting channel and secret
handling expectations. Never include real instance URLs, access tokens, SSH
keys, or production configuration in bug reports.
