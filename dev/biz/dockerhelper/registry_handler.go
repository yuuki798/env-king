package dockerhelper

type RegistryHandler struct {
	RegistryStore *RegistryStore
}

func (this *RegistryHandler) GetRegistryList() map[string]bool {
	this.RegistryStore.mu.Lock()
	defer this.RegistryStore.mu.Unlock()
	return this.RegistryStore.UrlDic
}
func (this *RegistryHandler) AddRegistry(url string) {
	this.RegistryStore.mu.Lock()
	defer this.RegistryStore.mu.Unlock()
	this.RegistryStore.UrlDic[url] = true
}
func (this *RegistryHandler) RemoveRegistry(url string) {
	this.RegistryStore.mu.Lock()
	defer this.RegistryStore.mu.Unlock()
	delete(this.RegistryStore.UrlDic, url)
}
