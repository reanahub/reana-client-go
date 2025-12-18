# REANA-Client-Go

[![image](https://github.com/reanahub/reana-client-go/workflows/CI/badge.svg)](https://github.com/reanahub/reana-client-go/actions)
[![image](https://codecov.io/gh/reanahub/reana-client-go/branch/master/graph/badge.svg)](https://codecov.io/gh/reanahub/reana-client-go)
[![image](https://img.shields.io/badge/discourse-forum-blue.svg)](https://forum.reana.io)
[![image](https://img.shields.io/github/license/reanahub/reana.svg)](https://github.com/reanahub/reana-client-go/blob/master/LICENSE)

# About

REANA-Client-Go is a component of the [REANA](https://www.reana.io/) reusable and
reproducible research data analysis platform. It provides a command-line tool that allows
researchers to submit, run, and manage their computational workflows.

- seed workspace with input code and data
- run computational workflows on remote compute clouds
- list submitted workflows and enquire about their statuses
- download results of finished workflows

# Usage

The detailed information on how to install and use REANA can be found in
[docs.reana.io](https://docs.reana.io).

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

# Useful links

- [REANA project home page](http://www.reana.io/)
- [REANA user documentation](https://docs.reana.io)
- [REANA user support forum](https://forum.reana.io)
- [REANA-Client-Go known issues](https://github.com/reanahub/reana-client-go/issues)
- [REANA-Client-Go source code](https://github.com/reanahub/reana-client-go)
