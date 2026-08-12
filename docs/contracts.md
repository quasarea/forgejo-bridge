# Adapter contracts

## Output

All adapters use schema version `1.0`. CLI stdout contains one JSON envelope and
MCP tools return the same envelope as structured output. Failures have stable
codes and do not serialize wrapped causes, tokens, or request headers.

## Instance selection

Selection order is explicit argument, `FORGEJO_BRIDGE_INSTANCE`, configured
default, or the only configured instance. Multiple candidates without a default
produce `instance_ambiguous`.

## Credentials

Configuration stores a credential reference, never a token. The MVP supports
`env:VARIABLE` and `file:/absolute/path`. Keyring and Authorized Integration
providers can implement the same resolver boundary without changing callers.

## Tool safety

Current MCP tools are read-only and annotated accordingly. R2 mutations will be
introduced only as separate plan/apply tools whose confirmation digest binds the
caller, instance, repository, inputs, observed state, and expiry.

## Consumer versions

Consumers must negotiate these independently:

```text
bridge product version
output schema version
MCP tool-set version
Forgejo compatibility major
```

Pi and T3 must depend on the CLI/MCP schema, not internal Forgejo DTOs.

## Implemented read taxonomy

The CLI and MCP adapters currently expose instances, repositories, branches,
issues, issue/PR comments, pull requests, pull-request reviews, repository
labels, releases, and Forgejo 16 Actions runs/jobs. Pagination is explicit and
never auto-drains an unbounded collection. Binary logs and artifacts, write
operations, administrative operations, and raw endpoint passthrough are not
part of schema `1.0`.
