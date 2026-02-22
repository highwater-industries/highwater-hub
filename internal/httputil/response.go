package httputil

type ListResponse[T any] struct {
    Total  int  `json:"total"`
    Offset int  `json:"offset"`
    Limit  int  `json:"limit"`
    Items  []T  `json:"items"`

}

type SingleResponse[T any] struct {
    Data T `json:"data"`
}

type ErrorResponse struct {
    Error string  `json:"error"`
    Code  string  `json:"code,omitempty"`
}


