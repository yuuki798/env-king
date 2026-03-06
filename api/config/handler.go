package config

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
)

func Register(r *gin.RouterGroup) {
	g := r.Group("/config")
	g.GET("", getConfig)
	g.PUT("", putConfig)
}

// getConfig 返回当前全部配置（viper 内存 + 文件），即刻反映最新值。
func getConfig(c *gin.Context) {
	m := viper.AllSettings()
	if m == nil {
		m = make(map[string]interface{})
	}
	c.JSON(http.StatusOK, m)
}

// putConfig 接受局部或完整配置（嵌套 JSON），合并进 viper 并即刻生效，并写回配置文件。
func putConfig(c *gin.Context) {
	var body map[string]interface{}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json: " + err.Error()})
		return
	}
	flat := flatten("", body)
	for k, v := range flat {
		viper.Set(k, v)
	}
	if err := viper.WriteConfig(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "write config: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// flatten 将嵌套 map 压平为 dot 键，如 "github.token" -> value。
func flatten(prefix string, m map[string]interface{}) map[string]interface{} {
	out := make(map[string]interface{})
	for k, v := range m {
		key := k
		if prefix != "" {
			key = prefix + "." + k
		}
		switch val := v.(type) {
		case map[string]interface{}:
			for k2, v2 := range flatten(key, val) {
				out[k2] = v2
			}
		default:
			out[key] = v
		}
	}
	return out
}
