# feishu-bot

飞书机器人本地服务 — 通过飞书消息远程操控 Claude Code 和本地程序。

## 特性

- **WebSocket 长连接**: 无需公网 IP，本地直接运行
- **双模式 Claude Code 集成**:
  - `/ask` — 无状态模式，`claude -p` 一次性调用
  - `/session` + `/s` — tmux 持久会话，保持上下文
- **通用命令框架**: 可扩展的命令路由，新增命令只需实现接口
- **Claude Code Hooks 回调**: 内置 HTTP 端点，支持双向通信
- **安全控制**: 用户/群聊白名单、Shell 命令白名单

## 快速开始

### 1. 飞书应用配置

1. 登录 [飞书开放平台](https://open.feishu.cn) → 创建企业自建应用
2. 添加「机器人」能力
3. 权限: `im:message`、`im:message:send_as_bot`、`im:message:patch`
4. 事件订阅 → **WebSocket 模式** → 订阅 `im.message.receive_v1`
5. 发布应用版本

### 2. 配置

```bash
cp config.yaml config.local.yaml
# 编辑 config.local.yaml，填入 app_id 和 app_secret
```

或使用环境变量:

```bash
export FEISHU_APP_ID="cli_xxx"
export FEISHU_APP_SECRET="xxx"
```

### 3. 运行

```bash
# 编译
go build -o feishu-bot .

# 后台启动（日志写入 logs/ 目录，按天自动切换）
./feishu-bot start --config config.local.yaml

# 查看状态
./feishu-bot status

# 停止
./feishu-bot stop

# 重启
./feishu-bot restart --config config.local.yaml

# 前台运行（日志输出到终端，调试用）
./feishu-bot console --config config.local.yaml
```

### 日志

- `start` 模式日志写入 `logs/feishu-bot.log`
- 跨天自动归档为 `logs/feishu-bot-YYYY-MM-DD.log.gz`（gzip 压缩）
- `console` 模式日志直接输出到终端

## 命令列表

| 命令 | 说明 | 示例 |
|------|------|------|
| `/ask <提示词>` | Claude Code 无状态问答 | `/ask 帮我看看有什么文件` |
| `/ask --cwd <目录> <提示词>` | 指定工作目录 | `/ask --cwd /path/to/project 分析结构` |
| `/ask @别名 <提示词>` | 用项目别名 | `/ask @server 看看迁移方案` |
| `/session start [目录]` | 启动 tmux 持久会话 | `/session start /path/to/project` |
| `/session status` | 查看当前会话详情 | `/session status` |
| `/session list` | 列出所有活跃会话 | `/session list` |
| `/session kill <会话名>` | 终止指定会话 | `/session kill cc-chat-xxx` |
| `/session stop` | 关闭当前会话 | `/session stop` |
| `/s <消息>` | 发送到活跃会话 | `/s 帮我重构这个函数` |
| `/shell <命令>` | 执行白名单 shell 命令 | `/shell docker ps` |
| `/project` 或 `/p` | 查看已配置的项目别名 | `/project` |
| `/status` | 查看系统状态和活跃会话 | `/status` |
| `/danger on\|off` | 切换 Claude Code 权限绕过模式 | `/danger on` |
| `/reload` | 热重载配置文件 | `/reload` |
| `/help [命令]` | 帮助信息 | `/help ask` |

**非命令消息**: 如有活跃 tmux 会话则发送到会话，否则等同 `/ask`。

## 配置说明

### 超时配置

默认情况下，Claude Code 执行超时为 50 分钟（相比之前的 5 分钟大幅提升）。你可以在 `config.yaml` 中自定义超时时间：

```yaml
# Claude Code 超时配置（分钟）
claude_ask_timeout: 50        # /ask 命令超时时间
claude_session_timeout: 50    # /session 会话响应超时
```

### 消息分块与长输出处理

系统会自动将超过 3500 字符的长消息分块发送，避免消息截断导致信息丢失。分块时会智能选择段落、句子边界，确保内容完整性。每条分块消息会标注序号（如 [1/3], [2/3], [3/3]）。

你可以自定义分块大小：

```yaml
# 消息分块配置
max_chunk_size: 3500    # 长消息分块大小（默认 3500 字符）
```

### 交互式输入处理

当 Claude Code 在 `/session` 会话中遇到交互式提示（如确认操作、yes/no 问题等）时，系统会自动检测并提示你：

```
⚠️ 检测到交互式提示，Claude Code 正在等待输入。
💡 请使用 /s 命令发送您的响应。
```

你可以通过 `/s yes` 或 `/s n` 等命令回应交互提示。系统会检测以下常见交互模式：
- `(y/n)`, `[yes/no]` - 确认问题
- `continue?`, `proceed?` - 继续提示
- `press enter`, `按回车` - 等待确认
- 以及其他常见的交互式提示

### 嵌套会话问题

本项目已修复嵌套 Claude Code 会话问题。系统会自动过滤可能导致嵌套会话检测的环境变量（如 `CLAUDECODE`、`ANTHROPIC_*` 等），确保在 Claude Code 环境中也能正常启动子进程。

### 状态查询和项目管理

**会话管理** (`/session`):
- `/session start [目录]` - 启动新的 Claude Code 持久会话
- `/session status` - 查看当前会话的详细信息（会话名、工作目录、创建时间、运行时间、状态）
- `/session list` - 列出所有活跃的会话及其工作目录和运行时间
- `/session kill <会话名>` - 终止指定名称的会话（通过 `/session list` 查看会话名）
- `/session stop` - 关闭当前会话

**查看系统状态** (`/status`):
- 显示系统信息（OS、架构、运行时间）
- 显示默认工作目录
- 显示活跃的 Claude Code 会话及其工作目录和运行时间
- 显示 tmux 会话列表
- 显示 Claude Code 版本信息

**查看项目列表** (`/project` 或 `/p`):
- 显示所有配置的项目别名及其对应目录
- 提供项目别名的使用示例

在 `config.yaml` 中配置项目：
```yaml
projects:
  server: "/Volumes/data/sources/server_migration"
  devops: "/Volumes/data/sources/devops"
  webapp: "/path/to/webapp"
```

使用项目别名：
```
/ask @server 分析最新的代码变更
/session start @webapp
```

## Claude Code Hooks 集成

### 从 Claude Code 通知飞书

在 `~/.claude/settings.json` 中配置 hooks:

```json
{
  "hooks": {
    "PostToolUse": [
      {
        "matcher": "Write|Edit",
        "hooks": [
          {
            "type": "command",
            "command": "curl -sS http://localhost:9876/notify -H 'Content-Type: application/json' -d '{\"event\":\"file_changed\",\"message\":\"Claude 修改了文件\",\"chat_id\":\"oc_xxx\"}'"
          }
        ]
      }
    ]
  }
}
```

### Hook HTTP API

```
POST http://localhost:9876/notify
Content-Type: application/json

{
  "event": "task_complete",
  "message": "任务已完成: 重构了认证模块",
  "chat_id": "oc_xxx"       // 可选，指定推送到哪个飞书聊天
}
```

```
GET http://localhost:9876/health
→ 200 ok
```

## 项目别名

在 `config.yaml` 中配置项目目录别名:

```yaml
projects:
  server: "/Volumes/data/sources/server_migration"
  devops: "/Volumes/data/sources/devops"
  webapp: "/Volumes/data/sources/webapp"
```

使用: `/ask @server 看看迁移方案的进度`

## 扩展命令

实现 `commands.Command` 接口并注册到 router:

```go
type Command interface {
    Name() string
    Aliases() []string
    Description() string
    Usage() string
    Execute(ctx context.Context, args string, meta *MessageMeta) (string, error)
}
```

在 `main.go` 中注册:

```go
router.Register(NewMyCommand())
```

## 为什么不直接使用 OpenClaw？

[OpenClaw](https://github.com/openclaw/openclaw) 是 Anthropic 官方提供的用于在受限环境或多租户场景下运行 Claude Code 的基础设施。虽然 OpenClaw 和本项目都涉及远程访问 Claude Code，但它们针对不同的使用场景和架构需求。

### OpenClaw vs feishu-bot 主要区别

| 维度 | OpenClaw | feishu-bot |
|------|----------|------------|
| **定位** | 企业级多租户 Claude Code 托管平台 | 个人/团队本地 Claude Code 远程控制网关 |
| **部署架构** | 云端集中式部署，需要服务器基础设施 | 本地部署，无需公网 IP 或服务器 |
| **访问方式** | Web UI / API，需要网络暴露 | 飞书聊天界面，通过 WebSocket 反向连接 |
| **用户场景** | 多用户共享的 Claude Code 环境 | 个人工作站的远程 Claude Code 操作 |
| **网络要求** | 需要公网访问、负载均衡、TLS 证书等 | 仅需出站 WebSocket 连接（无需公网 IP） |
| **隔离模型** | 容器化/沙箱化，强隔离 | 本地 tmux 会话隔离 |
| **状态管理** | 无状态设计，适合云端横向扩展 | 支持有状态会话（tmux），保持上下文 |
| **扩展性** | 通过编排工具水平扩展 | 单实例本地运行 |
| **权限控制** | 基于角色的访问控制（RBAC） | 基于白名单的用户/群组控制 |
| **集成方式** | RESTful API / gRPC | 飞书机器人 + Claude Code Hooks |

### 何时选择 feishu-bot？

**适合以下场景**：

1. **个人开发者或小团队**
   - 已有飞书/Lark 作为沟通工具
   - 希望在本地工作站远程执行 Claude Code
   - 不想维护云端服务器基础设施

2. **企业内网环境**
   - 严格的网络安全策略，禁止暴露服务端口到公网
   - 需要在办公室外访问办公室内的开发机器
   - 通过飞书企业自建应用已有的安全审核机制

3. **移动办公场景**
   - 通过手机飞书 App 远程控制家里或办公室的开发环境
   - 外出时需要触发 Claude Code 执行任务
   - 希望通过聊天界面而非 SSH/VPN 进行操作

4. **嵌套会话需求**
   - 需要在 Claude Code 环境中启动子 Claude Code 实例
   - 本项目已解决环境变量过滤问题，支持嵌套场景

5. **交互式会话管理**
   - 需要长时间运行的 Claude Code 会话，保持上下文
   - 需要处理交互式提示（y/n 确认等）
   - 希望查看和管理多个活跃会话

### 何时选择 OpenClaw？

**适合以下场景**：

1. **企业级多租户服务**
   - 需要为多个团队/部门提供统一的 Claude Code 服务
   - 需要资源配额、计费、审计等企业级功能
   - 需要强隔离保证不同用户之间的安全性

2. **云端托管需求**
   - 希望 Claude Code 运行环境与本地工作站解耦
   - 需要高可用性和负载均衡
   - 需要集中管理和监控所有 Claude Code 实例

3. **API 集成需求**
   - 需要通过 RESTful API 或 gRPC 集成到现有系统
   - 需要编程式访问而非聊天界面
   - 需要与 CI/CD 流水线集成

### 两者可以结合使用吗？

**可以！** 两个项目并不互斥，可以组合使用：

- **OpenClaw** 作为后端提供 Claude Code 托管服务
- **feishu-bot** 作为前端提供飞书聊天界面
- 通过修改 `commands/ask.go` 中的 Claude Code 调用逻辑，将请求转发到 OpenClaw API

这种组合可以获得两者的优势：
- OpenClaw 的企业级管理能力
- feishu-bot 的便捷聊天交互体验

### 技术优势对比

**feishu-bot 的独特优势**：

1. **零基础设施要求**
   - 不需要云服务器、域名、SSL 证书
   - 不需要配置防火墙规则或负载均衡器
   - 仅需一个飞书企业自建应用

2. **即时可用性**
   - 5 分钟内完成配置并开始使用
   - 无需学习 Kubernetes/Docker/云平台操作
   - 配置文件热重载，无需重启服务

3. **本地化优势**
   - 直接访问本地文件系统和开发环境
   - 无网络延迟（Claude Code 在本地执行）
   - 支持访问内网资源（数据库、内部服务等）

4. **聊天原生体验**
   - 支持飞书的富文本、代码块、图片等功能
   - 消息自动分块，避免超长内容截断
   - 支持群聊协作和多人共享会话

5. **灵活的会话模式**
   - 无状态模式（`/ask`）：快速一次性查询
   - 有状态模式（`/session`）：保持上下文的长对话
   - 两种模式可根据场景自由切换

## 安全注意

- `/shell` 仅执行白名单内的命令
- 建议配置 `allowed_users` 限制响应范围
- `claude_danger_mode: true` 会跳过所有 Claude Code 权限检查，仅在可控环境使用
- 不要将 `config.yaml` 中的 `app_secret` 提交到版本控制
