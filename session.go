package main

import (
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"time"
)

// Session 表示一个 tmux 中运行的 Claude Code 会话
type Session struct {
	Name      string
	CWD       string
	CreatedAt time.Time
	Active    bool
}

// SessionManager 管理多个 tmux Claude Code 会话
type SessionManager struct {
	mu       sync.RWMutex
	sessions map[string]*Session // key: chatID 或 userID
	config   *Config
}

func NewSessionManager(cfg *Config) *SessionManager {
	return &SessionManager{
		sessions: make(map[string]*Session),
		config:   cfg,
	}
}

// Start 创建一个新的 tmux 会话并启动 Claude Code
func (sm *SessionManager) Start(key, cwd string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if s, ok := sm.sessions[key]; ok && s.Active {
		return fmt.Errorf("会话 %s 已存在，请先 /session stop", s.Name)
	}

	name := fmt.Sprintf("feishu-claude-%s", sanitizeName(key))
	resolvedCWD := sm.config.ResolveCWD(cwd)

	// 构建 claude 命令
	claudeCmd := sm.config.ClaudeBin
	if sm.config.ClaudeDangerMode {
		claudeCmd += " --dangerously-skip-permissions"
	}

	// 创建 tmux 会话
	cmd := exec.Command("tmux", "new-session", "-d", "-s", name,
		"-c", resolvedCWD,
		fmt.Sprintf("cd %s && %s", shellQuote(resolvedCWD), claudeCmd))

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("创建 tmux 会话失败: %w", err)
	}

	// 等待 Claude Code 启动
	time.Sleep(2 * time.Second)

	sm.sessions[key] = &Session{
		Name:      name,
		CWD:       resolvedCWD,
		CreatedAt: time.Now(),
		Active:    true,
	}

	return nil
}

// Send 向 tmux 会话发送消息并等待响应
func (sm *SessionManager) Send(key, message string) (string, error) {
	sm.mu.RLock()
	s, ok := sm.sessions[key]
	sm.mu.RUnlock()

	if !ok || !s.Active {
		return "", fmt.Errorf("没有活跃的会话，请先 /session start [目录]")
	}

	// 记录发送前的 pane 内容行数
	beforeContent, err := sm.capturePane(s.Name)
	if err != nil {
		return "", fmt.Errorf("捕获会话内容失败: %w", err)
	}
	beforeLines := len(strings.Split(beforeContent, "\n"))

	// 发送消息到 tmux
	cmd := exec.Command("tmux", "send-keys", "-t", s.Name, message, "Enter")
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("发送消息失败: %w", err)
	}

	// 轮询等待输出稳定
	response, err := sm.waitForResponse(s.Name, beforeLines)
	if err != nil {
		return "", err
	}

	return response, nil
}

// Stop 关闭 tmux 会话
func (sm *SessionManager) Stop(key string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	s, ok := sm.sessions[key]
	if !ok {
		return fmt.Errorf("没有找到会话")
	}

	// 先发 exit，再 kill
	exec.Command("tmux", "send-keys", "-t", s.Name, "exit", "Enter").Run()
	time.Sleep(500 * time.Millisecond)
	exec.Command("tmux", "kill-session", "-t", s.Name).Run()

	s.Active = false
	delete(sm.sessions, key)

	return nil
}

// GetSession 获取会话信息
func (sm *SessionManager) GetSession(key string) (*Session, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	s, ok := sm.sessions[key]
	return s, ok
}

// ListSessions 列出所有活跃会话
func (sm *SessionManager) ListSessions() []*Session {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	var result []*Session
	for _, s := range sm.sessions {
		if s.Active {
			result = append(result, s)
		}
	}
	return result
}

// capturePane 捕获 tmux pane 内容
func (sm *SessionManager) capturePane(name string) (string, error) {
	cmd := exec.Command("tmux", "capture-pane", "-t", name, "-p", "-S", "-500")
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// waitForResponse 轮询 tmux 输出直到稳定
func (sm *SessionManager) waitForResponse(name string, beforeLines int) (string, error) {
	// 先等一小段时间让 Claude 开始处理
	time.Sleep(1 * time.Second)

	var lastContent string
	stableCount := 0
	maxWait := 300 // 最长等待 300 秒 (5 分钟)

	for i := 0; i < maxWait*2; i++ { // 每 500ms 检查一次
		content, err := sm.capturePane(name)
		if err != nil {
			return "", fmt.Errorf("捕获输出失败: %w", err)
		}

		if content == lastContent && content != "" {
			stableCount++
			// 连续 4 次（2秒）没有变化，认为输出完成
			if stableCount >= 4 {
				// 提取新增的输出
				return extractNewOutput(content, beforeLines), nil
			}
		} else {
			stableCount = 0
			lastContent = content
		}

		time.Sleep(500 * time.Millisecond)
	}

	// 超时，返回已有输出
	if lastContent != "" {
		return extractNewOutput(lastContent, beforeLines) + "\n⚠️ [输出可能不完整，已超时]", nil
	}
	return "", fmt.Errorf("等待响应超时")
}

// extractNewOutput 从 pane 内容中提取新增输出
func extractNewOutput(content string, beforeLines int) string {
	lines := strings.Split(content, "\n")
	if beforeLines >= len(lines) {
		return strings.TrimSpace(content)
	}
	newLines := lines[beforeLines:]
	output := strings.Join(newLines, "\n")
	output = stripANSI(output)
	return strings.TrimSpace(output)
}

// stripANSI 移除 ANSI 转义码
func stripANSI(s string) string {
	re := regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]|\x1b\].*?\x07|\x1b\[.*?[@-~]`)
	return re.ReplaceAllString(s, "")
}

// sanitizeName 清理名称用于 tmux session name
func sanitizeName(s string) string {
	re := regexp.MustCompile(`[^a-zA-Z0-9_-]`)
	result := re.ReplaceAllString(s, "-")
	if len(result) > 20 {
		result = result[:20]
	}
	return result
}

// shellQuote 简单 shell 引号转义
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}
