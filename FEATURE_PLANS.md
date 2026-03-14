# Feature Plans

Four features planned for upcoming sessions. Each is self-contained and can be built independently.

---

## 1. Player Detail Page

**Goal:** Click any player name anywhere in the app → dedicated page showing their full profile, career stats, rankings, and game log.

### What already exists
- `GET /api/nflstats/players/{id}` — Go route + handler (returns Player with metadata)
- `getPlayer(id)` — frontend API function (defined but unused)
- `GET /api/nflstats/stats?player_id=XXX` — stats filtering by player_id already works
- `GET /api/nflstats/rankings?search=NAME` — rankings search works
- Player metadata includes: college, jersey_number, status, height, weight, birth_date, years_exp, headshot_url, etc.

### Backend work

**New Go endpoint — player stats aggregation:**
- `GET /api/nflstats/players/{player_id}/summary` — returns career totals, season-by-season totals, and recent game log in one response
- Add `GetPlayerSummary(ctx, playerID)` to `StatStore` interface
- SQL: aggregate stats grouped by season + ungrouped career totals + last 5 weekly rows
- This avoids the frontend making 3+ separate API calls

**Expose GetByPlayerID via HTTP:**
- `GET /api/nflstats/players/by-player-id/{player_id}` — lookup by NFL ID (gsis_id) instead of internal DB ID
- Already implemented in the store, just needs a handler + route

### Frontend work

**New route: `web/src/routes/players/[id]/+page.svelte`**

Layout (single page, scrollable):
```
┌─────────────────────────────────────────┐
│ PLAYER HEADER                           │
│ [Headshot]  Patrick Mahomes  #15        │
│ QB · KC · 6'3" 230 lbs · Texas Tech    │
│ Status: ACT · 9 yrs exp                │
├─────────────────────────────────────────┤
│ CAREER TOTALS (stats cards row)         │
│ [Pass Yd]  [Pass TD]  [Rush Yd]  [PPR] │
├─────────────────────────────────────────┤
│ SEASON-BY-SEASON TABLE                  │
│ Season | GP | PassYd | PassTD | RushYd… │
│ 2025   | 17 | 4,200  | 35     | 320    │
│ 2024   | 16 | 4,183  | 26     | 389    │
│ …                                       │
├─────────────────────────────────────────┤
│ RECENT GAME LOG (last 10 weeks)         │
│ Wk | Opp | PassYd | PassTD | RushYd…   │
├─────────────────────────────────────────┤
│ FANTASY RANKINGS (if available)         │
│ Type | Rank | ECR | Best | Worst        │
└─────────────────────────────────────────┘
```

**Link player names across the app:**
- Players page: wrap `player_name` in `<a href="/players/{id}">`
- Stats page: link player name → player detail
- Rankings page: link player name → player detail (requires looking up player ID from name/search)

### Components
- `PlayerHeader.svelte` — headshot, name, team, position, metadata badges
- `SeasonTable.svelte` — season-by-season aggregated stats
- `GameLog.svelte` — recent weekly stat lines

### Steps (ordered)
1. Add `GetPlayerSummary` to Go stat store (SQL aggregation query)
2. Add Go handler + route for the summary endpoint
3. Add `getPlayerSummary(id)` to frontend api.ts
4. Create `/players/[id]/+page.svelte` with header + career cards + tables
5. Link player names across all existing pages
6. Build & deploy

### Estimated effort: ~45 min

---

## 2. Season Comparison

**Goal:** Select 2–4 players and compare their stats side-by-side for a given season. Think ESPN player comparison tool.

### What already exists
- `listStats()` with `player_id` filter — can fetch stats for individual players
- `getLeaders()` — already does per-stat ranking

### Backend work

**New Go endpoint — multi-player season stats:**
- `GET /api/nflstats/compare?players=ID1,ID2,ID3&season=2025` 
- Returns each player's season totals (aggregated from weekly rows) in one response
- Add `ComparePlayers(ctx, playerIDs []string, season int)` to `StatStore`
- SQL: `SELECT player_id, SUM(passing_yards), SUM(passing_tds), ... FROM player_stats WHERE player_id IN (...) AND season = ? GROUP BY player_id`

### Frontend work

**New route: `web/src/routes/compare/+page.svelte`**

Layout:
```
┌─────────────────────────────────────────────────┐
│ SEASON COMPARISON                               │
│ Season: [2025 ▼]                                │
├─────────────────────────────────────────────────┤
│ PLAYER PICKER                                   │
│ [Search player...] → autocomplete dropdown      │
│ Selected: [Mahomes ✕] [Allen ✕] [Hurts ✕]      │
├──────────┬──────────┬──────────┬────────────────┤
│ Stat     │ Mahomes  │ Allen    │ Hurts          │
├──────────┼──────────┼──────────┼────────────────┤
│ Pass Yd  │ 4,200    │ 4,100    │ 3,800  ← best │
│ Pass TD  │ 35 ←best │ 28       │ 25             │
│ Rush Yd  │ 320      │ 580 ←b   │ 720 ← best    │
│ PPR Pts  │ 380      │ 350      │ 340            │
│ …        │          │          │                │
└──────────┴──────────┴──────────┴────────────────┘
```

**Player search autocomplete:**
- Use existing `listPlayers({ search: query, limit: 10 })` debounced on keystroke
- Show name + team + position in dropdown
- Clicking adds to comparison (max 4)

**Comparison table:**
- Columns = players, rows = stats
- Highlight the best value in each row (green/bold)
- Toggle between season totals vs per-game averages

### Navigation
- Add "Compare" to sidebar nav (icon: ⚖)
- Add "Compare" button to player detail page
- Add "Compare" checkbox column to players browse page

### Steps (ordered)
1. Add `ComparePlayers` to Go stat store
2. Add Go handler + route
3. Add `comparePlayers()` to frontend api.ts
4. Create player search autocomplete component
5. Create `/compare/+page.svelte` with picker + comparison table
6. Add nav link + cross-page "compare" buttons
7. Build & deploy

### Estimated effort: ~1 hour

---

## 3. Data Visualization (Charts)

**Goal:** Add interactive charts for stat trends — sparklines in tables, season-over-season line charts on player detail, and a standalone charts page for league-wide stats.

### Library choice
**Chart.js via svelte-chartjs** — lightweight, well-documented, works with Svelte 5. Alternatives: D3 (overkill), Recharts (React-only).

Install: `npm install chart.js svelte-chartjs`

### What already exists
- All the data is there — weekly stats, season-by-season, leaders — just not visualized
- Stats page has leaders by stat+season+week

### No backend changes needed
All data for charts is already available via existing endpoints. Frontend fetches and reshapes.

### Frontend work

**Phase A — Sparklines in tables (~20 min)**
- Tiny inline `<canvas>` (60×20px) showing a player's weekly stat trend
- Add to Stats browse table: one sparkline column per key stat
- Data: fetch weekly stats for visible players, draw mini line chart
- Component: `Sparkline.svelte` wrapping a tiny Chart.js line chart

**Phase B — Player detail charts (~20 min)**
- Requires Feature 1 (Player Detail Page) to exist
- Add to player detail: line chart of weekly fantasy points across the season
- Bar chart of season-by-season totals (passing yards, TDs, etc.)
- Component: `StatChart.svelte` — reusable, takes `{ labels, datasets }` prop

**Phase C — Standalone charts page (~30 min)**

New route: `web/src/routes/charts/+page.svelte`

```
┌──────────────────────────────────────────┐
│ // DATA VISUALIZATION                    │
│                                          │
│ [Stat: Pass Yards ▼] [Season: 2025 ▼]   │
│ [Position: QB ▼]                         │
│                                          │
│ ┌──────────────────────────────────┐     │
│ │  Top 10 QBs — Passing Yards     │     │
│ │  ████████████████████ Mahomes    │     │
│ │  █████████████████░░░ Allen      │     │
│ │  ████████████████░░░░ Hurts      │     │
│ │  …                               │     │
│ └──────────────────────────────────┘     │
│                                          │
│ ┌──────────────────────────────────┐     │
│ │  Weekly Trend — Top 5 QBs        │     │
│ │  (line chart, one line per player)│    │
│ └──────────────────────────────────┘     │
└──────────────────────────────────────────┘
```

- Horizontal bar chart: leader ranking for selected stat
- Multi-line chart: weekly trend for top N players
- Uses existing `getLeaders()` + `listStats()` endpoints

### Navigation
- Add "Charts" to sidebar nav (icon: 📈)

### Steps (ordered)
1. `npm install chart.js svelte-chartjs`
2. Create `Sparkline.svelte` component
3. Add sparklines to stats browse table
4. Create `StatChart.svelte` (reusable bar/line)
5. Create `/charts/+page.svelte` with leader bars + weekly trends
6. Add nav link
7. (Later) Add charts to player detail page once Feature 1 exists
8. Build & deploy

### Estimated effort: ~1 hour (all 3 phases)

---

## 4. Global Search

**Goal:** A single search bar in the header/sidebar that searches across players, stats, games — returning categorized results with direct links.

### What already exists
- `listPlayers({ search })` — name substring (`ILIKE`) search
- `listStats({ search })` — player name search within stats
- `listRankings({ search })` — player name search within rankings
- No game search by team/matchup exists yet

### Backend work

**New Go endpoint — unified search:**
- `GET /api/search?q=mahomes&limit=10`
- Returns categorized results in one response:
```json
{
  "players": [ { "id": 42, "player_name": "Patrick Mahomes", "team": "KC", "position": "QB" } ],
  "stats": { "total": 324, "sample": [ { "season": 2025, "week": 18, ... } ] },
  "games": { "total": 5, "sample": [ { "home_team": "KC", "away_team": "..." } ] },
  "rankings": { "total": 12, "sample": [ ... ] }
}
```
- Implementation: 4 parallel DB queries (one per entity), return top N of each
- Add `Search(ctx, query string, limit int)` that calls all 4 stores

**Add game search by team:**
- Modify `GameFilter` to support `search` parameter
- SQL: `WHERE away_team ILIKE $q OR home_team ILIKE $q`

### Frontend work

**Search component: `SearchBar.svelte`**
- Placed in layout sidebar, above nav links
- Input with debounced keystrokes (300ms)
- Dropdown overlay showing categorized results:

```
┌─────────────────────────────┐
│ 🔍 [mahom...]               │
├─────────────────────────────┤
│ PLAYERS                     │
│  Patrick Mahomes · QB · KC  │  → /players/42
├─────────────────────────────┤
│ STATS (324 results)         │
│  2025 Wk18: 312 pass yd    │  → /stats?search=mahomes
│  2025 Wk17: 289 pass yd    │  → …
├─────────────────────────────┤
│ GAMES (5 results)           │
│  KC vs BUF · 2025 Wk21     │  → /games?search=KC
├─────────────────────────────┤
│ View all results →          │
└─────────────────────────────┘
```

**Keyboard shortcuts:**
- `Ctrl+K` or `/` to focus the search bar
- `Escape` to close
- Arrow keys to navigate results
- `Enter` to go to selected result

### Steps (ordered)
1. Add game search support to Go GameStore + handler
2. Add unified search endpoint (Go handler + route)
3. Add `search()` to frontend api.ts
4. Create `SearchBar.svelte` component with dropdown
5. Add to `+layout.svelte` above nav links
6. Add keyboard shortcut support
7. Build & deploy

### Estimated effort: ~45 min

---

## Recommended Build Order

| Order | Feature                | Why                                     |
| ----- | ---------------------- | --------------------------------------- |
| 1     | **Player Detail**      | Foundation — other features link to it  |
| 2     | **Global Search**      | High daily-use value, enables discovery |
| 3     | **Season Comparison**  | Builds on player detail + search        |
| 4     | **Data Visualization** | Polish layer, builds on all above       |

Total estimated effort: ~3–4 hours across sessions.
