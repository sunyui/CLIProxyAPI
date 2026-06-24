@AGENTS.md

## Team distribution branch

- `team-build` is for internal team packaging artifacts, not upstream project fixes.
- Rebuild team packages with `./scripts/build_team_dist.sh`.
- The script recreates `dist/` and validates each archive contains only: `cli-proxy-api` or `cli-proxy-api.exe`, `config.example.yaml`, `installation.md`, and `team-config.yaml`.
- Override packaging toolchain with `CLIPROXYAPI_GOROOT` and `CLIPROXYAPI_GOPROXY`; defaults are `/usr/local/Cellar/go/1.26.2/libexec` and `https://goproxy.cn,direct`.
- Deploy the current team package locally with `./scripts/deploy_local_launchd.sh`; it installs to `~/.local/share/cliproxyapi-team`, keeps `~/.cli-proxy-api/config.yaml`, and restarts the `com.sunyan.cliproxyapi` LaunchAgent.
- Team config maps upstream `gpt-5.5` to client alias `claude-opus-4-7` and requires placeholders `YOUR_OPENCLAUDECODE_BASE_URL` and `YOUR_OPENCLAUDECODE_API_KEY` to be replaced before use.
