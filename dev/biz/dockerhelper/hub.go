package dockerhelper

import (
	"runtime"
	"sync"
)

type Hub struct {
	mode string //可以用于读取状态
	os   string // 操作系统类型

	//applying bool   // 模式是否已经应用
	mu sync.Mutex

	speedHandler    *SpeedHandler
	registryHandler *RegistryHandler
	//ProxyHandler    *ProxyHandler
}

func NewHub() *Hub {
	return &Hub{
		mode:            "mirror", // 默认镜像站模式
		os:              runtime.GOOS,
		speedHandler:    NewSpeedHandler(),
		registryHandler: NewRegistryHandler(runtime.GOOS),
		//ProxyHandler:    NewProxyHandler(),
	}
}

func (this *Hub) DoMirrorMode() bool {
	this.mu.Lock()
	this.mode = "mirror"
	this.mu.Unlock()
	return this.registryHandler.Do()
}

func (this *Hub) SpeedTest() (bool, []Url2Duration) {
	this.mu.Lock()
	defer this.mu.Unlock()

	if !this.speedHandler.FetchList() {
		return false, nil
	}
	this.speedHandler.SpeedTest()
	return true, this.speedHandler.Url2Duration
}
