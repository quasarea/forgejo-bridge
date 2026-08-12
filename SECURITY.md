# Security policy

## Reporting a vulnerability

Please report vulnerabilities through GitHub private vulnerability reporting
for this repository. Do not open a public issue containing credentials,
private instance URLs, repository names, configuration, or exploit details.

Include the affected bridge version, operation, and a minimal redacted
reproduction. Rotate any credential that may have been exposed before
collecting additional diagnostics.

## Deployment expectations

- Store credentials outside the repository and reference them through the
  supported `env:` or `file:` configuration schemes.
- Use a least-privilege Forgejo identity and an explicit repository allowlist.
- Treat Forgejo content as untrusted input.
- Keep TLS verification enabled and protect local configuration permissions.
- Do not add raw HTTP escape hatches that bypass policy or normalized errors.
