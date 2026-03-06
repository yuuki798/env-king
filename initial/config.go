package initial

import (
	"flag"
	"log"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

func InitConfig() {
	// 设置命令行参数
	var configFile string
	var mode string
	flag.StringVar(&configFile, "config", "", "配置文件路径，未指定时按 mode 选择")
	flag.StringVar(&mode, "mode", "dev", "运行模式: dev|prod，未指定 config 时使用 config.{mode}.yaml")
	flag.Parse()

	if configFile == "" {
		switch mode {
		case "prod":
			configFile = "./config.prod.yaml"
		default:
			configFile = "./config.dev.yaml"
		}
	}
	log.Printf("mode=%s 使用配置文件: %s", mode, configFile)
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

	// 热加载：监听配置文件变更并重新读取
	viper.WatchConfig()
	viper.OnConfigChange(func(_ fsnotify.Event) {
		log.Println("config file changed, reloading")
		if err := viper.ReadInConfig(); err != nil {
			log.Printf("config reload failed: %v", err)
			return
		}
		log.Println("config reload success")
	})
}
