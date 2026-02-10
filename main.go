package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	_ "github.com/mattn/go-sqlite3"
)

// TODO Toto pak nějak upravit, ať to tu není natvrdo...
const COUNTRY_ID int = 1
const COMPETITION_ID int = 1

func InitDb(db *sql.DB) {
	tx, _ := db.Begin()
	// Structure
	db.Exec(`DROP TABLE IF EXISTS "results"`)     // Info: bude odstraněno (prozatím z hist. důvodů)
	db.Exec(`DROP TABLE IF EXISTS "game_result"`) // Info: bude odstraněno (prozatím z hist. důvodů)
	db.Exec(`DROP TABLE IF EXISTS "game_results"`)
	db.Exec(`DROP TABLE IF EXISTS "result_types"`)
	db.Exec(`DROP TABLE IF EXISTS "games"`)
	db.Exec(`DROP TABLE IF EXISTS "teams"`)
	db.Exec(`DROP TABLE IF EXISTS "season_parts"`)
	db.Exec(`DROP TABLE IF EXISTS "seasons"`)
	db.Exec(`DROP TABLE IF EXISTS "competitions"`)
	db.Exec(`DROP TABLE IF EXISTS "competition_types"`)
	db.Exec(`DROP TABLE IF EXISTS "countries"`)

	db.Exec(`CREATE TABLE "competition_types" ("id" integer primary key autoincrement not null, "key" varchar not null)`)
	db.Exec(`CREATE TABLE "competitions" ("id" integer primary key autoincrement not null, "country_id" integer not null, "name" varchar not null, foreign key("country_id") references "countries"("id"))`)
	db.Exec(`CREATE TABLE "countries" ("id" integer primary key autoincrement not null, "code" varchar not null, "name" varchar not null)`)
	db.Exec(`CREATE TABLE "game_results" ("id" integer primary key autoincrement not null, "game_id" integer not null, "team_id" integer not null, "result_type_id" integer not null, "points" integer not null default 0, foreign key("game_id") references "games"("id"), foreign key("team_id") references "teams"("id"), foreign key("result_type_id") references "result_types"("id"))`)
	db.Exec(`CREATE TABLE "games" ("id" integer primary key autoincrement not null, "round" int, "game_no" int, "season_part_id" integer not null, "home_team_id" integer not null, "away_team_id" integer not null, "is_neutral_pitch" tinyint(1) not null default '0', "review_url" varchar, "played_at" datetime not null, "is_review_parsed" tinyint(1) not null default '0', foreign key("season_part_id") references "season_parts"("id"), foreign key("home_team_id") references "teams"("id"), foreign key("away_team_id") references "teams"("id"))`)
	db.Exec(`CREATE TABLE "result_types" ("id" integer primary key autoincrement not null, "key" varchar not null)`)
	db.Exec(`CREATE TABLE "season_parts" ("id" integer primary key autoincrement not null, "season_id" integer not null, "successor_id" integer, "competition_type_id" integer not null, "is_current" tinyint(1) not null default '0', "name" varchar not null, foreign key("season_id") references "seasons"("id"), foreign key("successor_id") references "season_parts"("id"), foreign key("competition_type_id") references "competition_types"("id"))`)
	db.Exec(`CREATE TABLE "seasons" ("id" integer primary key autoincrement not null, "competition_id" integer not null, "previous_id" integer, "name" varchar not null, "is_current" tinyint(1) not null default '0', foreign key("competition_id") references "competitions"("id"), foreign key("previous_id") references "seasons"("id"))`)
	db.Exec(`CREATE TABLE "teams" ("id" integer primary key autoincrement not null, "country_id" integer not null, "name" varchar not null, "name_2" varchar, "name_3" varchar, "logo_url" varchar, "logo_data" blob, "profile_url" varchar, "is_profile_parsed" tinyint(1) not null default '0', foreign key("country_id") references "countries"("id"))`)
	// Data
	db.Exec(`INSERT INTO "countries" (id, code, name) VALUES ('1', 'CZE', 'Czechia')`)
	db.Exec(`INSERT INTO "competitions" (id, country_id, name) VALUES ('1', '1', 'NBL')`)
	db.Exec(`INSERT INTO "competition_types" (id, key) VALUES ('1', 'cup'),('2', 'league'),('3', 'playoff'),('4', 'playout')`)
	db.Exec(`INSERT INTO "result_types" (id, key) VALUES ('1', '1st_quarter'),('2', '2nd_quarter'),('3', '3rd_quarter'),('4', '4th_quarter'),('5', '1st_overtime'),('6', '2nd_overtime'),('7', 'total')`)

	err := tx.Commit()
	if err != nil {
		log.Fatal(err)
	}
}

func GetTeamId(db *sql.DB, name string, logoUrl string) (int64, error) {
	row := db.QueryRow("SELECT id FROM teams WHERE country_id = ? AND name = ?", COUNTRY_ID, name)

	var id int64 = 0
	row.Scan(&id)
	if id == 0 {
		stmt, _ := db.Prepare(`INSERT INTO teams (country_id, name, logo_url) VALUES (?, ?, ?) RETURNING id`)
		res, err := stmt.Exec(COUNTRY_ID, name, logoUrl)
		if err != nil {
			return id, err
		}
		defer stmt.Close()

		return res.LastInsertId()
	}

	return id, nil
}

func ParseHomeAwayTeams(db *sql.DB, td *goquery.Selection) (int64, int64) {
	var homeTeamLogoUrl string = ""
	var awayTeamLogoUrl string = ""

	sel := td.Find("img")
	if sel.Length() == 2 {
		homeTeamLogoUrl = sel.Eq(0).AttrOr("src", "")
		if homeTeamLogoUrl != "" {
			homeTeamLogoUrl = "https://nbl.basketball" + homeTeamLogoUrl
		}
		awayTeamLogoUrl = sel.Eq(1).AttrOr("src", "")
		if awayTeamLogoUrl != "" {
			awayTeamLogoUrl = "https://nbl.basketball" + awayTeamLogoUrl
		}
	}

	var homeTeamName string = ""
	var awayTeamName string = ""

	sel = td.Find("div > div > div")
	if sel.Length() == 2 {
		homeTeamName = sel.Eq(0).Text()
		awayTeamName = sel.Eq(1).Text()
	}

	homeTeamId, _ := GetTeamId(db, homeTeamName, homeTeamLogoUrl)
	awayTeamId, _ := GetTeamId(db, awayTeamName, awayTeamLogoUrl)

	return homeTeamId, awayTeamId
}

func GetSeasonId(db *sql.DB, season string) (int64, error) {
	row := db.QueryRow("SELECT id FROM seasons WHERE competition_id = ? AND name = ?", COMPETITION_ID, season)

	var id int64 = 0
	row.Scan(&id)
	if id == 0 {
		res, err := db.Exec("INSERT INTO seasons (competition_id, name) VALUES (?, ?) RETURNING id", COMPETITION_ID, season)
		if err != nil {
			return id, err
		}

		return res.LastInsertId()
	}

	return id, nil
}

func ParseSeasonPart(db *sql.DB, td *goquery.Selection, seasonId int64) (int64, error) {
	name := td.Text()
	row := db.QueryRow("SELECT id FROM season_parts WHERE name = ?", name)

	var id int64 = 0
	row.Scan(&id)
	if id == 0 {
		res, err := db.Exec(`
			INSERT INTO season_parts (season_id, competition_type_id, name) 
			VALUES (?, ?, ?) RETURNING id
		`, seasonId, 2, name)
		if err != nil {
			return id, err
		}

		return res.LastInsertId()
	}

	return id, nil
}

/*
<td>

	<a href="/zapas/347323#tab-pane-one" class="text-primary d-block">
		79
		<div class="font-weight-bold">83</div>
	</a>

</td>
<td>

	<a href="https://www.fibalivestats.com/webcast/CBFFE/1746263/" target="_blank" class="text-gray">
		<span class="d-flex text-gray">
				<div class="mr-4">
					23<br>23
				</div>
				<div class="mr-4">
					43<br>41
				</div>
				<div class="mr-4">
					59<br>63
				</div>
		</span>
	</a>

</td>
*/
func ParseAndSaveGameResults(db *sql.DB, td1 *goquery.Selection, td2 *goquery.Selection, game Game) {
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

	// Výsledky jednotlivých třetin
	sel := td2.Find("a > span div")
	selLen := sel.Length()

	if selLen < 2 {
		// Toto se stává naprosto výjimečně (diskvalifikace), pak HTML vypadá takto:
		/*
			<td>
				<generic-tooltip class="tooltip" style="display: inline-block">
					<div slot="target" aria-labelledby="tooltip-bsu9baikg">*</div>
					<span slot="tooltip" hidden="" role="tooltip" id="tooltip-bsu9baikg">
						<div class="tooltip-body">Utkání bylo kontumováno ve prospěch USK Praha</div>
					</span>
				</generic-tooltip>
				<a href="https://www.fibalivestats.com/webcast/CBFFE/2209810/" target="_blank" class="text-gray">
					<span class="d-flex text-gray">
						<div class="mr-4">

						</div>
					</span>
				</a>
			</td>
		*/
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
			pointsHome[i] = pointsHome[i] - SumIntArray(pointsHome[0:i])
			pointsAway[i] = pointsAway[i] - SumIntArray(pointsAway[0:i])
		}
	})

	pointsHome[selLen] = totalHome - SumIntArray(pointsHome[0:selLen])
	pointsAway[selLen] = totalAway - SumIntArray(pointsAway[0:selLen])

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

func SumIntArray(numbers []int) int {
	result := 0
	for i := 0; i < len(numbers); i++ {
		result += numbers[i]
	}
	return result
}

/*
 * Přeformátuj datum/čas z "11. 9. 2020 Pá 18:00" na "2020-09-11 18:00:00"
 */
func ConvertPlayedAtDateFormat(dt string) (string, error) {
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

func main() {
	database := flag.String("database", "./data.db", "Path to the database")
	season := flag.String("season", "2020/21", "Season we want to grab")
	initDb := flag.Bool("initdb", false, "Initialize database - existing data will be erased")
	flag.Parse()

	year := strings.Split(*season, "/")[0]

	// 1. Inicializace SQLite databáze
	db, err := sql.Open("sqlite3", *database)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Případně vytvoříme stukturu a základní data
	if *initDb == true {
		InitDb(db)
	}

	// Získáme ID sezóny, kterou chceme stáhnout
	seasonId, err := GetSeasonId(db, *season)
	if err != nil {
		log.Fatal(err)
	}

	// 2. Získáme HTML data
	url := "https://nbl.basketball/zapasy?y=" + year + "&p1=0&c=0&d_od=&d_do=&k=0"
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

		/*
			! TODO Aby šlo do již stažených zápasů (schedule) ukládat výsledky,
			!      tak se tady nejprve musíme záznam o hře snažit získat!
		*/
		insert, err := db.Prepare(`
			INSERT INTO games (
				round, game_no, season_part_id, home_team_id, away_team_id, 
				is_neutral_pitch, played_at, review_url, is_review_parsed
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		`)
		if err != nil {
			log.Fatal(err)
		}
		defer insert.Close()

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
				g.PlayedAt, err = ConvertPlayedAtDateFormat(td.Text())
				if err != nil {
					skip = true
				}
			// 3 - domácí/hosté
			case 3:
				g.HomeTeamId, g.AwayTeamId = ParseHomeAwayTeams(db, td)
			// 4 - celkové skóre
			// 5 - výsledky čtvrtin
			//   - OBĚ PŘESKAKUJEME (dělá se až na konci)
			// 6 - fáze sezóny
			case 6:
				g.SeasonPartId, err = ParseSeasonPart(db, td, seasonId)
				if err != nil {
					log.Fatal(err)
				}
			// 7 - PŘESKAKUJEME
			// 8 - review (reviewUrl)
			case 8:
				g.ReviewUrl = td.Find("a").First().AttrOr("href", "")
				if g.ReviewUrl != "" {
					g.ReviewUrl = "https://nbl.basketball" + g.ReviewUrl
				}
				g.IsReviewParsed = false
			}
		})

		if skip != true {
			res, err := insert.Exec(g.Round, g.GameNo, g.SeasonPartId, g.HomeTeamId,
				g.AwayTeamId, g.IsNeutralPitch, g.PlayedAt, g.ReviewUrl, g.IsReviewParsed)
			if err != nil {
				log.Fatal(err)
			}

			g.Id, err = res.LastInsertId()
			if err != nil {
				log.Fatal()
			}

			ParseAndSaveGameResults(db, sel.Eq(4), sel.Eq(5), g)
		}
	})

	err = tx.Commit()
	if err != nil {
		log.Fatal(err)
	}
}
