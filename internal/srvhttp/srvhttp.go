package srvhttp

import (
	"apigw/internal/logger"
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	_ "github.com/mattn/go-sqlite3"
	"github.com/spf13/viper"
	"io"
	"log"
	"net/http"
	"os"
)

var sm = mux.NewRouter()
var getR = sm.Methods(http.MethodGet).Subrouter()

type sqlData struct {
	ReturnedRows int64         `json:"ReturnedRows"`
	Data         []interface{} `json:"Data"`
}

func createApiSql(apiName string) {
	logger.LogMsg(fmt.Sprintf("Requested api endpoint : %s", apiName), "info")

	apiMethod := viper.GetString(fmt.Sprintf("api.%s.method", apiName))

	if apiMethod == "get" {
		getR.HandleFunc(apiName, querySql)
		logger.LogMsg(fmt.Sprintf("Created GET api endpoint : %s", apiName), "info")
	}

}
func createApiApi(apiName string) {
	logger.LogMsg(fmt.Sprintf("Requested api endpoint : %s", apiName), "info")

	apiMethod := viper.GetString(fmt.Sprintf("api.%s.method", apiName))

	if apiMethod == "get" {
		getR.HandleFunc(apiName, queryApi)
		logger.LogMsg(fmt.Sprintf("Created GET api endpoint : %s", apiName), "info")
	}

}

func queryApi(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	apiName := r.URL
	logger.LogMsg(fmt.Sprintf("Call %s", r.URL), "info")

	apiSource := viper.GetString(fmt.Sprintf("api.%s.source", apiName))
	apiTargetMethod := viper.GetString(fmt.Sprintf("api.%s.target_method", apiName))
	apiTargetUrl := viper.GetString(fmt.Sprintf("sources.%s.url", apiSource))

	var response *http.Response
	var err error

	switch apiTargetMethod {
	case "get":
		response, err = http.Get(apiTargetUrl)
	//case "post":
	//	response, err = http.Post(apiTargetUrl)
	default:
		logger.LogMsg(
			fmt.Sprintf("Sorry but the http method %s is not recognised", apiTargetMethod),
			"critical",
		)

	}

	if err != nil {
		fmt.Print(err.Error())
		os.Exit(1)
	}

	_, err = io.Copy(w, response.Body)
	checkErr(err)
	response.Body.Close()

}

func querySql(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	apiName := r.URL
	logger.LogMsg(fmt.Sprintf("Call %s", r.URL), "info")

	dbQuery := viper.GetString(fmt.Sprintf("api.%s.query", apiName))
	dbSource := viper.GetString(fmt.Sprintf("api.%s.source", apiName))
	dbName := viper.GetString(fmt.Sprintf("sources.%s.dbname", dbSource))
	dbDriver := viper.GetString(fmt.Sprintf("sources.%s.engine", dbSource))

	var db *sql.DB
	var err error
	var sqlData sqlData
	var rowCount int64

	if dbDriver == "sqlite" {
		db, err = sql.Open("sqlite3", dbName)
		checkErr(err)
		logger.LogMsg(fmt.Sprintf("Open sqlite db %s", dbName), "info")
	}

	rows, err := db.Query(dbQuery)
	checkErr(err)
	logger.LogMsg(fmt.Sprintf("execute query : %s", dbQuery), "info")

	columnTypes, err := rows.ColumnTypes()
	checkErr(err)

	colCount := len(columnTypes)
	rowCount = 0

	for rows.Next() {

		scanArgs := make([]interface{}, colCount)

		for i, v := range columnTypes {

			switch v.DatabaseTypeName() {
			case "VARCHAR", "TEXT", "UUID", "TIMESTAMP":
				scanArgs[i] = new(sql.NullString)
				break
			case "BOOL":
				scanArgs[i] = new(sql.NullBool)
				break
			case "INT4":
				scanArgs[i] = new(sql.NullInt64)
				break
			default:
				scanArgs[i] = new(sql.NullString)
			}
		}

		err := rows.Scan(scanArgs...)
		checkErr(err)

		masterData := map[string]interface{}{}

		for i, v := range columnTypes {

			if z, ok := (scanArgs[i]).(*sql.NullBool); ok {
				masterData[v.Name()] = z.Bool
				continue
			}

			if z, ok := (scanArgs[i]).(*sql.NullString); ok {
				masterData[v.Name()] = z.String
				continue
			}

			if z, ok := (scanArgs[i]).(*sql.NullInt64); ok {
				masterData[v.Name()] = z.Int64
				continue
			}

			if z, ok := (scanArgs[i]).(*sql.NullFloat64); ok {
				masterData[v.Name()] = z.Float64
				continue
			}

			if z, ok := (scanArgs[i]).(*sql.NullInt32); ok {
				masterData[v.Name()] = z.Int32
				continue
			}

			masterData[v.Name()] = scanArgs[i]
		}

		sqlData.Data = append(sqlData.Data, masterData)
		rowCount += 1
	}

	logger.LogMsg(fmt.Sprintf("found : %d records", rowCount), "info")
	sqlData.ReturnedRows = rowCount

	rows.Close()
	json.NewEncoder(w).Encode(sqlData)

}

func checkErr(err error) {
	if err != nil {
		logger.LogMsg(fmt.Sprintf("An error occured %s", err), "critical")
		panic(err)
	}
}

func HandleRequests() {

	// create a serve mux
	//sm = mux.NewRouter()
	sm.StrictSlash(true)
	//myRouter := mux.NewRouter().StrictSlash(true)

	// register handlers
	//postR := sm.Methods(http.MethodPost).Subrouter()
	//getR = sm.Methods(http.MethodGet).Subrouter()
	//putR := sm.Methods(http.MethodPut).Subrouter()
	//deleteR := sm.Methods(http.MethodDelete).Subrouter()

	listApiDescriptors := viper.GetStringMap("api")
	for apiDescriptor, _ := range listApiDescriptors {

		logger.LogMsg(fmt.Sprintf("Info %s", apiDescriptor), "info")

		apiType := viper.GetString(fmt.Sprintf("api.%s.type", apiDescriptor))

		logger.LogMsg(fmt.Sprintf("Connect type: %s", apiType), "info")
		switch apiType {
		case "sql":
			createApiSql(apiDescriptor)
		case "api":
			createApiApi(apiDescriptor)
		default:
			logger.LogMsg(fmt.Sprintf("Sorry type : %s is not implemented", apiType), "warning")
		}

	}

	// Define GET Call
	//getR.HandleFunc("/users", users.AllUsers)

	// Define POST Call
	//postR.HandleFunc("/user/{Username}", users.CreateUser)

	// Define DELETE Call
	//deleteR.HandleFunc("/user/{username}", users.DeleteUser)

	// Define PUT Call
	//putR.HandleFunc("/user/{username}/{email}", updateUser)

	//// used the PathPrefix as workaround for scenarios where all the
	//// get requests must use the ValidateAccessToken middleware except
	//// the /refresh-token request which has to use ValidateRefreshToken middleware
	//refToken := sm.PathPrefix("/refresh-token").Subrouter()
	//refToken.HandleFunc("", uh.RefreshToken)
	//refToken.Use(uh.MiddlewareValidateRefreshToken)

	log.Fatal(http.ListenAndServe(":8081", sm))
}

func deleteUser(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Delete User Endpoint Hit")
}

func updateUser(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Update User Endpoint Hit")
}