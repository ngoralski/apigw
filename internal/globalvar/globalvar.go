package globalvar

import (
	"apigw/internal/logger"
	"context"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/shaj13/go-guardian/auth"
	"github.com/shaj13/go-guardian/auth/strategies/ldap"
	"github.com/shaj13/go-guardian/store"
	"github.com/spf13/viper"
	"log"
	"net/http"
	"time"
)

var Sm = mux.NewRouter()
var GetR = Sm.Methods(http.MethodGet).Subrouter()
var PostR = Sm.Methods(http.MethodPost).Subrouter()
var authenticator auth.Authenticator
var cache store.Cache

func CheckErr(err error) {
	if err != nil {
		logger.LogMsg(fmt.Sprintf("An error occured %s", err), "critical")
		panic(err)
	}
}

func SetupGoGuardian() {
	cfg := &ldap.Config{
		BaseDN:       viper.GetString("auth.basedn"),
		BindDN:       viper.GetString("auth.binddn"),
		Port:         viper.GetString("auth.port"),
		Host:         viper.GetString("auth.host"),
		BindPassword: viper.GetString("auth.bindpassword"),
		Filter:       viper.GetString("auth.filter"),
	}
	authenticator = auth.New()
	cache = store.NewFIFO(context.Background(), time.Minute*10)
	strategy := ldap.NewCached(cfg, cache)
	authenticator.EnableStrategy(ldap.StrategyKey, strategy)
}

func Middleware(next http.Handler) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		if viper.GetBool("auth.enabled") {
			log.Println("Executing Auth Middleware")
			user, err := authenticator.Authenticate(r)
			if err != nil {
				code := http.StatusUnauthorized
				http.Error(w, http.StatusText(code), code)
				return
			}
			log.Printf("User %s Authenticated\n", user.UserName())
		}
		next.ServeHTTP(w, r)
	})
}
