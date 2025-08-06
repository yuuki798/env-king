package dockerhelper

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"runtime"
	"sync"
)

type RegistryHandler struct {
	configPath string

	PublicUrlDic map[string]bool // 记录公开镜像站
	MuPublicUrl  sync.Mutex

	// todo 这里可以添加用户自定义的镜像站
	//UserUrlDic map[string]bool // 记录用户添加的镜像站，应该是从本地配置文件中读取的
	//MuUserUrl  sync.Mutex

	UrlList []string // 最终的镜像站列表

	SpeedHandler *SpeedHandler // 用于获取速度排序
}

func NewRegistryHandler(os string) *RegistryHandler {
	var configPath string
	switch os {
	case "linux":
		configPath = "/etc/docker/daemon.json" // 默认配置文件路
	case "windows":
		configPath = "C:\\ProgramData\\docker\\config\\daemon.json" // Windows默认配置文件路径
	case "darwin":
		configPath = ""
	}
	return &RegistryHandler{
		configPath:   configPath,
		PublicUrlDic: make(map[string]bool),
		//UserUrlDic:   make(map[string]bool),
		SpeedHandler: NewSpeedHandler(),
	}

}

func (this *RegistryHandler) Do() bool {
	var myUrls []string
	// 测速并生成排序后的镜像站列表
	myUrls = this.generateSortedTargetUrlList()
	// 写入配置
	ok := this.writeListIntoConfigFile(myUrls)
	if !ok {
		log.Println("Error writing registry mirrors to config file.")
		return false
	}
	var cmd *exec.Cmd
	// darwin是用户手动配置因此直接退出
	if runtime.GOOS == "darwin" {
		return true
	} else if runtime.GOOS == "windows" {
		cmd = exec.Command("powershell", "-Command", "Restart-Service docker")
	} else if runtime.GOOS == "linux" {
		cmd = exec.Command("systemctl", "restart", "docker")
	} else {
		log.Println("Unsupported OS:", runtime.GOOS)
		return false
	}
	// 重启Docker服务以应用配置
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		log.Println("Error restarting Docker service:", err)
		return false
	}
	log.Println("Docker registry mirrors updated successfully.")
	return true
}

func (this *RegistryHandler) writeListIntoConfigFile(myUrls []string) bool {
	var config map[string][]string
	//找到registry-mirrors
	// 读取配置文件内容
	content, err := os.ReadFile(this.configPath)
	if err != nil && !os.IsNotExist(err) {
		log.Println("Error reading config file:", err)
		return false
	}

	if len(content) > 0 {
		err = json.Unmarshal(content, &config)
		if err != nil {
			log.Println("Error unmarshalling config file:", err)
			return false
		}
	} else {
		config = make(map[string][]string)
	}

	// 替换registry-mirrors
	config["registry-mirrors"] = myUrls

	newContent, err := json.MarshalIndent(config, "", "	")
	if err != nil {
		log.Println("Error marshalling config to JSON:", err)
		return false
	}
	// 排除mac情况，这个需要用户手动复制
	if runtime.GOOS == "darwin" {
		fmt.Println(string(newContent))
		log.Println("Please copy the above content to your docker desktop settings.")
		return true
	}
	// 先检查是否存在路径
	if _, err := os.Stat(this.configPath); os.IsNotExist(err) {
		// 如果不存在，创建目录
		err = os.MkdirAll(this.configPath[:len(this.configPath)-len("daemon.json")], 0755) // 确保目录存在
		if err != nil {
			log.Println("Error creating directory:", err)
			return false
		}
	}
	// 写入配置文件
	err = os.WriteFile(this.configPath, newContent, 0644)
	if err != nil {
		log.Println("Error writing config file:", err)
		return false
	}
	return true
}

func (this *RegistryHandler) generateSortedTargetUrlList() []string {
	// 获取地址列表
	var myUrls []string
	// 应该使用speed_handler来获取速度排序
	ok := this.SpeedHandler.FetchList() //todo 这里可以穿进去用户自己配置的urls
	if !ok {
		log.Println("Error fetching speed list.")
		return myUrls
	}
	this.SpeedHandler.SpeedTest()
	this.SpeedHandler.LockUrl2Duration.Lock()
	for _, url2Duration := range this.SpeedHandler.Url2Duration {
		if url2Duration.Duration > 0 && url2Duration.Duration < 10000 {
			myUrls = append(myUrls, url2Duration.Url)
		}
	}
	this.SpeedHandler.LockUrl2Duration.Unlock()
	return myUrls
}
