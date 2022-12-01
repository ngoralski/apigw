package globalvar

import (
	"apigw/internal/logger"
	"fmt"
	"github.com/gorilla/mux"
	"net/http"
)

var Sm = mux.NewRouter()
var GetR = Sm.Methods(http.MethodGet).Subrouter()
var PostR = Sm.Methods(http.MethodPost).Subrouter()

func CheckErr(err error) {
	if err != nil {
		logger.LogMsg(fmt.Sprintf("An error occured %s", err), "critical")
		panic(err)
	}
}
