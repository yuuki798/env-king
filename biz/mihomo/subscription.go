package mihomo

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"go.yaml.in/yaml/v3"
)

const subscriptionsFileName = "subscriptions.json"
const defaultProxyGroupName = "订阅"

// Subscription 订阅链接
type Subscription struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	URL  string `json:"url"`
}

func subscriptionsPath() (string, error) {
	wd, err := getWorkDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(wd, subscriptionsFileName), nil
}

// ListSubscriptions 返回所有订阅
func ListSubscriptions() ([]Subscription, error) {
	p, err := subscriptionsPath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(p)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var list []Subscription
	if err := json.Unmarshal(data, &list); err != nil {
		return nil, err
	}
	return list, nil
}

// AddSubscription 添加订阅
func AddSubscription(name, url string) (Subscription, error) {
	name = strings.TrimSpace(name)
	url = strings.TrimSpace(url)
	if name == "" || url == "" {
		return Subscription{}, os.ErrInvalid
	}
	list, err := ListSubscriptions()
	if err != nil {
		return Subscription{}, err
	}
	sub := Subscription{ID: uuid.Must(uuid.NewRandom()).String(), Name: name, URL: url}
	list = append(list, sub)
	if err := writeSubscriptions(list); err != nil {
		return Subscription{}, err
	}
	return sub, nil
}

// DeleteSubscription 删除订阅
func DeleteSubscription(id string) error {
	list, err := ListSubscriptions()
	if err != nil {
		return err
	}
	for i, s := range list {
		if s.ID == id {
			list = append(list[:i], list[i+1:]...)
			return writeSubscriptions(list)
		}
	}
	return os.ErrNotExist
}

func writeSubscriptions(list []Subscription) error {
	p, err := subscriptionsPath()
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p, data, 0644)
}

// safeProviderName 生成可作为 proxy-provider 键名的名称（唯一且合法）
func safeProviderName(id string) string {
	if len(id) >= 8 {
		return "sub_" + strings.ReplaceAll(id[:8], "-", "")
	}
	return "sub_" + id
}

// ApplySubscriptionsToConfig 将订阅列表写入 config：添加 proxy-providers 与「订阅」选择组
func ApplySubscriptionsToConfig() error {
	wd, err := getWorkDir()
	if err != nil {
		return err
	}
	list, err := ListSubscriptions()
	if err != nil {
		return err
	}
	cfgPath, err := configPath()
	if err != nil {
		return err
	}
	data, err := os.ReadFile(cfgPath)
	if err != nil {
		return err
	}
	var raw map[string]interface{}
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return err
	}
	// proxy-providers
	providers, _ := raw["proxy-providers"].(map[string]interface{})
	if providers == nil {
		providers = make(map[string]interface{})
	}
	providerNames := make([]string, 0, len(list))
	for _, sub := range list {
		key := safeProviderName(sub.ID)
		providerNames = append(providerNames, key)
		providers[key] = map[string]interface{}{
			"type":     "http",
			"url":      sub.URL,
			"path":     "./providers/" + key + ".yaml",
			"interval": 3600,
			"health-check": map[string]interface{}{
				"enable":         true,
				"url":            "https://www.gstatic.com/generate_204",
				"interval":      300,
				"timeout":        5000,
				"expected-status": 204,
			},
		}
	}
	raw["proxy-providers"] = providers
	// proxy-groups: 添加或更新「订阅」选择组
	groups, _ := raw["proxy-groups"].([]interface{})
	if groups == nil {
		groups = []interface{}{}
	}
	// use 字段引用 proxy-provider（节点来自订阅），不追加 DIRECT/REJECT
	useList := make([]interface{}, 0, len(providerNames))
	for _, n := range providerNames {
		useList = append(useList, n)
	}
	subGroup := map[string]interface{}{
		"name":  defaultProxyGroupName,
		"type":  "select",
		"use":   useList,
	}
	found := false
	for i, g := range groups {
		gm, ok := g.(map[string]interface{})
		if !ok {
			continue
		}
		if n, _ := gm["name"].(string); n == defaultProxyGroupName {
			groups[i] = subGroup
			found = true
			break
		}
	}
	if !found {
		groups = append([]interface{}{subGroup}, groups...)
	}
	raw["proxy-groups"] = groups

	// 确保 mode 存在，默认 rule
	if _, ok := raw["mode"]; !ok {
		raw["mode"] = "rule"
	}

	// 如果没有 rules，写入默认规则（让「订阅」组兜底）
	if existing, ok := raw["rules"]; !ok || existing == nil {
		raw["rules"] = []interface{}{
			"GEOIP,CN,DIRECT",
			"MATCH," + defaultProxyGroupName,
		}
	}

	out, err := yaml.Marshal(raw)
	if err != nil {
		return err
	}
	providersDir := filepath.Join(wd, "providers")
	_ = os.MkdirAll(providersDir, 0755)
	return os.WriteFile(cfgPath, out, 0644)
}

// EnsureProviderFiles 确保所有 proxy-provider 的 path 文件存在，否则写入空 proxies 以便 mihomo 能启动
func EnsureProviderFiles() error {
	wd, err := getWorkDir()
	if err != nil {
		return err
	}
	data, err := GetConfig()
	if err != nil {
		return err
	}
	var raw map[string]interface{}
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return err
	}
	providers, _ := raw["proxy-providers"].(map[string]interface{})
	if providers == nil {
		return nil
	}
	providersDir := filepath.Join(wd, "providers")
	_ = os.MkdirAll(providersDir, 0755)
	emptyProxies := []byte("proxies: []\n")
	for _, p := range providers {
		pm, ok := p.(map[string]interface{})
		if !ok {
			continue
		}
		pathVal, _ := pm["path"].(string)
		if pathVal == "" {
			continue
		}
		// path 为 ./providers/xxx.yaml，转为绝对路径
		fullPath := filepath.Join(wd, pathVal)
		if _, err := os.Stat(fullPath); err == nil {
			continue
		}
		_ = os.WriteFile(fullPath, emptyProxies, 0644)
	}
	return nil
}
