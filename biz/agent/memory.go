package agent

import (
	"context"
	"strings"
)

// SimpleTurn 是传给 LLM 的精简对话单元。
type SimpleTurn struct {
	Role    string
	Content string
}

// 压缩相关的经验阈值（字符级粗估，避免强依赖具体 token 计算）。
const (
	maxHistoryChars       = 12000 // 超过则触发摘要
	recentMessageKeep     = 16    // 始终保留的最近消息条数（user/assistant 混合条数）
	maxSummaryChars       = 2000  // 摘要本身的最大长度
	maxSummarizeInputSize = 6000  // 传给摘要模型的旧消息文本上限
)

// CompressSessionHistory 对单个会话执行“短期记忆压缩”：
// - 若总长度未超过阈值且无历史摘要，则返回完整消息列表；
// - 否则，将较早的消息压缩进 Summary，仅保留最近若干条完整消息。
//
// 返回值：
// - summary: 更新后的会话摘要（可能为空）
// - history: 发送给 LLM 的精简 history（不含本轮用户消息）
// - changed: 是否修改了 sess.Summary（调用方可据此决定是否持久化会话）
func CompressSessionHistory(ctx context.Context, sess *Conversation) (summary string, history []SimpleTurn, changed bool, err error) {
	msgs := sess.Messages
	if len(msgs) == 0 {
		return strings.TrimSpace(sess.Summary), nil, false, nil
	}

	// 粗略估算当前会话文本总长度
	total := 0
	for _, m := range msgs {
		total += len(m.Content)
	}

	// 未超过阈值且没有历史摘要时，直接返回全部消息
	if total <= maxHistoryChars && strings.TrimSpace(sess.Summary) == "" {
		return "", convertMessagesToTurns(msgs), false, nil
	}

	// 需要触发摘要：保留最近 recentMessageKeep 条，其余合并进摘要
	keep := recentMessageKeep
	if keep > len(msgs) {
		keep = len(msgs)
	}
	cut := len(msgs) - keep
	if cut < 0 {
		cut = 0
	}
	oldPart := msgs[:cut]
	recentPart := msgs[cut:]

	// 基于已存在的 Summary + 旧消息生成新的摘要
	newSummary, sumErr := summarizeConversation(ctx, sess.Summary, oldPart)
	if sumErr != nil {
		// 摘要失败时，为了稳妥起见，退回到完整消息，不贸然丢信息
		return strings.TrimSpace(sess.Summary), convertMessagesToTurns(msgs), false, sumErr
	}

	newSummary = strings.TrimSpace(newSummary)
	if len(newSummary) > maxSummaryChars {
		newSummary = newSummary[:maxSummaryChars] + "…"
	}
	if newSummary != strings.TrimSpace(sess.Summary) {
		sess.Summary = newSummary
		changed = true
	}

	return sess.Summary, convertMessagesToTurns(recentPart), changed, nil
}

func convertMessagesToTurns(msgs []ConversationMessage) []SimpleTurn {
	out := make([]SimpleTurn, 0, len(msgs))
	for _, m := range msgs {
		out = append(out, SimpleTurn{
			Role:    m.Role,
			Content: m.Content,
		})
	}
	return out
}

