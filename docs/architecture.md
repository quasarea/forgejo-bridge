# Architecture

The executable has two adapters and one application core:

```text
CLI ----------\
               application service -> Forgejo client -> Forgejo /api/v1
MCP stdio ----/
```

The CLI and MCP server do not perform HTTP calls themselves. Domain models,
error envelopes, allowlist checks, retries, capability discovery, and policy
classification are shared.

## Compatibility

Forgejo guarantees compatibility within a major version, not between majors.
The bridge therefore records the server major and exposes an explicit capability
set. Forgejo 15 and 16 are certified by the matrix. The currently implemented
Forgejo 16-only surface is Actions workflow-run and job reads. Log and artifact
downloads are binary operations and remain withheld until bounded streaming,
content-type validation, destination handling, and checksum semantics are part
of the public contract.

The selected surface is hand-written and reviewed. There is intentionally no
raw HTTP escape-hatch tool and no generated exposure of the entire OpenAPI
document.

## Boundaries

Native Git remains responsible for checkout, fetch, pull, push, commits,
rebases, merges, tags, worktrees, LFS, submodules, and conflict resolution. The
bridge owns Forgejo collaboration resources and server-side metadata.

## References

- [Forgejo API usage](https://forgejo.org/docs/latest/user/api-usage/)
- [Forgejo token scopes](https://forgejo.org/docs/latest/user/authentication/token-scope/)
- [Forgejo 16 API additions](https://forgejo.org/2026-07-release-v16-0/)
- [MCP SDKs](https://modelcontextprotocol.io/docs/2026-07-28/sdk)
- [OpenAI plugin tool design](https://developers.openai.com/plugins/plan/tools)
