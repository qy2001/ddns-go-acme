package web

import (
	"encoding/json"
	"time"

	"net/http"

	"github.com/jeessy2/ddns-go/v6/config"
	"github.com/jeessy2/ddns-go/v6/util"
)

// AuthAPI 验证登录态，未登录或禁止访问时返回 JSON 而不是重定向
func AuthAPI(f ViewFunc) ViewFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		conf, _ := config.GetConfigCached()

		if conf.NotAllowWanAccess && !util.IsPrivateNetwork(r.RemoteAddr) {
			writeJSON(w, http.StatusForbidden, map[string]string{"result": "禁止从公网访问"})
			return
		}

		cookieInWeb, err := r.Cookie(cookieName)
		if err != nil ||
			cookieInSystem.Value == "" ||
			cookieInSystem.Value != cookieInWeb.Value ||
			!cookieInSystem.Expires.After(time.Now()) {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"result": "未登录"})
			return
		}

		f(w, r)
	}
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}
