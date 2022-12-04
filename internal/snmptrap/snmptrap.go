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

type SnmpMessage struct {
	Values []BodyMessage
}

type SnmpTarget struct {
	Ip           string `json:"ip"`
	Port         int    `json:"port"`
	Community    string `json:"community"`
	User         string `json:"user"`
	Pass         string `json:"pass"`
	Version      int    `json:"version"`
	Rootoid      string `json:"rootoid"`
	Specifictrap int    `json:"specific_trap"`
}

type PostData struct {
	Source SnmpSource      `json:"source"`
	Target SnmpTarget      `json:"target"`
	Data   []TrapDataLight `json:"msgdata"`
}

type TrapDataLight struct {
	Oid   string `json:"oid"`
	Value any    `json:"value"`
}

type returnMessage struct {
	Data []interface{} `json:"Data"`
}

type pduData struct {
	Data []gosnmp.SnmpPDU
}

func returnMsg(w io.Writer, rtnMessage returnMessage) {
	globalvar.CheckErr(json.NewEncoder(w).Encode(rtnMessage))
}

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

//func checkOverride(apiName any, varName string, postValue string) string {
func checkOverride(apiName any, varName string, key string, postValue any) any {
	if snmpOverride {
		logger.LogMsg(
			fmt.Sprintf(
				"%s can be overridden, change from '%s' to '%v'",
				varName,
				viper.GetString(fmt.Sprintf("api.%s.%s.%s", apiName, varName, key)),
				postValue,
			),
			"info",
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
			"info",
		)
		return postValue
	} else {
		//logger.LogMsg(fmt.Sprintf("%s is not supposed to be overridden, keep default value", varName), "info")
		return viper.GetString(fmt.Sprintf("api.%s.%s.%s", apiName, varName, key))
	}
}

func sendTrap(w http.ResponseWriter, r *http.Request) {

	//var snmptrap SnmpData
	var postdata PostData
	var errMessage string
	var rtnMessage returnMessage
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

	if viper.IsSet(fmt.Sprintf("api.%s", apiName)) {

		// If in config definition it's possible to override parameter take them into account.
		snmpOverride = viper.GetBool(fmt.Sprintf("api.%s.override", apiName))
		snmpSource := checkOverride(apiName, "source", "ip", postdata.Source.Ip)
		snmpTarget := checkOverride(apiName, "target", "ip", postdata.Target.Ip)
		//gosnmp.Default.Port = checkOverride(apiName, "target", "port", postdata.Target.Port).(uint16)
		gosnmp.Default.Community = checkOverride(apiName, "target", "community", postdata.Target.Community).(string)
		snmpUser := checkOverride(apiName, "target", "user", postdata.Target.User)
		snmpPass := checkOverride(apiName, "target", "pass", postdata.Target.Pass)
		snmpVersion := checkOverride(apiName, "target", "type", postdata.Target.Version)
		snmpRootOID := checkOverride(apiName, "target", "rootoid", postdata.Target.Rootoid)
		snmpSpecificTrap := checkOverride(apiName, "target", "specific_trap", postdata.Target.Specifictrap)
		//snmpData := checkOverride(apiName, "data", "values", postdata.Data).(ar)

		logger.LogMsg(
			fmt.Sprintf(
				"Will make this trap on call : %v, %s, %s, %s, %s, %s, %v, %s, %v",
				snmpOverride, snmpSource, snmpTarget, gosnmp.Default.Community, snmpUser,
				snmpPass, snmpVersion, snmpRootOID, snmpSpecificTrap,
			),
			"info",
		)

		if net.IsIPAddr(snmpTarget.(string)) || net.IsFQDN(snmpTarget.(string)) {
			gosnmp.Default.Target = snmpTarget.(string)
		} else {
			errMessage = fmt.Sprintf("Sorry your target host value (%s)is not correct", snmpTarget.(string))
			rtnMessage.Data = append(rtnMessage.Data, errMessage)
			logger.LogMsg(errMessage, "info")
		}

		/*
			It seems that we can't define a different ip source for snmptrap in this lib
		*/

		//if !net.IsIPAddr(snmpSource.(string)) || !net.IsFQDN(snmpSource.(string)) {
		if !net.IsIPAddr(snmpSource.(string)) {
			errMessage = fmt.Sprintf(
				"Sorry your Source host value (%s) is not correct",
				snmpSource.(string),
			)
			//errMessage = fmt.Sprintf("Sorry your Source host value (%s) is not correct", snmpSource.(string))
			rtnMessage.Data = append(rtnMessage.Data, errMessage)
			logger.LogMsg(errMessage, "info")
		}

		// If no error were discovered during processing posted data we can send the trap
		if len(rtnMessage.Data) > 0 {
			returnMsg(w, rtnMessage)
		} else {

			gosnmp.Default.Port = 162

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

			if !snmpOverride {

				if !viper.GetBool(fmt.Sprintf("api.%s.data.override", apiName)) {
					var defaultSnmpData []map[string]interface{}
					err = viper.UnmarshalKey(fmt.Sprintf("api.%s.data.values", apiName), &defaultSnmpData)
					if err != nil {
						fmt.Printf("err: %v\n", err)
					}

					for _, contents := range defaultSnmpData {

						vType := gosnmp.Integer
						vContent := contents["value"]
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

			}

			trap := gosnmp.SnmpTrap{
				Variables:    pdus.Data,
				Enterprise:   snmpRootOID.(string),
				AgentAddress: snmpSource.(string),
				GenericTrap:  0,
				SpecificTrap: 0,
				Timestamp:    300,
			}
			_, err = gosnmp.Default.SendTrap(trap)
			globalvar.CheckErr(err)

		}

	}

}
