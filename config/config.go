package config

import (
	"github.com/spf13/viper"
	"log"
)

// ✅ 确保 `ConfigStruct` 结构体只定义一次
type ConfigStruct struct {
	Ansible struct {
		PlaybookDir       string   `mapstructure:"playbook_dir"`
		InventoryDir      string   `mapstructure:"inventory_dir"`
		AllowedExtensions []string `mapstructure:"allowed_extensions"`
	}
}

var Conf ConfigStruct

func InitConfig() {
	viper.SetConfigFile("config/config.yaml") // 指定配置文件路径
	viper.AutomaticEnv()                      // 允许使用环境变量覆盖配置

	if err := viper.ReadInConfig(); err != nil {
		log.Fatalf("❌ 配置文件加载失败: %v", err)
	}

	// ✅ 确保 `playbook_dir` 存在
	if !viper.IsSet("ansible.playbook_dir") {
		log.Fatalf("❌ 配置文件中缺少 `ansible.playbook_dir`，请检查 config.yaml")
	}

	// ✅ 解析 `config.yaml`
	if err := viper.Unmarshal(&Conf); err != nil {
		log.Fatalf("❌ 配置解析失败: %v", err)
	}

	log.Println("✅ 配置加载成功: PlaybookDir =", Conf.Ansible.PlaybookDir)
}

func GetServerPort() string {
	return viper.GetString("server.port")
}

func GetJWTSecret() string {
	return viper.GetString("jwt.secret")
}
