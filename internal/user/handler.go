package user

import (
    "net/http"
    "strconv"
    "myproject/internal/httputil"
)
                                                                                                                     
// @Summary      Get a user
// @Description  Gets a user by id 
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        id         uuid      model.Uuid 
// @Success      200        {object}  model.SingleResponse[model.User]
// @Failure      400        {object}  model.ErrorResponse
// @Router       /api/users [get]
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

// @Summary      Get users 
// @Description  Gets all users 
// @Tags         users
// @
// @Produce      json
// @Success      200        {object}  model.ListResponse[model.Users]
// @Failure      400        {object}  model.ErrorResponse
// @Router       /api/users [post]
func HandleGetAll(store Store) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
        limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
        if limit == 0 {
            limit = 20
        }
        us, total, err := store.List(r.Context(), offset, limit)
        if err != nil {
            httputil.Encode(w, http.StatusInternalServerError, httputil.ErrorResponse{Error: "failed to list users"})
            return
        }

        httputil.Encode(w, http.StatusOK, httputil.ListResponse[User]{
            Items:  us,
            Total:  total,
            Offset: offset,
            Limit:  limit,
        })
    }
}

// @Summary      Create a user
// @Description  Creates a new user
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        body  body      model.CreateUserRequest 
// @Success      201   {object}  model.SingleResponse[model.User]
// @Failure      400   {object}  model.ErrorResponse
// @Router       /api/users [post]
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
