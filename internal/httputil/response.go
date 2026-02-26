package httputil

import "strconv"

type ListResponse[T any] struct {
    Items  []T `json:"items"`
    Total  int `json:"total"`
    Offset int `json:"offset"`
    Limit  int `json:"limit"`
}

type PaginationParams struct {
    Offset int
    Limit  int
}

const DefaultLimit = 20
const MaxLimit = 100

func ParsePagination(offsetStr, limitStr string) PaginationParams {
    offset := 0
    limit := DefaultLimit

    if v, err := strconv.Atoi(offsetStr); err == nil && v >= 0 {
        offset = v
    }
    if v, err := strconv.Atoi(limitStr); err == nil && v > 0 && v <= MaxLimit {
        limit = v
    }

    return PaginationParams{Offset: offset, Limit: limit}
}

type SingleResponse[T any] struct {
    Data T `json:"data"`
}

type ErrorResponse struct {
    Error string  `json:"error"`
    Code  string  `json:"code,omitempty"`
}


