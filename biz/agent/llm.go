package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"strings"
	"sync"

	"github.com/spf13/viper"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/ollama"
	"github.com/tmc/langchaingo/llms/openai"
)

const formFillPrefix = "[FORM_FILL:"

var (
	llmOnce sync.Once
	llm     llms.Model
	llmErr  error
)

func getLLM() (llms.Model, error) {
	llmOnce.Do(func() {
		typ := strings.TrimSpace(strings.ToLower(viper.GetString("agent.llm.type")))
		if typ == "" {
			typ = "ollama"
		}
		model := strings.TrimSpace(viper.GetString("agent.llm.model"))

		switch typ {
		case "openai":
			// OpenAI 兼容 API：智谱 GLM、OpenAI、Azure 等
			apiKey := strings.TrimSpace(viper.GetString("agent.llm.api_key"))
			baseURL := strings.TrimSpace(viper.GetString("agent.llm.base_url"))
			if baseURL == "" {
				baseURL = "https://api.openai.com/v1"
			}
			if model == "" {
				model = "gpt-4o-mini"
			}
			opts := []openai.Option{openai.WithToken(apiKey), openai.WithBaseURL(baseURL), openai.WithModel(model)}
			llm, llmErr = openai.New(opts...)
		default:
			// Ollama 本地服务
			base := strings.TrimSpace(viper.GetString("agent.llm.ollama_base"))
			if base == "" {
				base = "http://127.0.0.1:11434"
			}
			if model == "" {
				model = "qwen2.5:7b"
			}
			llm, llmErr = ollama.New(
				ollama.WithServerURL(base),
				ollama.WithModel(model),
			)
		}
	})
	return llm, llmErr
}

// FormFillAction 供前端自动填表的结构。
type FormFillAction struct {
	Page    string          `json:"page"`    // 如 script, config
	Form    string          `json:"form"`    // 如 createWorkflow
	Payload json.RawMessage `json:"payload"` // 表单字段 JSON，由前端按表单解析
}

const maxToolCallRounds = 4 // 最多循环几轮工具调用，防止无限循环

// ChatStream 流式对话：支持 ReAct 风格工具调用（[TOOL_CALL]...[/TOOL_CALL]），
// 若 LLM 输出中包含 TOOL_CALL 块则自动执行并将结果喂回，直到获得无工具调用的最终回复。
// activeSkills 为前端传入的已激活 skill ID 列表，空时退回基础 prompt。
func ChatStream(ctx context.Context, currentPage, userMessage string, history []struct{ Role, Content string }, activeSkills []string, streamFn func(ctx context.Context, chunk []byte) error) (fullReply string, formFill *FormFillAction, err error) {
	model, err := getLLM()
	if err != nil {
		return "", nil, err
	}

	systemPrompt := BuildSystemPromptFromSkills(currentPage, activeSkills)
	log.Printf("[agent] chat page=%s activeSkills=%v", currentPage, activeSkills)
	messages := buildMessages(systemPrompt, userMessage, history)

	for round := 0; round < maxToolCallRounds; round++ {
		// 先无流式地收集完整输出，方便检测 TOOL_CALL 块
		var full strings.Builder
		collectFn := func(_ context.Context, chunk []byte) error {
			full.Write(chunk)
			return nil
		}
		opts := []llms.CallOption{llms.WithStreamingFunc(collectFn)}
		_, genErr := model.GenerateContent(ctx, messages, opts...)
		rawOutput := full.String()

		toolCalls := ParseToolCalls(rawOutput)

		if len(toolCalls) == 0 || round == maxToolCallRounds-1 {
			// 无工具调用（或已达到最大轮数）=> 这是最终回复
			finalReply := StripToolCalls(rawOutput)
			if streamFn != nil {
				// 模拟流式输出：把收集到的内容分块推给调用方
				const chunkSize = 64
				for i := 0; i < len(finalReply); i += chunkSize {
					end := i + chunkSize
					if end > len(finalReply) {
						end = len(finalReply)
					}
					if ferr := streamFn(ctx, []byte(finalReply[i:end])); ferr != nil {
						break
					}
				}
			}
			if genErr != nil {
				return finalReply, nil, genErr
			}
			formFill = parseFormFill(rawOutput)
			log.Printf("[agent] chat user=%q", truncateForLog(userMessage, 200))
			log.Printf("[agent] chat reply=%q form_fill=%v tool_rounds=%d",
				truncateForLog(finalReply, 500), formFill != nil, round)
			return finalReply, formFill, nil
		}

		// 有工具调用 => 逐一执行，把结果构造成新的 human 消息加入历史
		log.Printf("[agent] tool_call round=%d calls=%d", round, len(toolCalls))

		// 把 LLM 的这轮回复（含 TOOL_CALL 块）加入消息历史
		messages = append(messages, llms.MessageContent{
			Role:  llms.ChatMessageTypeAI,
			Parts: []llms.ContentPart{llms.TextContent{Text: rawOutput}},
		})

		var resultParts []string
		for i, tc := range toolCalls {
			result, tcErr := ExecuteToolCall(ctx, tc)
			if tcErr != nil {
				result = fmt.Sprintf("工具调用失败: %v", tcErr)
			}
			log.Printf("[agent] tool_call[%d] %s %s => %s", i, tc.Method, tc.Path, truncateForLog(result, 200))
			resultParts = append(resultParts, fmt.Sprintf("[TOOL_RESULT %d]\n%s\n[/TOOL_RESULT]", i, result))
		}

		// 把工具结果作为 human 消息喂回给 LLM
		feedbackMsg := strings.Join(resultParts, "\n\n") +
			"\n\n请根据以上工具调用结果，用自然语言向用户汇报操作结果，不要再输出 [TOOL_CALL] 块。"
		messages = append(messages, llms.MessageContent{
			Role:  llms.ChatMessageTypeHuman,
			Parts: []llms.ContentPart{llms.TextContent{Text: feedbackMsg}},
		})
	}

	// 理论上不会走到这里
	return "", nil, fmt.Errorf("工具调用超过最大轮数 %d", maxToolCallRounds)
}

func truncateForLog(s string, max int) string {
	s = strings.TrimSpace(s)
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}


func buildMessages(systemPrompt, userMessage string, history []struct{ Role, Content string }) []llms.MessageContent {
	messages := make([]llms.MessageContent, 0, len(history)+3)
	messages = append(messages, llms.MessageContent{
		Role:  llms.ChatMessageTypeSystem,
		Parts: []llms.ContentPart{llms.TextContent{Text: systemPrompt}},
	})
	for _, h := range history {
		role := llms.ChatMessageTypeHuman
		if h.Role == "assistant" || h.Role == "ai" {
			role = llms.ChatMessageTypeAI
		}
		messages = append(messages, llms.MessageContent{
			Role:  role,
			Parts: []llms.ContentPart{llms.TextContent{Text: h.Content}},
		})
	}
	messages = append(messages, llms.MessageContent{
		Role:  llms.ChatMessageTypeHuman,
		Parts: []llms.ContentPart{llms.TextContent{Text: userMessage}},
	})
	return messages
}

var formFillHeadRe = regexp.MustCompile(`\[FORM_FILL:([^\]]+)\]\s*\n`)

func parseFormFill(reply string) *FormFillAction {
	idx := strings.Index(reply, formFillPrefix)
	if idx < 0 {
		return nil
	}
	rest := reply[idx:]
	loc := formFillHeadRe.FindStringIndex(rest)
	if loc == nil {
		return nil
	}
	pageForm := strings.TrimSpace(formFillHeadRe.FindStringSubmatch(rest)[1])
	jsonStart := loc[1]
	// 找第一个 {，再匹配到对应的 }
	brace := strings.Index(rest[jsonStart:], "{")
	if brace < 0 {
		return nil
	}
	start := jsonStart + brace
	depth := 0
	for i := start; i < len(rest); i++ {
		switch rest[i] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				payloadStr := rest[start : i+1]
				var payload json.RawMessage
				if err := json.Unmarshal([]byte(payloadStr), &payload); err != nil {
					return nil
				}
				parts := strings.SplitN(pageForm, ".", 2)
				page := pageForm
				form := ""
				if len(parts) == 2 {
					page = strings.TrimSpace(parts[0])
					form = strings.TrimSpace(parts[1])
				}
				return &FormFillAction{Page: page, Form: form, Payload: payload}
			}
		}
	}
	return nil
}

// summarizeConversation 使用 LLM 将一段会话消息压缩为简短摘要。
// previousSummary 为空时表示首次摘要；否则表示在已有摘要基础上合并新增轮次。
func summarizeConversation(ctx context.Context, previousSummary string, msgs []ConversationMessage) (string, error) {
	model, err := getLLM()
	if err != nil {
		return "", err
	}

	var b strings.Builder
	if strings.TrimSpace(previousSummary) != "" {
		b.WriteString("下面是当前会话的已有摘要：\n")
		b.WriteString(strings.TrimSpace(previousSummary))
		b.WriteString("\n\n")
	}
	b.WriteString("下面是本次需要合并进摘要的对话轮次（按时间顺序）：\n")
	currentSize := 0
	for _, m := range msgs {
		role := "用户"
		if m.Role != "" && m.Role != "user" {
			role = "助手"
		}
		content := strings.TrimSpace(m.Content)
		if content == "" {
			continue
		}
		if len(content) > 800 {
			content = content[:800] + "…"
		}
		line := "[" + role + "] " + content + "\n"
		if currentSize+len(line) > maxSummarizeInputSize {
			break
		}
		b.WriteString(line)
		currentSize += len(line)
	}
	if currentSize == 0 && strings.TrimSpace(previousSummary) != "" {
		// 没有新增内容时，直接返回已有摘要
		return strings.TrimSpace(previousSummary), nil
	}

	b.WriteString("\n请你用简洁的中文，将上述内容整合为一个新的会话摘要：")
	b.WriteString("\n- 保留关键事实、任务目标、已完成的步骤、重要约束；")
	b.WriteString("\n- 不要逐句复述，也不要包含无关寒暄；")
	b.WriteString("\n- 不超过约 800 字；")
	b.WriteString("\n- 直接输出摘要正文即可，不要再解释你的行为。\n")

	prompt := b.String()

	messages := []llms.MessageContent{
		{
			Role: llms.ChatMessageTypeSystem,
			Parts: []llms.ContentPart{
				llms.TextContent{Text: "你是一个专业的对话摘要助手，会用简洁清晰的中文总结对话。"},
			},
		},
		{
			Role: llms.ChatMessageTypeHuman,
			Parts: []llms.ContentPart{
				llms.TextContent{Text: prompt},
			},
		},
	}

	var full strings.Builder
	streamFunc := func(ctx context.Context, chunk []byte) error {
		full.Write(chunk)
		return nil
	}
	opts := []llms.CallOption{llms.WithStreamingFunc(streamFunc)}
	_, err = model.GenerateContent(ctx, messages, opts...)
	if err != nil {
		// 返回已生成的部分，尽量不让调用方完全失去摘要
		return strings.TrimSpace(full.String()), err
	}
	return strings.TrimSpace(full.String()), nil
}

