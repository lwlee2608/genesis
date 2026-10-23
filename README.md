# Genesis

A TUI tool for bootstrapping new Go projects.

This repo has two parts:

1. **CLI** — creates the project skeleton for a new project
2. **Skills** (`skills/`) — Claude Code skills to apply additional features: `genesis` (server, web, Docker, sqlc, Railway) copies from `reference/project-00`; `genesis-auth` (tiered authentication) writes auth to match the project's existing code

Typical workflow: start a new project with the CLI, then use the skill to add features.

## Installation

```bash
curl -fsSL https://raw.githubusercontent.com/lwlee2608/genesis/main/scripts/install.sh | bash
```

Or build from source:

```bash
make install
```

This will install the binary to `~/.local/bin/genesis`. Make sure `~/.local/bin` is in your PATH.

## Usage

```bash
cd /path/to/new/project
genesis
```

Non-interactive (for scripts and coding agents) — any flag, or `-y` to accept detected defaults:

```bash
genesis -name foo -module github.com/u/foo -fullstack=false
genesis -y
```

Run `genesis -h` for all flags. Flag mode refuses a non-empty directory unless `-force` is given, and skips the AI README unless `-readme` is given.

Every scaffold writes `.genesis.json` at the repo root, recording the genesis version, creation date, and options. Find genesis-made repos with `gh search code --filename .genesis.json`.
