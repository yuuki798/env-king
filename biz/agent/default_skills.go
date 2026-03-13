package agent

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"yuuki798/env-king/infra"
)

const (
	agentConfigBucket   = "config"
	agentConfigKeySkills = "default_skills"
)

var (
	agentStoreOnce sync.Once
	agentStore     *infra.Store
	agentStoreErr  error
)

// getAgentStore 打开 agent 配置用 bbolt（workdir/store/agent.db），用于默认技能等。
func getAgentStore() (*infra.Store, error) {
	agentStoreOnce.Do(func() {
		base := resolveWorkspaceDir()
		dir := filepath.Join(base, "store")
		if err := os.MkdirAll(dir, 0o755); err != nil {
			agentStoreErr = fmt.Errorf("创建 agent store 目录失败: %w", err)
			return
		}
		agentStore, agentStoreErr = infra.OpenStore(filepath.Join(dir, "agent.db"), nil)
	})
	return agentStore, agentStoreErr
}

// GetDefaultSkillIDs 返回持久化的默认技能 ID 列表；无配置或出错时返回 nil。
func GetDefaultSkillIDs() ([]string, error) {
	st, err := getAgentStore()
	if err != nil {
		return nil, err
	}
	raw, err := st.GetString(agentConfigBucket, agentConfigKeySkills)
	if err != nil || len(raw) == 0 {
		return nil, err
	}
	var ids []string
	if err := json.Unmarshal(raw, &ids); err != nil {
		return nil, err
	}
	// 去空、去重保持顺序
	seen := make(map[string]struct{})
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out, nil
}

// SetDefaultSkillIDs 将默认技能 ID 列表持久化到 bbolt。
func SetDefaultSkillIDs(ids []string) error {
	st, err := getAgentStore()
	if err != nil {
		return err
	}
	cleaned := make([]string, 0, len(ids))
	seen := make(map[string]struct{})
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		cleaned = append(cleaned, id)
	}
	raw, err := json.Marshal(cleaned)
	if err != nil {
		return err
	}
	return st.PutString(agentConfigBucket, agentConfigKeySkills, raw)
}
