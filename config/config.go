package config

import (
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
	"github.com/uopensail/ulib/commonconfig"
)

type AppConfig struct {
	commonconfig.ServerConfig `json:"server" toml:"server" yaml:"server"`
}

var AppConfigIns AppConfig

func (conf *AppConfig) Init(filePath string) {
	fData, err := os.ReadFile(filePath)
	if err != nil {
		fmt.Errorf("ioutil.ReadFile error: %s", err)
		panic(err)
	}
	_, err = toml.Decode(string(fData), conf)
	if err != nil {
		fmt.Errorf("Unmarshal error: %s", err)
		panic(err)
	}
	fmt.Printf("InitAppConfig:%v yaml:%s\n", conf, string(fData))
}
