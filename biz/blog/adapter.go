package blog

// PlatformAdapter 定义平台适配器接口
type PlatformAdapter interface {
	Auth() error
	// SaveToDraft is same as commit to local a storage
	SaveToDraft() error
	PreProcess() error
	Publish() error
	GetList() error
	Name() string
}
