package main

import (
	"flag"

	_ "net/http/pprof"

	"github.com/uopensail/example-service/boot"
	"github.com/uopensail/example-service/config"
	"github.com/uopensail/example-service/services"
)

func main() {
	configFilePath := flag.String("config", "conf/local/config.toml", "启动命令请设置配置文件目录")
	logDir := flag.String("log", "./logs", "启动命令请设置seelog.xml")
	flag.Parse()

	config.AppConfigIns.Init(*configFilePath)

	srv := services.NewServices()
	boot.Load(config.AppConfigIns.ServerConfig, *logDir, srv)

}
