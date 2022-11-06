package main

import (
	_ "apigw/internal/globalvar"
	"apigw/internal/logger"
	"apigw/internal/srvhttp"
	"fmt"
	"github.com/spf13/viper"
	"os"
)

var version = "0.3.0"

func main() {

	viper.AddConfigPath("./config/")
	viper.AddConfigPath("/etc/apigw/")
	viper.AddConfigPath("/usr/local/etc/apigw/")
	viper.SetConfigName("config")
	viper.SetConfigType("json")
	err := viper.ReadInConfig()
	if err != nil {
		fmt.Println("Sorry no configuration file were found in one of the following directory : " +
			" - ./config" +
			" - /etc/apigw" +
			" - /usr/local/etc/apigw",
		)
		os.Exit(1)
	}
	logger.InitLog()

	logger.LogMsg("Starting process", "info")
	logger.LogMsg("Read configfile config.json", "info")

	srvhttp.HandleRequests()

}
