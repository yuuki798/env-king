package mihomo

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"go.yaml.in/yaml/v3"
)

// ControllerInfo 从 config 解析出的 external-controller 与 secret
type ControllerInfo struct {
	BaseURL string // http://127.0.0.1:9090
	Secret  string
}

// GetControllerInfo 从当前 config 读取 external-controller 和 secret
func GetControllerInfo() (*ControllerInfo, error) {
	data, err := GetConfig()
	if err != nil {
		return nil, err
	}
	var raw map[string]interface{}
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	ec, _ := raw["external-controller"].(string)
	ec = strings.TrimSpace(ec)
	if ec == "" {
		return nil, nil
	}
	if !strings.HasPrefix(ec, "http") {
		ec = "http://" + ec
	}
	secret, _ := raw["secret"].(string)
	secret = strings.TrimSpace(secret)
	return &ControllerInfo{BaseURL: ec, Secret: secret}, nil
}

// ProxyRequest 向 mihomo 控制接口发请求（GET 或 PUT）
func ProxyRequest(method, path string, body interface{}) ([]byte, int, error) {
	info, err := GetControllerInfo()
	if err != nil || info == nil {
		return nil, 0, fmt.Errorf("未配置 external-controller，请在 config 中设置 external-controller: 127.0.0.1:9090")
	}
	url := strings.TrimSuffix(info.BaseURL, "/") + path
	var reqBody io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		reqBody = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	if info.Secret != "" {
		req.Header.Set("Authorization", "Bearer "+info.Secret)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, err
	}
	return data, resp.StatusCode, nil
}
