# CLIProxyAPI Team Quick Start

Creator: sunyan

## 介绍

CLIProxyAPI 是本地 API 代理，用于把 Claude Code 请求转发到团队配置的 OpenAI-compatible 上游。本团队默认配置把上游 `gpt-5.5` 暴露为 `claude-opus-4-7`，方便 Claude Code 使用 Opus 4.7 1M context 模式。

## 优势与核心能力

- 本地统一入口：默认监听 `http://127.0.0.1:8317`。
- Claude Code 兼容：使用 `claude-opus-4-7` 模型名，实际转发到上游 `gpt-5.5`。
- 1M 上下文模式：Claude Code 可选择 `opus[1m]`。
- 缓存成本优化：稳定 OpenAI prompt cache 前缀，长上下文会话通常可降低接近 10 倍输入成本。
- Thinking 兼容：默认支持 `medium`、`high`、`xhigh`，`max` 会映射到上游可用的 `xhigh`。
- 工具调用兼容：支持 Claude Code 的工具调用链路。
- 安全默认值：团队配置模板使用占位符，不包含真实 upstream key。

## 包内容

每个安装包只包含：

- `cli-proxy-api` / `cli-proxy-api.exe`
- `config.example.yaml`
- `installation.md`
- `team-config.yaml`

## 准备配置

解压后复制团队配置模板：

```sh
cp team-config.yaml config.yaml
```

编辑 `config.yaml`，替换：

```text
YOUR_OPENCLAUDECODE_BASE_URL
YOUR_OPENCLAUDECODE_API_KEY
```

默认本地客户端 token 是：

```text
sk-dummy
```

如果代理暴露给其他机器使用，请改成团队自己的强 token。

## macOS 启动

Apple Silicon 使用 `darwin_arm64` 包，Intel Mac 使用 `darwin_amd64` 包。

```sh
tar -xzf cli-proxy-api_team-dist_darwin_arm64.tar.gz
cp team-config.yaml config.yaml
./cli-proxy-api -config config.yaml
```

如果系统拦截二进制：

```sh
xattr -d com.apple.quarantine ./cli-proxy-api 2>/dev/null || true
./cli-proxy-api -config config.yaml
```

## Ubuntu/Linux x64 启动

```sh
tar -xzf cli-proxy-api_team-dist_linux_amd64.tar.gz
cp team-config.yaml config.yaml
./cli-proxy-api -config config.yaml
```

可选：安装为 systemd 服务。

```sh
sudo install -m 0755 cli-proxy-api /usr/local/bin/cli-proxy-api
sudo mkdir -p /etc/cliproxyapi
sudo cp config.yaml /etc/cliproxyapi/config.yaml
```

创建 `/etc/systemd/system/cliproxyapi.service`：

```ini
[Unit]
Description=CLIProxyAPI
After=network-online.target
Wants=network-online.target

[Service]
ExecStart=/usr/local/bin/cli-proxy-api -config /etc/cliproxyapi/config.yaml
Restart=on-failure
RestartSec=5

[Install]
WantedBy=multi-user.target
```

启动服务：

```sh
sudo systemctl daemon-reload
sudo systemctl enable --now cliproxyapi
```

## Windows x64 启动

解压 `cli-proxy-api_team-dist_windows_amd64.zip` 后，在 PowerShell 中运行：

```powershell
Copy-Item .\team-config.yaml .\config.yaml
.\cli-proxy-api.exe -config .\config.yaml
```

可选：安装为 Windows Service。

```powershell
New-Service -Name CLIProxyAPI `
  -BinaryPathName 'C:\path\to\cli-proxy-api.exe -config C:\path\to\config.yaml' `
  -DisplayName CLIProxyAPI `
  -StartupType Automatic
Start-Service CLIProxyAPI
```

请把路径替换为实际绝对路径。

## Claude Code 配置

`~/.claude/settings.json` 示例：

```json
{
  "model": "opus[1m]",
  "env": {
    "ANTHROPIC_BASE_URL": "http://127.0.0.1:8317",
    "ANTHROPIC_AUTH_TOKEN": "sk-dummy",
    "ANTHROPIC_MODEL": "claude-opus-4-7",
    "ANTHROPIC_DEFAULT_OPUS_MODEL": "claude-opus-4-7",
    "ANTHROPIC_DEFAULT_SONNET_MODEL": "claude-opus-4-7",
    "ANTHROPIC_DEFAULT_HAIKU_MODEL": "claude-opus-4-7",
    "CLAUDE_CODE_MAX_OUTPUT_TOKENS": "128000"
  }
}
```

修改后重启 Claude Code。

## 验证

检查模型列表：

```sh
curl -sS http://127.0.0.1:8317/v1/models \
  -H 'Authorization: Bearer sk-dummy'
```

期望看到：

```text
claude-opus-4-7
```

在 Claude Code 中运行：

```text
/model
```

期望显示：

```text
Opus 4.7 (1M context)
```
