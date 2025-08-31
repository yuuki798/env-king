package initial

import (
	"flag"
	"github.com/spf13/viper"
	"log"
)

func InitConfig() {
	// 设置命令行参数
	var configFile string
	flag.StringVar(&configFile, "config", "./config.dev.yaml", "配置文件路径")
	flag.Parse()
	if configFile == "config.dev.yaml" {
		log.Println("使用默认配置文件: config.dev.yaml")
	} else {
		log.Printf("使用配置文件: %s", configFile)
	}
	// 设置viper配置
	viper.SetConfigFile(configFile)
	// 设置配置文件类型
	viper.SetConfigType("yaml")
	// 设置配置文件的搜索路径
	viper.AddConfigPath(".") // 当前目录
	if err := viper.ReadInConfig(); err != nil {
		log.Fatalf("viper read config failed: %s", err)
	}
	log.Println("viper read config success")
}
