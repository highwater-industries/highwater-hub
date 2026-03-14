package nflstats

import (
	"net/http"
	"strconv"

	"myproject/internal/httputil"
)

// =====================================================================
// Player Stats handlers
// =====================================================================

// HandleListStats lists player stats with filtering and pagination.
//
//	GET /api/nflstats/stats?player_id=00-0022531&team=KC&position=QB&season=2024&week=1&search=mahomes
func HandleListStats(store StatStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		p := httputil.ParsePagination(r.URL.Query().Get("offset"), r.URL.Query().Get("limit"))
		filter := parseStatFilter(r)

		var stats []PlayerStat
		var total int
		var err error

		if filter.GroupBy == "season" {
			stats, total, err = store.ListSeasonStats(r.Context(), filter, p.Offset, p.Limit)
		} else {
			stats, total, err = store.ListStats(r.Context(), filter, p.Offset, p.Limit)
		}
		if err != nil {
			httputil.Encode(w, http.StatusInternalServerError, httputil.ErrorResponse{Error: "failed to list stats"})
			return
		}

		httputil.Encode(w, http.StatusOK, httputil.ListResponse[PlayerStat]{
			Items: stats, Total: total, Offset: p.Offset, Limit: p.Limit,
		})
	}
}

// HandleGetLeaders returns the top N players for a given stat.
//
//	GET /api/nflstats/leaders?stat=passing_yards&season=2024&week=1&position=QB&limit=25
func HandleGetLeaders(store StatStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		stat := r.URL.Query().Get("stat")
		if stat == "" {
			httputil.Encode(w, http.StatusBadRequest, httputil.ErrorResponse{Error: "stat parameter required"})
			return
		}

		seasonStr := r.URL.Query().Get("season")
		season, err := strconv.Atoi(seasonStr)
		if err != nil || season < 1920 {
			httputil.Encode(w, http.StatusBadRequest, httputil.ErrorResponse{Error: "valid season parameter required"})
			return
		}

		week := 0
		if v := r.URL.Query().Get("week"); v != "" {
			week, _ = strconv.Atoi(v)
		}

		position := r.URL.Query().Get("position")

		limit := 25
		if v := r.URL.Query().Get("limit"); v != "" {
			if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 100 {
				limit = n
			}
		}

		leaders, err := store.GetLeaders(r.Context(), stat, season, week, position, limit)
		if err != nil {
			httputil.Encode(w, http.StatusBadRequest, httputil.ErrorResponse{Error: err.Error()})
			return
		}

		httputil.Encode(w, http.StatusOK, map[string]any{
			"stat":     stat,
			"season":   season,
			"week":     week,
			"position": position,
			"items":    leaders,
		})
	}
}

// =====================================================================
// Games handlers
// =====================================================================

// HandleListGames lists games with filtering and pagination.
//
//	GET /api/nflstats/games?season=2024&week=1&team=KC
func HandleListGames(store GameStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		p := httputil.ParsePagination(r.URL.Query().Get("offset"), r.URL.Query().Get("limit"))
		filter := parseGameFilter(r)

		games, total, err := store.ListGames(r.Context(), filter, p.Offset, p.Limit)
		if err != nil {
			httputil.Encode(w, http.StatusInternalServerError, httputil.ErrorResponse{Error: "failed to list games"})
			return
		}

		httputil.Encode(w, http.StatusOK, httputil.ListResponse[Game]{
			Items: games, Total: total, Offset: p.Offset, Limit: p.Limit,
		})
	}
}

// HandleGetGame returns a single game by game_id.
//
//	GET /api/nflstats/games/{game_id}
func HandleGetGame(store GameStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		gameID := r.PathValue("game_id")
		if gameID == "" {
			httputil.Encode(w, http.StatusBadRequest, httputil.ErrorResponse{Error: "game_id required"})
			return
		}

		game, err := store.GetGame(r.Context(), gameID)
		if err != nil {
			httputil.Encode(w, http.StatusNotFound, httputil.ErrorResponse{Error: "game not found"})
			return
		}

		httputil.Encode(w, http.StatusOK, httputil.SingleResponse[Game]{Data: game})
	}
}

// =====================================================================
// Fantasy Rankings handlers
// =====================================================================

// HandleListRankings lists fantasy rankings with filtering and pagination.
//
//	GET /api/nflstats/rankings?rank_type=draft&pos=QB&team=KC&search=mahomes
func HandleListRankings(store RankingStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		p := httputil.ParsePagination(r.URL.Query().Get("offset"), r.URL.Query().Get("limit"))
		filter := parseRankingFilter(r)

		rankings, total, err := store.ListRankings(r.Context(), filter, p.Offset, p.Limit)
		if err != nil {
			httputil.Encode(w, http.StatusInternalServerError, httputil.ErrorResponse{Error: "failed to list rankings"})
			return
		}

		httputil.Encode(w, http.StatusOK, httputil.ListResponse[FantasyRanking]{
			Items: rankings, Total: total, Offset: p.Offset, Limit: p.Limit,
		})
	}
}

// =====================================================================
// Filter parsers
// =====================================================================

func parseStatFilter(r *http.Request) StatFilter {
	var f StatFilter
	if v := r.URL.Query().Get("player_id"); v != "" {
		f.PlayerID = &v
	}
	if v := r.URL.Query().Get("team"); v != "" {
		f.Team = &v
	}
	if v := r.URL.Query().Get("position"); v != "" {
		f.Position = &v
	}
	if v := r.URL.Query().Get("season"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			f.Season = &n
		}
	}
	if v := r.URL.Query().Get("week"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			f.Week = &n
		}
	}
	if v := r.URL.Query().Get("search"); v != "" {
		f.Search = &v
	}
	if v := r.URL.Query().Get("stat_type"); v != "" {
		f.StatType = &v
	}
	if v := r.URL.Query().Get("season_type"); v != "" {
		f.SeasonType = &v
	}
	if v := r.URL.Query().Get("source"); v != "" {
		f.Source = &v
	}

	f.Sort = r.URL.Query().Get("sort")
	f.Order = r.URL.Query().Get("order")
	f.GroupBy = r.URL.Query().Get("group_by")

	return f
}

func parseGameFilter(r *http.Request) GameFilter {
	var f GameFilter
	if v := r.URL.Query().Get("season"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			f.Season = &n
		}
	}
	if v := r.URL.Query().Get("week"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			f.Week = &n
		}
	}
	if v := r.URL.Query().Get("team"); v != "" {
		f.Team = &v
	}

	f.Sort = r.URL.Query().Get("sort")
	f.Order = r.URL.Query().Get("order")

	return f
}

func parseRankingFilter(r *http.Request) RankingFilter {
	var f RankingFilter
	if v := r.URL.Query().Get("rank_type"); v != "" {
		f.RankType = &v
	}
	if v := r.URL.Query().Get("pos"); v != "" {
		f.Pos = &v
	}
	if v := r.URL.Query().Get("team"); v != "" {
		f.Team = &v
	}
	if v := r.URL.Query().Get("search"); v != "" {
		f.Search = &v
	}
	if v := r.URL.Query().Get("season"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			f.Season = &n
		}
	}
	if v := r.URL.Query().Get("week"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			f.Week = &n
		}
	}
	if v := r.URL.Query().Get("source"); v != "" {
		f.Source = &v
	}

	f.Sort = r.URL.Query().Get("sort")
	f.Order = r.URL.Query().Get("order")

	return f
}
