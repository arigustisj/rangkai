# Rangkai

A command line tool that builds a ready-to-use AI coding environment for Hermes or Claude Code: skills, MCP servers, hooks and model settings, set up with one command.

## What it is

`rangkai init` writes a complete Hermes profile (config, install script, verify script) for a chosen stack and model. Other commands add or remove skills and MCP servers, switch the model, check health, back up and restore a profile, and export a profile as a shareable pack.

## Requirements

- Go 1.25 or newer
- Hermes Agent, to use the profiles it creates

## Run locally

```bash
git clone https://github.com/arigustisj/rangkai
cd rangkai
make build                       # builds bin/rangkai from ./cmd/rangkai
./bin/rangkai init --profile personal --stack go --model openrouter:nemotron
./bin/rangkai verify
```

`cmd/rangkai/main.go` is part of this repository, so `make build` works out of the box.

## Commands

| Command | Status |
|---------|--------|
| `init` | Implemented |
| `verify` | Implemented |
| `doctor` | Stub |
| `add-skill` | Implemented |
| `list-skills` | Implemented |
| `remove-skill` | Implemented |
| `sync-skills` | Implemented |
| `add-mcp` | Implemented |
| `list-mcp` | Implemented |
| `remove-mcp` | Implemented |
| `sync-mcp` | Implemented |
| `switch-model` | Stub |
| `backup` | Stub |
| `restore` | Stub |
| `update` | Stub |
| `eject` | Stub |
| `pack export` | Implemented |
| `pack import` | Implemented |
| `pack sanitize` | Implemented |
| `pack onboard` | Implemented |

## Environment variables

None are needed. The tool only reads `HOME` to find where to write the profile.

## Pack guidelines (public vs private)

- `pack export` exports a **shareable** pack. The exported archive contains the sanitized structure (skills, hooks, public config template) and is safe to distribute.
- `pack import` imports the pack into a **target** profile.
- Personal memory and credentials must NOT go into the shareable pack. Keep them in `memory/` (usually `.gitignore`d) and in the source profile's private settings.

## Install (once)

The one-time install script URL (0xlzy.tech/install/rangkai) is planned but not yet available. Build and use the binary locally instead.

## License

MIT — Copyright (c) Rangkai contributors
