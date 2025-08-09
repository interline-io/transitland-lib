package util

import (
	"encoding/json"
	"net/http"
)

func WriteJsonError(w http.ResponseWriter, msg string, statusCode int) {
	a := map[string]string{
		"error": msg,
	}
	jj, _ := json.Marshal(&a)
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	w.Write(jj)
}
