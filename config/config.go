package config

import (
	"github.com/spf13/viper"
	"log"
)

func InitConfig() {
	viper.SetConfigFile("config/config.yaml")
	err := viper.ReadInConfig()
	if err != nil {
		log.Fatalf("Config load failed: %v", err)
	}
	log.Println("✅ Config loaded")
}

func GetServerPort() string {
	return viper.GetString("server.port")
}

func GetJWTSecret() string {
	return viper.GetString("jwt.secret")
}
