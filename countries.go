package main

import (
	"strings"
)

// countryNameToCode mapuje běžné názvy zemí (včetně českých a anglických variant) na ISO kódy
var countryNameToCode = map[string]string{
	// Česko
	"česko":           "CZE",
	"česká republika": "CZE",
	"czech republic":  "CZE",
	"czechia":         "CZE",
	"čr":              "CZE",

	// Slovensko
	"slovensko": "SVK",
	"slovakia":  "SVK",
	"sr":        "SVK",

	// USA
	"usa":           "USA",
	"united states": "USA",
	"spojené státy": "USA",
	"amerika":       "USA",

	// Polsko
	"polsko": "POL",
	"poland": "POL",

	// Německo
	"německo": "DEU",
	"germany": "DEU",

	// Francie
	"francie": "FRA",
	"france":  "FRA",

	// Španělsko
	"španělsko": "ESP",
	"spain":     "ESP",

	// Itálie
	"itálie": "ITA",
	"italy":  "ITA",

	// Velká Británie
	"velká británie": "GBR",
	"great britain":  "GBR",
	"uk":             "GBR",
	"united kingdom": "GBR",
	"anglie":         "GBR",
	"england":        "GBR",

	// Rusko
	"rusko":  "RUS",
	"russia": "RUS",

	// Ukrajina
	"ukrajina": "UKR",
	"ukraine":  "UKR",

	// Srbsko
	"srbsko": "SRB",
	"serbia": "SRB",

	// Chorvatsko
	"chorvatsko": "HRV",
	"croatia":    "HRV",

	// Slovinsko
	"slovinsko": "SVN",
	"slovenia":  "SVN",

	// Litva
	"litva":     "LTU",
	"lithuania": "LTU",

	// Lotyšsko
	"lotyšsko": "LVA",
	"latvia":   "LVA",

	// Estonsko
	"estonsko": "EST",
	"estonia":  "EST",

	// Bosna a Hercegovina
	"bosna a hercegovina": "BIH",
	"bosnia":              "BIH",

	// Makedonie
	"makedonie":         "MKD",
	"macedonia":         "MKD",
	"severní makedonie": "MKD",

	// Černá Hora
	"černá hora": "MNE",
	"montenegro": "MNE",

	// Řecko
	"řecko":  "GRC",
	"greece": "GRC",

	// Turecko
	"turecko": "TUR",
	"turkey":  "TUR",

	// Maďarsko
	"maďarsko": "HUN",
	"hungary":  "HUN",

	// Rakousko
	"rakousko": "AUT",
	"austria":  "AUT",

	// Švýcarsko
	"švýcarsko":   "CHE",
	"switzerland": "CHE",

	// Belgie
	"belgie":  "BEL",
	"belgium": "BEL",

	// Nizozemsko
	"nizozemsko":  "NLD",
	"netherlands": "NLD",
	"holandsko":   "NLD",

	// Portugalsko
	"portugalsko": "PRT",
	"portugal":    "PRT",

	// Dánsko
	"dánsko":  "DNK",
	"denmark": "DNK",

	// Švédsko
	"švédsko": "SWE",
	"sweden":  "SWE",

	// Norsko
	"norsko": "NOR",
	"norway": "NOR",

	// Finsko
	"finsko":  "FIN",
	"finland": "FIN",

	// Island
	"island":  "ISL",
	"iceland": "ISL",

	// Irsko
	"irsko":   "IRL",
	"ireland": "IRL",

	// Rumunsko
	"rumunsko": "ROU",
	"romania":  "ROU",

	// Bulharsko
	"bulharsko": "BGR",
	"bulgaria":  "BGR",

	// Kanada
	"kanada": "CAN",
	"canada": "CAN",

	// Austrálie
	"austrálie": "AUS",
	"australia": "AUS",

	// Nový Zéland
	"nový zéland": "NZL",
	"new zealand": "NZL",

	// Čína
	"čína":  "CHN",
	"china": "CHN",

	// Japonsko
	"japonsko": "JPN",
	"japan":    "JPN",

	// Jižní Korea
	"jižní korea": "KOR",
	"south korea": "KOR",
	"korea":       "KOR",

	// Argentina
	"argentina": "ARG",

	// Brazílie
	"brazílie": "BRA",
	"brazil":   "BRA",

	// Senegal
	"senegal": "SEN",

	// Nigérie
	"nigérie": "NGA",
	"nigeria": "NGA",

	// Kamerun
	"kamerun":  "CMR",
	"cameroon": "CMR",

	// Pobřeží slonoviny
	"pobřeží slonoviny": "CIV",
	"ivory coast":       "CIV",

	// Izrael
	"izrael": "ISR",
	"israel": "ISR",
}

// countryCodeToName mapuje ISO kódy zemí na jejich oficiální anglické názvy
var countryCodeToName = map[string]string{
	"CZE": "Czechia",
	"SVK": "Slovakia",
	"USA": "United States",
	"POL": "Poland",
	"DEU": "Germany",
	"FRA": "France",
	"ESP": "Spain",
	"ITA": "Italy",
	"GBR": "United Kingdom",
	"RUS": "Russia",
	"UKR": "Ukraine",
	"SRB": "Serbia",
	"HRV": "Croatia",
	"SVN": "Slovenia",
	"LTU": "Lithuania",
	"LVA": "Latvia",
	"EST": "Estonia",
	"BIH": "Bosnia and Herzegovina",
	"MKD": "North Macedonia",
	"MNE": "Montenegro",
	"GRC": "Greece",
	"TUR": "Turkey",
	"HUN": "Hungary",
	"AUT": "Austria",
	"CHE": "Switzerland",
	"BEL": "Belgium",
	"NLD": "Netherlands",
	"PRT": "Portugal",
	"DNK": "Denmark",
	"SWE": "Sweden",
	"NOR": "Norway",
	"FIN": "Finland",
	"ISL": "Iceland",
	"IRL": "Ireland",
	"ROU": "Romania",
	"BGR": "Bulgaria",
	"CAN": "Canada",
	"AUS": "Australia",
	"NZL": "New Zealand",
	"CHN": "China",
	"JPN": "Japan",
	"KOR": "South Korea",
	"ARG": "Argentina",
	"BRA": "Brazil",
	"SEN": "Senegal",
	"NGA": "Nigeria",
	"CMR": "Cameroon",
	"CIV": "Ivory Coast",
	"ISR": "Israel",
}

// normalizeCountryName normalizuje název země a vrací ISO kód a oficiální název
// Pokud zemi nelze identifikovat, vrací prázdné stringy
func normalizeCountryName(countryName string) (code string, name string) {
	// Normalizujeme název země
	normalized := strings.ToLower(strings.TrimSpace(countryName))

	// Pokud je název prázdný, vrátíme prázdné hodnoty
	if normalized == "" {
		return "", ""
	}

	// Pokusíme se najít kód země
	code, found := countryNameToCode[normalized]
	if !found {
		// Pokud nenajdeme, zkusíme to jako ISO kód
		uppercaseName := strings.ToUpper(normalized)
		if _, exists := countryCodeToName[uppercaseName]; exists {
			code = uppercaseName
		} else {
			// Nemůžeme identifikovat zemi
			return "", ""
		}
	}

	// Zjistíme oficiální název země
	name, exists := countryCodeToName[code]
	if !exists {
		name = countryName // Fallback na původní název
	}

	return code, name
}
