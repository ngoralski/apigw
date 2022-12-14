package main

import (
	"apigw/internal/globalvar"
	_ "apigw/internal/globalvar"
	"apigw/internal/logger"
	"apigw/internal/srvhttp"
	"flag"
	"fmt"
	"github.com/spf13/viper"
	"os"
)

//	@title			API GW documentation
//	@version		1.0
//	@description	This is a small documentation about API GW documentation
//	@termsOfService	http://swagger.io/terms/

//	@host		localhost:8080
//	@BasePath	/

var softVersion = "0.3.1"

func main() {

	extraConfigPath := flag.String("config", "./", "Configuration folder location")
	version := flag.Bool("version", false, "Display software version")

	flag.Parse()

	viper.AddConfigPath("./config/")
	viper.AddConfigPath("/etc/apigw/")
	viper.AddConfigPath("/usr/local/etc/apigw/")
	viper.SetConfigName("config")
	viper.SetConfigType("json")
	viper.AddConfigPath(*extraConfigPath)

	if *version {
		fmt.Printf("Version %s\n", softVersion)
		os.Exit(1)
	}

	err := viper.ReadInConfig()
	if err != nil {
		msg := fmt.Sprintf(`Sorry no configuration file were found in one of the following directory : 
			- ./config,
			- /etc/apigw
			- /usr/local/etc/apigw
			- %s`, *extraConfigPath)
		fmt.Println(msg)
		os.Exit(1)
	}
	logger.InitLog()

	globalvar.SetupGoGuardian()

	//threads := int(viper.Get("threads").(float64))
	logger.LogMsg("Starting process", "info")
	logger.LogMsg("Read configfile config.json", "info")

	//alarm := make(chan struct{})
	//for i := 0; i < threads; i++ {
	//	go srvhttp.HandleRequests()
	//}

	srvhttp.HandleRequests()

}
