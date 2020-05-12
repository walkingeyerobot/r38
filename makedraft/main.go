package main

import (
	"bufio"
	"database/sql"
	"encoding/csv"
	"flag"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"log"
	"math"
	"math/rand"
	"os"
	"sort"
	"strconv"
	_ "time"
)

var database *sql.DB
var oM map[int]int
var oR map[int]int
var oU map[int]int
var oC map[int]int

const MAX_M = 20
const MAX_R = 30
const MAX_U = 60
const MAX_C = 80
const PACK_COLOR_STDEV = 1.55
const RATING_MIN = 1.8
const RATING_MAX = 3
const POOL_COLOR_STDEV = 5.0

func main() {
	oM = make(map[int]int)
	oR = make(map[int]int)
	oU = make(map[int]int)
	oC = make(map[int]int)

	draftNamePtr := flag.String("name", "untitled draft", "string")
	filenamePtr := flag.String("filename", "ktk.csv", "string")
	databasePtr := flag.String("database", "draft.db", "string")
	flag.Parse()

	name := *draftNamePtr

	//rand.Seed(time.Now().UnixNano())
	rand.Seed(1)
	log.Printf("generating draft %s.", name)

	var err error
	var packIds [24]int64

	database, err = sql.Open("sqlite3", *databasePtr)
	if err != nil {
		log.Printf("error opening database %s: %s", *databasePtr, err)
		return
	}
	err = database.Ping()
	if err != nil {
		log.Printf("error pinging database: %s", err)
		return
	}
	/*
		packIds, err = generateEmptyDraft(name)
		if err != nil {
			return
		}
	*/
	err = generateStandardDraft(packIds, *filenamePtr)

	// err = generateCubeDraft(packIds, *filenamePtr)
	if err != nil {
		return
	}

	fmt.Printf("%v\n%v\n%v\n%v\n", oM, oR, oU, oC)
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

func readCsv(filename string) ([][]string, error) {
	var lines [][]string
	file, err := os.Open(filename)
	if err != nil {
		log.Printf("could not open file %s: %s", filename, err)
		return lines, err
	}
	defer file.Close()

	// read the first line as a text file and throw it away
	normalReader := bufio.NewReader(file)
	_, _, err = normalReader.ReadLine()
	if err != nil {
		log.Printf("error discarding first line of file %s: %s", filename, err)
		return lines, err
	}

	reader := csv.NewReader(normalReader)
	if err != nil {
		log.Printf("error processing CSV file %s: %s", filename, err)
		return lines, err
	}

	lines, err = reader.ReadAll()
	if err != nil {
		log.Printf("error reading CSV file %s: %s", filename, err)
		return lines, err
	}

	return lines, nil
}

func generateCubeDraft(packIds [24]int64, filename string) error {
	lines, err := readCsv(filename)
	if err != nil {
		return err
	}

	query := `INSERT INTO cards (pack, original_pack, edition, number, tags, name, cmc, type, color, mtgo) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	for i := 539; i > 179; i-- {
		j := rand.Intn(i)
		lines[i], lines[j] = lines[j], lines[i]
		packId := packIds[(539-i)/15]
		finish := lines[i][7]
		mtgoId := lines[i][12]
		if finish == "Foil" {
			// if a card is foil, increment the mtgo id
			mtgoIdInt, err := strconv.Atoi(mtgoId)
			if err != nil {
				log.Printf("could not convert foil version %s: %s", mtgoId, err)
				return err
			}
			mtgoIdInt++
			mtgoId = fmt.Sprintf("%d", mtgoIdInt)
		}
		database.Exec(query, packId, packId, lines[i][4], lines[i][5], lines[i][10], lines[i][0], lines[i][1], lines[i][2], lines[i][3], mtgoId)
	}
	log.Printf("done generating new cube draft\n")
	return nil
}

func generateStandardDraft(packIds [24]int64, filename string) error {
	lines, err := readCsv(filename)
	if err != nil {
		return err
	}

	var allCards CardSet

	for _, line := range lines {
		cmc, err := strconv.ParseInt(line[6], 10, 64)
		if err != nil {
			return err
		}
		rating, err := strconv.ParseFloat(line[8], 64)
		if err != nil {
			return err
		}
		card := Card{
			Mtgo:          line[0],
			Number:        line[1],
			Rarity:        line[2],
			Name:          line[3],
			Color:         line[4],
			ColorIdentity: line[5],
			Cmc:           cmc,
			Type:          line[7],
			Rating:        rating}
		switch card.Rarity {
		case "M":
			allCards.Mythics = append(allCards.Mythics, card)
		case "R":
			allCards.Rares = append(allCards.Rares, card)
		case "U":
			allCards.Uncommons = append(allCards.Uncommons, card)
		case "C":
			allCards.Commons = append(allCards.Commons, card)
		case "B":
			allCards.Basics = append(allCards.Basics, card)
		default:
			return fmt.Errorf("Error determining rarity of %v", line)
		}
	}

	var hoppers [15]Hopper

	resetHoppers := func() {
		hoppers[0] = MakeNormalHopper(allCards.Mythics, allCards.Rares, allCards.Rares)

		/*
			hoppers[1] = MakeNormalHopper(allCards.Uncommons, allCards.Uncommons)
			hoppers[2] = MakeNormalHopper(allCards.Uncommons, allCards.Uncommons)
			hoppers[3] = MakeNormalHopper(allCards.Uncommons, allCards.Uncommons)
		*/
		hoppers[1] = MakeUncommonHopper(allCards.Uncommons, allCards.Uncommons, allCards.Uncommons, allCards.Uncommons, allCards.Uncommons)
		hoppers[2] = hoppers[1]
		hoppers[3] = hoppers[1]

		hoppers[4] = MakeNormalHopper(allCards.Commons, allCards.Commons)
		hoppers[5] = hoppers[4]
		hoppers[6] = MakeNormalHopper(allCards.Commons, allCards.Commons)
		hoppers[7] = hoppers[6]
		hoppers[8] = MakeNormalHopper(allCards.Commons, allCards.Commons)
		hoppers[9] = hoppers[8]
		hoppers[10] = MakeNormalHopper(allCards.Commons, allCards.Commons)
		hoppers[11] = hoppers[10]
		hoppers[12] = MakeNormalHopper(allCards.Commons, allCards.Commons)
		hoppers[13] = MakeFoilHopper(&hoppers[12],
			allCards.Mythics, allCards.Rares, allCards.Rares,
			allCards.Uncommons, allCards.Uncommons, allCards.Uncommons,
			allCards.Commons, allCards.Commons, allCards.Commons, allCards.Commons,
			allCards.Basics, allCards.Basics, allCards.Basics, allCards.Basics)

		hoppers[14] = MakeBasicLandHopper(allCards.Basics)
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
				log.Printf("%s\t%v\t%s", card.Rarity, card.Foil, card.Name)
			}

			if okPack(packs[i]) {
				i++
			}
		}
		if !resetDraft && okDraft(packs) {
			break
		}
		log.Printf("RESETTING DRAFT")
	}

	log.Printf("draft attempts: %d", draftAttempts)
	log.Printf("pack attempts: %d", packAttempts)

	log.Printf("inserting into db...")
	//	query := `INSERT INTO cards (pack, original_pack, edition, number, tags, name, cmc, type, color, mtgo) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	/*
		for i, pack := range packs {
			for _, card := range pack {
				packId := packIds[i]
				var tags string
				if card.Foil {
					tags = "foil"
				}
				database.Exec(query, packId, packId, "ktk", card.Number, tags, card.Name, card.Cmc, card.Type, card.ColorIdentity, card.Mtgo)
			}
		}
	*/
	log.Printf("done!")
	return nil
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
		cardHash[card.Name]++
		if cardHash[card.Name] > 1 {
			log.Printf("found duplicated card %s", card.Name)
			passes = false
		}
		if card.Rarity == "C" {
			for _, color := range card.ColorIdentity {
				colorHash[color]++
			}
			ratings = append(ratings, card.Rating)
			totalCommons++
		} else if card.Rarity == "U" {
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
	passes := true

	cardHash := make(map[string]int)
	colorHash := make(map[rune]float64)
	q := make(map[string]int)
	for _, pack := range packs {
		for _, card := range pack {
			cardHash[card.Name]++
			qty := cardHash[card.Name]
			if card.Rarity != "B" && qty > q[card.Rarity] {
				q[card.Rarity] = qty
			}
			switch card.Rarity {
			case "M":
				if qty > MAX_M {
					log.Printf("found %d %s, which is more than MAX_M %d", qty, card.Name, MAX_M)
					passes = false
				}
			case "R":
				if qty > MAX_R {
					log.Printf("found %d %s, which is more than MAX_R %d", qty, card.Name, MAX_R)
					passes = false
				}
			case "U":
				if qty > MAX_U {
					log.Printf("found %d %s, which is more than MAX_U %d", qty, card.Name, MAX_U)
					passes = false
				}
			case "C":
				if qty > MAX_C {
					log.Printf("found %d %s, which is more than MAX_C %d", qty, card.Name, MAX_C)
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
		oM[q["M"]]++
		oR[q["R"]]++
		oU[q["U"]]++
		oC[q["C"]]++
		fmt.Printf("%d, %d, %d, %d\n", q["M"], q["R"], q["U"], q["C"])
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
