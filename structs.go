package main

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

type CompetitionType struct {
	Id  int64
	Key string
}

type Country struct {
	Id   int64
	Code string
	Name string
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

type SeasonPart struct {
	Id                int64
	SeasonId          int
	SuccessorId       int // Předcházející část sezóny (je-li)
	CompetitionTypeId int
	Name              string
	IsCurrent         bool
}

type Team struct {
	Id              int64
	CountryId       int64
	Name            string
	Name2           string
	Name3           string
	LogoUrl         string
	LogoData        string // Base64 encoded ikonka
	ProfileUrl      string
	IsProfileParsed bool
}
