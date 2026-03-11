package commands

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

type StatusCommand struct{}

func NewStatusCommand() *StatusCommand {
	return &StatusCommand{}
}

func (c *StatusCommand) Name() string        { return "status" }
func (c *StatusCommand) Aliases() []string    { return nil }
func (c *StatusCommand) Description() string  { return "查看本地服务和系统状态" }
func (c *StatusCommand) Usage() string        { return `/status` }

func (c *StatusCommand) Execute(ctx context.Context, args string, meta *MessageMeta) (string, error) {
	var sb strings.Builder
	sb.WriteString("📊 系统状态\n\n")

	// 系统信息
	sb.WriteString(fmt.Sprintf("  OS: %s/%s\n", runtime.GOOS, runtime.GOARCH))

	// uptime
	if out, err := exec.Command("uptime").Output(); err == nil {
		sb.WriteString(fmt.Sprintf("  Uptime: %s\n", strings.TrimSpace(string(out))))
	}

	// tmux 会话
	sb.WriteString("\n📺 tmux 会话:\n")
	if out, err := exec.Command("tmux", "list-sessions").Output(); err == nil {
		sessions := strings.TrimSpace(string(out))
		if sessions != "" {
			for _, line := range strings.Split(sessions, "\n") {
				sb.WriteString(fmt.Sprintf("  %s\n", line))
			}
		}
	} else {
		sb.WriteString("  (无活跃会话)\n")
	}

	// Claude Code 版本
	sb.WriteString("\n🤖 Claude Code:\n")
	if out, err := exec.Command("claude", "--version").Output(); err == nil {
		sb.WriteString(fmt.Sprintf("  版本: %s\n", strings.TrimSpace(string(out))))
	} else {
		sb.WriteString("  未安装或不在 PATH 中\n")
	}

	sb.WriteString(fmt.Sprintf("\n⏱️ 查询时间: %s", time.Now().Format("2006-01-02 15:04:05")))

	return sb.String(), nil
}
