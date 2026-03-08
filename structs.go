package main

import (
	"database/sql"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
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
		"SELECT id, code, name FROM countries WHERE code = ? OR name = ?",
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
	ReviewParsed   sql.NullString
}

func (g Game) FindOrCreate(db *sql.DB) (Game, error) {
	// Pokus o nalezení existujícího záznamu
	err := db.QueryRow(
		"SELECT id, round, game_no, season_part_id, home_team_id, away_team_id, is_neutral_pitch, played_at, review_url, review_parsed FROM games WHERE home_team_id = ? AND away_team_id = ? AND played_at = ?",
		g.HomeTeamId, g.AwayTeamId, g.PlayedAt,
	).Scan(&g.Id, &g.Round, &g.GameNo, &g.SeasonPartId, &g.HomeTeamId, &g.AwayTeamId, &g.IsNeutralPitch, &g.PlayedAt, &g.ReviewUrl, &g.ReviewParsed)

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
		"INSERT INTO games (round, game_no, season_part_id, home_team_id, away_team_id, is_neutral_pitch, played_at, review_url, review_parsed) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)",
		g.Round, g.GameNo, g.SeasonPartId, g.HomeTeamId, g.AwayTeamId, g.IsNeutralPitch, g.PlayedAt, g.ReviewUrl, g.ReviewParsed,
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

func (gr GameResult) FindOrCreate(db *sql.DB) (GameResult, error) {
	tmp := GameResult{}

	// Pokus o nalezení existujícího záznamu
	err := db.QueryRow(
		"SELECT id, game_id, team_id, result_type_id, points FROM game_results WHERE game_id = ? AND team_id = ? AND result_type_id = ?",
		gr.GameId, gr.TeamId, gr.ResultTypeId,
	).Scan(&tmp.Id, &tmp.GameId, &tmp.TeamId, &tmp.ResultTypeId, &tmp.Points)

	if err == nil {
		// Záznam nalezen, aktualizujeme body
		gr.Id = tmp.Id
		_, err = db.Exec(
			"UPDATE game_results SET points = ? WHERE id = ?",
			gr.Points, tmp.Id,
		)
		return gr, err
	}

	if err != sql.ErrNoRows {
		// Jiná chyba než "not found"
		return gr, err
	}

	// Záznam neexistuje, vytvoříme nový
	result, err := db.Exec(
		"INSERT INTO game_results (game_id, team_id, result_type_id, points) VALUES (?, ?, ?, ?)",
		gr.GameId, gr.TeamId, gr.ResultTypeId, gr.Points,
	)
	if err != nil {
		return gr, err
	}

	lastId, err := result.LastInsertId()
	if err != nil {
		return gr, err
	}

	gr.Id = lastId

	return gr, nil
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
	Id            int64
	CountryId     int64
	Name          string
	Name2         string
	Name3         string
	LogoUrl       string
	LogoData      string
	ProfileUrl    string
	ProfileParsed sql.NullString
}

func (t Team) FindById(db *sql.DB) (Team, error) {
	err := db.QueryRow(
		"SELECT id, country_id, name, name_2, name_3, logo_url, logo_data, profile_url, profile_parsed FROM teams WHERE id = ?",
		t.Id,
	).Scan(&t.Id, &t.CountryId, &t.Name, &t.Name2, &t.Name3, &t.LogoUrl, &t.LogoData, &t.ProfileUrl, &t.ProfileParsed)

	return t, err
}

func (t Team) FindOrCreate(db *sql.DB) (Team, error) {
	// Pokus o nalezení existujícího záznamu
	err := db.QueryRow(
		"SELECT id, country_id, name, name_2, name_3, logo_url, logo_data, profile_url, profile_parsed FROM teams WHERE name = ? OR logo_url = ?",
		t.Name, t.LogoUrl,
	).Scan(&t.Id, &t.CountryId, &t.Name, &t.Name2, &t.Name3, &t.LogoUrl, &t.LogoData, &t.ProfileUrl, &t.ProfileParsed)

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
		"INSERT INTO teams (country_id, name, name_2, name_3, logo_url, logo_data, profile_url, profile_parsed) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
		t.CountryId, t.Name, t.Name2, t.Name3, t.LogoUrl, t.LogoData, t.ProfileUrl, nil,
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

// DownloadLogo stáhne logo týmu z LogoUrl a naplní LogoData
func (t *Team) DownloadLogo() error {
	if t.LogoUrl == "" {
		return fmt.Errorf("logo_url is empty")
	}

	resp, err := http.Get(t.LogoUrl)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("špatný stavový kód: %d", resp.StatusCode)
	}

	bytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	// Převod na base64 data URI
	mimeType := http.DetectContentType(bytes)
	base64Str := base64.StdEncoding.EncodeToString(bytes)
	t.LogoData = "data:" + mimeType + ";base64," + base64Str

	return nil
}

type Player struct {
	Id            int64
	FirstName     string
	LastName      string
	CountryId     int64
	Birthdate     string
	ProfileUrl    string
	ProfileParsed sql.NullString
	PhotoUrl      string
	PhotoData     string
}

func (p Player) FullName() string {
	return p.FirstName + " " + p.LastName
}

func (p *Player) DownloadPhoto() error {
	if p.PhotoUrl == "" {
		return fmt.Errorf("photo_url is empty")
	}

	resp, err := http.Get(p.PhotoUrl)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("špatný stavový kód: %d", resp.StatusCode)
	}

	bytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	// Převod na base64 data URI
	mimeType := http.DetectContentType(bytes)
	base64Str := base64.StdEncoding.EncodeToString(bytes)
	p.PhotoData = "data:" + mimeType + ";base64," + base64Str

	return nil
}

func (p Player) FindOrCreate(db *sql.DB) (Player, error) {
	var existing Player
	// Pokus o nalezení existujícího záznamu podle profile_url
	err := db.QueryRow(
		"SELECT id, first_name, last_name, country_id, birthdate, profile_url, profile_parsed, photo_url, photo_data FROM players WHERE profile_url = ?",
		p.ProfileUrl,
	).Scan(&existing.Id, &existing.FirstName, &existing.LastName, &existing.CountryId, &existing.Birthdate, &existing.ProfileUrl, &existing.ProfileParsed, &existing.PhotoUrl, &existing.PhotoData)

	if err == nil {
		// Záznam nalezen - aktualizujeme hodnoty, pokud jsou nové lepší
		needUpdate := false
		if p.CountryId > 0 && existing.CountryId != p.CountryId {
			existing.CountryId = p.CountryId
			needUpdate = true
		}
		if p.Birthdate != "" && existing.Birthdate != p.Birthdate {
			existing.Birthdate = p.Birthdate
			needUpdate = true
		}
		if needUpdate {
			_, _ = db.Exec(
				"UPDATE players SET country_id = ?, birthdate = ? WHERE id = ?",
				existing.CountryId, existing.Birthdate, existing.Id,
			)
		}
		return existing, nil
	}

	if err != sql.ErrNoRows {
		// Jiná chyba než "not found"
		return p, err
	}

	// Pokus o nalezení podle jména
	err = db.QueryRow(
		"SELECT id, first_name, last_name, country_id, birthdate, profile_url, profile_parsed, photo_url, photo_data FROM players WHERE first_name = ? AND last_name = ?",
		p.FirstName, p.LastName,
	).Scan(&existing.Id, &existing.FirstName, &existing.LastName, &existing.CountryId, &existing.Birthdate, &existing.ProfileUrl, &existing.ProfileParsed, &existing.PhotoUrl, &existing.PhotoData)

	if err == nil {
		// Záznam nalezen podle jména - aktualizujeme hodnoty
		needUpdate := false
		if p.ProfileUrl != "" && existing.ProfileUrl != p.ProfileUrl {
			existing.ProfileUrl = p.ProfileUrl
			needUpdate = true
		}
		if p.CountryId > 0 && existing.CountryId != p.CountryId {
			existing.CountryId = p.CountryId
			needUpdate = true
		}
		if p.Birthdate != "" && existing.Birthdate != p.Birthdate {
			existing.Birthdate = p.Birthdate
			needUpdate = true
		}
		if needUpdate {
			_, _ = db.Exec(
				"UPDATE players SET profile_url = ?, country_id = ?, birthdate = ? WHERE id = ?",
				existing.ProfileUrl, existing.CountryId, existing.Birthdate, existing.Id,
			)
		}
		return existing, nil
	}

	if err != sql.ErrNoRows {
		// Jiná chyba než "not found"
		return p, err
	}

	// Záznam neexistuje, vytvoříme nový
	result, err := db.Exec(
		"INSERT INTO players (first_name, last_name, country_id, birthdate, profile_url, profile_parsed, photo_url, photo_data) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
		p.FirstName, p.LastName, p.CountryId, p.Birthdate, p.ProfileUrl, nil, p.PhotoUrl, p.PhotoData,
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
	Id              int64
	PlayerId        int64
	TeamId          int64
	SeasonId        int64
	DateFrom        string
	DateTo          string
	Number          int
	Birthdate       string
	Position        string
	PreviousTeamId  int64
	SeasonsInLeague int
	Height          int
	GamesInSeason   int
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
