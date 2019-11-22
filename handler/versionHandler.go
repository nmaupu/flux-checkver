package handler

import (
	"encoding/json"
	"github.com/prometheus/common/log"
	"net/http"
)

var (
	_ Handler = AppInfo{}
)

type AppInfo struct {
	AppName    string `json:"appName"`
	AppDesc    string `json:"appDesc"`
	AppVersion string `json:"appVersion"`
}

func (ai AppInfo) Handle(w http.ResponseWriter, r *http.Request) {
	err := json.NewEncoder(w).Encode(ai)
	if err != nil {
		log.Errorf("Error encoding json: %+v", err)
	}
}
