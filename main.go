package main

import (
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
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

func InitDb(db *sql.DB) {
	tx, _ := db.Begin()
	// Smazání tabulek (pokud existují)
	db.Exec(`DROP TABLE IF EXISTS "game_results"`)
	db.Exec(`DROP TABLE IF EXISTS "result_types"`)
	db.Exec(`DROP TABLE IF EXISTS "player_teams"`)
	db.Exec(`DROP TABLE IF EXISTS "players"`)
	db.Exec(`DROP TABLE IF EXISTS "game_team_stats"`)
	db.Exec(`DROP TABLE IF EXISTS "game_player_stats"`)
	db.Exec(`DROP TABLE IF EXISTS "games"`)
	db.Exec(`DROP TABLE IF EXISTS "teams"`)
	db.Exec(`DROP TABLE IF EXISTS "season_parts"`)
	db.Exec(`DROP TABLE IF EXISTS "seasons"`)
	db.Exec(`DROP TABLE IF EXISTS "competitions"`)
	db.Exec(`DROP TABLE IF EXISTS "competition_types"`)
	db.Exec(`DROP TABLE IF EXISTS "countries"`)
	// Vytvoření tabulek
	db.Exec(`CREATE TABLE "competition_types" ("id" integer primary key autoincrement not null, "key" varchar not null)`)
	db.Exec(`CREATE TABLE "competitions" ("id" integer primary key autoincrement not null, "country_id" integer not null, "name" varchar not null, foreign key("country_id") references "countries"("id"))`)
	db.Exec(`CREATE TABLE "countries" ("id" integer primary key autoincrement not null, "code" varchar not null, "name" varchar not null)`)
	db.Exec(`CREATE TABLE "game_results" ("id" integer primary key autoincrement not null, "game_id" integer not null, "team_id" integer not null, "result_type_id" integer not null, "points" integer not null default 0, foreign key("game_id") references "games"("id"), foreign key("team_id") references "teams"("id"), foreign key("result_type_id") references "result_types"("id"))`)
	db.Exec(`CREATE TABLE "games" ("id" integer primary key autoincrement not null, "round" int, "game_no" int, "season_part_id" integer not null, "home_team_id" integer not null, "away_team_id" integer not null, "is_neutral_pitch" tinyint(1) not null default '0', "review_url" varchar, "played_at" datetime not null, "is_review_parsed" tinyint(1) not null default '0', foreign key("season_part_id") references "season_parts"("id"), foreign key("home_team_id") references "teams"("id"), foreign key("away_team_id") references "teams"("id"))`)
	db.Exec(`CREATE TABLE "result_types" ("id" integer primary key autoincrement not null, "key" varchar not null)`)
	db.Exec(`CREATE TABLE "season_parts" ("id" integer primary key autoincrement not null, "season_id" integer not null, "successor_id" integer, "competition_type_id" integer not null, "is_current" tinyint(1) not null default '0', "name" varchar not null, foreign key("season_id") references "seasons"("id"), foreign key("successor_id") references "season_parts"("id"), foreign key("competition_type_id") references "competition_types"("id"))`)
	db.Exec(`CREATE TABLE "seasons" ("id" integer primary key autoincrement not null, "competition_id" integer not null, "previous_id" integer, "name" varchar not null, "is_current" tinyint(1) not null default '0', foreign key("competition_id") references "competitions"("id"), foreign key("previous_id") references "seasons"("id"))`)
	db.Exec(`CREATE TABLE "teams" ("id" integer primary key autoincrement not null, "country_id" integer not null, "name" varchar not null, "name_2" varchar, "name_3" varchar, "logo_url" varchar, "logo_data" blob, "profile_url" varchar, "is_profile_parsed" tinyint(1) not null default '0', foreign key("country_id") references "countries"("id"))`)
	db.Exec(`CREATE TABLE "players" ("id" integer primary key autoincrement not null, "first_name" varchar not null, "last_name" varchar not null, "country_id" integer not null, "birthdate" varchar, "profile_url" varchar, "is_profile_parsed" tinyint(1) not null default '0', foreign key("country_id") references "countries"("id"))`)
	db.Exec(`CREATE TABLE "player_teams" ("id" integer primary key autoincrement not null, "player_id" integer not null, "team_id" integer not null, "date_from" date, "date_to" date, "number" integer, foreign key("player_id") references "players"("id"), foreign key("team_id") references "teams"("id"))`)
	db.Exec(`CREATE TABLE "game_team_stats" ("id" integer primary key autoincrement not null, "game_id" integer not null, "team_id" integer not null, "two_pt_pct" real, "three_pt_pct" real, "ft_pct" real, "rebounds" integer, "turnovers" integer, foreign key("game_id") references "games"("id"), foreign key("team_id") references "teams"("id"))`)
	db.Exec(`CREATE TABLE "game_player_stats" ("id" integer primary key autoincrement not null, "game_id" integer not null, "team_id" integer not null, "player_id" integer, "player_number" integer, "stats_json" text, foreign key("game_id") references "games"("id"), foreign key("team_id") references "teams"("id"), foreign key("player_id") references "players"("id"))`)
	// Defaultní data
	db.Exec(`INSERT INTO "countries" (id, code, name) VALUES ('1', 'CZE', 'Czechia')`)
	db.Exec(`INSERT INTO "competitions" (id, country_id, name) VALUES ('1', '1', 'NBL')`)
	db.Exec(`INSERT INTO "competition_types" (id, key) VALUES ('1', 'cup'),('2', 'league'),('3', 'playoff'),('4', 'playout')`)
	db.Exec(`INSERT INTO "result_types" (id, key) VALUES ('1', '1st_quarter'),('2', '2nd_quarter'),('3', '3rd_quarter'),('4', '4th_quarter'),('5', '1st_overtime'),('6', '2nd_overtime'),('7', 'total')`)

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

func getOrCreatePlayer(db *sql.DB, fullName string, profileUrl string) (int64, error) {
	if fullName == "" || profileUrl == "" {
		return 0, nil
	}

	firstName, lastName := fullnameToFirstLast(fullName)
	player, err := (Player{FirstName: firstName, LastName: lastName, ProfileUrl: profileUrl}).FindOrCreate(db)
	if err != nil {
		return 0, err
	}

	return player.Id, nil
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

func parseGameDetails(db *sql.DB, gameId int64, homeTeamId int64, awayTeamId int64, reviewUrl string) error {
	if reviewUrl == "" {
		return nil
	}

	url := reviewUrl
	if !strings.HasPrefix(url, "http") {
		url = BASE_URL + url
	}
	res, err := http.Get(url)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		return err
	}

	homeTeam, _ := (Team{Id: homeTeamId}).FindById(db)
	awayTeam, _ := (Team{Id: awayTeamId}).FindById(db)

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
			log.Printf("No table found for team: %s", teamName)
			return nil
		}
		log.Printf("Parsing table for team: %s, found %d rows", teamName, table.Find("tr").Length())

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
			playerId, err := getOrCreatePlayer(db, playerName, profileUrl)
			if err != nil {
				return err
			}

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

	_, err = db.Exec("UPDATE games SET is_review_parsed = 1 WHERE id = ?", gameId)
	return err
}

func importGameReviews(db *sql.DB, limit int) {
	rows, err := db.Query(`
		SELECT id, home_team_id, away_team_id, review_url
		FROM games
		WHERE review_url IS NOT NULL AND review_url != ''
		  AND is_review_parsed = 0
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
		if err := parseGameDetails(db, game.gameId, game.homeTeamId, game.awayTeamId, game.reviewUrl); err != nil {
			log.Printf("Failed to parse game %d: %v", game.gameId, err)
		}
	}
}

func parseHomeAwayTeams(db *sql.DB, td *goquery.Selection) (int64, int64) {
	var homeTeamLogoUrl string = ""
	var awayTeamLogoUrl string = ""

	sel := td.Find("img")
	if sel.Length() == 2 {
		homeTeamLogoUrl = sel.Eq(0).AttrOr("src", "")
		if homeTeamLogoUrl != "" {
			homeTeamLogoUrl = BASE_URL + homeTeamLogoUrl
		}
		awayTeamLogoUrl = sel.Eq(1).AttrOr("src", "")
		if awayTeamLogoUrl != "" {
			awayTeamLogoUrl = BASE_URL + awayTeamLogoUrl
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

	sql := `
		INSERT INTO game_results (game_id, team_id, result_type_id, points) 
		VALUES (?, ?, ?, ?) RETURNING id
	`
	a1 := strings.TrimSpace(td1.Find("a").Text())
	a2 := strings.TrimSpace(td1.Find("a div").Text())

	if strings.Index(a1, a2) == 0 {
		totalHome, _ = strconv.Atoi(a2)
		totalAway, _ = strconv.Atoi(strings.TrimSpace(strings.Replace(a1, a2, "", -1)))
	} else {
		totalAway, _ = strconv.Atoi(a2)
		totalHome, _ = strconv.Atoi(strings.TrimSpace(strings.Replace(a1, a2, "", -1)))
	}

	_, _ = db.Exec(sql, game.Id, game.HomeTeamId, RT_TOTAL, totalHome)
	_, _ = db.Exec(sql, game.Id, game.AwayTeamId, RT_TOTAL, totalAway)

	// Výsledky jednotlivých čtvrtin a případných prodloužení
	sel := td2.Find("a > span div")
	selLen := sel.Length()

	if selLen < 2 {
		// Toto se stává naprosto výjimečně (např. diskvalifikace).
		return
	}

	pointsHome := [6]int{-1, -1, -1, -1, -1, -1}
	pointsAway := [6]int{-1, -1, -1, -1, -1, -1}

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

			_, _ = db.Exec(sql, game.Id, game.HomeTeamId, resultType, pointsHome[i])
			_, _ = db.Exec(sql, game.Id, game.AwayTeamId, resultType, pointsAway[i])
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

// Přeformátuj datum/čas z "11. 9. 2020 Pá 18:00" na "2020-09-11 18:00:00"
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

func importTeamLogos(db *sql.DB, limit int) {
	// 1. Načtení prvních 100 záznamů (URL je plné, data jsou prázdná)
	rows, err := db.Query(`
		SELECT id, logo_url 
		FROM teams 
		WHERE (logo_url IS NOT NULL AND logo_url != '') 
		  AND (logo_data IS NULL OR length(logo_data) = 0)
		LIMIT ?
	`, limit)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	var teams []Team
	for rows.Next() {
		var t Team
		err = rows.Scan(&t.Id, &t.LogoUrl)
		if err != nil {
			log.Fatal(err)
		}
		teams = append(teams, t)
	}

	fmt.Printf("Nalezeno %d týmů ke zpracování.\n", len(teams))

	// 2. Stažení a aktualizace
	for _, team := range teams {
		fmt.Printf("Stahuji logo pro tým ID %d z: %s\n", team.Id, team.LogoUrl)

		imgBytes, err := downloadImage(team.LogoUrl)
		if err != nil {
			log.Printf("Nepodařilo se stáhnout %s: %v\n", team.LogoUrl, err)
			continue
		}

		// Uložení binárních dat (BLOB) zpět do databáze
		_, err = db.Exec("UPDATE teams SET logo_data = ? WHERE id = ?", getBase64Image(imgBytes), team.Id)
		if err != nil {
			log.Printf("Chyba při ukládání do DB pro ID %d: %v\n", team.Id, err)
		} else {
			fmt.Printf("Logo pro tým ID %d úspěšně uloženo.\n", team.Id)
		}
	}
}

// downloadImage stáhne obsah z URL a vrátí ho jako slice bajtů
func downloadImage(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("špatný stavový kód: %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}

func getBase64Image(bytes []byte) string {
	// Optional: Detect MIME type to create a Data URI (e.g., "data:image/png;base64,..." )
	mimeType := http.DetectContentType(bytes)
	base64Str := base64.StdEncoding.EncodeToString(bytes)

	return "data:" + mimeType + ";base64," + base64Str
}

func main() {
	database := flag.String("database", "./data.db", "Path to the database")
	initDb := flag.Bool("initdb", false, "Initialize database - existing data will be erased")
	// Volba pro import zápasů/výsledků dle sezóny
	argSeason := flag.String("season", "", "Season we want to grab (eg '2020/21')")
	// Volba pro import log jednotlivých týmů
	logos := flag.Bool("logos", false, "Download logos of single teams (either \"-logos\" or \"-season\" alone is possible)")
	// Volba pro import detailů jednotlivých zápasů
	reviews := flag.Bool("reviews", false, "Download details of single games (either \"-reviews\" or \"-season\" alone is possible)")
	reviewsLimit := flag.Int("reviews-limit", 50, "Limit of games to parse for details")
	flag.Parse()

	// 1. Inicializace SQLite databáze
	db, err := sql.Open("sqlite3", *database+"?_busy_timeout=1000&_journal_mode=WAL")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Případně vytvoříme stukturu a základní data
	if *initDb == true {
		InitDb(db)
	}

	// Odbočka na stahování log jednotlivých týmů a stahování detailů jednotlivých zápasů
	if *argSeason == "" && *logos == true && *reviews == false {
		limit := flag.Int("limit", 100, "Limit of items to parse")
		importTeamLogos(db, *limit)
		return
	} else if *argSeason == "" && *logos == false && *reviews == true {
		importGameReviews(db, *reviewsLimit)
		return
	} else if *argSeason == "" && *logos == false && *reviews == false {
		log.Fatal("Either -season, -logos, or -reviews flag must be provided.")
	} else if *argSeason != "" && (*logos == true || *reviews == true) {
		log.Fatal("Only one of -season, -logos, or -reviews flag can be provided.")
	}

	// Pokračujeme se stahováním zápasů a výsledků pro zadanou sezónu
	year := strings.Split(*argSeason, "/")[0]

	// Získáme ID sezóny, kterou chceme stáhnout
	season, err := (Season{Name: *argSeason, CompetitionId: int64(COMPETITION_ID)}).FindOrCreate(db)
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
				seasonPart, err := (SeasonPart{Name: strings.TrimSpace(td.Text()), SeasonId: season.Id}).FindOrCreate(db)
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
				g.IsReviewParsed = false
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
