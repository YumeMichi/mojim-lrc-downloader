//
// Copyright (C) 2022 YumeMichi
//
// SPDX-License-Identifier: Apache-2.0
//

package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/YumeMichi/mojim-lrc-downloader/utils"

	"gopkg.in/yaml.v3"
)

type ProxyConfig struct {
	Enabled  bool   `yaml:"enabled"`
	Protocol string `yaml:"protocol"`
	Ip       string `yaml:"ip"`
	Port     string `yaml:"port"`
}

type LrcConfig struct {
	LoadFileName string `yaml:"load_file_name"`
	SaveFileName string `yaml:"save_file_name"`
}

type Config struct {
	Proxy *ProxyConfig `yaml:"proxy"`
	Lrc   *LrcConfig   `yaml:"lrc"`
}

func DefaultConfigs() *Config {
	return &Config{
		Proxy: &ProxyConfig{
			Enabled:  false,
			Protocol: "socks5",
			Ip:       "127.0.0.1",
			Port:     "1080",
		},
		Lrc: &LrcConfig{
			LoadFileName: "lrc-in.txt",
			SaveFileName: "lrc-out.txt",
		},
	}
}

func Load(p string) *Config {
	if !utils.PathExists(p) {
		_ = DefaultConfigs().Save(p)
	}
	c := Config{}
	err := yaml.Unmarshal([]byte(utils.ReadAllText(p)), &c)
	if err != nil {
		fmt.Println("尝试加载配置文件失败: 读取文件失败！")
		fmt.Println("原配置文件已备份！")
		_ = os.Rename(p, p+".backup"+strconv.FormatInt(time.Now().Unix(), 10))
		_ = DefaultConfigs().Save(p)
	}
	c = Config{}
	_ = yaml.Unmarshal([]byte(utils.ReadAllText(p)), &c)
	// xlog.Info("配置加载完毕！")
	return &c
}

func (c *Config) Save(p string) error {
	data, err := yaml.Marshal(c)
	if err != nil {
		fmt.Println("写入新的配置文件失败！")
		return err
	}
	utils.WriteAllText(p, string(data))
	return nil
}
