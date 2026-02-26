package user

import (
	"myproject/internal/httputil"
	"net/http"
)

func HandleGetById(store Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")

		u, err := store.Get(r.Context(), id)
		if err != nil {
			httputil.Encode(w, http.StatusNotFound, httputil.ErrorResponse{Error: "user not found"})
			return
		}

		httputil.Encode(w, http.StatusOK, httputil.SingleResponse[User]{Data: u})
	}
}

func HandleGetAll(store Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		p := httputil.ParsePagination(
			r.URL.Query().Get("offset"),
			r.URL.Query().Get("limit"),
		)

		us, total, err := store.List(r.Context(), p.Offset, p.Limit)
		if err != nil {
			httputil.Encode(w, http.StatusInternalServerError, httputil.ErrorResponse{Error: "failed to list users"})
			return
		}

		httputil.Encode(w, http.StatusOK, httputil.ListResponse[User]{
			Items:  us,
			Total:  total,
			Offset: p.Offset,
			Limit:  p.Limit,
		})
	}
}

func HandleCreateSingle(store Store) http.HandlerFunc {
	type request struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		req, err := httputil.Decode[request](r)
		if err != nil {
			httputil.Encode(w, http.StatusBadRequest, httputil.ErrorResponse{Error: "invalid JSON"})
			return
		}

		u, err := store.Create(r.Context(), req.Name, req.Email)
		if err != nil {
			httputil.Encode(w, http.StatusInternalServerError, httputil.ErrorResponse{Error: "failed to create user"})
			return
		}

		httputil.Encode(w, http.StatusCreated, httputil.SingleResponse[User]{Data: u})
	}
}
