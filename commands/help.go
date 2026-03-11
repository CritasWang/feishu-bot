package commands

import (
	"context"
	"fmt"
	"strings"
)

type HelpCommand struct {
	commands []Command // 注入所有已注册的命令
}

func NewHelpCommand() *HelpCommand {
	return &HelpCommand{}
}

// SetCommands 注入所有命令列表（在 router 注册完成后调用）
func (c *HelpCommand) SetCommands(cmds []Command) {
	c.commands = cmds
}

func (c *HelpCommand) Name() string        { return "help" }
func (c *HelpCommand) Aliases() []string    { return []string{"h", "?"} }
func (c *HelpCommand) Description() string  { return "显示帮助信息" }
func (c *HelpCommand) Usage() string        { return `/help [命令名]` }

func (c *HelpCommand) Execute(ctx context.Context, args string, meta *MessageMeta) (string, error) {
	target := strings.TrimSpace(args)

	// 指定命令的详细帮助
	if target != "" {
		for _, cmd := range c.commands {
			if cmd.Name() == target {
				aliases := ""
				if len(cmd.Aliases()) > 0 {
					aliases = fmt.Sprintf("\n别名: /%s", strings.Join(cmd.Aliases(), ", /"))
				}
				return fmt.Sprintf("📖 /%s — %s%s\n\n%s", cmd.Name(), cmd.Description(), aliases, cmd.Usage()), nil
			}
		}
		return fmt.Sprintf("未知命令: %s", target), nil
	}

	// 列出所有命令
	var sb strings.Builder
	sb.WriteString("📋 可用命令:\n\n")
	for _, cmd := range c.commands {
		aliases := ""
		if len(cmd.Aliases()) > 0 {
			aliases = fmt.Sprintf(" (/%s)", strings.Join(cmd.Aliases(), ", /"))
		}
		sb.WriteString(fmt.Sprintf("  /%s%s — %s\n", cmd.Name(), aliases, cmd.Description()))
	}
	sb.WriteString("\n输入 /help <命令名> 查看详细用法")
	return sb.String(), nil
}
