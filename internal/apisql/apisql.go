package apisql

import (
	"apigw/internal/globalvar"
	"apigw/internal/logger"
	"database/sql"
	"encoding/json"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
	"github.com/spf13/viper"
	"io"
	"net/http"
	"strings"
)

type sqlData struct {
	ReturnedRows int64         `json:"ReturnedRows"`
	Data         []interface{} `json:"Data"`
}

type Field struct {
	Field    string   `json:"field"`
	Criteria string   `json:"criteria"`
	Value    string   `json:"value"`
	Values   []string `json:"values"`
}

type Filter struct {
	Condition string  `json:"condition"`
	Filter    []Field `json:"filter"`
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
	w.Header().Set("Content-Type", "application/json")
	apiName := r.URL
	logger.LogMsg(fmt.Sprintf("Call %s", apiName), "info")

	decoder := json.NewDecoder(r.Body)
	var filter Filter
	// Try to use parameterized queries
	var mParam string
	err := decoder.Decode(&filter)
	if err != nil {
		panic(err)
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
		var sqlData sqlData
		var rowCount int64
		var dbConnect bool
		var endResponse *strings.Reader

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

			for i := range filter.Filter {
				// Try to use parameterized queries

				mParam = ""

				switch strings.ToLower(dbDriver) {
				case "postgres":
					//mParam = fmt.Sprintf("$%v", i+1)
					for j := range filter.Filter[i].Values {
						if mParam == "" {
							mParam = fmt.Sprintf("$%v", j+1)
						} else {
							mParam += fmt.Sprintf(", $%v", j+1)
						}
						//sqlParam = append(sqlParam, filter.Filter[i].Values[j])
					}
					//mParam = strings.Join(strings.Split(strings.Repeat("$", len(filter.Filter[i].Values)), ""), ", ")
				case "mysql":
					mParam = strings.Join(strings.Split(strings.Repeat("?", len(filter.Filter[i].Values)), ""), ", ")
					//for j := range filter.Filter[i].Values {
					//	if mParam == "" {
					//		mParam = fmt.Sprintf("?", j+1)
					//	} else {
					//		mParam += fmt.Sprintf(", $%v", j+1)
					//	}
					//	//sqlParam = append(sqlParam, filter.Filter[i].Values[j])
					//}
				}

				// If it's first condition
				if len(queryConditions) == 0 {
					// Try to use parameterized queries
					//queryConditions = fmt.Sprintf("%s = %s", filter.Filter[i].Field, mParam)
					//sqlParam = append(sqlParam, filter.Filter[i].Value)
					//queryConditions = fmt.Sprintf("%s = '%s'", filter.Filter[i].Field, filter.Filter[i].Value)

					sqlParam = make([]any, 0, len(filter.Filter[i].Values))
					//if len(filter.Filter[i].Values) == 0 {
					//	sqlParam
					//} else {
					//	for j := range filter.Filter[i].Values {
					//		sqlParam = append(sqlParam, filter.Filter[i].Values[j])
					//	}
					//}

					for j := range filter.Filter[i].Values {
						sqlParam = append(sqlParam, filter.Filter[i].Values[j])
					}
					//for val :=
					//sqlParam = append(sqlParam, filter.Filter[i].Values)

					/*
						// Works on a standard query
						if strings.ToUpper(filter.Filter[i].Criteria) == "IN" {
							value := strings.Join(filter.Filter[i].Values, "','")
							queryConditions += fmt.Sprintf(
								"%s IN ('%s')",
								filter.Filter[i].Field,
								value,
							)
						} else {
							queryConditions += fmt.Sprintf(
								"%s %s '%s'",
								filter.Filter[i].Field,
								filter.Filter[i].Criteria,
								filter.Filter[i].Value,
							)
						}
					*/

					if strings.ToUpper(filter.Filter[i].Criteria) == "IN" {
						//value := strings.Join(filter.Filter[i].Values, "','")
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
					// from here there is more than 1 filter

					// Try to use parameterized queries
					//queryConditions += fmt.Sprintf(
					//	" %s %s = %s",
					//	strings.ToLower(filter.Condition),
					//	filter.Filter[i].Field,
					//	mParam,
					//)
					//sqlParam = append(sqlParam, filter.Filter[i].Value)
					if strings.ToUpper(filter.Filter[i].Criteria) == "IN" {
						value := strings.Join(filter.Filter[i].Values, "','")
						queryConditions += fmt.Sprintf(
							" %s %s IN ('%s')",
							strings.ToUpper(filter.Condition),
							filter.Filter[i].Field,
							value,
						)
					} else {
						queryConditions += fmt.Sprintf(
							" %s %s %s '%s'",
							strings.ToUpper(filter.Condition),
							filter.Filter[i].Field,
							strings.ToUpper(filter.Filter[i].Criteria),
							filter.Filter[i].Value,
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

			// Try to use parameterized queries
			logger.LogMsg(fmt.Sprintf("execute query : %s", dbQuery), "info")

			rows, err := db.Query(dbQuery, sqlParam...)
			//
			//rows, err := db.Query(dbQuery)

			//var age []string
			//var age []interface{}
			//age := make([]any, 0, 3)
			//var ages
			//age = append(age, "RNaseP")
			//age = append(age, "U1")
			//age = append(age, "banana")
			//age = append(age, "orange")
			//age = append(age, "apple")
			//age = append(age, "banana")
			//ages := ages[0:len(age)]
			//ages := age[:]

			//rows, err := db.Query("SELECT * FROM clan WHERE id IN (?,?,?) LIMIT 10", age...)
			//rows, err := db.Query("SELECT * FROM fruits WHERE name IN ($1,$2,$3) LIMIT 10", age...)

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

				sqlData.Data = append(sqlData.Data, masterData)
				rowCount += 1
			}

			logger.LogMsg(fmt.Sprintf("found : %d records", rowCount), "info")
			sqlData.ReturnedRows = rowCount

			globalvar.CheckErr(rows.Close())
			globalvar.CheckErr(json.NewEncoder(w).Encode(sqlData))
			globalvar.CheckErr(db.Close())

		} else {

			_, err := io.Copy(w, endResponse)
			globalvar.CheckErr(err)

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
