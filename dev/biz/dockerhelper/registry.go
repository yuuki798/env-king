package dockerhelper

import "sync"

// 目前支持公开访问的镜像站
type RegistryStore struct {
	UrlDic map[string]bool
	Speed  map[string]int // 记录每个镜像站的速度
	mu     sync.Mutex
}
