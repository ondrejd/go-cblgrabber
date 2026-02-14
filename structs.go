package main

import (
	"database/sql"
)

/*
 * Competition types
 */
const CT_CUP int = 1
const CT_LEAGUE int = 2
const CT_PLAYOFF int = 3
const CT_PLAYOUT int = 4

/*
 * Result types
 */
const RT_1ST_QUARTER int = 1
const RT_2ND_QUARTER int = 2
const RT_3RD_QUARTER int = 3
const RT_4TH_QUARTER int = 4
const RT_1ST_OVERTIME int = 5
const RT_2ND_OVERTIME int = 6
const RT_TOTAL int = 7

type Competition struct {
	Id        int64
	CountryId int64
	Name      string
}

func (c Competition) FindOrCreate(db *sql.DB) (Competition, error) {
	// Pokus o nalezení existujícího záznamu
	err := db.QueryRow(
		"SELECT id, country_id, name FROM competitions WHERE country_id = ? AND name = ?",
		c.CountryId, c.Name,
	).Scan(&c.Id, &c.CountryId, &c.Name)

	if err == nil {
		// Záznam nalezen
		return c, nil
	}

	if err != sql.ErrNoRows {
		// Jiná chyba než "not found"
		return c, err
	}

	// Záznam neexistuje, vytvoříme nový
	result, err := db.Exec(
		"INSERT INTO competitions (country_id, name) VALUES (?, ?)",
		c.CountryId, c.Name,
	)
	if err != nil {
		return c, err
	}

	lastId, err := result.LastInsertId()
	if err != nil {
		return c, err
	}

	c.Id = lastId

	return c, nil
}

type CompetitionType struct {
	Id  int64
	Key string
}

type Country struct {
	Id   int64
	Code string
	Name string
}

func (c Country) FindOrCreate(db *sql.DB) (Country, error) {
	// Pokus o nalezení existujícího záznamu
	err := db.QueryRow(
		"SELECT id, code, name FROM countries WHERE code = ? AND name = ?",
		c.Code, c.Name,
	).Scan(&c.Id, &c.Code, &c.Name)

	if err == nil {
		// Záznam nalezen
		return c, nil
	}

	if err != sql.ErrNoRows {
		// Jiná chyba než "not found"
		return c, err
	}

	// Záznam neexistuje, vytvoříme nový
	result, err := db.Exec(
		"INSERT INTO countries (code, name) VALUES (?, ?)",
		c.Code, c.Name,
	)
	if err != nil {
		return c, err
	}

	lastId, err := result.LastInsertId()
	if err != nil {
		return c, err
	}

	c.Id = lastId

	return c, nil
}

type Game struct {
	Id             int64
	Round          int // Číslo kola v sezóně
	GameNo         int // Číslo hry v sezóně
	SeasonPartId   int64
	HomeTeamId     int64
	AwayTeamId     int64
	IsNeutralPitch bool
	PlayedAt       string
	ReviewUrl      string
	IsReviewParsed bool
}

func (g Game) FindOrCreate(db *sql.DB) (Game, error) {
	// Pokus o nalezení existujícího záznamu
	err := db.QueryRow(
		"SELECT id, round, game_no, season_part_id, home_team_id, away_team_id, is_neutral_pitch, played_at, review_url, is_review_parsed FROM games WHERE home_team_id = ? AND away_team_id = ? AND played_at = ?",
		g.HomeTeamId, g.AwayTeamId, g.PlayedAt,
	).Scan(&g.Id, &g.Round, &g.GameNo, &g.SeasonPartId, &g.HomeTeamId, &g.AwayTeamId, &g.IsNeutralPitch, &g.PlayedAt, &g.ReviewUrl, &g.IsReviewParsed)

	if err == nil {
		// Záznam nalezen
		return g, nil
	}

	if err != sql.ErrNoRows {
		// Jiná chyba než "not found"
		return g, err
	}

	// Záznam neexistuje, vytvoříme nový
	result, err := db.Exec(
		"INSERT INTO games (round, game_no, season_part_id, home_team_id, away_team_id, is_neutral_pitch, played_at, review_url, is_review_parsed) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)",
		g.Round, g.GameNo, g.SeasonPartId, g.HomeTeamId, g.AwayTeamId, g.IsNeutralPitch, g.PlayedAt, g.ReviewUrl, g.IsReviewParsed,
	)
	if err != nil {
		return g, err
	}

	lastId, err := result.LastInsertId()
	if err != nil {
		return g, err
	}

	g.Id = lastId

	return g, nil
}

type GameResult struct {
	Id           int64
	GameId       int64
	TeamId       int64
	ResultTypeId int64
	Points       int
}

type ResultType struct {
	Id  int64
	Key string
}

type Season struct {
	Id            int64
	CompetitionId int64
	PreviousId    int64 // Předcházející sezóna (je-li)
	Name          string
	IsCurrent     bool
}

func (s Season) FindOrCreate(db *sql.DB) (Season, error) {
	// Pokus o nalezení existujícího záznamu
	err := db.QueryRow(
		"SELECT id, competition_id, previous_id, name, is_current FROM seasons WHERE competition_id = ? AND name = ?",
		s.CompetitionId, s.Name,
	).Scan(&s.Id, &s.CompetitionId, &s.PreviousId, &s.Name, &s.IsCurrent)

	if err == nil {
		// Záznam nalezen
		return s, nil
	}

	if err != sql.ErrNoRows {
		// Jiná chyba než "not found"
		return s, err
	}

	// Záznam neexistuje, vytvoříme nový
	result, err := db.Exec(
		"INSERT INTO seasons (competition_id, previous_id, name, is_current) VALUES (?, ?, ?, ?)",
		s.CompetitionId, s.PreviousId, s.Name, false,
	)
	if err != nil {
		return s, err
	}

	lastId, err := result.LastInsertId()
	if err != nil {
		return s, err
	}

	s.Id = lastId

	return s, nil
}

type SeasonPart struct {
	Id                int64
	SeasonId          int64
	SuccessorId       int64 // Předcházející část sezóny (je-li)
	CompetitionTypeId int64
	Name              string
	IsCurrent         bool
}

func (sp SeasonPart) FindOrCreate(db *sql.DB) (SeasonPart, error) {
	// Pokus o nalezení existujícího záznamu
	err := db.QueryRow(
		"SELECT id, season_id, successor_id, competition_type_id, name, is_current FROM season_parts WHERE season_id = ? AND name = ?",
		sp.SeasonId, sp.Name,
	).Scan(&sp.Id, &sp.SeasonId, &sp.SuccessorId, &sp.CompetitionTypeId, &sp.Name, &sp.IsCurrent)

	if err == nil {
		// Záznam nalezen
		return sp, nil
	}

	if err != sql.ErrNoRows {
		// Jiná chyba než "not found"
		return sp, err
	}

	// Záznam neexistuje, vytvoříme nový
	result, err := db.Exec(
		"INSERT INTO season_parts (season_id, successor_id, competition_type_id, name, is_current) VALUES (?, ?, ?, ?, ?)",
		sp.SeasonId, sp.SuccessorId, sp.CompetitionTypeId, sp.Name, false,
	)
	if err != nil {
		return sp, err
	}

	lastId, err := result.LastInsertId()
	if err != nil {
		return sp, err
	}

	sp.Id = lastId

	return sp, nil
}

type Team struct {
	Id              int64
	CountryId       int64
	Name            string
	Name2           string
	Name3           string
	LogoUrl         string
	LogoData        string
	ProfileUrl      string
	IsProfileParsed bool
}

func (t Team) FindById(db *sql.DB) (Team, error) {
	err := db.QueryRow(
		"SELECT id, country_id, name, name_2, name_3, logo_url, logo_data, profile_url, is_profile_parsed FROM teams WHERE id = ?",
		t.Id,
	).Scan(&t.Id, &t.CountryId, &t.Name, &t.Name2, &t.Name3, &t.LogoUrl, &t.LogoData, &t.ProfileUrl, &t.IsProfileParsed)

	return t, err
}

func (t Team) FindOrCreate(db *sql.DB) (Team, error) {
	// Pokus o nalezení existujícího záznamu
	err := db.QueryRow(
		"SELECT id, country_id, name, name_2, name_3, logo_url, logo_data, profile_url, is_profile_parsed FROM teams WHERE name = ? OR logo_url = ?",
		t.Name, t.LogoUrl,
	).Scan(&t.Id, &t.CountryId, &t.Name, &t.Name2, &t.Name3, &t.LogoUrl, &t.LogoData, &t.ProfileUrl, &t.IsProfileParsed)

	if err == nil {
		// Záznam nalezen
		return t, nil
	}

	if err != sql.ErrNoRows {
		// Jiná chyba než "not found"
		return t, err
	}

	// Záznam neexistuje, vytvoříme nový
	result, err := db.Exec(
		"INSERT INTO teams (country_id, name, name_2, name_3, logo_url, logo_data, profile_url, is_profile_parsed) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
		t.CountryId, t.Name, t.Name2, t.Name3, t.LogoUrl, t.LogoData, t.ProfileUrl, false,
	)
	if err != nil {
		return t, err
	}

	lastId, err := result.LastInsertId()
	if err != nil {
		return t, err
	}

	t.Id = lastId

	return t, nil
}

type Player struct {
	Id              int64
	FirstName       string
	LastName        string
	CountryId       int64
	Birthdate       string
	ProfileUrl      string
	IsProfileParsed bool
}

func (p Player) FullName() string {
	return p.FirstName + " " + p.LastName
}
func (p Player) FindOrCreate(db *sql.DB) (Player, error) {
	// Pokus o nalezení existujícího záznamu podle profile_url
	err := db.QueryRow(
		"SELECT id, first_name, last_name, country_id, birthdate, profile_url, is_profile_parsed FROM players WHERE profile_url = ?",
		p.ProfileUrl,
	).Scan(&p.Id, &p.FirstName, &p.LastName, &p.CountryId, &p.Birthdate, &p.ProfileUrl, &p.IsProfileParsed)

	if err == nil {
		// Záznam nalezen
		return p, nil
	}

	if err != sql.ErrNoRows {
		// Jiná chyba než "not found"
		return p, err
	}

	// Pokus o nalezení podle jména
	err = db.QueryRow(
		"SELECT id, first_name, last_name, country_id, birthdate, profile_url, is_profile_parsed FROM players WHERE first_name = ? AND last_name = ?",
		p.FirstName, p.LastName,
	).Scan(&p.Id, &p.FirstName, &p.LastName, &p.CountryId, &p.Birthdate, &p.ProfileUrl, &p.IsProfileParsed)

	if err == nil {
		// Záznam nalezen podle jména
		return p, nil
	}

	if err != sql.ErrNoRows {
		// Jiná chyba než "not found"
		return p, err
	}

	// Záznam neexistuje, vytvoříme nový
	result, err := db.Exec(
		"INSERT INTO players (first_name, last_name, country_id, birthdate, profile_url, is_profile_parsed) VALUES (?, ?, ?, ?, ?, ?)",
		p.FirstName, p.LastName, p.CountryId, p.Birthdate, p.ProfileUrl, false,
	)
	if err != nil {
		return p, err
	}

	lastId, err := result.LastInsertId()
	if err != nil {
		return p, err
	}

	p.Id = lastId

	return p, nil
}

type PlayerTeam struct {
	Id       int64
	PlayerId int64
	TeamId   int64
	DateFrom string
	DateTo   string
	Number   int
}

type GameTeamStats struct {
	Id         int64
	GameId     int64
	TeamId     int64
	TwoPtPct   float64
	ThreePtPct float64
	FtPct      float64
	Rebounds   int
	Turnovers  int
}

type GamePlayerStats struct {
	Id           int64
	GameId       int64
	TeamId       int64
	PlayerId     int64
	PlayerNumber int
	StatsJson    string
}

// PlayerStats represents parsed boxscore stats stored in game_player_stats.stats_json.
type PlayerStats map[string]string
