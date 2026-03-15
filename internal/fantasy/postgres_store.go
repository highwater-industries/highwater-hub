package fantasy

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
)

// PostgresStore implements Store by reading the fantasy_* tables.
type PostgresStore struct {
	db *sql.DB
}

// NewPostgresStore creates a new store backed by the given database connection.
func NewPostgresStore(db *sql.DB) *PostgresStore {
	return &PostgresStore{db: db}
}

// ListLeagues returns paginated leagues with optional platform/season filter.
func (s *PostgresStore) ListLeagues(ctx context.Context, filter LeagueFilter, offset, limit int) ([]League, int, error) {
	where := "WHERE 1=1"
	args := []any{}

	if filter.Platform != "" {
		args = append(args, filter.Platform)
		where += fmt.Sprintf(" AND platform = $%d", len(args))
	}
	if filter.Season > 0 {
		args = append(args, filter.Season)
		where += fmt.Sprintf(" AND season = $%d", len(args))
	}

	// Count
	var total int
	err := s.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM fantasy_leagues "+where, args...,
	).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count leagues: %w", err)
	}

	// Fetch page
	pageArgs := append(args, limit, offset)
	rows, err := s.db.QueryContext(ctx,
		fmt.Sprintf(`SELECT id, external_league_id, league_name, platform, season,
                            num_teams, scoring_type, settings, created_at, updated_at
                     FROM fantasy_leagues %s
                     ORDER BY season DESC, league_name
                     LIMIT $%d OFFSET $%d`, where, len(args)+1, len(args)+2),
		pageArgs...,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("query leagues: %w", err)
	}
	defer rows.Close()

	leagues := make([]League, 0)
	for rows.Next() {
		var l League
		var numTeams sql.NullInt64
		var scoringType sql.NullString
		var settingsJSON sql.NullString

		err := rows.Scan(
			&l.ID, &l.ExternalLeagueID, &l.LeagueName, &l.Platform, &l.Season,
			&numTeams, &scoringType, &settingsJSON,
			&l.CreatedAt, &l.UpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("scan league: %w", err)
		}

		if numTeams.Valid {
			v := int(numTeams.Int64)
			l.NumTeams = &v
		}
		if scoringType.Valid {
			l.ScoringType = &scoringType.String
		}
		if settingsJSON.Valid {
			json.Unmarshal([]byte(settingsJSON.String), &l.Settings)
		}

		leagues = append(leagues, l)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate leagues: %w", err)
	}

	return leagues, total, nil
}

// GetLeague returns a single league by ID.
func (s *PostgresStore) GetLeague(ctx context.Context, id int) (League, error) {
	var l League
	var numTeams sql.NullInt64
	var scoringType sql.NullString
	var settingsJSON sql.NullString

	err := s.db.QueryRowContext(ctx,
		`SELECT id, external_league_id, league_name, platform, season,
                num_teams, scoring_type, settings, created_at, updated_at
         FROM fantasy_leagues WHERE id = $1`, id,
	).Scan(
		&l.ID, &l.ExternalLeagueID, &l.LeagueName, &l.Platform, &l.Season,
		&numTeams, &scoringType, &settingsJSON,
		&l.CreatedAt, &l.UpdatedAt,
	)
	if err != nil {
		return League{}, fmt.Errorf("get league %d: %w", id, err)
	}

	if numTeams.Valid {
		v := int(numTeams.Int64)
		l.NumTeams = &v
	}
	if scoringType.Valid {
		l.ScoringType = &scoringType.String
	}
	if settingsJSON.Valid {
		json.Unmarshal([]byte(settingsJSON.String), &l.Settings)
	}

	return l, nil
}

// ListTeams returns all teams for a given league, ordered by standing rank.
func (s *PostgresStore) ListTeams(ctx context.Context, leagueID int) ([]Team, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, league_id, external_team_id, team_name, owner_name,
                wins, losses, ties, points_for, points_against,
                standing_rank, playoff_seed, created_at, updated_at
         FROM fantasy_teams
         WHERE league_id = $1
         ORDER BY COALESCE(standing_rank, 9999), team_name`, leagueID,
	)
	if err != nil {
		return nil, fmt.Errorf("query teams: %w", err)
	}
	defer rows.Close()

	teams := make([]Team, 0)
	for rows.Next() {
		var t Team
		var extID, owner sql.NullString
		var rank, seed sql.NullInt64

		err := rows.Scan(
			&t.ID, &t.LeagueID, &extID, &t.TeamName, &owner,
			&t.Wins, &t.Losses, &t.Ties, &t.PointsFor, &t.PointsAgainst,
			&rank, &seed,
			&t.CreatedAt, &t.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan team: %w", err)
		}

		if extID.Valid {
			t.ExternalTeamID = &extID.String
		}
		if owner.Valid {
			t.OwnerName = &owner.String
		}
		if rank.Valid {
			v := int(rank.Int64)
			t.StandingRank = &v
		}
		if seed.Valid {
			v := int(seed.Int64)
			t.PlayoffSeed = &v
		}

		teams = append(teams, t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate teams: %w", err)
	}

	return teams, nil
}

// GetTeam returns a single team by ID.
func (s *PostgresStore) GetTeam(ctx context.Context, teamID int) (Team, error) {
	var t Team
	var extID, owner sql.NullString
	var rank, seed sql.NullInt64

	err := s.db.QueryRowContext(ctx,
		`SELECT id, league_id, external_team_id, team_name, owner_name,
                wins, losses, ties, points_for, points_against,
                standing_rank, playoff_seed, created_at, updated_at
         FROM fantasy_teams WHERE id = $1`, teamID,
	).Scan(
		&t.ID, &t.LeagueID, &extID, &t.TeamName, &owner,
		&t.Wins, &t.Losses, &t.Ties, &t.PointsFor, &t.PointsAgainst,
		&rank, &seed,
		&t.CreatedAt, &t.UpdatedAt,
	)
	if err != nil {
		return Team{}, fmt.Errorf("get team %d: %w", teamID, err)
	}

	if extID.Valid {
		t.ExternalTeamID = &extID.String
	}
	if owner.Valid {
		t.OwnerName = &owner.String
	}
	if rank.Valid {
		v := int(rank.Int64)
		t.StandingRank = &v
	}
	if seed.Valid {
		v := int(seed.Int64)
		t.PlayoffSeed = &v
	}

	return t, nil
}

// ListRoster returns all roster entries for a given team.
func (s *PostgresStore) ListRoster(ctx context.Context, teamID int) ([]RosterEntry, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, team_id, player_id, player_name, player_position,
                nfl_team, roster_position, external_player_id, matched, created_at
         FROM fantasy_rosters
         WHERE team_id = $1
         ORDER BY player_position, player_name`, teamID,
	)
	if err != nil {
		return nil, fmt.Errorf("query roster: %w", err)
	}
	defer rows.Close()

	roster := make([]RosterEntry, 0)
	for rows.Next() {
		var r RosterEntry
		var pid, nflTeam, rosterPos, extPID sql.NullString

		err := rows.Scan(
			&r.ID, &r.TeamID, &pid, &r.PlayerName, &r.PlayerPosition,
			&nflTeam, &rosterPos, &extPID, &r.Matched, &r.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan roster entry: %w", err)
		}

		if pid.Valid {
			r.PlayerID = &pid.String
		}
		if nflTeam.Valid {
			r.NFLTeam = &nflTeam.String
		}
		if rosterPos.Valid {
			r.RosterPosition = &rosterPos.String
		}
		if extPID.Valid {
			r.ExternalPlayerID = &extPID.String
		}

		roster = append(roster, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate roster: %w", err)
	}

	return roster, nil
}
