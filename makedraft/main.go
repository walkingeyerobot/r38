package main

import (
	"database/sql"
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
	"time"
)

var database *sql.DB
var oM map[int]int
var oR map[int]int
var oU map[int]int
var oC map[int]int

const MAX_M = 2
const MAX_R = 3
const MAX_U = 4
const MAX_C = 6
const PACK_COLOR_STDEV = 1.55
const RATING_MIN = 1.8
const RATING_MAX = 3
const POOL_COLOR_STDEV = 5.0

var CUBE_MODE bool

func main() {
	oM = make(map[int]int)
	oR = make(map[int]int)
	oU = make(map[int]int)
	oC = make(map[int]int)

	draftNamePtr := flag.String("name", "untitled draft", "string")
	filenamePtr := flag.String("filename", "makedraft/cube.json", "string")
	databasePtr := flag.String("database", "draft.db", "string")
	flag.Parse()

	name := *draftNamePtr

	rand.Seed(time.Now().UnixNano())
	// rand.Seed(1)
	log.Printf("generating draft %s.", name)

	var err error
	var packIds [24]int64

	database, err = sql.Open("sqlite3", *databasePtr)
	if err != nil {
		log.Printf("error opening database %s: %s", *databasePtr, err.Error())
		return
	}
	err = database.Ping()
	if err != nil {
		log.Printf("error pinging database: %s", err.Error())
		return
	}

	jsonFile, err := os.Open(*filenamePtr)
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

	for _, flag := range cfg.Flags {
		if flag == "cube=false" {
			CUBE_MODE = false
		} else if flag == "cube=true" {
			CUBE_MODE = true
		}
	}

	var allCards CardSet

	for _, card := range cfg.Cards {
		allCards.All = append(allCards.All, card)
		switch card.Rarity {
		case "mythic":
			allCards.Mythics = append(allCards.Mythics, card)
		case "rare":
			allCards.Rares = append(allCards.Rares, card)
		case "uncommon":
			allCards.Uncommons = append(allCards.Uncommons, card)
		case "common":
			allCards.Commons = append(allCards.Commons, card)
		case "basic":
			allCards.Basics = append(allCards.Basics, card)
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
				hoppers[i] = MakeNormalHopper(allCards.Mythics, allCards.Rares, allCards.Rares)
			case "UncommonHopper":
				hoppers[i] = MakeNormalHopper(allCards.Uncommons, allCards.Uncommons)
			case "CommonHopper":
				hoppers[i] = MakeNormalHopper(allCards.Commons, allCards.Commons)
			case "BasicLandHopper":
				hoppers[i] = MakeBasicLandHopper(allCards.Basics)
			case "CubeHopper":
				hoppers[i] = MakeNormalHopper(allCards.All)
			case "Pointer":
				hoppers[i] = hoppers[hopdef.Refs[0]]
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

	packIds, err = generateEmptyDraft(name)
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
	uncommonColors := make(map[string]int)
	var ratings []float64
	totalCommons := 0
	for _, card := range pack {
		if card.Foil {
			continue
		}
		cardHash[card.Id]++
		if cardHash[card.Id] > 1 {
			log.Printf("found duplicated card %s", card.Id)
			passes = false
		}
		if card.Rarity == "common" {
			for _, color := range card.ColorIdentity {
				colorHash[color]++
			}
			ratings = append(ratings, card.Rating)
			totalCommons++
		} else if card.Rarity == "uncommon" {
			sortedColor := stringSort(card.ColorIdentity)
			uncommonColors[sortedColor]++
			if uncommonColors[sortedColor] > 1 && len(sortedColor) == 3 {
				log.Printf("found more than one %s card", sortedColor)
				passes = false
			}
			if uncommonColors[sortedColor] >= 3 {
				log.Printf("all uncommons are %s", sortedColor)
				passes = false
			}
		}
	}

	// calculate stdev for color
	var colors []float64
	for _, v := range colorHash {
		colors = append(colors, v)
	}

	if len(colors) != 5 {
		log.Printf("a color is missing")
		passes = false
		for {
			colors = append(colors, 0)
			if len(colors) == 5 {
				break
			}
		}
	}

	colorStdev := stdev(colors)
	ratingMean := mean(ratings)
	log.Printf("color stdev:\t%f", colorStdev)
	log.Printf("rating mean:\t%f", ratingMean)

	if colorStdev > PACK_COLOR_STDEV {
		log.Printf("color stdev too high")
		passes = false
	}
	if ratingMean > RATING_MAX {
		log.Printf("rating mean too high")
		passes = false
	} else if ratingMean < RATING_MIN {
		log.Printf("rating mean too low")
		passes = false
	}

	if passes {
		log.Printf("pack passes!")
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
				if qty > MAX_M {
					log.Printf("found %d %s, which is more than MAX_M %d", qty, card.Id, MAX_M)
					passes = false
				}
			case "rare":
				if qty > MAX_R {
					log.Printf("found %d %s, which is more than MAX_R %d", qty, card.Id, MAX_R)
					passes = false
				}
			case "uncommon":
				if qty > MAX_U {
					log.Printf("found %d %s, which is more than MAX_U %d", qty, card.Id, MAX_U)
					passes = false
				}
			case "common":
				if qty > MAX_C {
					log.Printf("found %d %s, which is more than MAX_C %d", qty, card.Id, MAX_C)
					passes = false
				}
				if !card.Foil {
					for _, color := range card.ColorIdentity {
						colorHash[color]++
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

	log.Printf("all commons color stdev:\t%f", colorStdev)

	if colorStdev > POOL_COLOR_STDEV {
		log.Printf("color stdev too high")
		passes = false
	}

	if passes {
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
