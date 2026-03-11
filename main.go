package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	lark "github.com/larksuite/oapi-sdk-go/v3"
	larkcore "github.com/larksuite/oapi-sdk-go/v3/core"
	larkws "github.com/larksuite/oapi-sdk-go/v3/ws"

	"feishu-bot/commands"
)

func main() {
	configPath := flag.String("config", "config.yaml", "配置文件路径")
	flag.Parse()

	// 加载配置
	cfg, err := LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	// 验证必要配置
	if cfg.AppID == "" || cfg.AppSecret == "" {
		// 尝试从环境变量读取
		if id := os.Getenv("FEISHU_APP_ID"); id != "" {
			cfg.AppID = id
		}
		if secret := os.Getenv("FEISHU_APP_SECRET"); secret != "" {
			cfg.AppSecret = secret
		}
		if cfg.AppID == "" || cfg.AppSecret == "" {
			log.Fatal("请配置 app_id 和 app_secret（配置文件或环境变量 FEISHU_APP_ID / FEISHU_APP_SECRET）")
		}
	}

	// 创建飞书 API 客户端
	larkClient := lark.NewClient(cfg.AppID, cfg.AppSecret)

	// 创建各模块
	replier := NewReplier(larkClient)
	sessionMgr := NewSessionManager(cfg)
	router := NewRouter()

	// 注册命令
	helpCmd := commands.NewHelpCommand()

	router.Register(commands.NewAskCommand(commands.AskConfig{
		ClaudeBin:    cfg.ClaudeBin,
		DefaultCWD:   cfg.DefaultCWD,
		AllowedTools: cfg.ClaudeAllowedTools,
		DangerMode:   cfg.ClaudeDangerMode,
		ResolveCWD:   cfg.ResolveCWD,
	}))
	router.Register(commands.NewSessionCommand(sessionMgr))
	router.Register(commands.NewSendCommand(sessionMgr))
	router.Register(commands.NewShellCommand(cfg.ShellWhitelist))
	router.Register(commands.NewStatusCommand())
	router.Register(helpCmd)

	// 让 help 命令能列出所有命令
	helpCmd.SetCommands(router.AllCommands())

	// 启动 Hook HTTP 服务（供 Claude Code hooks 回调）
	hookServer := NewHookServer(cfg.HookPort, replier)
	hookServer.Start()

	// 创建飞书事件处理器
	eventHandler := NewEventHandler(cfg, router, replier)

	// 创建 WebSocket 客户端
	wsLogLevel := larkcore.LogLevelInfo
	if cfg.LogLevel == "debug" {
		wsLogLevel = larkcore.LogLevelDebug
	}

	wsClient := larkws.NewClient(cfg.AppID, cfg.AppSecret,
		larkws.WithEventHandler(eventHandler),
		larkws.WithLogLevel(wsLogLevel),
	)

	// 优雅关闭
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigCh
		fmt.Println("\n正在关闭...")
		cancel()
	}()

	// 启动 WebSocket 连接
	log.Println("🚀 飞书机器人启动中...")
	log.Printf("   App ID: %s", cfg.AppID[:8]+"...")
	log.Printf("   Hook 端口: %d", cfg.HookPort)
	log.Printf("   默认工作目录: %s", cfg.DefaultCWD)
	log.Printf("   Claude 权限模式: %s", func() string {
		if cfg.ClaudeDangerMode {
			return "danger (全部放行)"
		}
		return fmt.Sprintf("白名单 (%d 个工具)", len(cfg.ClaudeAllowedTools))
	}())

	if err := wsClient.Start(ctx); err != nil {
		log.Fatalf("WebSocket 连接失败: %v", err)
	}
}
