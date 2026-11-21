<div align="center">

# Rangkai

**One command to a ready-to-use AI coding environment for Hermes and Claude Code.**

[![License: MIT](https://img.shields.io/github/license/arigustisj/rangkai?color=blue)](LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/arigustisj/rangkai)](https://goreportcard.com/report/github.com/arigustisj/rangkai)
[![Go 1.25+](https://img.shields.io/badge/go-1.25%2B-00ADD8?logo=go&logoColor=white)](go.mod)
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg)](#contributing)

</div>

---

Rangkai writes a complete agent profile for your stack and model: config, install script, and verify script. From there you add skills, wire MCP servers, set hooks, and route models. When the setup works, export it as a pack and hand it to a teammate. That replaces hand-assembling directories and copy-pasting config files.

## Quickstart

```bash
git clone https://github.com/arigustisj/rangkai
cd rangkai
make build
./bin/rangkai init --profile personal --stack go --model openrouter:nemotron
./bin/rangkai verify
```

`make build` works out of the box. The entry point lives at `cmd/rangkai/main.go`.

## Requirements

- Go 1.25 or newer
- Hermes Agent, to run the profiles Rangkai creates

## Commands

Status legend: **Ready** means implemented, **Stub** means a placeholder still pending implementation.

| Command | What it does | Status |
| --- | --- | :---: |
| `init` | Create a profile with config, install and verify scripts | Ready |
| `verify` | Health-check a profile | Ready |
| `doctor` | Diagnose and fix environment issues | Stub |
| `add-skill` | Link a skill into the profile | Ready |
| `list-skills` | List installed and available skills | Ready |
| `remove-skill` | Remove a skill from the profile | Ready |
| `sync-skills` | Sync core skills for the chosen stack | Ready |
| `add-mcp` | Wire an MCP server into the profile | Ready |
| `list-mcp` | List installed and available MCP servers | Ready |
| `remove-mcp` | Remove an MCP server from the profile | Ready |
| `sync-mcp` | Sync core MCP servers | Ready |
| `switch-model` | Change the active model | Stub |
| `backup` | Back up profile, memory, and skills | Stub |
| `restore` | Restore from a backup | Stub |
| `update` | Update Hermes, skills, and MCP servers | Stub |
| `eject` | Export config as plain files, no lock-in | Stub |
| `pack export` | Export a shareable, sanitized pack | Ready |
| `pack import` | Import a pack into a target profile | Ready |
| `pack sanitize` | Strip credentials and replace sensitive patterns | Ready |
| `pack onboard` | Onboard a new team member from a pack | Ready |

Run `rangkai <command> -h` for flags. `init` takes `--profile` (personal, team-member, ci-bot), `--stack` (go, node, laravel, python), `--model` (`openrouter:...`, `ollama:...`, `vllm:...`), `--server`, `--shared-skills`, `--variant`, and `--force`.

## Packs: share the setup, keep the secrets

A pack is a portable archive of a working environment.

- `pack export` produces a **shareable** pack: sanitized structure, skills, hooks, and a public config template. Safe to distribute.
- `pack import` applies a pack to a **target** profile.
- `pack sanitize` strips credentials and replaces sensitive patterns before anything leaves your machine.

Personal memory and credentials never belong in a shareable pack. Keep them in `memory/` (gitignored) and in the source profile's private settings.

## Environment variables

None required. Rangkai reads `HOME` to locate where the profile is written.

## Roadmap

- Fill in the stub commands: `doctor`, `switch-model`, `backup`, `restore`, `update`, `eject`.
- A one-line install script at `0xlzy.tech/install/rangkai` (planned, not yet available). Build from source for now.
- Encrypted personal packs with private hub sync, so a setup moves between machines without losing memory.

## Contributing

Issues and pull requests are welcome. Run `make test` and `make lint` before opening a PR.

## License

MIT. Copyright (c) Rangkai contributors. See [LICENSE](LICENSE).
