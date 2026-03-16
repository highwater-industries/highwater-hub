package fantasy

import (
	"database/sql"
	"net/http"
	"strconv"

	"myproject/internal/httputil"
)

// HandleListLeagues returns paginated fantasy leagues.
//
//	GET /api/fantasy/leagues?platform=yahoo&season=2024
func HandleListLeagues(store Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		p := httputil.ParsePagination(
			r.URL.Query().Get("offset"),
			r.URL.Query().Get("limit"),
		)

		var filter LeagueFilter
		filter.Platform = r.URL.Query().Get("platform")
		if s := r.URL.Query().Get("season"); s != "" {
			if v, err := strconv.Atoi(s); err == nil {
				filter.Season = v
			}
		}

		leagues, total, err := store.ListLeagues(r.Context(), filter, p.Offset, p.Limit)
		if err != nil {
			httputil.Encode(w, http.StatusInternalServerError, httputil.ErrorResponse{
				Error: "failed to list leagues: " + err.Error(),
			})
			return
		}

		httputil.Encode(w, http.StatusOK, httputil.ListResponse[League]{
			Items:  leagues,
			Total:  total,
			Offset: p.Offset,
			Limit:  p.Limit,
		})
	}
}

// HandleGetLeague returns a single league with its teams.
//
//	GET /api/fantasy/leagues/{id}
func HandleGetLeague(store Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.Atoi(r.PathValue("id"))
		if err != nil {
			httputil.Encode(w, http.StatusBadRequest, httputil.ErrorResponse{
				Error: "invalid league id",
			})
			return
		}

		league, err := store.GetLeague(r.Context(), id)
		if err != nil {
			if err == sql.ErrNoRows {
				httputil.Encode(w, http.StatusNotFound, httputil.ErrorResponse{
					Error: "league not found",
				})
				return
			}
			httputil.Encode(w, http.StatusInternalServerError, httputil.ErrorResponse{
				Error: "failed to get league: " + err.Error(),
			})
			return
		}

		teams, err := store.ListTeams(r.Context(), league.ID)
		if err != nil {
			httputil.Encode(w, http.StatusInternalServerError, httputil.ErrorResponse{
				Error: "failed to get teams: " + err.Error(),
			})
			return
		}

		httputil.Encode(w, http.StatusOK, LeagueDetail{
			League: league,
			Teams:  teams,
		})
	}
}

// HandleGetTeam returns a team with its roster.
//
//	GET /api/fantasy/teams/{id}
func HandleGetTeam(store Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.Atoi(r.PathValue("id"))
		if err != nil {
			httputil.Encode(w, http.StatusBadRequest, httputil.ErrorResponse{
				Error: "invalid team id",
			})
			return
		}

		team, err := store.GetTeam(r.Context(), id)
		if err != nil {
			if err == sql.ErrNoRows {
				httputil.Encode(w, http.StatusNotFound, httputil.ErrorResponse{
					Error: "team not found",
				})
				return
			}
			httputil.Encode(w, http.StatusInternalServerError, httputil.ErrorResponse{
				Error: "failed to get team: " + err.Error(),
			})
			return
		}

		roster, err := store.ListRoster(r.Context(), team.ID)
		if err != nil {
			httputil.Encode(w, http.StatusInternalServerError, httputil.ErrorResponse{
				Error: "failed to get roster: " + err.Error(),
			})
			return
		}

		httputil.Encode(w, http.StatusOK, TeamDetail{
			Team:   team,
			Roster: roster,
		})
	}
}

// HandleListMatchups returns all weekly matchup scores for a league.
//
//	GET /api/fantasy/leagues/{id}/matchups
func HandleListMatchups(store Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.Atoi(r.PathValue("id"))
		if err != nil {
			httputil.Encode(w, http.StatusBadRequest, httputil.ErrorResponse{
				Error: "invalid league id",
			})
			return
		}

		matchups, err := store.ListMatchups(r.Context(), id)
		if err != nil {
			httputil.Encode(w, http.StatusInternalServerError, httputil.ErrorResponse{
				Error: "failed to get matchups: " + err.Error(),
			})
			return
		}

		httputil.Encode(w, http.StatusOK, matchups)
	}
}

// HandleStartImport dispatches a fantasy league import to the Python service.
//
//	POST /api/fantasy/import
func HandleStartImport(client *Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		req, err := httputil.Decode[ImportRequest](r)
		if err != nil {
			httputil.Encode(w, http.StatusBadRequest, httputil.ErrorResponse{
				Error: "invalid JSON",
			})
			return
		}

		// Validate required fields
		if req.Platform == "" || req.LeagueID == "" || req.Season == 0 {
			httputil.Encode(w, http.StatusBadRequest, httputil.ErrorResponse{
				Error: "platform, league_id, and season are required",
			})
			return
		}

		// ESPN requires cookies
		if req.Platform == "espn" && (req.SWID == "" || req.EspnS2 == "") {
			httputil.Encode(w, http.StatusBadRequest, httputil.ErrorResponse{
				Error: "ESPN imports require swid and espn_s2 cookies",
			})
			return
		}

		accepted, err := client.StartImport(r.Context(), req)
		if err != nil {
			httputil.Encode(w, http.StatusBadGateway, httputil.ErrorResponse{
				Error: "failed to start import: " + err.Error(),
			})
			return
		}

		httputil.Encode(w, http.StatusAccepted, accepted)
	}
}
