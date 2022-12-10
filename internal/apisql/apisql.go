package apisql

import (
	"apigw/internal/globalvar"
	"apigw/internal/logger"
	"database/sql"
	"encoding/json"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"github.com/golang/gddo/httputil/header"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
	"github.com/spf13/viper"
	"io"
	"net/http"
	"regexp"
	"strings"
)

type jsonMessage struct {
	ReturnedRows int64         `json:"ReturnedRows"`
	Status       string        `json:"Status"`
	Message      string        `json:"Message"`
	Data         []interface{} `json:"Data"`
}

type Field struct {
	Field    string   `json:"field"`
	Criteria string   `json:"criteria"`
	Value    string   `json:"value"`
	Values   []string `json:"values"`
}

type Order struct {
	Field string `json:"field"`
	Order string `json:"order"`
}

type Filter struct {
	Condition string  `json:"condition"`
	Filter    []Field `json:"filter"`
	Order     []Order `json:"order"`
	Limit     int     `json:"limit"`
}

type Fil struct {
	Name string `json:"name"`
}

func CreateApiSql(apiName string) {
	logger.LogMsg(fmt.Sprintf("Requested api endpoint : %s", apiName), "info")

	apiMethod := viper.GetString(fmt.Sprintf("api.%s.method", apiName))

	if apiMethod == "get" {
		globalvar.GetR.HandleFunc(apiName, querySql)
		logger.LogMsg(fmt.Sprintf("Created GET api endpoint : %s", apiName), "info")
	}

	if apiMethod == "post" {
		globalvar.PostR.HandleFunc(apiName, querySql)
		logger.LogMsg(fmt.Sprintf("Created GET api endpoint : %s", apiName), "info")
	}

}

func querySql(w http.ResponseWriter, r *http.Request) {

	var mParam string
	var filter Filter
	var jsonMessage jsonMessage

	w.Header().Set("Content-Type", "application/json")
	apiName := r.URL
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
		err := decoder.Decode(&filter)
		globalvar.CheckErr(err)
	}

	if viper.IsSet(fmt.Sprintf("api.%s", apiName)) {

		dbQuery := viper.GetString(fmt.Sprintf("api.%s.query", apiName))
		dbSource := viper.GetString(fmt.Sprintf("api.%s.source", apiName))
		dbName := viper.GetString(fmt.Sprintf("sources.%s.dbname", dbSource))
		dbUsername := viper.GetString(fmt.Sprintf("sources.%s.username", dbSource))
		dbPassword := viper.GetString(fmt.Sprintf("sources.%s.password", dbSource))
		dbHost := viper.GetString(fmt.Sprintf("sources.%s.host", dbSource))
		dbPort := viper.GetInt(fmt.Sprintf("sources.%s.port", dbSource))
		dbDriver := viper.GetString(fmt.Sprintf("sources.%s.engine", dbSource))

		var db *sql.DB
		var err error
		var rowCount int64
		var dbConnect bool
		var endResponse *strings.Reader
		var cntParam int

		switch strings.ToLower(dbDriver) {
		case "sqlite":
			db, err = sql.Open("sqlite3", dbName)
			globalvar.CheckErr(err)
			logger.LogMsg(fmt.Sprintf("Open sqlite db %s", dbName), "info")
			dbConnect = true

		case "mysql":
			dbCnxString := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s", dbUsername, dbPassword, dbHost, dbPort, dbName)
			db, err = sql.Open("mysql", dbCnxString)
			globalvar.CheckErr(err)
			mParam = "?"

			if err != nil {
				logger.LogMsg(fmt.Sprintf("Can't open mysql db %s", dbName), "info")
				dbConnect = false
				endResponse = strings.NewReader(
					fmt.Sprintf("{\"error\" : \"Sorry the call %s is unable to join the target endpoint, "+
						"please contact an administrator\"}\n",
						apiName,
					),
				)
			} else {
				logger.LogMsg(fmt.Sprintf("Open mysql db %s", dbName), "info")
				dbConnect = true
			}

		case "postgres":
			dbCnxString := fmt.Sprintf("postgresql://%s:%s@%s:%d/%s", dbUsername, dbPassword, dbHost, dbPort, dbName)
			db, err = sql.Open("postgres", dbCnxString)
			globalvar.CheckErr(err)
			cntParam = 0

			if err != nil {
				logger.LogMsg(fmt.Sprintf("Can't open postgres db %s", dbName), "info")
				dbConnect = false
				endResponse = strings.NewReader(
					fmt.Sprintf("{\"error\" : \"Sorry the call %s is unable to join the target endpoint, "+
						"please contact an administrator\"}\n",
						apiName,
					),
				)
			} else {
				logger.LogMsg(fmt.Sprintf("Open postgres db %s", dbName), "info")
				dbConnect = true
			}

		default:
			dbConnect = false
			logger.LogMsg(
				fmt.Sprintf(
					"Sorry the call %s is misconfigured sql driver %s is not supported, ",
					apiName, dbDriver,
				),
				"info",
			)
			endResponse = strings.NewReader(
				fmt.Sprintf("{\"error\" : \"Sorry the call %s was misconfigured contact the administrator\"}\n",
					apiName,
				),
			)

		}

		if dbConnect {

			var queryConditions string
			//var sqlParam []string
			var sqlParam []any

			if len(filter.Filter) > 0 {

				var cntFilter int
				for i := range filter.Filter {
					cntFilter = cntFilter + len(filter.Filter[i].Values)
				}
				sqlParam = make([]any, 0, cntFilter)

				for i := range filter.Filter {
					// Try to use parameterized queries

					mParam = ""

					switch strings.ToLower(dbDriver) {
					case "postgres":
						for _ = range filter.Filter[i].Values {
							cntParam++
							if mParam == "" {
								mParam = fmt.Sprintf("$%v", cntParam)
							} else {
								mParam += fmt.Sprintf(", $%v", cntParam)
							}
						}
					case "mysql":
						mParam = strings.Join(strings.Split(strings.Repeat("?", len(filter.Filter[i].Values)), ""), ", ")
					}

					// If it's first condition
					if len(queryConditions) == 0 {

						for j := range filter.Filter[i].Values {
							sqlParam = append(sqlParam, filter.Filter[i].Values[j])
						}

						if strings.ToUpper(filter.Filter[i].Criteria) == "IN" {
							queryConditions += fmt.Sprintf(
								"%s IN (%s)",
								filter.Filter[i].Field,
								mParam,
							)
						} else {
							queryConditions += fmt.Sprintf(
								"%s %s %s",
								filter.Filter[i].Field,
								filter.Filter[i].Criteria,
								mParam,
							)
						}

					} else {

						//sqlParam = make([]any, 0, len(filter.Filter[i].Values))
						for j := range filter.Filter[i].Values {
							sqlParam = append(sqlParam, filter.Filter[i].Values[j])
						}

						if strings.ToUpper(filter.Filter[i].Criteria) == "IN" {
							queryConditions += fmt.Sprintf(
								" %s %s IN (%s)",
								strings.ToUpper(filter.Condition),
								filter.Filter[i].Field,
								mParam,
							)
						} else {
							queryConditions += fmt.Sprintf(
								" %s %s %s %s",
								strings.ToUpper(filter.Condition),
								filter.Filter[i].Field,
								filter.Filter[i].Criteria,
								mParam,
							)
						}

					}

					logger.LogMsg(
						fmt.Sprintf(
							"Filter on %s on value %s", filter.Filter[i].Field, filter.Filter[i].Value,
						),
						"info",
					)
				}

			}

			if len(queryConditions) > 0 {
				if strings.Contains(dbQuery, "WHERE") {
					dbQuery += " AND " + queryConditions
				} else {
					dbQuery += " WHERE " + queryConditions
				}
			}

			logger.LogMsg(
				fmt.Sprintf(
					"DB Query: %s, with params %s", dbQuery, sqlParam,
				),
				"info",
			)

			if len(filter.Order) > 0 {
				dbQueryOrder := ""
				//var re = regexp.MustCompile("\W+")
				for i := range filter.Order {
					name, _ := regexp.MatchString(`^(\W+)$`, filter.Order[i].Field)
					order, _ := regexp.MatchString(`^(ASC|DESC)$`, strings.ToUpper(filter.Order[i].Order))
					if name && order {
						dbQueryOrder += fmt.Sprintf(" %s %s", filter.Order[i].Field, filter.Order[i].Order)
					}

					//dbQuery += " %"
				}
				if len(dbQueryOrder) > 0 {
					dbQuery += fmt.Sprintf(" ORDER BY %s", dbQueryOrder)
				}
			}

			if filter.Limit > 0 {
				dbQuery += fmt.Sprintf(" LIMIT %v", filter.Limit)
			}

			logger.LogMsg(fmt.Sprintf("execute query : %s", dbQuery), "info")

			rows, err := db.Query(dbQuery, sqlParam...)

			globalvar.CheckErr(err)

			columnTypes, err := rows.ColumnTypes()
			globalvar.CheckErr(err)

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
					case "INT4", "INTEGER", "numeric":
						scanArgs[i] = new(sql.NullInt64)
						break
					default:
						scanArgs[i] = new(sql.NullString)
					}
				}

				err := rows.Scan(scanArgs...)
				globalvar.CheckErr(err)

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

				jsonMessage.Data = append(jsonMessage.Data, masterData)
				rowCount += 1
			}

			logger.LogMsg(fmt.Sprintf("found : %d records", rowCount), "info")
			jsonMessage.ReturnedRows = rowCount
			jsonMessage.Status = "ok"

			globalvar.CheckErr(rows.Close())
			globalvar.CheckErr(json.NewEncoder(w).Encode(jsonMessage))
			globalvar.CheckErr(db.Close())

		} else {

			_, err := io.Copy(w, endResponse)
			globalvar.CheckErr(err)

		}

	} else {

		jsonMessage.Message = fmt.Sprintf("{\"error\" : \"Sorry the call %s was undefined\"}\n", apiName)
		jsonMessage.Status = "error"
		globalvar.CheckErr(json.NewEncoder(w).Encode(jsonMessage))

		//endResponse := strings.NewReader(
		//
		//)
		//_, err := io.Copy(w, endResponse)
		//globalvar.CheckErr(err)
		logger.LogMsg(fmt.Sprintf("Sorry the call %s was undefined", apiName), "info")

	}

}
