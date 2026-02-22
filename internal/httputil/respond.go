package httputil 

import (
    "encoding/json"
    "net/http"
)

func Encode (w http.ResponseWriter, status int, data any) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(status)
    json.NewEncoder(w).Encode(data)
}

func Decode[T any](r *http.Request) (T, error) {
    var v T
    err := json.NewDecoder(r.Body).Decode(&v)
    return v, err
}
