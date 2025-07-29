package dockerhelper

import "sync"

// 目前支持公开访问的镜像站
type RegistryStore struct {
	PublicUrlDic map[string]bool // 记录公开镜像站
	MuPublicUrl  sync.Mutex

	UserUrlDic map[string]bool // 记录用户添加的镜像站
	MuUserUrl  sync.Mutex

	SpeedDic map[string]int // 记录每个镜像站的速度
	MuSpeed  sync.Mutex
}
