package snmptrap

import (
	"apigw/internal/globalvar"
	"apigw/internal/logger"
	"encoding/json"
	"fmt"
	"github.com/THREATINT/go-net"
	"github.com/golang/gddo/httputil/header"
	"github.com/gosnmp/gosnmp"
	"github.com/spf13/viper"
	"io"
	"net/http"
	"strings"
)

var snmpOverride bool
var apiName any

type SnmpData struct {
	Version      int
	community    string
	user         string
	pass         string
	ipFrom       string
	oid          string
	specificTrap int
	trapType     int
	trapData     []TrapData
}

type TrapData struct {
	Oid      string `json:"oid"`
	DataType string
	Value    string `json:"value"`
}

type SnmpSource struct {
	Ip string `json:"ip"`
}

type BodyMessage struct {
	oid   string
	value string
}

type jsonMessage struct {
	Status  string        `json:"status"`
	Message []interface{} `json:"message"`
}

type SnmpTarget struct {
	// Define the ip of the snmp trap destination
	// in: ipv4,ipv6
	Ip           string `json:"ip"`
	Port         uint16 `json:"port"`
	Community    string `json:"community"`
	User         string `json:"user"`
	Pass         string `json:"pass"`
	Version      string `json:"version"`
	Rootoid      string `json:"rootoid"`
	Specifictrap int    `json:"specific_trap"`
}

type PostData struct {
	// Enable the help for the api call
	// in: bool
	Help   bool            `json:"help"`
	Source SnmpSource      `json:"source"`
	Target SnmpTarget      `json:"target"`
	Data   []TrapDataLight `json:"msgdata"`
}

type TrapDataLight struct {
	Oid   string `json:"oid"`
	Value any    `json:"value"`
}

//type returnMessage struct {
//	Data []interface{} `json:"Data"`
//}

type pduData struct {
	Data []gosnmp.SnmpPDU
}

//func returnMsg(w io.Writer, rtnMessage returnMessage) {
//	globalvar.CheckErr(json.NewEncoder(w).Encode(rtnMessage))
//}

func CreateApiSnmpTrap(apiName string) {
	logger.LogMsg(fmt.Sprintf("Requested api endpoint : %s", apiName), "info")

	apiMethod := viper.GetString(fmt.Sprintf("api.%s.method", apiName))

	if apiMethod == "get" {
		globalvar.GetR.HandleFunc(apiName, sendTrap)
		logger.LogMsg(fmt.Sprintf("Created GET api endpoint : %s", apiName), "info")
	}

	if apiMethod == "post" {
		globalvar.PostR.HandleFunc(apiName, sendTrap)
		logger.LogMsg(fmt.Sprintf("Created GET api endpoint : %s", apiName), "info")
	}

}

func checkOverrideUint16(apiName any, varName string, key string, postValue uint16) uint16 {
	if postValue == 0 {
		logger.LogMsg(
			fmt.Sprintf(
				"PostValue for %s.%s is nil, use config value '%s'",
				varName,
				key,
				viper.GetString(fmt.Sprintf("api.%s.%s.%s", apiName, varName, key)),
			),
			"debug",
		)
		return viper.GetUint16(fmt.Sprintf("api.%s.%s.%s", apiName, varName, key))
	}

	if snmpOverride {
		logger.LogMsg(
			fmt.Sprintf(
				"%s can be overridden, change from '%s' to '%v'",
				varName,
				viper.GetString(fmt.Sprintf("api.%s.%s.%s", apiName, varName, key)),
				postValue,
			),
			"debug",
		)
		return postValue
	} else if viper.GetBool(fmt.Sprintf("api.%s.%s.override", apiName, varName)) {
		logger.LogMsg(
			fmt.Sprintf(
				"%s can be overridden, change from '%s' to '%v'",
				varName,
				viper.GetString(fmt.Sprintf("api.%s.%s.%s", apiName, varName, key)),
				postValue,
			),
			"debug",
		)
		return postValue
	} else {
		//logger.LogMsg(fmt.Sprintf("%s is not supposed to be overridden, keep default value", varName), "info")
		return viper.GetUint16(fmt.Sprintf("api.%s.%s.%s", apiName, varName, key))
	}
}

func checkOverrideInt(apiName any, varName string, key string, postValue int) int {
	if postValue == 0 {
		logger.LogMsg(
			fmt.Sprintf(
				"PostValue for %s.%s is nil, use config value '%s'",
				varName,
				key,
				viper.GetString(fmt.Sprintf("api.%s.%s.%s", apiName, varName, key)),
			),
			"debug",
		)
		return viper.GetInt(fmt.Sprintf("api.%s.%s.%s", apiName, varName, key))
	}

	if snmpOverride {
		logger.LogMsg(
			fmt.Sprintf(
				"%s can be overridden, change from '%s' to '%v'",
				varName,
				viper.GetString(fmt.Sprintf("api.%s.%s.%s", apiName, varName, key)),
				postValue,
			),
			"debug",
		)
		return postValue
	} else if viper.GetBool(fmt.Sprintf("api.%s.%s.override", apiName, varName)) {
		logger.LogMsg(
			fmt.Sprintf(
				"%s can be overridden, change from '%s' to '%v'",
				varName,
				viper.GetString(fmt.Sprintf("api.%s.%s.%s", apiName, varName, key)),
				postValue,
			),
			"debug",
		)
		return postValue
	} else {
		//logger.LogMsg(fmt.Sprintf("%s is not supposed to be overridden, keep default value", varName), "info")
		return viper.GetInt(fmt.Sprintf("api.%s.%s.%s", apiName, varName, key))
	}
}

//func checkOverride(apiName any, varName string, postValue string) string {
func checkOverrideString(apiName any, varName string, key string, postValue string) string {

	if postValue == "" {
		logger.LogMsg(
			fmt.Sprintf(
				"PostValue for %s.%s is nil, use config value '%s'",
				varName,
				key,
				viper.GetString(fmt.Sprintf("api.%s.%s.%s", apiName, varName, key)),
			),
			"debug",
		)
		return viper.GetString(fmt.Sprintf("api.%s.%s.%s", apiName, varName, key))
	}

	if snmpOverride {
		logger.LogMsg(
			fmt.Sprintf(
				"%s can be overridden, change from '%s' to '%v'",
				varName,
				viper.GetString(fmt.Sprintf("api.%s.%s.%s", apiName, varName, key)),
				postValue,
			),
			"debug",
		)
		return postValue
	} else if viper.GetBool(fmt.Sprintf("api.%s.%s.override", apiName, varName)) {
		logger.LogMsg(
			fmt.Sprintf(
				"%s can be overridden, change from '%s' to '%v'",
				varName,
				viper.GetString(fmt.Sprintf("api.%s.%s.%s", apiName, varName, key)),
				postValue,
			),
			"debug",
		)
		return postValue
	} else {
		return viper.GetString(fmt.Sprintf("api.%s.%s.%s", apiName, varName, key))
	}
}

// sendTrap godoc
//	@Summary		Send a snmp trap via api call with predefined or posted values
//	@Description	Send a snmp trap
//	@Tags			snmptrap
//	@Accept			json
//	@Produce		json
//	@Param			message	body		PostData	true	"snmptrap form"
//	@Success		200		{object}	string		"Trap Sent"
//	@Failure		400		{object}	string		"Error processing Data"
//	@Router			/generic/snmptrap [get]
func sendTrap(w http.ResponseWriter, r *http.Request) {

	//var snmptrap SnmpData
	var postdata PostData
	var errMessage string
	//var rtnMessage returnMessage
	var jsonMessage jsonMessage
	//var snmpMessage []SnmpMessage{}

	w.Header().Set("Content-Type", "application/json")
	apiName = r.URL
	logger.LogMsg(fmt.Sprintf("Call %s", apiName), "info")

	logger.LogMsg(fmt.Sprintf("Header : %s", r.Header.Get("Content-Type")), "info")

	if r.Header.Get("Content-Type") != "" {
		value, _ := header.ParseValueAndParams(r.Header, "Content-Type")
		if value != "application/json" {
			msg := "Content-Type header is not application/json"
			http.Error(w, msg, http.StatusUnsupportedMediaType)
			return
		}
		decoder := json.NewDecoder(r.Body)
		// Try to use parameterized queries
		err := decoder.Decode(&postdata)
		globalvar.CheckErr(err)
	}

	// TODO
	// Check if posted json format / value match expectation (string is string, int is int..)

	if viper.IsSet(fmt.Sprintf("api.%s", apiName)) {

		// If in config definition it's possible to override parameter take them into account.
		snmpOverride = viper.GetBool(fmt.Sprintf("api.%s.override", apiName))
		snmpSource := checkOverrideString(apiName, "source", "ip", postdata.Source.Ip)
		snmpTarget := checkOverrideString(apiName, "target", "ip", postdata.Target.Ip)

		//if postdata.Target.Port nil || postdata.Target.Port == "" || postdata.Target.Port == 0 {
		//	postdata.Target.Port = viper.GetString(fmt.Sprintf("api.%s.%s.%s", apiName, "target", "port"))
		//}

		gosnmp.Default.Port = checkOverrideUint16(apiName, "target", "port", postdata.Target.Port)
		gosnmp.Default.Community = checkOverrideString(apiName, "target", "community", postdata.Target.Community)
		//snmpUser := checkOverride(apiName, "target", "user", postdata.Target.User)
		//snmpPass := checkOverride(apiName, "target", "pass", postdata.Target.Pass)
		snmpVersion := checkOverrideString(apiName, "target", "type", postdata.Target.Version)
		snmpRootOID := checkOverrideString(apiName, "target", "rootoid", postdata.Target.Rootoid)
		snmpSpecificTrap := checkOverrideInt(apiName, "target", "specific_trap", postdata.Target.Specifictrap)

		if net.IsIPAddr(snmpTarget) || net.IsFQDN(snmpTarget) {
			gosnmp.Default.Target = snmpTarget
		} else {
			errMessage = fmt.Sprintf("Sorry your target host value (%s) is not correct", snmpTarget)
			jsonMessage.Message = append(jsonMessage.Message, errMessage)
			logger.LogMsg(errMessage, "info")
		}

		if !net.IsIPAddr(snmpSource) {
			errMessage = fmt.Sprintf(
				"Sorry your Source host value (%s) is not correct",
				snmpSource,
			)
			//errMessage = fmt.Sprintf("Sorry your Source host value (%s) is not correct", snmpSource.(string))
			jsonMessage.Message = append(jsonMessage.Message, errMessage)
			logger.LogMsg(errMessage, "info")
		}

		// If no error were discovered during processing posted data we can send the trap
		if len(jsonMessage.Message) > 0 {
			jsonMessage.Status = "error"
			globalvar.CheckErr(json.NewEncoder(w).Encode(jsonMessage))

			//returnMsg(w, rtnMessage)
		} else {

			switch snmpVersion {
			case "3":
				gosnmp.Default.Version = gosnmp.Version3
			case "2|2c":
				gosnmp.Default.Version = gosnmp.Version2c
			default:
				gosnmp.Default.Version = gosnmp.Version1
			}

			err := gosnmp.Default.Connect()
			globalvar.CheckErr(err)
			defer gosnmp.Default.Conn.Close()

			var pdus pduData

			//
			// TODO
			// Actually get global override.
			//

			if !viper.GetBool(fmt.Sprintf("api.%s.data.override", apiName)) {
				var defaultSnmpData []map[string]interface{}
				err = viper.UnmarshalKey(fmt.Sprintf("api.%s.data.values", apiName), &defaultSnmpData)
				if err != nil {
					fmt.Printf("err: %v\n", err)
				}

				for _, contents := range defaultSnmpData {

					vType := gosnmp.Integer
					vContent := contents["value"]
					logger.LogMsg(fmt.Sprintf("Append new OID %s : %s", contents["oid"].(string), vContent), "debug")

					switch contents["value"].(type) {
					//case float64:
					//	vType = gosnmp.Integer
					//	vContent = int(contents["value"].(float64))
					case string:
						vType = gosnmp.OctetString
						vContent = contents["value"].(string)
					default:
						vType = gosnmp.Integer
						vContent = int(contents["value"].(float64))
					}

					pdus.Data = append(
						pdus.Data,
						gosnmp.SnmpPDU{
							Name:  contents["oid"].(string),
							Type:  vType,
							Value: vContent,
						},
					)

				}

			} else {

				for idx := range postdata.Data {
					logger.LogMsg(
						fmt.Sprintf(
							"DBG : Data : ##%v##", postdata.Data[idx].Oid,
						),
						"info",
					)

					vType := gosnmp.Integer
					vContent := postdata.Data[idx].Value
					switch postdata.Data[idx].Value.(type) {
					case string:
						vType = gosnmp.OctetString
						vContent = postdata.Data[idx].Value.(string)
					default:
						vType = gosnmp.Integer
						vContent = int(postdata.Data[idx].Value.(float64))
					}

					pdus.Data = append(
						pdus.Data,
						gosnmp.SnmpPDU{
							Name:  postdata.Data[idx].Oid,
							Type:  vType,
							Value: vContent,
						},
					)
				}

			}

			trap := gosnmp.SnmpTrap{
				Variables:    pdus.Data,
				Enterprise:   snmpRootOID,
				AgentAddress: snmpSource,
				GenericTrap:  0,
				SpecificTrap: snmpSpecificTrap,
				Timestamp:    300,
			}
			_, err = gosnmp.Default.SendTrap(trap)
			globalvar.CheckErr(err)

			if err == nil {
				jsonMessage.Status = "ok"
				jsonMessage.Message = append(jsonMessage.Message, "your snmptrap was submitted")
				globalvar.CheckErr(json.NewEncoder(w).Encode(jsonMessage))
				//
				//endResponse := strings.NewReader(
				//	fmt.Sprintf("{\"message\" : \"your snmptrap was submitted\"}\n"),
				//)
				//_, err := io.Copy(w, endResponse)
				globalvar.CheckErr(err)
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
