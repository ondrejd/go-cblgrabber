package main

import (
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	_ "github.com/mattn/go-sqlite3"
)

// TODO Toto pak nějak upravit, ať to tu není natvrdo...
const COUNTRY_ID int = 1
const COMPETITION_ID int = 1

const BASE_URL = "https://nbl.basketball"

func ConnectDb(database string) *sql.DB {
	db, err := sql.Open("sqlite3", database+"?_busy_timeout=1000&_journal_mode=WAL")
	if err != nil {
		log.Fatal(err)
	}

	return db
}

func InitDb(db *sql.DB) {
	// Definice všech SQL příkazů
	queries := []string{
		// Smazání tabulek (pokud existují)
		`DROP TABLE IF EXISTS "game_results"`,
		`DROP TABLE IF EXISTS "result_types"`,
		`DROP TABLE IF EXISTS "player_teams"`,
		`DROP TABLE IF EXISTS "players"`,
		`DROP TABLE IF EXISTS "game_team_stats"`,
		`DROP TABLE IF EXISTS "game_player_stats"`,
		`DROP TABLE IF EXISTS "games"`,
		`DROP TABLE IF EXISTS "teams"`,
		`DROP TABLE IF EXISTS "season_parts"`,
		`DROP TABLE IF EXISTS "seasons"`,
		`DROP TABLE IF EXISTS "competitions"`,
		`DROP TABLE IF EXISTS "competition_types"`,
		`DROP TABLE IF EXISTS "countries"`,
		// Vytvoření tabulek
		`CREATE TABLE "competition_types" (
			"id" integer primary key autoincrement not null,
			"key" varchar not null
		)`,
		`CREATE TABLE "competitions" (
			"id" integer primary key autoincrement not null,
			"country_id" integer not null,
			"previous_id" integer,
			"name" varchar not null,
			foreign key("country_id") references "countries"("id"),
			foreign key("previous_id") references "competitions"("id")
		)`,
		`CREATE TABLE "countries" (
			"id" integer primary key autoincrement not null,
			"code" varchar not null,
			"name" varchar not null
		)`,
		`CREATE TABLE "game_results" (
			"id" integer primary key autoincrement not null,
			"game_id" integer not null,
			"team_id" integer not null,
			"result_type_id" integer not null,
			"points" integer not null default 0,
			foreign key("game_id") references "games"("id"),
			foreign key("team_id") references "teams"("id"),
			foreign key("result_type_id") references "result_types"("id")
		)`,
		`CREATE TABLE "games" (
			"id" integer primary key autoincrement not null,
			"round" int,
			"game_no" int,
			"season_part_id" integer not null,
			"home_team_id" integer not null,
			"away_team_id" integer not null,
			"is_neutral_pitch" tinyint(1) not null default '0',
			"review_url" varchar,
			"played_at" datetime not null,
			"review_parsed" datetime,
			foreign key("season_part_id") references "season_parts"("id"),
			foreign key("home_team_id") references "teams"("id"),
			foreign key("away_team_id") references "teams"("id")
		)`,
		`CREATE TABLE "result_types" (
			"id" integer primary key autoincrement not null,
			"key" varchar not null
		)`,
		`CREATE TABLE "season_parts" (
			"id" integer primary key autoincrement not null,
			"season_id" integer not null,
			"previous_id" integer,
			"competition_type_id" integer not null,
			"is_current" tinyint(1) not null default '0',
			"name" varchar not null,
			foreign key("season_id") references "seasons"("id"),
			foreign key("previous_id") references "season_parts"("id"),
			foreign key("competition_type_id") references "competition_types"("id")
		)`,
		`CREATE TABLE "seasons" (
			"id" integer primary key autoincrement not null,
			"competition_id" integer not null,
			"previous_id" integer,
			"name" varchar not null,
			"is_current" tinyint(1) not null default '0',
			foreign key("competition_id") references "competitions"("id"),
			foreign key("previous_id") references "seasons"("id")
		)`,
		`CREATE TABLE "teams" (
			"id" integer primary key autoincrement not null,
			"country_id" integer not null,
			"name" varchar not null,
			"name_2" varchar,
			"name_3" varchar,
			"logo_url" varchar,
			"logo_data" blob,
			"profile_url" varchar,
			"profile_parsed" datetime,
			foreign key("country_id") references "countries"("id")
		)`,
		`CREATE TABLE "players" (
			"id" integer primary key autoincrement not null,
			"first_name" varchar not null,
			"last_name" varchar not null,
			"country_id" integer not null,
			"birthdate" varchar,
			"profile_url" varchar,
			"profile_parsed" datetime,
			"photo_url" varchar,
			"photo_data" blob,
			foreign key("country_id") references "countries"("id")
		)`,
		`CREATE TABLE "player_teams" (
			"id" integer primary key autoincrement not null,
			"player_id" integer not null,
			"team_id" integer not null,
			"season_id" integer,
			"date_from" date,
			"date_to" date,
			"number" integer,
			"birthdate" varchar,
			"position" varchar,
			"previous_team_id" integer,
			"seasons_in_league" integer,
			"height" integer,
			"games_in_season" integer,
			foreign key("player_id") references "players"("id"),
			foreign key("team_id") references "teams"("id"),
			foreign key("season_id") references "seasons"("id"),
			foreign key("previous_team_id") references "teams"("id")
		)`,
		`CREATE TABLE "game_team_stats" (
			"id" integer primary key autoincrement not null,
			"game_id" integer not null,
			"team_id" integer not null,
			"two_pt_pct" real,
			"three_pt_pct" real,
			"ft_pct" real,
			"rebounds" integer,
			"turnovers" integer,
			foreign key("game_id") references "games"("id"),
			foreign key("team_id") references "teams"("id")
		)`,
		`CREATE TABLE "game_player_stats" (
			"id" integer primary key autoincrement not null,
			"game_id" integer not null,
			"team_id" integer not null,
			"player_id" integer,
			"player_number" integer,
			"stats_json" text,
			foreign key("game_id") references "games"("id"),
			foreign key("team_id") references "teams"("id"),
			foreign key("player_id") references "players"("id")
		)`,
		// Defaultní data
		`INSERT INTO "countries" (id, code, name) VALUES ('1', 'CZE', 'Czechia')`,
		`INSERT INTO "competitions" (id, country_id, name) VALUES ('1', '1', 'NBL')`,
		`INSERT INTO "competition_types" (id, key) VALUES ('1', 'cup'),('2', 'league'),('3', 'playoff'),('4', 'playout')`,
		`INSERT INTO "result_types" (id, key) VALUES ('1', '1st_quarter'),('2', '2nd_quarter'),('3', '3rd_quarter'),('4', '4th_quarter'),('5', '1st_overtime'),('6', '2nd_overtime'),('7', 'total')`,
	}

	// Spuštění všech příkazů v rámci jedné transakce
	tx, _ := db.Begin()

	for _, query := range queries {
		_, err := tx.Exec(query)
		if err != nil {
			tx.Rollback()
			log.Fatal(err)
		}
	}

	err := tx.Commit()
	if err != nil {
		log.Fatal(err)
	}
}

var statValueRegex = regexp.MustCompile(`[-+]?[0-9]+(?:\.[0-9]+)?`)

func fullnameToFirstLast(fullName string) (string, string) {
	firstName := ""
	lastName := ""
	parts := strings.Fields(fullName)
	if len(parts) == 1 {
		lastName = parts[0]
	} else if len(parts) > 1 {
		firstName = strings.Join(parts[:len(parts)-1], " ")
		lastName = parts[len(parts)-1]
	}
	return firstName, lastName
}

func extractValueAfterName(text string, name string) (string, bool) {
	idx := strings.Index(text, name)
	if idx == -1 {
		return "", false
	}
	tail := text[idx+len(name):]
	match := statValueRegex.FindString(tail)
	if match == "" {
		return "", false
	}
	return match, true
}

func extractFirstTwoNumbers(text string) (string, string, bool) {
	matches := statValueRegex.FindAllString(text, -1)
	if len(matches) < 2 {
		return "", "", false
	}
	return matches[0], matches[1], true
}

func parseTeamStats(doc *goquery.Document, homeName string, awayName string) (float64, float64, float64, float64, float64, float64, int, int, int, int, error) {
	var homeTwoPct float64
	var awayTwoPct float64
	var homeThreePct float64
	var awayThreePct float64
	var homeFtPct float64
	var awayFtPct float64
	var homeReb int
	var awayReb int
	var homeTov int
	var awayTov int

	parsePair := func(label string) (string, string, bool) {
		var pairText string
		doc.Find("h4").EachWithBreak(func(_ int, h4 *goquery.Selection) bool {
			if strings.TrimSpace(h4.Text()) == label {
				pairText = strings.TrimSpace(h4.Parent().Text())
				return false
			}
			return true
		})
		if pairText == "" {
			return "", "", false
		}
		if homeName != "" && awayName != "" {
			homeVal, okHome := extractValueAfterName(pairText, homeName)
			awayVal, okAway := extractValueAfterName(pairText, awayName)
			if okHome && okAway {
				return homeVal, awayVal, true
			}
		}
		v1, v2, ok := extractFirstTwoNumbers(pairText)
		return v1, v2, ok
	}

	if homeVal, awayVal, ok := parsePair("Střelba za 2B"); ok {
		homeTwoPct, _ = strconv.ParseFloat(homeVal, 64)
		awayTwoPct, _ = strconv.ParseFloat(awayVal, 64)
	}
	if homeVal, awayVal, ok := parsePair("Střelba za 3B"); ok {
		homeThreePct, _ = strconv.ParseFloat(homeVal, 64)
		awayThreePct, _ = strconv.ParseFloat(awayVal, 64)
	}
	if homeVal, awayVal, ok := parsePair("Trestné hody"); ok {
		homeFtPct, _ = strconv.ParseFloat(homeVal, 64)
		awayFtPct, _ = strconv.ParseFloat(awayVal, 64)
	}
	if homeVal, awayVal, ok := parsePair("Doskoky"); ok {
		homeReb, _ = strconv.Atoi(homeVal)
		awayReb, _ = strconv.Atoi(awayVal)
	}
	if homeVal, awayVal, ok := parsePair("Ztráty"); ok {
		homeTov, _ = strconv.Atoi(homeVal)
		awayTov, _ = strconv.Atoi(awayVal)
	}

	return homeTwoPct, awayTwoPct, homeThreePct, awayThreePct, homeFtPct, awayFtPct, homeReb, awayReb, homeTov, awayTov, nil
}

type boxscoreRow struct {
	values     []string
	profileUrl string
}

func parseBoxscoreTable(table *goquery.Selection) ([]string, []boxscoreRow) {
	var headers []string
	var rows []boxscoreRow

	table.Find("thead th").Each(func(_ int, th *goquery.Selection) {
		headers = append(headers, strings.TrimSpace(th.Text()))
	})

	rowSel := table.Find("tbody tr")
	if rowSel.Length() == 0 {
		rowSel = table.Find("tr")
	}

	rowSel.Each(func(_ int, tr *goquery.Selection) {
		var values []string
		tr.Find("td").Each(func(_ int, td *goquery.Selection) {
			values = append(values, strings.TrimSpace(td.Text()))
		})
		if len(values) == 0 {
			return
		}

		profileUrl := ""
		if link := tr.Find("a[href*='/hrac/']").First(); link.Length() > 0 {
			profileUrl = strings.TrimSpace(link.AttrOr("href", ""))
			if profileUrl != "" && !strings.HasPrefix(profileUrl, "http") {
				profileUrl = BASE_URL + profileUrl
			}
		}

		rows = append(rows, boxscoreRow{values: values, profileUrl: profileUrl})
	})

	if len(headers) == 0 && len(rows) > 0 {
		for i := range rows[0].values {
			headers = append(headers, fmt.Sprintf("col_%d", i+1))
		}
	}

	return headers, rows
}

func mapRowToStats(headers []string, values []string) map[string]string {
	stats := make(map[string]string)
	for i, header := range headers {
		if i >= len(values) {
			break
		}
		key := strings.TrimSpace(header)
		if key == "" {
			key = fmt.Sprintf("col_%d", i+1)
		}
		stats[key] = values[i]
	}
	return stats
}

func parseGameDetailsFromDoc(db *sql.DB, gameId int64, homeTeamId int64, awayTeamId int64, doc *goquery.Document) error {
	homeTeam, _ := (Team{Id: homeTeamId}).FindById(db)
	awayTeam, _ := (Team{Id: awayTeamId}).FindById(db)
	var err error

	_, err = db.Exec("DELETE FROM game_team_stats WHERE game_id = ?", gameId)
	if err != nil {
		return err
	}
	_, err = db.Exec("DELETE FROM game_player_stats WHERE game_id = ?", gameId)
	if err != nil {
		return err
	}

	homeTwoPct, awayTwoPct, homeThreePct, awayThreePct, homeFtPct, awayFtPct, homeReb, awayReb, homeTov, awayTov, err := parseTeamStats(doc, homeTeam.Name, awayTeam.Name)
	if err != nil {
		return err
	}

	_, err = db.Exec(
		`INSERT INTO game_team_stats (game_id, team_id, two_pt_pct, three_pt_pct, ft_pct, rebounds, turnovers)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		gameId, homeTeamId, homeTwoPct, homeThreePct, homeFtPct, homeReb, homeTov,
	)
	if err != nil {
		return err
	}
	_, err = db.Exec(
		`INSERT INTO game_team_stats (game_id, team_id, two_pt_pct, three_pt_pct, ft_pct, rebounds, turnovers)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		gameId, awayTeamId, awayTwoPct, awayThreePct, awayFtPct, awayReb, awayTov,
	)
	if err != nil {
		return err
	}

	var parseTeamTable = func(teamName string, teamId int64) error {
		var table *goquery.Selection
		doc.Find("h4").EachWithBreak(func(_ int, h4 *goquery.Selection) bool {
			if strings.TrimSpace(h4.Text()) == teamName {
				// Try NextAllFiltered first
				table = h4.NextAllFiltered("table").First()
				if table.Length() == 0 {
					// If not found, try parent's next sibling table or nearest table
					table = h4.Parent().Find("table").First()
				}
				if table.Length() == 0 {
					// Last resort: find any nearby table
					table = h4.Parent().NextAllFiltered("table").First()
				}
				return false
			}
			return true
		})
		if table == nil || table.Length() == 0 {
			log.Printf("No table found for team: %s\n", teamName)
			return nil
		}
		log.Printf("Parsing table for team: %s, found %d rows\n", teamName, table.Find("tr").Length())

		headers, rows := parseBoxscoreTable(table)
		log.Printf("Team %s: found %d headers, %d data rows", teamName, len(headers), len(rows))
		for _, row := range rows {
			values := row.values
			if len(values) < 2 {
				continue
			}
			playerNumberStr := strings.TrimSpace(values[0])
			playerName := strings.TrimSpace(values[1])
			if playerName == "" {
				continue
			}
			// Skip team/coach rows
			if strings.Contains(playerName, "Tým/") {
				continue
			}

			var playerNumber sql.NullInt64
			if playerNumberStr != "" {
				if num, err := strconv.Atoi(playerNumberStr); err == nil {
					playerNumber = sql.NullInt64{Int64: int64(num), Valid: true}
				}
			}

			profileUrl := row.profileUrl
			firstName, lastName := fullnameToFirstLast(playerName)
			player, err := (Player{FirstName: firstName, LastName: lastName, ProfileUrl: profileUrl}).FindOrCreate(db)
			if err != nil {
				return err
			}
			playerId := player.Id

			statsMap := mapRowToStats(headers, values)
			statsJSON, err := json.Marshal(statsMap)
			if err != nil {
				return err
			}

			_, err = db.Exec(
				`INSERT INTO game_player_stats (game_id, team_id, player_id, player_number, stats_json)
				 VALUES (?, ?, ?, ?, ?)`,
				gameId, teamId, playerId, playerNumber, string(statsJSON),
			)
			if err != nil {
				return err
			}
		}
		return nil
	}

	if err := parseTeamTable(homeTeam.Name, homeTeamId); err != nil {
		return err
	}
	if err := parseTeamTable(awayTeam.Name, awayTeamId); err != nil {
		return err
	}

	_, err = db.Exec("UPDATE games SET review_parsed = datetime('now') WHERE id = ?", gameId)
	return err
}

func normalizeTeamProfileUrl(profileUrl string) string {
	profileUrl = strings.TrimSpace(profileUrl)
	if profileUrl == "" {
		return ""
	}
	if !strings.HasPrefix(profileUrl, "http") {
		return BASE_URL + profileUrl
	}
	return profileUrl
}

func updateTeamProfileUrlsFromReviewDoc(db *sql.DB, homeTeam Team, awayTeam Team, doc *goquery.Document) {
	homeTeamName := strings.TrimSpace(homeTeam.Name)
	awayTeamName := strings.TrimSpace(awayTeam.Name)
	if homeTeamName == "" || awayTeamName == "" {
		return
	}

	homeTeamProfileUrl := ""
	awayTeamProfileUrl := ""

	doc.Find("a[href*='/tym/']").Each(func(_ int, a *goquery.Selection) {
		linkText := strings.TrimSpace(a.Text())
		if homeTeamProfileUrl == "" && strings.EqualFold(linkText, homeTeamName) {
			homeTeamProfileUrl = normalizeTeamProfileUrl(a.AttrOr("href", ""))
		}
		if awayTeamProfileUrl == "" && strings.EqualFold(linkText, awayTeamName) {
			awayTeamProfileUrl = normalizeTeamProfileUrl(a.AttrOr("href", ""))
		}
	})

	if homeTeamProfileUrl != "" && homeTeam.ProfileUrl != homeTeamProfileUrl {
		_, _ = db.Exec("UPDATE teams SET profile_url = ? WHERE id = ?", homeTeamProfileUrl, homeTeam.Id)
	}
	if awayTeamProfileUrl != "" && awayTeam.ProfileUrl != awayTeamProfileUrl {
		_, _ = db.Exec("UPDATE teams SET profile_url = ? WHERE id = ?", awayTeamProfileUrl, awayTeam.Id)
	}
}

func importGameReviews(db *sql.DB, limit int) {
	rows, err := db.Query(`
		SELECT id, home_team_id, away_team_id, review_url
		FROM games
		WHERE review_url IS NOT NULL AND review_url != ''
		  AND (review_parsed IS NULL OR review_parsed < datetime('now', '-1 year'))
		LIMIT ?
	`, limit)
	if err != nil {
		log.Fatal(err)
	}

	type gameRow struct {
		gameId     int64
		homeTeamId int64
		awayTeamId int64
		reviewUrl  string
	}
	var games []gameRow
	for rows.Next() {
		var gameId int64
		var homeTeamId int64
		var awayTeamId int64
		var reviewUrl string
		if err := rows.Scan(&gameId, &homeTeamId, &awayTeamId, &reviewUrl); err != nil {
			_ = rows.Close()
			log.Fatal(err)
		}
		games = append(games, gameRow{
			gameId:     gameId,
			homeTeamId: homeTeamId,
			awayTeamId: awayTeamId,
			reviewUrl:  reviewUrl,
		})
	}
	_ = rows.Close()

	for _, game := range games {
		if game.reviewUrl == "" {
			continue
		}

		url := game.reviewUrl
		if !strings.HasPrefix(url, "http") {
			url = BASE_URL + url
		}
		res, err := http.Get(url)
		if err != nil {
			log.Printf("Failed to download review for game %d: %v", game.gameId, err)
			continue
		}
		func() {
			defer res.Body.Close()
			doc, err := goquery.NewDocumentFromReader(res.Body)
			if err != nil {
				log.Printf("Failed to parse review HTML for game %d: %v", game.gameId, err)
				return
			}

			homeTeam, err := (Team{Id: game.homeTeamId}).FindById(db)
			if err != nil {
				log.Printf("Failed to load home team for game %d: %v", game.gameId, err)
				return
			}
			awayTeam, err := (Team{Id: game.awayTeamId}).FindById(db)
			if err != nil {
				log.Printf("Failed to load away team for game %d: %v", game.gameId, err)
				return
			}

			updateTeamProfileUrlsFromReviewDoc(db, homeTeam, awayTeam, doc)

			if err := parseGameDetailsFromDoc(db, game.gameId, game.homeTeamId, game.awayTeamId, doc); err != nil {
				log.Printf("Failed to parse game %d: %v", game.gameId, err)
			}
		}()
	}
}

func parseHomeAwayTeams(db *sql.DB, td *goquery.Selection) (int64, int64) {
	var homeTeamLogoUrl string = ""
	var awayTeamLogoUrl string = ""

	sel := td.Find("img")
	if sel.Length() == 2 {
		homeTeamLogoUrl = sel.Eq(0).AttrOr("src", "")
		if homeTeamLogoUrl != "" {
			// Extrahujeme skutečný URL obrázku (pokud je v URL parametru "file=")
			homeTeamLogoUrl = extractFileParameterFromUrl(homeTeamLogoUrl)
			if !strings.HasPrefix(homeTeamLogoUrl, "http") {
				homeTeamLogoUrl = BASE_URL + homeTeamLogoUrl
			}
		}
		awayTeamLogoUrl = sel.Eq(1).AttrOr("src", "")
		if awayTeamLogoUrl != "" {
			// Extrahujeme skutečný URL obrázku (pokud je v URL parametru "file=")
			awayTeamLogoUrl = extractFileParameterFromUrl(awayTeamLogoUrl)
			if !strings.HasPrefix(awayTeamLogoUrl, "http") {
				awayTeamLogoUrl = BASE_URL + awayTeamLogoUrl
			}
		}
	}

	var homeTeamName string = ""
	var awayTeamName string = ""

	sel = td.Find("div > div > div")
	if sel.Length() == 2 {
		homeTeamName = sel.Eq(0).Text()
		awayTeamName = sel.Eq(1).Text()
	}

	homeTeam, _ := (Team{CountryId: int64(COUNTRY_ID), Name: homeTeamName, LogoUrl: homeTeamLogoUrl}).FindOrCreate(db)
	awayTeam, _ := (Team{CountryId: int64(COUNTRY_ID), Name: awayTeamName, LogoUrl: awayTeamLogoUrl}).FindOrCreate(db)

	return homeTeam.Id, awayTeam.Id
}

func parseAndSaveGameResults(db *sql.DB, td1 *goquery.Selection, td2 *goquery.Selection, game Game) {
	// Celkový výsledek
	var totalHome int = 0
	var totalAway int = 0

	a1 := strings.TrimSpace(td1.Find("a").Text())
	a2 := strings.TrimSpace(td1.Find("a div").Text())

	if strings.Index(a1, a2) == 0 {
		totalHome, _ = strconv.Atoi(a2)
		totalAway, _ = strconv.Atoi(strings.TrimSpace(strings.Replace(a1, a2, "", -1)))
	} else {
		totalAway, _ = strconv.Atoi(a2)
		totalHome, _ = strconv.Atoi(strings.TrimSpace(strings.Replace(a1, a2, "", -1)))
	}

	_, _ = (GameResult{GameId: game.Id, TeamId: game.HomeTeamId, ResultTypeId: int64(RT_TOTAL), Points: totalHome}).FindOrCreate(db)
	_, _ = (GameResult{GameId: game.Id, TeamId: game.AwayTeamId, ResultTypeId: int64(RT_TOTAL), Points: totalAway}).FindOrCreate(db)

	// Výsledky jednotlivých čtvrtin a případných prodloužení
	sel := td2.Find("a > span div")
	selLen := sel.Length()

	if selLen < 2 {
		// Toto se stává naprosto výjimečně (např. diskvalifikace).
		return
	}

	pointsHome := [7]int{-1, -1, -1, -1, -1, -1, -1}
	pointsAway := [7]int{-1, -1, -1, -1, -1, -1, -1}

	sel.Each(func(i int, div *goquery.Selection) {
		html, _ := div.Html()
		parts := strings.Split(strings.TrimSpace(html), "<br/>")
		pointsHome[i], _ = strconv.Atoi(parts[0])
		pointsAway[i], _ = strconv.Atoi(parts[1])

		if i > 0 && i < selLen {
			pointsHome[i] = pointsHome[i] - sumIntArray(pointsHome[0:i])
			pointsAway[i] = pointsAway[i] - sumIntArray(pointsAway[0:i])
		}
	})

	pointsHome[selLen] = totalHome - sumIntArray(pointsHome[0:selLen])
	pointsAway[selLen] = totalAway - sumIntArray(pointsAway[0:selLen])

	for i := range 6 {
		if pointsHome[i] != -1 && pointsAway[i] != -1 {
			var resultType int
			switch i {
			case 0:
				resultType = RT_1ST_QUARTER
			case 1:
				resultType = RT_2ND_QUARTER
			case 2:
				resultType = RT_3RD_QUARTER
			case 3:
				resultType = RT_4TH_QUARTER
			case 4:
				resultType = RT_1ST_OVERTIME
			case 5:
				resultType = RT_2ND_OVERTIME
			}

			_, _ = (GameResult{GameId: game.Id, TeamId: game.HomeTeamId, ResultTypeId: int64(resultType), Points: pointsHome[i]}).FindOrCreate(db)
			_, _ = (GameResult{GameId: game.Id, TeamId: game.AwayTeamId, ResultTypeId: int64(resultType), Points: pointsAway[i]}).FindOrCreate(db)
		}
	}
}

func sumIntArray(numbers []int) int {
	result := 0
	for i := 0; i < len(numbers); i++ {
		result += numbers[i]
	}
	return result
}

// extractFileParameterFromUrl parsuje URL a hledá parametr "file="
// Pokud je nalezen, vrátí jeho hodnotu, jinak vrátí původní URL
func extractFileParameterFromUrl(urlStr string) string {
	idx := strings.Index(urlStr, "file=")
	if idx == -1 {
		// Parametr "file" není v URL, vrátíme původní URL
		return urlStr
	}

	// Posuneme se na začátek hodnoty parametru
	valueStart := idx + len("file=")

	// Hledáme konec hodnoty (buď "&" nebo konec stringu)
	valueEnd := strings.IndexByte(urlStr[valueStart:], '&')
	if valueEnd == -1 {
		// Není "&", takže hodnota pokračuje do konce stringu
		return urlStr[valueStart:]
	}

	// Vrátíme hodnotu mezi valueStart a valueEnd
	return urlStr[valueStart : valueStart+valueEnd]
}

// Přeformátuj datum/čas z "11. 9. 2020 Pá 18:00" na "2020-09-11 18:00:00"
// convertBirthdateFormat konvertuje datum narození z "d. m. yyyy" na "yyyy-mm-dd"
func convertBirthdateFormat(dt string) (string, error) {
	dt = strings.TrimSpace(dt)
	if dt == "" {
		return "", fmt.Errorf("empty date")
	}

	// Parsování formátu "7. 3. 2000"
	parts := strings.Fields(dt)

	// Pokud máme jen rok (např. "2002"), vrátíme prázdný řetězec - není to kompletní datum
	if len(parts) == 1 {
		// Zkusíme zda je to platný rok
		year, err := strconv.Atoi(parts[0])
		if err == nil && year >= 1900 && year <= 2100 {
			// Je to validní rok, ale není to kompletní datum - vrátíme prázdný řetězec
			return "", nil
		}
		return "", fmt.Errorf("invalid date format: %s", dt)
	}

	if len(parts) < 3 {
		return "", fmt.Errorf("invalid date format: %s", dt)
	}

	// Extrahujeme den, měsíc, rok
	day, err1 := strconv.Atoi(strings.TrimSuffix(parts[0], "."))
	month, err2 := strconv.Atoi(strings.TrimSuffix(parts[1], "."))
	year, err3 := strconv.Atoi(parts[2])

	if err1 != nil || err2 != nil || err3 != nil || day < 1 || day > 31 || month < 1 || month > 12 || year < 1800 || year > 2100 {
		return "", fmt.Errorf("invalid date values: d=%d, m=%d, y=%d", day, month, year)
	}

	// Formátujeme na YYYY-MM-DD
	result := fmt.Sprintf("%04d-%02d-%02d", year, month, day)

	// Validujeme že je to platné datum
	_, err := time.Parse("2006-01-02", result)
	return result, err
}

func convertPlayedAtDateFormat(dt string) (string, error) {
	ret := ""
	parts := strings.Split(strings.TrimSpace(strings.ReplaceAll(dt, "\n", "")), " ")
	partsLen := len(parts)

	if partsLen >= 5 {
		// YYYY
		ret = parts[2] + "-"

		// MM
		m, _ := strconv.Atoi(strings.ReplaceAll(parts[1], ".", ""))
		if m < 10 {
			ret = ret + "0" + strconv.Itoa(m) + "-"
		} else {
			ret = ret + strconv.Itoa(m) + "-"
		}

		// DD
		d, _ := strconv.Atoi(strings.ReplaceAll(parts[0], ".", ""))
		if d < 10 {
			ret = ret + "0" + strconv.Itoa(d) + " "
		} else {
			ret = ret + strconv.Itoa(d) + " "
		}

		// H:i:s
		ret = ret + parts[partsLen-1] + ":00"
	}

	// Datum zkontrolujeme - chybu řešíme v místě použití
	_, err := time.Parse(time.DateTime, ret)

	return ret, err
}

func importTeamPlayers(db *sql.DB, year string, limit int) {
	year = strings.Split(year, "/")[0]

	// 0. Vytvoříme/najdeme sezónu
	yearNum, _ := strconv.Atoi(year)
	seasonName := year + "/" + fmt.Sprintf("%02d", (yearNum+1)%100)
	season, err := (Season{Name: seasonName, CompetitionId: int64(COMPETITION_ID)}).FindOrCreate(db)
	if err != nil {
		log.Fatal(err)
	}

	// 1. Načteme týmy z databáze
	rows, err := db.Query(`
		SELECT id, name, profile_url, logo_url, logo_data 
		FROM teams 
		WHERE profile_url IS NOT NULL AND profile_url != ''
		  AND (profile_parsed IS NULL OR profile_parsed < datetime('now', '-1 year'))
		LIMIT ?
	`, limit)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	var teams []Team
	for rows.Next() {
		var t Team
		err = rows.Scan(&t.Id, &t.Name, &t.ProfileUrl, &t.LogoUrl, &t.LogoData)
		if err != nil {
			log.Fatal(err)
		}
		teams = append(teams, t)
	}

	fmt.Printf("Nalezeno %d týmů ke zpracování hráčů pro sezónu %s.\n", len(teams), seasonName)

	// 2. Pro každý tým parsujeme hráče
	for _, team := range teams {
		fmt.Printf("Parsuju hráče pro tým %s (ID %d) za sezonu %s\n", team.Name, team.Id, seasonName)

		url := strings.TrimSpace(team.ProfileUrl)
		// TODO V databázi by už měly být s "https://..." Zkouknout v debuggeru...
		if !strings.HasPrefix(url, "http") {
			url = BASE_URL + url
		}
		url = url + "?y=" + year

		res, err := http.Get(url)
		if err != nil {
			log.Printf("Chyba při stahování %s: %v\n", url, err)
			continue
		}
		defer res.Body.Close()

		doc, err := goquery.NewDocumentFromReader(res.Body)
		if err != nil {
			log.Printf("Chyba při parsování %s: %v\n", url, err)
			continue
		}

		// 3. Parsujeme tabulku v tab-pane-one
		table := doc.Find("#tab-pane-one table").First()
		if table.Length() == 0 {
			log.Printf("Tabulka s hráči nebyla nalezena pro tým %s\n", team.Name)
			continue
		}

		headers, rows := parseBoxscoreTable(table)

		for _, row := range rows {
			if len(row.values) < 2 {
				continue
			}

			// Extrakt dat z buněk
			number := strings.TrimSpace(row.values[0])
			fullName := strings.TrimSpace(row.values[1])

			if fullName == "" {
				continue
			}

			playerProfileUrl := row.profileUrl

			// Mapujeme statistiky do mapy
			statsMap := mapRowToStats(headers, row.values)

			birthdateValue := ""
			if val, ok := statsMap["datum narození"]; ok {
				birthdateValue = strings.TrimSpace(val)
			}

			// Formátujeme datum narození na YYYY-MM-DD
			var birthdateFormatted string
			if birthdateValue != "" {
				formatted, err := convertBirthdateFormat(birthdateValue)
				if err != nil {
					// Logujeme jen skutečné chyby (ne když je to jen rok)
					log.Printf("Chyba při formatování data %s pro hráče %s: %v\n", birthdateValue, fullName, err)
				} else if formatted != "" {
					// Uložíme jen pokud máme kompletní datum (ne jen rok)
					birthdateFormatted = formatted
				}
			}

			// Vytvoříme nebo najdeme hráče
			firstName, lastName := fullnameToFirstLast(fullName)
			player, err := (Player{
				FirstName:  firstName,
				LastName:   lastName,
				CountryId:  int64(COUNTRY_ID),
				Birthdate:  birthdateFormatted,
				ProfileUrl: playerProfileUrl,
			}).FindOrCreate(db)
			if err != nil || player.Id == 0 {
				log.Printf("Chyba při vytváření hráče %s: %v\n", fullName, err)
				continue
			}
			playerId := player.Id

			// Parsujeme číslo dresu
			playerNumber := 0
			if number != "" {
				playerNumber, _ = strconv.Atoi(number)
			}

			// Extrahujeme hodnoty do samostatných proměnných
			birthdate := sql.NullString{}
			if birthdateFormatted != "" {
				birthdate = sql.NullString{String: birthdateFormatted, Valid: true}
			}

			position := sql.NullString{}
			if val, ok := statsMap["post"]; ok && val != "" {
				position = sql.NullString{String: val, Valid: true}
			}

			// Předchozí klub - hledáme v databázi nebo vytvoříme
			var previousTeamId sql.NullInt64
			if prevClubName, ok := statsMap["předchozí klub"]; ok && prevClubName != "" {
				prevTeam, err := (Team{CountryId: int64(COUNTRY_ID), Name: prevClubName}).FindOrCreate(db)
				if err == nil {
					previousTeamId = sql.NullInt64{Int64: prevTeam.Id, Valid: true}
				}
			}

			seasonsInLeague := sql.NullInt64{}
			if val, ok := statsMap["sezony v lize"]; ok && val != "" {
				if num, err := strconv.Atoi(val); err == nil {
					seasonsInLeague = sql.NullInt64{Int64: int64(num), Valid: true}
				}
			}

			height := sql.NullInt64{}
			if val, ok := statsMap["výška"]; ok && val != "" {
				// Odstraníme " cm" a převedeme na číslo
				val = strings.TrimSpace(strings.Replace(val, " cm", "", 1))
				if num, err := strconv.Atoi(val); err == nil {
					height = sql.NullInt64{Int64: int64(num), Valid: true}
				}
			}

			gamesInSeason := sql.NullInt64{}
			if val, ok := statsMap["zápasů v sezoně"]; ok && val != "" {
				if num, err := strconv.Atoi(val); err == nil {
					gamesInSeason = sql.NullInt64{Int64: int64(num), Valid: true}
				}
			}

			// Uložení do databáze - nejdřív zkontrolujeme, zda záznam existuje
			_, err = db.Exec(
				`INSERT OR REPLACE INTO player_teams (player_id, team_id, season_id, number, birthdate, position, previous_team_id, seasons_in_league, height, games_in_season) 
				 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
				playerId, team.Id, season.Id, playerNumber, birthdate, position, previousTeamId, seasonsInLeague, height, gamesInSeason,
			)
			if err != nil {
				log.Printf("Chyba při ukládání hráče %s do týmu: %v\n", fullName, err)
			}
		}
		res.Body.Close()

		// Stáhneme logo týmu, pokud ještě není v databázi
		if team.LogoUrl != "" && team.LogoData == "" {
			// Extrahujeme skutečný URL loga (pokud je v URL parametru "file=")
			logoUrl := extractFileParameterFromUrl(team.LogoUrl)
			if !strings.HasPrefix(logoUrl, "http") {
				logoUrl = BASE_URL + logoUrl
			}
			fmt.Printf("Stahuji logo pro tým %s (ID %d) z: %s\n", team.Name, team.Id, logoUrl)
			team.LogoUrl = logoUrl
			err := team.DownloadLogo()
			if err != nil {
				log.Printf("Nepodařilo se stáhnout logo pro tým %s: %v\n", team.Name, err)
			} else {
				// Uložení do databáze
				_, err = db.Exec("UPDATE teams SET logo_data = ? WHERE id = ?", team.LogoData, team.Id)
				if err != nil {
					log.Printf("Chyba při ukládání loga do DB pro tým %s: %v\n", team.Name, err)
				} else {
					fmt.Printf("Logo pro tým %s úspěšně uloženo.\n", team.Name)
				}
			}
		}

		// Označíme tým jako parsovaný
		_, err = db.Exec("UPDATE teams SET profile_parsed = datetime('now') WHERE id = ?", team.Id)
		if err != nil {
			log.Printf("Chyba při označení týmu %s jako parsovaný: %v\n", team.Name, err)
		}
	}
}

func importPlayerDetails(db *sql.DB, limit int) {
	rows, err := db.Query(`
		SELECT id, first_name, last_name, country_id, birthdate, profile_url
		FROM players
		WHERE profile_url IS NOT NULL AND profile_url != ''
		  AND (profile_parsed IS NULL OR profile_parsed < datetime('now', '-1 year'))
		LIMIT ?
	`, limit)
	if err != nil {
		log.Fatal(err)
	}

	var players []Player
	for rows.Next() {
		var p Player
		err = rows.Scan(&p.Id, &p.FirstName, &p.LastName, &p.CountryId, &p.Birthdate, &p.ProfileUrl)
		if err != nil {
			log.Fatal(err)
		}
		players = append(players, p)
	}
	_ = rows.Close()

	fmt.Printf("Nalezeno %d hráčů ke zpracování detailů.\n", len(players))

	for _, player := range players {
		fmt.Printf("Parsuju detaily pro hráče %s %s (ID %d)\n", player.FirstName, player.LastName, player.Id)

		url := strings.TrimSpace(player.ProfileUrl)
		if !strings.HasPrefix(url, "http") {
			url = BASE_URL + url
		}

		res, err := http.Get(url)
		if err != nil {
			log.Printf("Chyba při stahování %s: %v\n", url, err)
			continue
		}
		defer res.Body.Close()

		doc, err := goquery.NewDocumentFromReader(res.Body)
		if err != nil {
			log.Printf("Chyba při parsování %s: %v\n", url, err)
			res.Body.Close()
			continue
		}

		// Parsujeme informace o hráči
		var birthdate string
		var countryId int64 = int64(COUNTRY_ID) // Default
		var countryName string
		var photoUrl string

		// Hledáme datum narození - očekáváme formát "Datum narození: 15. 3. 1995"
		doc.Find(".player-info, .profile-info, p, div").Each(func(_ int, s *goquery.Selection) {
			text := s.Text()
			// Datum narození
			if strings.Contains(text, "Datum narození") || strings.Contains(text, "Narození") {
				// Extrahujeme datum
				parts := strings.Split(text, ":")
				if len(parts) >= 2 {
					// Vezmeme jen první řádek a validujeme formát data
					dateStr := strings.TrimSpace(strings.Split(parts[1], "\n")[0])
					// Validace: musí obsahovat tečky a čísla (formát "d. m. yyyy")
					if strings.Contains(dateStr, ".") && len(dateStr) < 20 {
						birthdate = dateStr
					}
				}
			}
			// Země/národnost
			if strings.Contains(text, "Národnost") || strings.Contains(text, "Země") {
				// Extrahujeme název země
				parts := strings.Split(text, ":")
				if len(parts) >= 2 {
					// Vezmeme jen první řádek
					countryName = strings.TrimSpace(strings.Split(parts[1], "\n")[0])
				}
			}
		})

		// Hledáme fotografii hráče
		doc.Find("img").Each(func(_ int, s *goquery.Selection) {
			if photoUrl != "" {
				return // Už jsme našli foto
			}
			src, exists := s.Attr("src")
			if !exists || src == "" {
				return
			}
			// Vyhledáváme typické atributy pro fotografii hráče
			alt, _ := s.Attr("alt")
			class, _ := s.Attr("class")

			// Kontrola alt textu nebo třídy obsahující "photo", "player", "portrait" atp.
			lowerAlt := strings.ToLower(alt)
			lowerClass := strings.ToLower(class)
			lowerSrc := strings.ToLower(src)

			if strings.Contains(lowerAlt, "photo") || strings.Contains(lowerAlt, "player") || strings.Contains(lowerAlt, "portrait") ||
				strings.Contains(lowerClass, "photo") || strings.Contains(lowerClass, "player") || strings.Contains(lowerClass, "portrait") ||
				strings.Contains(lowerSrc, "player") || strings.Contains(lowerSrc, "photo") {
				// Extrahujeme skutečný URL obrázku (pokud je v URL parametru "file=")
				photoUrl = extractFileParameterFromUrl(src)
			}
		})

		// Pokud jsme našli název země, pokusíme se ji najít/vytvořit v databázi
		if countryName != "" {
			code, name := normalizeCountryName(countryName)
			if code != "" {
				country, err := (Country{Code: code, Name: name}).FindOrCreate(db)
				if err != nil {
					log.Printf("Chyba při hledání/vytváření země %s: %v\n", countryName, err)
				} else {
					countryId = country.Id
				}
			}
		}

		// Konvertujeme birthdate do formátu YYYY-MM-DD
		var birthdateFormatted string
		if birthdate != "" {
			formatted, err := convertBirthdateFormat(birthdate)
			if err != nil {
				// Logujeme jen skutečné chyby (ne když je to jen rok)
				log.Printf("Chyba při formatování data %s pro hráče %s %s: %v\n", birthdate, player.FirstName, player.LastName, err)
			} else if formatted != "" {
				// Uložíme jen pokud máme kompletní datum (ne jen rok)
				birthdateFormatted = formatted
			}
		}

		// Stáhneme fotografii hráče, pokud máme URL
		var playerPhotoData string
		var normalizedPhotoUrl string
		if photoUrl != "" {
			// Normalizujeme URL
			normalizedPhotoUrl = strings.TrimSpace(photoUrl)
			// Extrahujeme skutečný URL obrázku (pokud je v URL parametru "file=")
			normalizedPhotoUrl = extractFileParameterFromUrl(normalizedPhotoUrl)
			if !strings.HasPrefix(normalizedPhotoUrl, "http") {
				normalizedPhotoUrl = BASE_URL + normalizedPhotoUrl
			}

			fmt.Printf("Stahuji foto pro hráče %s %s z: %s\n", player.FirstName, player.LastName, normalizedPhotoUrl)
			playerObj := Player{PhotoUrl: normalizedPhotoUrl}
			err := playerObj.DownloadPhoto()
			if err != nil {
				log.Printf("Nepodařilo se stáhnout foto pro hráče %s %s: %v\n", player.FirstName, player.LastName, err)
			} else {
				playerPhotoData = playerObj.PhotoData
				fmt.Printf("Foto pro hráče %s %s úspěšně staženo.\n", player.FirstName, player.LastName)
			}
		}

		// Aktualizujeme hráče v databázi
		needsUpdate := false
		if birthdateFormatted != "" && birthdateFormatted != player.Birthdate {
			needsUpdate = true
		}
		if countryId > 0 && countryId != player.CountryId {
			needsUpdate = true
		}
		if normalizedPhotoUrl != "" {
			needsUpdate = true
		}
		if playerPhotoData != "" {
			needsUpdate = true
		}

		if needsUpdate {
			_, err = db.Exec(
				"UPDATE players SET birthdate = ?, country_id = ?, photo_url = ?, photo_data = ? WHERE id = ?",
				birthdateFormatted, countryId, normalizedPhotoUrl, playerPhotoData, player.Id,
			)
			if err != nil {
				log.Printf("Chyba při aktualizaci hráče %s %s: %v\n", player.FirstName, player.LastName, err)
			}
		}

		// Označíme hráče jako parsovaného
		_, err = db.Exec("UPDATE players SET profile_parsed = datetime('now') WHERE id = ?", player.Id)
		if err != nil {
			log.Printf("Chyba při označení hráče %s %s jako parsovaný: %v\n", player.FirstName, player.LastName, err)
		}

		res.Body.Close()
	}
}

func importSeasonGames(db *sql.DB, initDb bool, season string) {
	// Případně vytvoříme stukturu a základní data
	if initDb == true {
		InitDb(db)
	}

	// Pokračujeme se stahováním zápasů a výsledků pro zadanou sezónu
	year := strings.Split(season, "/")[0]

	// Získáme ID sezóny, kterou chceme stáhnout
	seasonObj, err := (Season{Name: season, CompetitionId: int64(COMPETITION_ID)}).FindOrCreate(db)
	if err != nil {
		log.Fatal(err)
	}

	// 2. Získáme HTML data
	url := BASE_URL + "/zapasy?y=" + year + "&p1=0&c=0&d_od=&d_do=&k=0"
	fmt.Printf("Grabbing URL: %s\n", url)
	res, err := http.Get(url)
	if err != nil {
		log.Fatal(err)
	}
	defer res.Body.Close()

	// Načteme HTML dokument
	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		log.Fatal(err)
	}

	// 3. Vyparsujeme a uložíme data
	tx, _ := db.Begin()
	doc.Find("#start-of-content .table tbody tr").Each(func(i int, tr *goquery.Selection) {
		g := Game{}

		skip := false
		sel := tr.Find("td")
		sel.Each(func(ii int, td *goquery.Selection) {
			switch ii {
			// 0 - kolo (round)
			case 0:
				g.Round, _ = strconv.Atoi(strings.Replace(td.Text(), ".", "", 1))
			// 1 - číslo (gameNo)
			case 1:
				g.GameNo, _ = strconv.Atoi(td.Text())
			// 2 - datum a čas (playedAt)
			case 2:
				g.PlayedAt, err = convertPlayedAtDateFormat(td.Text())
				if err != nil {
					skip = true
				}
			// 3 - domácí/hosté
			case 3:
				g.HomeTeamId, g.AwayTeamId = parseHomeAwayTeams(db, td)
			// 4 - celkové skóre
			// 5 - výsledky čtvrtin
			//   - OBĚ PŘESKAKUJEME (dělá se až na konci)
			// 6 - fáze sezóny
			case 6:
				seasonPart, err := (SeasonPart{Name: strings.TrimSpace(td.Text()), SeasonId: seasonObj.Id}).FindOrCreate(db)
				if err != nil {
					log.Fatal(err)
				}
				g.SeasonPartId = seasonPart.Id
			// 7 - PŘESKAKUJEME
			// 8 - review (reviewUrl)
			case 8:
				g.ReviewUrl = td.Find("a").First().AttrOr("href", "")
				if g.ReviewUrl != "" {
					g.ReviewUrl = BASE_URL + g.ReviewUrl
				}
				g.ReviewParsed = sql.NullString{}
			}
		})

		if skip != true {
			g, err := g.FindOrCreate(db)
			if err != nil {
				log.Fatal(err)
			}

			parseAndSaveGameResults(db, sel.Eq(4), sel.Eq(5), g)
		}
	})

	err = tx.Commit()
	if err != nil {
		log.Fatal(err)
	}
}

func main() {

	help := "Add on of these subcommands: 'games', 'players', 'reviews' or 'teams'!"

	// Např. `./go-cligrabber season -database ./data.db -season 2020/2021`
	// nebo `./go-cligrabber teams -database ./data.db -season 2020/2021 -limit 50`
	// nebo `./go-cligrabber reviews -database ./data.db -limit 50`
	// nebo `./go-cligrabber players -database ./data.db -limit 50`
	if len(os.Args) < 4 {
		fmt.Fprintln(os.Stderr, help)
		os.Exit(0)
	}

	var database, season string
	var initDb bool
	var limit int

	// Volba pro import zápasů/výsledků dle sezóny
	fsSeason := flag.NewFlagSet("season", flag.ExitOnError)
	fsSeason.StringVar(&database, "database", "./data.sqlite", "Path to the database")
	fsSeason.BoolVar(&initDb, "initdb", false, "Initialize database - existing data will be erased")
	fsSeason.StringVar(&season, "season", "", "Season we want to grab (eg '2020/21')")

	// Volba pro import hráčů jednotlivých týmů
	fsTeams := flag.NewFlagSet("teams", flag.ExitOnError)
	fsTeams.StringVar(&database, "database", "./data.sqlite", "Path to the database")
	fsTeams.StringVar(&season, "season", "", "Season we want to grab (eg '2020/21')")
	fsTeams.IntVar(&limit, "limit", 50, "Limit of items to download and parse")

	// Volba pro import detailů jednotlivých zápasů
	fsReviews := flag.NewFlagSet("reviews", flag.ExitOnError)
	fsReviews.StringVar(&database, "database", "./data.sqlite", "Path to the database")
	fsReviews.IntVar(&limit, "limit", 50, "Limit of items to download and parse")

	// Volba pro import detailů jednotlivých hráčů
	fsPlayers := flag.NewFlagSet("players", flag.ExitOnError)
	fsPlayers.StringVar(&database, "database", "./data.sqlite", "Path to the database")
	fsPlayers.IntVar(&limit, "limit", 50, "Limit of items to download and parse")
	flag.Parse()

	// Hlavní rozcestník
	switch os.Args[1] {
	case "season":
		if err := fsSeason.Parse(os.Args[2:]); err == nil {
			db := ConnectDb(database)
			defer db.Close()
			importSeasonGames(db, initDb, season)
		}
	case "teams":
		if err := fsTeams.Parse(os.Args[2:]); err == nil {
			db := ConnectDb(database)
			defer db.Close()
			importTeamPlayers(db, season, limit)
		}
	case "reviews":
		if err := fsReviews.Parse(os.Args[2:]); err == nil {
			db := ConnectDb(database)
			defer db.Close()
			importGameReviews(db, limit)
		}
	case "players":
		if err := fsPlayers.Parse(os.Args[2:]); err == nil {
			db := ConnectDb(database)
			defer db.Close()
			importPlayerDetails(db, limit)
		}
	default:
		fmt.Fprintln(os.Stderr, help)
		os.Exit(0)
	}
}
