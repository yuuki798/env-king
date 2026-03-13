package feishu

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	lark "github.com/larksuite/oapi-sdk-go/v3"
	larkcore "github.com/larksuite/oapi-sdk-go/v3/core"
	"github.com/larksuite/oapi-sdk-go/v3/event/dispatcher"
	larkim "github.com/larksuite/oapi-sdk-go/v3/service/im/v1"
	larkws "github.com/larksuite/oapi-sdk-go/v3/ws"
	"github.com/spf13/viper"

	bizagent "yuuki798/env-king/biz/agent"
	bizscript "yuuki798/env-king/biz/script"
)

var mentionRe = regexp.MustCompile(`@_user_\d+`)

// Connector 将飞书群消息通过长连接桥接到 env-king Agent 对话系统。
// - 每个 chat_id 对应独立的 Agent 会话
// - 只响应明确 @机器人 的消息
// - 收到消息立即先发"思考中…"占位卡，LLM 结束后 patch 替换
// - 支持运行时开启/关闭消息监听
type Connector struct {
	cli          *lark.Client
	bot          *bizagent.FeishuBot
	appID        string
	appSecret    string
	botOpenID    string             // 机器人自身的 open_id，用于 @mention 检测
	allowedChats map[string]struct{} // 非空时只响应白名单内的群
	dedup        sync.Map           // 防止同一消息重复处理

	// enabled 控制是否处理收到的消息（1=开启，0=关闭）
	enabled int32
	// wsCancel 用于停止 WebSocket 长连接
	wsCancel context.CancelFunc
	wsMu     sync.Mutex
	// jobHookID 注册的 job 完成通知 hook id，用于注销
	jobHookID string
}

var (
	connOnce  sync.Once
	singleton *Connector
)

// GetConnector 返回已初始化的单例（未初始化时返回 nil）。
func GetConnector() *Connector { return singleton }

// IsEnabled 返回当前飞书监听是否处于开启状态。
func (c *Connector) IsEnabled() bool {
	return atomic.LoadInt32(&c.enabled) == 1
}

// Enable 开启消息监听（若已开启则无操作）。
func (c *Connector) Enable() {
	atomic.StoreInt32(&c.enabled, 1)
	log.Println("[feishu] 消息监听已开启")
}

// Disable 关闭消息监听（已收到的消息不再处理；WebSocket 连接保持，节省重连开销）。
func (c *Connector) Disable() {
	atomic.StoreInt32(&c.enabled, 0)
	log.Println("[feishu] 消息监听已关闭")
}

// FromConfig 读取 viper 配置创建 Connector 单例；未配置 app_id/app_secret 时返回 nil。
func FromConfig() *Connector {
	appID := strings.TrimSpace(viper.GetString("feishu.app_id"))
	appSecret := strings.TrimSpace(viper.GetString("feishu.app_secret"))
	if appID == "" || appSecret == "" {
		return nil
	}
	connOnce.Do(func() {
		chatIDs := viper.GetStringSlice("feishu.chat_ids")
		allowed := make(map[string]struct{}, len(chatIDs))
		for _, id := range chatIDs {
			if id = strings.TrimSpace(id); id != "" {
				allowed[id] = struct{}{}
			}
		}
		singleton = newConnector(appID, appSecret, allowed)
	})
	return singleton
}

func newConnector(appID, appSecret string, allowedChats map[string]struct{}) *Connector {
	cli := lark.NewClient(appID, appSecret,
		lark.WithLogLevel(larkcore.LogLevelWarn),
		lark.WithReqTimeout(15*time.Second),
		lark.WithEnableTokenCache(true),
		lark.WithHttpClient(http.DefaultClient),
	)
	c := &Connector{
		cli:          cli,
		bot:          bizagent.NewFeishuBot(),
		appID:        appID,
		appSecret:    appSecret,
		allowedChats: allowedChats,
	}

	// 尝试从配置读取 bot_open_id；若未配置则通过 API 自动获取
	if id := strings.TrimSpace(viper.GetString("feishu.bot_open_id")); id != "" {
		c.botOpenID = id
		log.Printf("[feishu] bot open_id (from config): %s", id)
	} else {
		c.botOpenID = fetchBotOpenID(cli)
		if c.botOpenID != "" {
			log.Printf("[feishu] bot open_id (from API): %s", c.botOpenID)
		} else {
			log.Println("[feishu] 警告：无法获取 bot open_id，将降级为响应所有群消息（无 @mention 过滤）")
		}
	}

	if len(allowedChats) > 0 {
		ids := make([]string, 0, len(allowedChats))
		for id := range allowedChats {
			ids = append(ids, id)
		}
		log.Printf("[feishu] 群聊白名单: %v", ids)
	}
	// 默认开启
	atomic.StoreInt32(&c.enabled, 1)
	return c
}

// fetchBotOpenID 调用飞书 Bot Info API 获取当前应用机器人的 open_id。
func fetchBotOpenID(cli *lark.Client) string {
	type botInfoResp struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
		Bot  *struct {
			OpenID string `json:"open_id"`
		} `json:"bot"`
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, err := cli.Get(ctx, "/open-apis/bot/v3/info", nil, larkcore.AccessTokenTypeApp)
	if err != nil {
		log.Printf("[feishu] fetchBotOpenID 请求失败: %v", err)
		return ""
	}
	if resp.StatusCode != http.StatusOK {
		log.Printf("[feishu] fetchBotOpenID HTTP 状态异常: %d", resp.StatusCode)
		return ""
	}
	var result botInfoResp
	if err := json.Unmarshal(resp.RawBody, &result); err != nil {
		log.Printf("[feishu] fetchBotOpenID 解析响应失败: %v", err)
		return ""
	}
	if result.Code != 0 {
		log.Printf("[feishu] fetchBotOpenID 业务异常: code=%d msg=%s", result.Code, result.Msg)
		return ""
	}
	if result.Bot == nil {
		return ""
	}
	return result.Bot.OpenID
}

// Start 启动飞书 WebSocket 长连接（阻塞），并注册 job 完成通知 hook。
func (c *Connector) Start(ctx context.Context) {
	// 注册 job 完成后的飞书通知 hook
	c.wsMu.Lock()
	if c.jobHookID == "" {
		c.jobHookID = bizscript.RegisterJobDoneHook(func(job *bizscript.JobView) {
			if !c.IsEnabled() {
				return
			}
			c.notifyJobDone(job)
		})
	}
	c.wsMu.Unlock()

	handler := dispatcher.NewEventDispatcher("", "").
		OnP2MessageReceiveV1(func(ctx context.Context, event *larkim.P2MessageReceiveV1) error {
			return c.onMessageReceive(ctx, event)
		})

	wsCli := larkws.NewClient(c.appID, c.appSecret,
		larkws.WithEventHandler(handler),
		larkws.WithLogLevel(larkcore.LogLevelWarn),
	)
	log.Println("[feishu] WebSocket 长连接启动中…")
	if err := wsCli.Start(ctx); err != nil {
		log.Printf("[feishu] WebSocket 连接异常退出: %v", err)
	}
}

// notifyJobDone 查找 job 对应的 chat_id，并向飞书群发送任务完成通知。
func (c *Connector) notifyJobDone(job *bizscript.JobView) {
	if job == nil || job.ID == "" {
		return
	}
	st, err := bizagent.GetFeishuStoreExported()
	if err != nil {
		return
	}
	raw, err := st.GetString(bizagent.FeishuBucketJobNotify, job.ID)
	if err != nil || len(raw) == 0 {
		return
	}
	chatID := strings.TrimSpace(string(raw))
	if chatID == "" {
		return
	}
	// 清理已消费的映射
	_ = st.DeleteString(bizagent.FeishuBucketJobNotify, job.ID)

	// 构建通知内容
	statusEmoji := "✅"
	if job.Status == "failed" {
		statusEmoji = "❌"
	} else if job.Status == "canceled" {
		statusEmoji = "⏹️"
	}

	var msg strings.Builder
	msg.WriteString(fmt.Sprintf("%s **任务执行完毕**\n\n", statusEmoji))
	msg.WriteString(fmt.Sprintf("- **任务 ID**：`%s`\n", job.ID))
	msg.WriteString(fmt.Sprintf("- **状态**：`%s`\n", job.Status))
	if job.Error != "" {
		msg.WriteString(fmt.Sprintf("- **错误**：%s\n", job.Error))
	}
	if job.Logs != "" {
		logs := job.Logs
		const maxLogLen = 800
		if len(logs) > maxLogLen {
			logs = "..." + logs[len(logs)-maxLogLen:]
		}
		msg.WriteString(fmt.Sprintf("\n**日志（末尾）：**\n```\n%s\n```", logs))
	}

	if sendErr := c.sendMarkdownToChat(context.Background(), chatID, msg.String()); sendErr != nil {
		log.Printf("[feishu] 发送任务完成通知失败 chat_id=%s job_id=%s: %v", chatID, job.ID, sendErr)
	}
}

// onMessageReceive 处理飞书收到的消息事件。
func (c *Connector) onMessageReceive(_ context.Context, event *larkim.P2MessageReceiveV1) error {
	// 监听开关关闭时丢弃所有消息
	if !c.IsEnabled() {
		return nil
	}
	if event == nil || event.Event == nil {
		return nil
	}
	sender := event.Event.Sender
	msg := event.Event.Message

	// 1. 只处理真实用户消息
	if sender == nil || sender.SenderType == nil || *sender.SenderType != "user" {
		return nil
	}

	chatID := strPtr(msg.ChatId)
	msgID := strPtr(msg.MessageId)
	msgType := strPtr(msg.MessageType)
	if chatID == "" || msgID == "" {
		return nil
	}

	// 2. 群聊白名单过滤
	if len(c.allowedChats) > 0 {
		if _, ok := c.allowedChats[chatID]; !ok {
			return nil
		}
	}

	// 3. @mention 过滤：只响应明确 @机器人 的消息
	//    若 botOpenID 为空（获取失败），则跳过此过滤降级为全响应
	if c.botOpenID != "" && !isBotMentioned(msg.Mentions, c.botOpenID) {
		return nil
	}

	// 4. 去重
	if _, existed := c.dedup.LoadOrStore(msgID, struct{}{}); existed {
		return nil
	}
	go func() {
		time.Sleep(5 * time.Minute)
		c.dedup.Delete(msgID)
	}()

	// 5. 提取纯文本（去掉 @mention 占位符）
	text := extractText(msgType, msg.Content)
	if text == "" {
		return nil
	}

	log.Printf("[feishu] @mention chat_id=%s msg_id=%s text=%q", chatID, msgID, truncate(text, 100))

	// 6. 立即发"思考中…"占位卡，然后异步生成回复并 patch 替换
	go c.handleAndReply(chatID, text)
	return nil
}

// handleAndReply：先发占位卡，Agent 返回后 patch 更新。
func (c *Connector) handleAndReply(chatID, text string) {
	bgCtx := context.Background()

	// 立即发出"思考中…"卡片，让用户知道机器人在处理
	placeholderMsgID, err := c.createMarkdownCard(bgCtx, chatID, "💬 正在思考中，请稍候…")
	if err != nil {
		log.Printf("[feishu] 发送占位卡失败 chat_id=%s: %v", chatID, err)
	}

	// 给 Agent 调用 3 分钟超时
	agentCtx, cancel := context.WithTimeout(bgCtx, 3*time.Minute)
	defer cancel()

	reply, err := c.bot.HandleIncomingMessage(agentCtx, chatID, text)
	if err != nil {
		log.Printf("[feishu] Agent 处理失败 chat_id=%s: %v", chatID, err)
		reply = "⚠️ 处理消息时出了点问题，请稍后再试。"
	}

	if placeholderMsgID != "" {
		// 有占位消息 → patch 更新为实际回复
		if patchErr := c.patchMarkdownCard(bgCtx, placeholderMsgID, reply); patchErr != nil {
			log.Printf("[feishu] patch 卡片失败，尝试新发 chat_id=%s: %v", chatID, patchErr)
			// patch 失败时兜底发新消息
			if sendErr := c.sendMarkdownToChat(bgCtx, chatID, reply); sendErr != nil {
				log.Printf("[feishu] 兜底发送也失败 chat_id=%s: %v", chatID, sendErr)
			}
		}
	} else {
		// 占位卡发送失败时，直接发实际回复
		if sendErr := c.sendMarkdownToChat(bgCtx, chatID, reply); sendErr != nil {
			log.Printf("[feishu] 发送回复失败 chat_id=%s: %v", chatID, sendErr)
		}
	}
}

// isBotMentioned 判断 mentions 列表中是否包含指定 open_id 的机器人。
func isBotMentioned(mentions []*larkim.MentionEvent, botOpenID string) bool {
	for _, m := range mentions {
		if m == nil || m.Id == nil || m.Id.OpenId == nil {
			continue
		}
		if *m.Id.OpenId == botOpenID {
			return true
		}
	}
	return false
}

// buildMarkdownCardContent 构建 schema 2.0 markdown 卡片 JSON 字符串。
func buildMarkdownCardContent(text string) (string, error) {
	card := map[string]interface{}{
		"schema": "2.0",
		"config": map[string]interface{}{
			"wide_screen_mode": true,
		},
		"body": map[string]interface{}{
			"elements": []interface{}{
				map[string]interface{}{
					"tag":     "markdown",
					"content": text,
				},
			},
		},
	}
	b, err := json.Marshal(card)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// createMarkdownCard 发送一条 interactive card 消息到群，返回 message_id。
func (c *Connector) createMarkdownCard(ctx context.Context, chatID, text string) (string, error) {
	content, err := buildMarkdownCardContent(text)
	if err != nil {
		return "", fmt.Errorf("序列化卡片失败: %w", err)
	}
	resp, err := c.cli.Im.V1.Message.Create(ctx,
		larkim.NewCreateMessageReqBuilder().
			ReceiveIdType("chat_id").
			Body(larkim.NewCreateMessageReqBodyBuilder().
				ReceiveId(chatID).
				MsgType("interactive").
				Content(content).
				Build()).
			Build(),
	)
	if err != nil {
		return "", fmt.Errorf("飞书 Create 失败: %w", err)
	}
	if !resp.Success() {
		return "", fmt.Errorf("飞书 Create 非成功: code=%d msg=%s", resp.Code, resp.Msg)
	}
	if resp.Data == nil || resp.Data.MessageId == nil {
		return "", fmt.Errorf("飞书 Create 返回空 message_id")
	}
	return *resp.Data.MessageId, nil
}

// patchMarkdownCard 用新内容更新已发送的 interactive card 消息。
func (c *Connector) patchMarkdownCard(ctx context.Context, msgID, text string) error {
	content, err := buildMarkdownCardContent(text)
	if err != nil {
		return fmt.Errorf("序列化卡片失败: %w", err)
	}
	resp, err := c.cli.Im.V1.Message.Patch(ctx,
		larkim.NewPatchMessageReqBuilder().
			MessageId(msgID).
			Body(larkim.NewPatchMessageReqBodyBuilder().
				Content(content).
				Build()).
			Build(),
	)
	if err != nil {
		return fmt.Errorf("飞书 Patch 失败: %w", err)
	}
	if !resp.Success() {
		return fmt.Errorf("飞书 Patch 非成功: code=%d msg=%s", resp.Code, resp.Msg)
	}
	return nil
}

// sendMarkdownToChat 直接发新消息（patchMarkdownCard 失败时的兜底）。
func (c *Connector) sendMarkdownToChat(ctx context.Context, chatID, text string) error {
	_, err := c.createMarkdownCard(ctx, chatID, text)
	return err
}

// extractText 从飞书消息 content JSON 中提取纯文本，并去除 @mention 占位符。
func extractText(msgType string, rawContent *string) string {
	if rawContent == nil {
		return ""
	}
	content := strings.TrimSpace(*rawContent)
	if content == "" {
		return ""
	}

	var text string
	switch msgType {
	case "text":
		var obj struct {
			Text string `json:"text"`
		}
		if err := json.Unmarshal([]byte(content), &obj); err == nil {
			text = obj.Text
		}
	case "post":
		var obj struct {
			ZhCN *struct {
				Title   string `json:"title"`
				Content [][]struct {
					Tag  string `json:"tag"`
					Text string `json:"text"`
				} `json:"content"`
			} `json:"zh_cn"`
		}
		if err := json.Unmarshal([]byte(content), &obj); err == nil && obj.ZhCN != nil {
			var parts []string
			if t := strings.TrimSpace(obj.ZhCN.Title); t != "" {
				parts = append(parts, t)
			}
			for _, row := range obj.ZhCN.Content {
				for _, el := range row {
					if (el.Tag == "text" || el.Tag == "md") && strings.TrimSpace(el.Text) != "" {
						parts = append(parts, strings.TrimSpace(el.Text))
					}
				}
			}
			text = strings.Join(parts, " ")
		}
	default:
		var obj struct {
			Text string `json:"text"`
		}
		if err := json.Unmarshal([]byte(content), &obj); err == nil {
			text = obj.Text
		}
	}

	text = mentionRe.ReplaceAllString(text, "")
	return strings.TrimSpace(text)
}

func strPtr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "..."
}
