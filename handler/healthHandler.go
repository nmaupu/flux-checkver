package handler

import (
	"encoding/json"
	"github.com/prometheus/common/log"
	"net/http"
)

var (
	_ Handler = Health{}
)

type Health struct {
	Status string `json:"status"`
}

func (h Health) Handle(w http.ResponseWriter, r *http.Request) {
	err := json.NewEncoder(w).Encode(h)
	if err != nil {
		log.Errorf("Error encoding json: %+v", err)
	}
}
