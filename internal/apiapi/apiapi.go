package apiapi

import (
	"apigw/internal/globalvar"
	"apigw/internal/logger"
	"encoding/json"
	"fmt"
	"github.com/spf13/viper"
	"io"
	"net"
	"net/http"
	"strings"
	"time"
)

type apiData struct {
	Message string        `json:"Message"`
	Data    []interface{} `json:"Data"`
}

func httpClose(c io.Closer) {
	err := c.Close()
	globalvar.CheckErr(err)
}

func CreateApiApi(apiName string) {
	//logger.LogMsg(fmt.Sprintf("Requested api endpoint : %s", apiName), "info")

	apiMethod := viper.GetString(fmt.Sprintf("api.%s.method", apiName))

	if apiMethod == "get" {
		globalvar.GetR.HandleFunc(apiName, queryApi)
		logger.LogMsg(fmt.Sprintf("Created GET api endpoint : %s", apiName), "info")
	}

}

func queryApi(w http.ResponseWriter, r *http.Request) {

	apiName := r.URL
	var apiData apiData

	logger.LogMsg(fmt.Sprintf("Call %s", r.URL), "info")

	if viper.IsSet(fmt.Sprintf("api.%s", apiName)) {

		var response *http.Response
		var err error

		if viper.IsSet(fmt.Sprintf("api.%s", apiName)) {

			endResponse := strings.NewReader(
				fmt.Sprintf("{\"error\" : \"Sorry the call %s is linked to an unexisting source\"}\n", apiName),
			)
			_, err := io.Copy(w, endResponse)
			globalvar.CheckErr(err)
			logger.LogMsg(fmt.Sprintf("{\"error\" : \"Sorry the call %s is linked to an unexisting source\"}\n", apiName), "info")

		} else {
			var timeout time.Duration
			timeout = time.Duration(viper.GetInt("http_client_timeout")) * time.Second

			apiSource := viper.GetString(fmt.Sprintf("api.%s.source", apiName))
			apiTargetMethod := viper.GetString(fmt.Sprintf("api.%s.target_method", apiName))
			apiTargetUrl := viper.GetString(fmt.Sprintf("sources.%s.url", apiSource))
			apiExpectedStatusCode := viper.GetInt(fmt.Sprintf("sources.%s.expected_status_code", apiSource))

			if viper.IsSet(fmt.Sprintf("sources.%s.http_client_timeout", apiSource)) {
				timeout = time.Duration(
					viper.GetInt(fmt.Sprintf("sources.%s.http_client_timeout", apiSource)),
				) * time.Second
			}

			tr := &http.Transport{
				MaxIdleConns: 10,
				//IdleConnTimeout:    30 * time.Second,
				ResponseHeaderTimeout: timeout,
				DisableCompression:    true,
			}
			httpClient := &http.Client{Transport: tr}

			switch apiTargetMethod {
			case "get":
				response, err = httpClient.Get(apiTargetUrl)

			//case "post":
			//	response, err = http.Post(apiTargetUrl)
			default:
				logger.LogMsg(
					fmt.Sprintf("Sorry but the http call %s is not recognized", apiTargetMethod),
					"critical",
				)

			}

			if err != nil {
				switch err := err.(type) {
				case net.Error:
					if err.Timeout() {
						fmt.Println("This was a net.Error with a Timeout")
						w.WriteHeader(http.StatusGatewayTimeout)
						apiData.Message = "Sorry but the endpoint did not answer in the expected timeframe"
						logger.LogMsg("Sorry but the endpoint did not answer in the expected timeframe", "info")
						globalvar.CheckErr(json.NewEncoder(w).Encode(apiData))
					}
				default:
					globalvar.CheckErr(err)
					httpClose(response.Body)

				}

			} else {

				w.Header().Set("Content-Type", "application/json")

				if response.StatusCode == apiExpectedStatusCode {
					_, err = io.Copy(w, response.Body)
					globalvar.CheckErr(err)

					// Add an CR at the end of the json stream
					endResponse := strings.NewReader("\n")
					_, err = io.Copy(w, endResponse)
					globalvar.CheckErr(err)

				} else {

					var data map[string]interface{}
					_ = json.NewDecoder(response.Body).Decode(&data)

					w.WriteHeader(http.StatusExpectationFailed)
					apiData.Data = append(apiData.Data, data)
					apiData.Message = "Sorry but the endpoint did not return expected StatusCode"
					globalvar.CheckErr(json.NewEncoder(w).Encode(apiData))

				}
				httpClose(response.Body)

			}

		}

	} else {

		endResponse := strings.NewReader(
			fmt.Sprintf("{\"error\" : \"Sorry the call %s was undefined\"}\n", apiName),
		)
		_, err := io.Copy(w, endResponse)
		globalvar.CheckErr(err)
		logger.LogMsg(fmt.Sprintf("Sorry the call %s was undefined", apiName), "info")

	}

}
