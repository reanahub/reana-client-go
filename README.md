# REANA-Client-Go

[![image](https://github.com/reanahub/reana-client-go/workflows/CI/badge.svg)](https://github.com/reanahub/reana-client-go/actions)
[![image](https://codecov.io/gh/reanahub/reana-client-go/branch/master/graph/badge.svg)](https://codecov.io/gh/reanahub/reana-client-go)
[![image](https://img.shields.io/badge/discourse-forum-blue.svg)](https://forum.reana.io)
[![image](https://img.shields.io/github/license/reanahub/reana.svg)](https://github.com/reanahub/reana-client-go/blob/master/LICENSE)

## About

REANA-Client-Go is a component of the [REANA](https://www.reana.io/) reusable
and reproducible research data analysis platform. It provides a command-line
tool that allows researchers to submit, run, and manage their computational
workflows.

- seed workspace with input code and data
- run computational workflows on remote compute clouds
- test completed workflow runs using Gherkin feature files
- list submitted workflows and enquire about their statuses
- download results of finished workflows

## Usage

The detailed information on how to install and use REANA can be found in
[docs.reana.io](https://docs.reana.io).

### Installation with Go

Install the published prerelease with Go 1.25.13 or later:

```console
$ go install github.com/reanahub/reana-client-go@v0.95.0-alpha.1
```

The executable is installed into `go env GOBIN`, or into `$(go env GOPATH)/bin`
when `GOBIN` is not configured. Add that directory to `PATH`. This release
requires an OIDC-enabled REANA deployment.

### Installation from a source checkout

Install the executable into the Go binary directory:

```console
$ make install
```

The destination defaults to the value of `go env GOBIN`, or to
`$(go env GOPATH)/bin` when `GOBIN` is not configured. Make sure that this
directory is present in `PATH`.

Set `BINDIR` to an absolute path to choose a different destination, such as a
Python virtual environment's scripts directory:

```console
$ make install BINDIR=/path/to/virtualenv/bin
```

Use the same destination to remove the executable:

```console
$ make uninstall BINDIR=/path/to/virtualenv/bin
```

### Testing completed workflows

Use `test` to evaluate a finished workflow run against Gherkin feature files:

```console
$ reana-client-go test -w myanalysis.1
```

By default, the command reads feature file paths from `tests.files` in the
stored `reana.yaml` specification and opens those paths from the current local
project. Repeat `-n/--test-files` to override the specification:

```console
$ reana-client-go test -w myanalysis.1 \
    -n tests/log-messages.feature \
    -n tests/workspace-files.feature
```

### Bundling additional workflow source files

The `create` and `validate` commands upload a scoped specification bundle for
server-side workflow loading. Declare every imported source explicitly under
`workflow.files` or `workflow.directories` so it is available to the loader.

For example, given this Snakemake project:

```text
analysis/
├── reana.yaml
├── Snakefile
└── rules/
    └── common.smk
```

where `Snakefile` contains `include: "rules/common.smk"`, declare the included
source in `reana.yaml`:

```yaml
version: 0.9.0
workflow:
  type: snakemake
  file: Snakefile
  directories:
    - rules
```

Paths are relative to the directory containing the selected specification.
Absolute paths, paths that escape through `..`, and symbolic links are rejected.
Use `workflow.files` only for workflow definitions and configuration needed
while loading the workflow; input datasets belong under `inputs.files` or
`inputs.directories`.

Validation snapshots accept at most 1,000 files, 2,000 directories, 100 MiB of
file content, and 64 relative path components. Symbolic links are not followed.

`reana-client-go validate --environments` performs offline image-reference
checks and reports effective runtime identities. Add `--pull` to verify
availability and inspect those images with your local container runtime and
registry credentials; the REANA server does not contact image registries.

### Authentication

Authenticate against a REANA deployment using the browser-based OIDC flow:

```console
$ reana-client-go login --server-url https://reana.example.org
```

On a remote or browserless machine, use the interactive device flow instead:

```console
$ reana-client-go login --headless --server-url https://reana.example.org
```

Browser login normally uses an OS-assigned loopback port. If the identity
provider requires an exact redirect URI, set
`REANA_CLIENT_LOGIN_LOOPBACK_PORT=<port>` and register
`http://127.0.0.1:<port>/callback`. Login fails with an actionable error if the
configured port is invalid or already occupied.

The client stores renewable OIDC credentials in the shared REANA client
configuration at `~/.config/reana/reana-client.json`, or at the path selected by
`REANA_CLIENT_CONFIG`. The file is permission-restricted to `0600`. Use
`reana-client-go logout` to revoke and remove the credentials. The
`--access-token` option remains available as an explicit per-command override.

For CI, obtain credentials once with an interactive browser or headless login,
store the JSON as a protected secret, and restore it to a unique private file:

```console
$ credential_file="$(mktemp)"
$ trap 'rm -f "$credential_file"' EXIT
$ chmod 600 "$credential_file"
$ printf '%s' "$REANA_CREDENTIALS_SECRET" > "$credential_file"
$ export REANA_CLIENT_CONFIG="$credential_file"
```

This immutable-secret pattern is safe only when the issuer permits reuse of the
stored refresh token. If refresh-token rotation invalidates the previous token,
the client writes the replacement only to the current job's local file; later or
concurrent jobs restoring the original secret will fail. Such deployments need a
mutable, serialised secret store or an issuer-supported non-rotating service
credential.

TLS certificate verification is enabled by default. For local deployments, set
`REANA_SERVER_CA_CERTS` to a trusted CA bundle (PEM) for both REANA and the
identity provider. This takes precedence over `REANA_SERVER_TLS_VERIFY`.

`REANA_SERVER_TLS_VERIFY` accepts `1`/`true`/`yes`/`on` to enable verification
and `0`/`false`/`no`/`off` to disable it for requests to the REANA server's
HTTPS hostname and port (local testing). This includes bundled Keycloak
endpoints under `/keycloak`. Identity providers on other hostnames or ports are
always verified. Values are case-insensitive and ignore surrounding whitespace.
Unset or empty values enable verification; other values are errors.

## Shell completion

The `reana-client-go` supports shell completion for Bash and Zsh. To enable the
auto-completion of commands and options, add the following to your shell
configuration file:

**Bash** (add to `~/.bashrc`):

```bash
source <(reana-client-go completion bash)
```

**Zsh** (add to `~/.zshrc`):

```bash
source <(reana-client-go completion zsh)
compdef _reana-client-go reana-client-go
```

## Useful links

- [REANA project home page](http://www.reana.io/)
- [REANA user documentation](https://docs.reana.io)
- [REANA user support forum](https://forum.reana.io)
- [REANA-Client-Go known issues](https://github.com/reanahub/reana-client-go/issues)
- [REANA-Client-Go source code](https://github.com/reanahub/reana-client-go)
