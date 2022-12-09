package srvhttp

import (
	_ "apigw/docs"
	"apigw/internal/apiapi"
	"apigw/internal/apisql"
	"apigw/internal/globalvar"
	"apigw/internal/logger"
	"apigw/internal/snmptrap"
	"fmt"
	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
	httpSwagger "github.com/swaggo/http-swagger"
	"net/http"
)

func createEndpoints() {
	listApiDescriptors := viper.GetStringMap("api")
	for apiDescriptor := range listApiDescriptors {

		logger.LogMsg(fmt.Sprintf("Info %s", apiDescriptor), "info")

		apiType := viper.GetString(fmt.Sprintf("api.%s.type", apiDescriptor))

		logger.LogMsg(fmt.Sprintf("Connect type: %s", apiType), "info")
		switch apiType {
		case "sql":
			apisql.CreateApiSql(apiDescriptor)
		case "api":
			apiapi.CreateApiApi(apiDescriptor)
		case "snmptrap":
			snmptrap.CreateApiSnmpTrap(apiDescriptor)
		default:
			logger.LogMsg(fmt.Sprintf("Sorry type : %s is not implemented", apiType), "warning")
		}

	}
	globalvar.Sm.PathPrefix("/documentation/").Handler(httpSwagger.WrapHandler)

}

func HandleRequests() {

	var err error

	globalvar.Sm.StrictSlash(true)

	viper.OnConfigChange(func(e fsnotify.Event) {
		logger.LogMsg(fmt.Sprintf("Config file changed: %s", e.Name), "info")
		createEndpoints()
	})
	viper.WatchConfig()
	createEndpoints()

	listeningPort := viper.GetString("listening_port")

	err = http.ListenAndServe(":"+listeningPort, globalvar.Sm)
	globalvar.CheckErr(err)

}
