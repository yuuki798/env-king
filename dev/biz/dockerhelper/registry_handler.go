package dockerhelper

import (
	"encoding/json"
	"log"
	"os"
	"os/exec"
	"sort"
)

type RegistryHandler struct {
	RegistryStore   *RegistryStore
	ApplyUserList   bool
	ApplyPublicList bool
	configPath      string
}

func (this *RegistryHandler) Do() bool {
	var myUrls []string
	myUrls = this.generateSortedTargetUrlList()
	ok := this.writeListIntoConfigFile(myUrls)
	if !ok {
		log.Println("Error writing registry mirrors to config file.")
		return false
	}
	cmd := exec.Command("systemctl", "restart", "docker")
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
	if this.ApplyPublicList {
		this.RegistryStore.MuPublicUrl.Lock()
		for url := range this.RegistryStore.PublicUrlDic {
			myUrls = append(myUrls, url)
		}
		this.RegistryStore.MuPublicUrl.Unlock()
	}
	if this.ApplyUserList {
		this.RegistryStore.MuUserUrl.Lock()
		for url := range this.RegistryStore.UserUrlDic {
			myUrls = append(myUrls, url)
		}
		this.RegistryStore.MuUserUrl.Unlock()
	}

	// 按照speed排序
	this.RegistryStore.MuSpeed.Lock()
	sort.Slice(myUrls, func(i, j int) bool {
		if _, ok := this.RegistryStore.SpeedDic[myUrls[i]]; !ok {
			return false // 如果没有速度数据，认为速度较慢
		}
		if _, ok := this.RegistryStore.SpeedDic[myUrls[j]]; !ok {
			return true // 如果没有速度数据，认为速度较慢
		}
		return this.RegistryStore.SpeedDic[myUrls[i]] > this.RegistryStore.SpeedDic[myUrls[j]]
	})
	this.RegistryStore.MuSpeed.Unlock()
	return myUrls
}
