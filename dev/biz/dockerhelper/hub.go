package dockerhelper

import "sync"

type Hub struct {
	mode            string //镜像站模式或代理模式
	applying        bool   // 模式是否已经应用
	mu              sync.Mutex
	speedHandler    *SpeedHandler
	registryHandler *RegistryHandler
	ProxyHandler    *ProxyHandler
}

func (this *Hub) SetMode(mode string) bool {
	//分为镜像站模式和代理模式
	this.mu.Lock()
	defer this.mu.Unlock()
	switch mode {
	case "mirror":
		this.mode = "mirror" // 镜像站模式
	case "proxy":
		this.mode = "proxy" // 代理模式
	default:
		return false // 无效模式
	}
	return true
}

func (this *Hub) Undo() {

}
func (this *Hub) Do() {

}
