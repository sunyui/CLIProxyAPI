# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build and test

- Go module: `github.com/router-for-me/CLIProxyAPI/v6`; use Go 1.26+.
- Server entrypoint: `./cmd/server`.
- Build locally with `go build -o cli-proxy-api ./cmd/server`.
- Run focused translator tests with `go test ./internal/translator/openai/claude`.
- Run the full suite with `go test ./...` when changes are broad.
- CI's PR compile check is effectively `go build -o test-output ./cmd/server && rm -f test-output`.

## Runtime/config gotchas

- Default config file is `config.yaml` in the working directory; `config.example.yaml` is the public template.
- `.env` is auto-loaded from the working directory.
- Use `./cli-proxy-api --local-model` when you need to avoid the runtime remote model updater.
- If `PGSTORE_DSN` is set, Postgres storage takes precedence over git storage; object storage is chosen before git storage when Postgres is unset.

## PR constraints

- Do not submit standalone PRs touching `internal/translator/**`; upstream CI rejects translator changes and asks for an issue instead.
- Do not modify `AGENTS.md` in PRs; upstream automation comments on and closes those PRs.
- PRs targeting `main` may be retargeted to `dev` by workflow unless they are `dev -> main`.

## Team distribution branch

- `team-build` is for internal team packaging artifacts, not upstream project fixes.
- Rebuild team packages with `./scripts/build_team_dist.sh`.
- The script recreates `dist/` and validates each archive contains only: `cli-proxy-api` or `cli-proxy-api.exe`, `config.example.yaml`, `installation.md`, and `team-config.yaml`.
- Override packaging toolchain with `CLIPROXYAPI_GOROOT` and `CLIPROXYAPI_GOPROXY`; defaults are `/usr/local/Cellar/go/1.26.2/libexec` and `https://goproxy.cn,direct`.
- Deploy the current team package locally with `./scripts/deploy_local_launchd.sh`; it installs to `~/.local/share/cliproxyapi-team`, keeps `~/.cli-proxy-api/config.yaml`, and restarts the `com.sunyan.cliproxyapi` LaunchAgent.
- Team config maps upstream `gpt-5.5` to client alias `claude-opus-4-7` and requires placeholders `YOUR_OPENCLAUDECODE_BASE_URL` and `YOUR_OPENCLAUDECODE_API_KEY` to be replaced before use.
