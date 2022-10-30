package main

import (
	"apigw/internal/logger"
	"apigw/internal/srvhttp"
	"github.com/spf13/viper"
)

func main() {

	viper.AddConfigPath("./config/")
	viper.AddConfigPath("/etc/apigw/")
	viper.AddConfigPath("/usr/local/etc/apigw/")
	viper.SetConfigName("config")
	viper.SetConfigType("json")
	viper.ReadInConfig()
	logger.InitLog()

	logger.LogMsg("Starting process", "info")
	logger.LogMsg("Read configfile config.json", "info")

	srvhttp.HandleRequests()

}
