package commands

import (
	"context"
	"fmt"
	"strings"
)

// tmuxKeyMap 友好名称 → tmux send-keys 参数
var tmuxKeyMap = map[string][]string{
	// 方向键
	"up":    {"Up"},
	"down":  {"Down"},
	"left":  {"Left"},
	"right": {"Right"},

	// 功能键
	"enter": {"Enter"},
	"tab":   {"Tab"},
	"esc":   {"Escape"},
	"space": {"Space"},

	// Ctrl 组合键
	"ctrl+c": {"C-c"},
	"ctrl+d": {"C-d"},
	"ctrl+z": {"C-z"},
	"ctrl+l": {"C-l"},
	"ctrl+a": {"C-a"},
	"ctrl+e": {"C-e"},
	"ctrl+r": {"C-r"},
	"ctrl+p": {"C-p"},
	"ctrl+n": {"C-n"},

	// 常用别名
	"y":     {"y", "Enter"},    // 快速确认 yes
	"n":     {"n", "Enter"},    // 快速确认 no
	"yes":   {"y", "e", "s", "Enter"},
	"no":    {"n", "o", "Enter"},
}

// KeySender 向 tmux 会话发送原始按键的接口
type KeySender interface {
	SendKeys(key string, tmuxKeys ...string) error
}

type KeyCommand struct {
	sm KeySender
}

func NewKeyCommand(sm KeySender) *KeyCommand {
	return &KeyCommand{sm: sm}
}

func (c *KeyCommand) Name() string      { return "key" }
func (c *KeyCommand) Aliases() []string  { return []string{"k"} }
func (c *KeyCommand) Description() string { return "向活跃会话发送特殊按键（方向键、回车、Tab等）" }
func (c *KeyCommand) Usage() string {
	return `/key <按键>        发送单个按键
/key <按键> <次数>  连续发送多次

方向键:  up / down / left / right
功能键:  enter / tab / esc / space
Ctrl键:  ctrl+c / ctrl+d / ctrl+z / ctrl+l
快捷:    y (确认) / n (拒绝)

示例:
  /key enter          发送回车
  /key up             发送上箭头
  /key tab            发送 Tab
  /key esc            发送 Escape
  /key ctrl+c         发送 Ctrl+C
  /key y              发送 y + 回车（快速确认）
  /key up 3           连续发送 3 次上箭头`
}

func (c *KeyCommand) Execute(ctx context.Context, args string, meta *MessageMeta) (string, error) {
	args = strings.TrimSpace(args)
	if args == "" {
		return c.Usage(), nil
	}

	parts := strings.Fields(args)
	keyName := strings.ToLower(parts[0])

	// 解析重复次数
	repeat := 1
	if len(parts) > 1 {
		if _, err := fmt.Sscanf(parts[1], "%d", &repeat); err != nil || repeat < 1 {
			repeat = 1
		}
		if repeat > 20 {
			repeat = 20 // 安全上限
		}
	}

	// 查找 tmux 按键映射
	tmuxKeys, ok := tmuxKeyMap[keyName]
	if !ok {
		// 如果不在映射表中，尝试直接作为 tmux key name 发送
		tmuxKeys = []string{keyName}
	}

	sessionKey := meta.SessionKey()

	for i := 0; i < repeat; i++ {
		if err := c.sm.SendKeys(sessionKey, tmuxKeys...); err != nil {
			return fmt.Sprintf("❌ 发送按键失败: %s", err), nil
		}
	}

	// 格式化反馈
	keyDisplay := keyName
	if displayNames, exists := keyDisplayMap[keyName]; exists {
		keyDisplay = displayNames
	}
	if repeat > 1 {
		return fmt.Sprintf("⌨️ 已发送 %s ×%d", keyDisplay, repeat), nil
	}
	return fmt.Sprintf("⌨️ 已发送 %s", keyDisplay), nil
}

// keyDisplayMap 用于显示友好的按键名称
var keyDisplayMap = map[string]string{
	"up":     "↑",
	"down":   "↓",
	"left":   "←",
	"right":  "→",
	"enter":  "↵ Enter",
	"tab":    "⇥ Tab",
	"esc":    "⎋ Esc",
	"space":  "␣ Space",
	"ctrl+c": "Ctrl+C",
	"ctrl+d": "Ctrl+D",
	"ctrl+z": "Ctrl+Z",
	"ctrl+l": "Ctrl+L",
	"ctrl+a": "Ctrl+A",
	"ctrl+e": "Ctrl+E",
	"ctrl+r": "Ctrl+R",
	"ctrl+p": "Ctrl+P",
	"ctrl+n": "Ctrl+N",
	"y":      "y↵ (确认)",
	"n":      "n↵ (拒绝)",
	"yes":    "yes↵",
	"no":     "no↵",
}
