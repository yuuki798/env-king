package initial

import (
	"context"
	"log"

	"yuuki798/env-king/biz/feishu"
)

func InitAll() {
	InitConfig()
	InitDockerHelperHub()
	initFeishu()
}

// initFeishu 读取配置，若配置了 feishu.app_id / feishu.app_secret，则启动飞书机器人长连接。
// 连接在独立 goroutine 中运行，不阻塞主服务。
func initFeishu() {
	conn := feishu.FromConfig()
	if conn == nil {
		log.Println("[feishu] 未配置 app_id/app_secret，跳过飞书机器人启动")
		return
	}
	log.Println("[feishu] 飞书机器人已配置，正在启动 WebSocket 长连接…")
	go func() {
		conn.Start(context.Background())
	}()
}
