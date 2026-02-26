package nflstats

import (
	"net/http"
	"strconv"

	"myproject/internal/httputil"
)

// HandleListPlayers returns a filtered, paginated list of players.
//
//	GET /api/nflstats/players?team=KC&position=QB&search=mahomes&offset=0&limit=20
func HandleListPlayers(store Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Parse pagination
		p := httputil.ParsePagination(
			r.URL.Query().Get("offset"),
			r.URL.Query().Get("limit"),
		)

		// Parse filters from query params
		filter := parseFilter(r)

		players, total, err := store.List(r.Context(), filter, p.Offset, p.Limit)
		if err != nil {
			httputil.Encode(w, http.StatusInternalServerError, httputil.ErrorResponse{
				Error: "failed to list players",
			})
			return
		}

		httputil.Encode(w, http.StatusOK, httputil.ListResponse[Player]{
			Items:  players,
			Total:  total,
			Offset: p.Offset,
			Limit:  p.Limit,
		})
	}
}

// HandleGetPlayer returns a single player by internal database ID.
//
//	GET /api/nflstats/players/{id}
func HandleGetPlayer(store Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := r.PathValue("id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			httputil.Encode(w, http.StatusBadRequest, httputil.ErrorResponse{
				Error: "invalid player id",
			})
			return
		}

		player, err := store.Get(r.Context(), id)
		if err != nil {
			httputil.Encode(w, http.StatusNotFound, httputil.ErrorResponse{
				Error: "player not found",
			})
			return
		}

		httputil.Encode(w, http.StatusOK, httputil.SingleResponse[Player]{Data: player})
	}
}

// parseFilter extracts a PlayerFilter from URL query parameters.
// Missing params result in nil fields (no filtering on that dimension).
func parseFilter(r *http.Request) PlayerFilter {
	var f PlayerFilter

	if v := r.URL.Query().Get("team"); v != "" {
		f.Team = &v
	}
	if v := r.URL.Query().Get("position"); v != "" {
		f.Position = &v
	}
	if v := r.URL.Query().Get("source"); v != "" {
		f.Source = &v
	}
	if v := r.URL.Query().Get("search"); v != "" {
		f.Search = &v
	}

	return f
}
