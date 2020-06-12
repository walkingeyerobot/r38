package main

import (
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"flag"
	_ "github.com/mattn/go-sqlite3"
	"io/ioutil"
	"log"
	"math"
	"math/rand"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"
)

var database *sql.DB
var oM map[int]int
var oR map[int]int
var oU map[int]int
var oC map[int]int

type Settings struct {
	Set                                       *string
	Database                                  *string
	Name                                      *string
	MaxMythic                                 *int
	MaxRare                                   *int
	MaxUncommon                               *int
	MaxCommon                                 *int
	PackCommonColorStdevMax                   *float64
	PackCommonRatingMin                       *float64
	PackCommonRatingMax                       *float64
	DraftCommonColorStdevMax                  *float64
	PackCommonColorIdentityStdevMax           *float64
	DraftCommonColorIdentityStdevMax          *float64
	DfcMode                                   *bool
	AbortMissingCommonColor                   *bool
	AbortMissingCommonColorIdentity           *bool
	AbortDuplicateThreeColorIdentityUncommons *bool
}

var settings Settings

const MAX_M = 2
const MAX_R = 3
const MAX_U = 4
const MAX_C = 6
const PACK_COLOR_STDEV = 0.8
const RATING_MIN = 1.8
const RATING_MAX = 3
const POOL_COLOR_STDEV = 3.0

var CUBE_MODE bool
var DFC_MODE bool

var packReasons map[int]int
var draftReasons map[int]int
var badPack map[int]int

func main() {
	oM = make(map[int]int)
	oR = make(map[int]int)
	oU = make(map[int]int)
	oC = make(map[int]int)
	packReasons = make(map[int]int)
	draftReasons = make(map[int]int)
	badPack = make(map[int]int)

	flagSet := flag.NewFlagSet(os.Args[0], flag.ContinueOnError)

	settings.Set = flagSet.String(
		"set", "sets/isd.json",
		"A .json file containing relevant set data.")
	settings.Database = flagSet.String(
		"database", "draft.db",
		"The sqlite3 database to insert to.")
	settings.Name = flagSet.String(
		"name", "untitled draft",
		"The name of the draft.")
	settings.MaxMythic = flagSet.Int(
		"max-mythic", 2,
		"Maximum number of copies of a given mythic allowed in a draft. 0 to disable.")
	settings.MaxRare = flagSet.Int(
		"max-rare", 3,
		"Maximum number of copies of a given rare allowed in a draft. 0 to disable.")
	settings.MaxUncommon = flagSet.Int(
		"max-uncommon", 4,
		"Maximum number of copies of a given uncommon allowed in a draft. 0 to disable.")
	settings.MaxCommon = flagSet.Int(
		"max-common", 6,
		"Maximum number of copies of a given common allowed in a draft. 0 to disable.")
	settings.PackCommonColorStdevMax = flagSet.Float64(
		"pack-common-color-stdev-max", 0,
		"Maximum standard deviation allowed in a pack of color distribution among commons. 0 to disable.")
	settings.PackCommonRatingMin = flagSet.Float64(
		"pack-common-rating-min", 0,
		"Minimum average rating allowed in a pack among commons. 0 to disable.")
	settings.PackCommonRatingMax = flagSet.Float64(
		"pack-common-rating-max", 0,
		"Maximum average rating allowed in a pack among commons. 0 to disable.")
	settings.DraftCommonColorStdevMax = flagSet.Float64(
		"draft-common-color-stdev-max", 0,
		"Maximum standard deviation allowed in the entire draft of color distribution among commons. 0 to disable.")
	settings.PackCommonColorIdentityStdevMax = flagSet.Float64(
		"pack-common-color-identity-stdev-max", 0,
		"Maximum standard deviation allowed in a pack of color identity distribution among commons. 0 to disable.")
	settings.DraftCommonColorIdentityStdevMax = flagSet.Float64(
		"draft-common-color-identity-stdev-max", 0,
		"Maximum standard deviation allowed in the entire draft of color identity distribution among commons. 0 to disable.")
	settings.DfcMode = flagSet.Bool(
		"dfc-mode", false,
		"If true, include DFCs only in DFC specific hoppers and exclude them from color distribution stats.")
	settings.AbortMissingCommonColor = flagSet.Bool(
		"abort-missing-common-color", false,
		"If true, every color will be represented in the colors of commons in every pack.")
	settings.AbortMissingCommonColorIdentity = flagSet.Bool(
		"abort-missing-common-color-identity", false,
		"If true, every color will be represented in the color identities of commons in every pack.")
	settings.AbortDuplicateThreeColorIdentityUncommons = flagSet.Bool(
		"abort-duplicate-three-color-identity-uncommons", false,
		"If true, only one uncommon of a color identity triplet will be allowed per pack.")

	flagSet.Parse(os.Args[1:])

	if *settings.Set == "" {
		log.Printf("you must specify a set json file to continue")
		return
	}

	jsonFile, err := os.Open(*settings.Set)
	if err != nil {
		log.Printf("error opening json file: %s", err.Error())
		return
	}
	defer jsonFile.Close()

	byteValue, err := ioutil.ReadAll(jsonFile)
	if err != nil {
		log.Printf("error readalling: %s", err.Error())
		return
	}

	var cfg DraftConfig
	err = json.Unmarshal(byteValue, &cfg)
	if err != nil {
		log.Printf("error unmarshalling: %s", err.Error())
		return
	}

	if len(cfg.Flags) != 0 {
		jsonFlags := strings.Join(cfg.Flags, " ")
		r := csv.NewReader(strings.NewReader(jsonFlags))
		r.Comma = ' '
		fields, err := r.Read()
		if err != nil {
			log.Printf("error parsing json flags: %s", err.Error())
			return
		}
		var allFlags []string
		for _, flag := range fields {
			if flag != "" {
				allFlags = append(allFlags, flag)
			}
		}
		allFlags = append(allFlags, os.Args[1:]...)

		flagSet.Parse(allFlags)
	}

	log.Printf("settings: %v", settings)

	rand.Seed(time.Now().UnixNano())
	// rand.Seed(1)
	log.Printf("generating draft %s.", *settings.Name)

	var packIds [24]int64

	database, err = sql.Open("sqlite3", *settings.Database)
	if err != nil {
		log.Printf("error opening database %s: %s", *settings.Database, err.Error())
		return
	}
	err = database.Ping()
	if err != nil {
		log.Printf("error pinging database: %s", err.Error())
		return
	}

	var allCards CardSet
	var dfcCards CardSet

	log.Printf("dfc mode: %t", *settings.DfcMode)

	for _, card := range cfg.Cards {
		var currentSet *CardSet
		if *settings.DfcMode && card.Dfc {
			currentSet = &dfcCards
		} else {
			currentSet = &allCards
		}
		currentSet.All = append(currentSet.All, card)

		switch card.Rarity {
		case "mythic":
			currentSet.Mythics = append(currentSet.Mythics, card)
		case "rare":
			currentSet.Rares = append(currentSet.Rares, card)
		case "uncommon":
			currentSet.Uncommons = append(currentSet.Uncommons, card)
		case "common":
			currentSet.Commons = append(currentSet.Commons, card)
		case "basic":
			currentSet.Basics = append(currentSet.Basics, card)
		default:
			log.Printf("error with determining rarity for %v", card)
			return
		}
	}

	var hoppers [15]Hopper
	resetHoppers := func() {
		for i, hopdef := range cfg.Hoppers {
			switch hopdef.Type {
			case "RareHopper":
				hoppers[i] = MakeNormalHopper(false, allCards.Mythics, allCards.Rares, allCards.Rares)
			case "RareRefillHopper":
				hoppers[i] = MakeNormalHopper(true, allCards.Mythics, allCards.Rares, allCards.Rares)
			case "UncommonHopper":
				hoppers[i] = MakeNormalHopper(false, allCards.Uncommons, allCards.Uncommons)
			case "UncommonRefillHopper":
				hoppers[i] = MakeNormalHopper(true, allCards.Uncommons, allCards.Uncommons)
			case "CommonHopper":
				hoppers[i] = MakeNormalHopper(false, allCards.Commons, allCards.Commons)
			case "CommonRefillHopper":
				hoppers[i] = MakeNormalHopper(true, allCards.Commons, allCards.Commons)
			case "BasicLandHopper":
				hoppers[i] = MakeBasicLandHopper(allCards.Basics)
			case "CubeHopper":
				hoppers[i] = MakeNormalHopper(false, allCards.All)
			case "Pointer":
				hoppers[i] = hoppers[hopdef.Refs[0]]
			case "DfcHopper":
				hoppers[i] = MakeNormalHopper(false,
					dfcCards.Mythics,
					dfcCards.Rares, dfcCards.Rares,
					dfcCards.Uncommons, dfcCards.Uncommons, dfcCards.Uncommons,
					dfcCards.Uncommons, dfcCards.Uncommons, dfcCards.Uncommons,
					dfcCards.Commons, dfcCards.Commons, dfcCards.Commons, dfcCards.Commons,
					dfcCards.Commons, dfcCards.Commons, dfcCards.Commons, dfcCards.Commons,
					dfcCards.Commons, dfcCards.Commons, dfcCards.Commons)
			case "DfcRefillHopper":
				hoppers[i] = MakeNormalHopper(true,
					dfcCards.Mythics,
					dfcCards.Rares, dfcCards.Rares,
					dfcCards.Uncommons, dfcCards.Uncommons, dfcCards.Uncommons,
					dfcCards.Uncommons, dfcCards.Uncommons, dfcCards.Uncommons,
					dfcCards.Commons, dfcCards.Commons, dfcCards.Commons, dfcCards.Commons,
					dfcCards.Commons, dfcCards.Commons, dfcCards.Commons, dfcCards.Commons,
					dfcCards.Commons, dfcCards.Commons, dfcCards.Commons)
			case "FoilHopper":
				hoppers[i] = MakeFoilHopper(&hoppers[hopdef.Refs[0]], &hoppers[hopdef.Refs[1]], &hoppers[hopdef.Refs[2]],
					allCards.Mythics,
					allCards.Rares, allCards.Rares,
					allCards.Uncommons, allCards.Uncommons, allCards.Uncommons,
					allCards.Commons, allCards.Commons, allCards.Commons, allCards.Commons,
					allCards.Basics, allCards.Basics, allCards.Basics, allCards.Basics)
			}
		}
	}

	var packs [24][15]Card
	packAttempts := 0
	draftAttempts := 0

	for {
		resetHoppers()
		resetDraft := false
		draftAttempts++
		for i := 0; i < 24; { // we'll manually increment i
			packAttempts++
			for j, hopper := range hoppers {
				var empty bool
				packs[i][j], empty = hopper.Pop()
				if empty {
					resetDraft = true
					draftReasons[6]++
					badPack[j]++
					break
				}
			}

			if resetDraft {
				break
			}

			for _, card := range packs[i] {
				log.Printf("%s\t%v\t%s", card.Rarity, card.Foil, card.Data)
			}

			if CUBE_MODE || okPack(packs[i]) {
				i++
			}
		}
		if !resetDraft && (CUBE_MODE || okDraft(packs)) {
			break
		}
		log.Printf("RESETTING DRAFT")
	}

	log.Printf("draft attempts: %d", draftAttempts)
	log.Printf("pack attempts: %d", packAttempts)

	packIds, err = generateEmptyDraft(*settings.Name)
	if err != nil {
		return
	}
	// \"FOIL_STATUS\"
	re := regexp.MustCompile(`"FOIL_STATUS"`)
	log.Printf("inserting into db...")
	// query := `INSERT INTO cards (pack, original_pack, data) VALUES (?, ?, ?)`
	query := `INSERT INTO cards (pack, original_pack, edition, number, tags, name, cmc, type, color, mtgo, data) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	for i, pack := range packs {
		for _, card := range pack {
			packId := packIds[i]
			var data string
			var tags string
			if card.Foil {
				data = re.ReplaceAllString(card.Data, "true")
				tags = "foil"
			} else {
				data = re.ReplaceAllString(card.Data, "false")
			}
			// database.Exec(query, packId, packId, data)
			database.Exec(query, packId, packId, card.Set, card.CollectorNumber, tags, card.Name, card.Cmc, card.TypeLine, card.ColorIdentity, card.MtgoId, data)
		}
	}

	log.Printf("done!")

	// fmt.Printf("%v\n%v\n%v\n%v\n", oM, oR, oU, oC)
}

func generateEmptyDraft(name string) ([24]int64, error) {
	var packIds [24]int64

	query := `INSERT INTO drafts (name) VALUES (?);`
	res, err := database.Exec(query, name)
	if err != nil {
		log.Printf("error creating draft: %s", err)
		return packIds, err
	}

	draftId, err := res.LastInsertId()
	if err != nil {
		log.Printf("could not get draft ID: %s", err)
		return packIds, err
	}

	query = `INSERT INTO seats (position, draft) VALUES (?, ?)`
	var seatIds [8]int64
	for i := 0; i < 8; i++ {
		res, err = database.Exec(query, i, draftId)
		if err != nil {
			log.Printf("could not create seats in draft: %s", err)
			return packIds, err
		}
		seatIds[i], err = res.LastInsertId()
		if err != nil {
			log.Printf("could not finalize seat creation: %s", err)
			return packIds, err
		}
	}

	query = `INSERT INTO packs (seat, original_seat, modified, round) VALUES (?, ?, 0, ?)`
	for i := 0; i < 8; i++ {
		for j := 0; j < 4; j++ {
			res, err = database.Exec(query, seatIds[i], seatIds[i], j)
			if err != nil {
				log.Printf("error creating packs: %s", err)
				return packIds, err
			}
			if j != 0 {
				packIds[(3*i)+(j-1)], err = res.LastInsertId()
				if err != nil {
					log.Printf("error creating packs: %s", err)
					return packIds, err
				}
			}
		}
	}

	return packIds, nil
}

func okPack(pack [15]Card) bool {
	passes := true
	cardHash := make(map[string]int)
	colorHash := make(map[rune]float64)
	colorIdentityHash := make(map[rune]float64)
	uncommonColorIdentities := make(map[string]int)
	var ratings []float64
	totalCommons := 0
	for _, card := range pack {
		if card.Foil || (*settings.DfcMode && card.Dfc) {
			continue
		}
		cardHash[card.Id]++
		if cardHash[card.Id] > 1 {
			log.Printf("found duplicated card %s", card.Id)
			passes = false
			packReasons[1]++
		}
		if card.Rarity == "common" {
			for _, color := range card.Color {
				colorHash[color]++
			}
			for _, color := range card.ColorIdentity {
				colorIdentityHash[color]++
			}
			ratings = append(ratings, card.Rating)
			totalCommons++
		} else if card.Rarity == "uncommon" {
			if *settings.AbortDuplicateThreeColorIdentityUncommons {
				sortedColor := stringSort(card.ColorIdentity)
				uncommonColorIdentities[sortedColor]++
				if uncommonColorIdentities[sortedColor] > 1 && len(sortedColor) == 3 {
					log.Printf("found more than one %s card", sortedColor)
					packReasons[2]++
					passes = false
				}
				if uncommonColorIdentities[sortedColor] >= 3 {
					log.Printf("all uncommons are %s", sortedColor)
					packReasons[3]++
					passes = false
				}
			}
		}
	}

	// calculate stdev for color
	var colors []float64
	for _, v := range colorHash {
		colors = append(colors, v)
	}

	if *settings.AbortMissingCommonColor && len(colors) != 5 {
		log.Printf("a color is missing")
		packReasons[4]++
		passes = false
		for {
			colors = append(colors, 0)
			if len(colors) == 5 {
				break
			}
		}
	}

	colorStdev := stdev(colors)

	var colorIdentities []float64
	for _, v := range colorIdentityHash {
		colorIdentities = append(colorIdentities, v)
	}

	if *settings.AbortMissingCommonColorIdentity && len(colors) != 5 {
		log.Printf("a color identity is missing")
		packReasons[4]++
		passes = false
		for {
			colorIdentities = append(colorIdentities, 0)
			if len(colorIdentities) == 5 {
				break
			}
		}
	}

	colorIdentityStdev := stdev(colorIdentities)

	ratingMean := mean(ratings)
	log.Printf("color stdev:\t%f", colorStdev)
	log.Printf("color identity stdev:\t%f", colorIdentityStdev)
	log.Printf("rating mean:\t%f", ratingMean)

	if *settings.PackCommonColorStdevMax != 0 && colorStdev > *settings.PackCommonColorStdevMax {
		log.Printf("color stdev too high")
		packReasons[5]++
		passes = false
	}
	if *settings.PackCommonColorIdentityStdevMax != 0 && colorIdentityStdev > *settings.PackCommonColorIdentityStdevMax {
		log.Printf("color identity stdev too high")
		packReasons[5]++
		passes = false
	}
	if *settings.PackCommonRatingMax != 0 && ratingMean > *settings.PackCommonRatingMax {
		log.Printf("rating mean too high")
		packReasons[6]++
		passes = false
	} else if *settings.PackCommonRatingMin != 0 && ratingMean < *settings.PackCommonRatingMin {
		log.Printf("rating mean too low")
		packReasons[7]++
		passes = false
	}

	if passes {
		log.Printf("pack passes!")
		packReasons[0]++
	} else {
		log.Printf("pack fails :(")
	}

	return passes
}

func okDraft(packs [24][15]Card) bool {
	log.Printf("analyzing entire draft pool...")
	passes := true

	cardHash := make(map[string]int)
	colorHash := make(map[rune]float64)
	colorIdentityHash := make(map[rune]float64)
	q := make(map[string]int)
	for _, pack := range packs {
		for _, card := range pack {
			cardHash[card.Id]++
			qty := cardHash[card.Id]
			if card.Rarity != "B" && qty > q[card.Rarity] {
				q[card.Rarity] = qty
			}
			switch card.Rarity {
			case "mythic":
				if *settings.MaxMythic != 0 && qty > *settings.MaxMythic {
					log.Printf("found %d %s, which is more than MAX_M %d", qty, card.Id, MAX_M)
					draftReasons[1]++
					passes = false
				}
			case "rare":
				if *settings.MaxRare != 0 && qty > *settings.MaxRare {
					log.Printf("found %d %s, which is more than MAX_R %d", qty, card.Id, MAX_R)
					draftReasons[2]++
					passes = false
				}
			case "uncommon":
				if *settings.MaxUncommon != 0 && qty > *settings.MaxUncommon {
					log.Printf("found %d %s, which is more than MAX_U %d", qty, card.Id, MAX_U)
					draftReasons[3]++
					passes = false
				}
			case "common":
				if *settings.MaxCommon != 0 && qty > *settings.MaxCommon {
					log.Printf("found %d %s, which is more than MAX_C %d", qty, card.Id, MAX_C)
					draftReasons[4]++
					passes = false
				}
				if !(card.Foil || (*settings.DfcMode && card.Dfc)) {
					for _, color := range card.Color {
						colorHash[color]++
					}
					for _, color := range card.ColorIdentity {
						colorIdentityHash[color]++
					}
				}
			}
		}
	}

	// calculate stdev for color
	var colors []float64
	for _, v := range colorHash {
		colors = append(colors, v)
	}

	colorStdev := stdev(colors)

	var colorIdentities []float64
	for _, v := range colorIdentityHash {
		colorIdentities = append(colorIdentities, v)
	}

	colorIdentityStdev := stdev(colorIdentities)

	log.Printf("all commons color stdev:\t%f\t%v", colorStdev, colorHash)
	log.Printf("all commons color identity stdev:\t%f\t%v", colorIdentityStdev, colorIdentityHash)

	if *settings.DraftCommonColorStdevMax != 0 && colorStdev > *settings.DraftCommonColorStdevMax {
		log.Printf("color stdev too high")
		draftReasons[5]++
		passes = false
	}

	if *settings.DraftCommonColorIdentityStdevMax != 0 && colorIdentityStdev > *settings.DraftCommonColorIdentityStdevMax {
		log.Printf("color identity stdev too high")
		draftReasons[5]++
		passes = false
	}

	if passes {
		draftReasons[0]++
		log.Printf("draft passes!")
		oM[q["mythic"]]++
		oR[q["rare"]]++
		oU[q["uncommon"]]++
		oC[q["common"]]++
		// fmt.Printf("%d, %d, %d, %d\n", q["mythic"], q["rare"], q["uncommon"], q["common"])
	} else {
		log.Printf("draft fails :(")
	}

	return passes
}

func stdev(list []float64) float64 {
	avg := mean(list)

	var sum float64
	for _, val := range list {
		sum += math.Pow(val-avg, 2)
	}
	return math.Sqrt(sum / float64(len(list)))
}

func mean(list []float64) float64 {
	var sum float64
	for _, val := range list {
		sum += val
	}
	return sum / float64(len(list))
}

// from https://stackoverflow.com/questions/22688651/golang-how-to-sort-string-or-byte
// go generics when
type sortRunes []rune

func (s sortRunes) Less(i, j int) bool {
	return s[i] < s[j]
}

func (s sortRunes) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s sortRunes) Len() int {
	return len(s)
}

func stringSort(s string) string {
	r := []rune(s)
	sort.Sort(sortRunes(r))
	return string(r)
}
