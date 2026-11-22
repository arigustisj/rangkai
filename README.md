<div align="center">

<img src="assets/rangkai-wordmark.svg" alt="Rangkai" width="560">

**One command to a ready-to-use AI coding environment for Hermes and Claude Code.**

[![License: MIT](https://img.shields.io/github/license/arigustisj/rangkai?color=blue)](LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/arigustisj/rangkai)](https://goreportcard.com/report/github.com/arigustisj/rangkai)
[![Go 1.25+](https://img.shields.io/badge/go-1.25%2B-00ADD8?logo=go&logoColor=white)](go.mod)
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg)](#contributing)

</div>

---

Rangkai writes a complete agent profile for your stack and model: config, install script, and verify script. From there you add skills, wire MCP servers, set hooks, and route models. When the setup works, export it as a pack and hand it to a teammate. That replaces hand-assembling directories and copy-pasting config files.

## Quickstart

**One-line install** (downloads the release binary, verifies its SHA256, installs to `~/.local/bin`):

```bash
curl -fsSL https://0xlzy.tech/install/rangkai | sh
```

The vanity URL redirects to the GitHub release installer (`https://github.com/arigustisj/rangkai/releases/latest/download/install.sh`).

**Or build from source**:

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
| `backup` | Back up the full personal environment to an encrypted archive | Ready |
| `restore` | Restore a personal environment from an encrypted archive | Ready |
| `update` | Update Hermes, skills, and MCP servers | Stub |
| `eject` | Wipe the installed environment and secrets from this machine | Ready |
| `push` | Encrypt and upload the environment to a private hub over SSH | Ready |
| `pull` | Download and restore the environment from a private hub | Ready |
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

## Carry your environment between machines

`backup`, `restore`, and `eject` move a whole personal environment (profile, memory, skills, secrets, and SSOT) as one age-encrypted archive.

```bash
# Old machine: encrypt everything into one archive
RANGKAI_PASSPHRASE='your-passphrase' ./bin/rangkai backup --profile personal --output carry.tar.gz.age

# Carry carry.tar.gz.age to the new machine, then:
RANGKAI_PASSPHRASE='your-passphrase' ./bin/rangkai restore --source carry.tar.gz.age

# Wipe the old machine (dry-run by default; add --yes to actually delete)
./bin/rangkai eject --profile personal --yes
```

The archive is age ciphertext and the passphrase never leaves your machine. `restore` decrypts fully before writing, refuses to overwrite an existing profile without `--force`, and rejects path traversal. `eject` removes only the resolved profile and secrets directories, never `$HOME`, and reports residue it cannot control (shell history, swap, host logs).

This is the manual-carry flow. To sync without carrying the file yourself, use `push`/`pull` with a private hub (below). A one-line install is on the roadmap.

## Hub sync: move between machines automatically

`push` and `pull` sync the encrypted archive with a private hub over SSH (for example a server reachable only on your Tailscale network). The hub stores only ciphertext; the passphrase never leaves your machine.

```bash
# Old machine: encrypt and upload
RANGKAI_PASSPHRASE='your-passphrase' ./bin/rangkai push --hub your-hub:rangkai-hub --profile personal

# New machine: download and restore
RANGKAI_PASSPHRASE='your-passphrase' ./bin/rangkai pull --hub your-hub:rangkai-hub --profile personal
```

`--hub` is an SCP target (`ssh-host:path`); an ssh_config alias works. `push` creates the hub directory if needed. `pull` inherits the restore guards: decrypt-first, `--force` to overwrite, path-traversal rejection. SSH runs in BatchMode with trust-on-first-use host keys and fails closed.

## Environment variables

- `RANGKAI_PASSPHRASE` (optional): passphrase for `backup`, `restore`, `push`, and `pull`, as an alternative to `--passphrase`.

Rangkai reads `HOME` to locate where the profile is written.

## Roadmap

- Fill in the remaining stub commands: `doctor`, `switch-model`, `update`.
- Vanity URL `0xlzy.tech/install/rangkai` redirects to the GitHub-hosted `install.sh` (a redirect service behind the portfolio Cloudflare Tunnel).
- Layered public/private pack export with a sanitize lint for sharing.

## Contributing

Issues and pull requests are welcome. Run `make test` and `make lint` before opening a PR.

## License

MIT. Copyright (c) Rangkai contributors. See [LICENSE](LICENSE).
