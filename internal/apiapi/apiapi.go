package apiapi

import (
	"apigw/internal/globalvar"
	"apigw/internal/logger"
	"fmt"
	"github.com/spf13/viper"
	"io"
	"net/http"
	"strings"
)

func httpClose(c io.Closer) {
	err := c.Close()
	globalvar.CheckErr(err)
}

func CreateApiApi(apiName string) {
	logger.LogMsg(fmt.Sprintf("Requested api endpoint : %s", apiName), "info")

	apiMethod := viper.GetString(fmt.Sprintf("api.%s.method", apiName))

	if apiMethod == "get" {
		globalvar.GetR.HandleFunc(apiName, queryApi)
		logger.LogMsg(fmt.Sprintf("Created GET api endpoint : %s", apiName), "info")
	}

}

func queryApi(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "application/json")
	apiName := r.URL
	logger.LogMsg(fmt.Sprintf("Call %s", r.URL), "info")

	if viper.IsSet(fmt.Sprintf("api.%s", apiName)) {

		var response *http.Response
		var err error

		apiSource := viper.GetString(fmt.Sprintf("api.%s.source", apiName))
		apiTargetMethod := viper.GetString(fmt.Sprintf("api.%s.target_method", apiName))
		apiTargetUrl := viper.GetString(fmt.Sprintf("sources.%s.url", apiSource))

		switch apiTargetMethod {
		case "get":
			response, err = http.Get(apiTargetUrl)
		//case "post":
		//	response, err = http.Post(apiTargetUrl)
		default:
			logger.LogMsg(
				fmt.Sprintf("Sorry but the http call %s is not recognized", apiTargetMethod),
				"critical",
			)

		}

		globalvar.CheckErr(err)

		_, err = io.Copy(w, response.Body)
		globalvar.CheckErr(err)

		// Add an CR at the end of the json stream
		endResponse := strings.NewReader("\n")
		_, err = io.Copy(w, endResponse)
		globalvar.CheckErr(err)
		httpClose(response.Body)

	} else {

		endResponse := strings.NewReader(
			fmt.Sprintf("{\"error\" : \"Sorry the call %s was undefined\"}\n", apiName),
		)
		_, err := io.Copy(w, endResponse)
		globalvar.CheckErr(err)
		logger.LogMsg(fmt.Sprintf("Sorry the call %s was undefined", apiName), "info")

	}

}
