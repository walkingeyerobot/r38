package main

import (
	"bufio"
	"database/sql"
	"encoding/csv"
	"flag"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"log"
	"os"
	"strconv"

	"math"
	"math/rand"
	"time"
)

type Card struct {
	Mtgo   string
	Number string
	Rarity string
	Name   string
	Color  string
	Cmc    int64
	Type   string
	Rating float64
}

type CardSet struct {
	Mythics   []Card
	Rares     []Card
	Uncommons []Card
	Commons   []Card
	Foils     []Card
}

var database *sql.DB

const MAX_M = 2
const MAX_R = 3
const MAX_U = 5
const MAX_C = 8

type hopperRefill func(h *Hopper)

func main() {
	draftNamePtr := flag.String("name", "untitled draft", "string")
	filenamePtr := flag.String("filename", "ktk.csv", "string")
	databasePtr := flag.String("database", "draft2.db", "string")
	flag.Parse()

	name := *draftNamePtr

	rand.Seed(time.Now().UnixNano())

	var err error
	var lol [24]int64
	err = generateStandardDraft(lol, *filenamePtr)
	if err == nil {
		return
	}

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

	packIds, err := generateEmptyDraft(name)
	if err != nil {
		return
	}

	err = generateCubeDraft(packIds, *filenamePtr)
	if err != nil {
		return
	}
}

type Hopper struct {
	Cards  []Card
	Refill hopperRefill
}

func (h *Hopper) Pop() Card {
	ret := h.Cards[0]
	h.Cards = h.Cards[1:]
	if len(h.Cards) == 0 {
		h.Refill(h)
	}
	return ret
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
		cmc, err := strconv.ParseInt(line[5], 10, 64)
		if err != nil {
			return err
		}
		rating, err := strconv.ParseFloat(line[7], 64)
		if err != nil {
			return err
		}
		card := Card{Mtgo: line[0], Number: line[1], Rarity: line[2], Name: line[3], Color: line[4], Type: line[6], Cmc: cmc, Rating: rating}
		switch card.Rarity {
		case "M":
			allCards.Mythics = append(allCards.Mythics, card)
		case "R":
			allCards.Rares = append(allCards.Rares, card)
		case "U":
			allCards.Uncommons = append(allCards.Uncommons, card)
		case "C":
			allCards.Commons = append(allCards.Commons, card)
		default:
			return fmt.Errorf("Error determining rarity of %v", line)
		}
	}

	/*
		mythicCount := len(allCards.Mythics)
		rareCount := len(allCards.Rares)
		uncommonCount := len(allCards.Uncommons)
		commonCount := len(allCards.Commons)
	*/

	refillRares := func(h *Hopper) {
		if len(h.Cards) == 0 {
			return
		}
		h.Cards = append(append(allCards.Mythics, allCards.Rares...), allCards.Rares...)
		shuffle(h.Cards)
	}
	refillUncommons := func(h *Hopper) {
		if len(h.Cards) == 0 {
			return
		}
		h.Cards = append(allCards.Uncommons, allCards.Uncommons...)
		shuffle(h.Cards)
	}
	refillCommons := func(h *Hopper) {
		if len(h.Cards) == 0 {
			return
		}
		h.Cards = append(allCards.Commons, allCards.Commons...)
		shuffle(h.Cards)
	}

	var hoppers [14]*Hopper
	hoppers[0] = &(Hopper{Refill: refillRares})

	hoppers[1] = &(Hopper{Refill: refillUncommons})
	hoppers[2] = &(Hopper{Refill: refillUncommons})
	hoppers[3] = &(Hopper{Refill: refillUncommons})

	hoppers[4] = &(Hopper{Refill: refillCommons})
	hoppers[5] = hoppers[4]
	hoppers[6] = &(Hopper{Refill: refillCommons})
	hoppers[7] = hoppers[6]
	hoppers[8] = &(Hopper{Refill: refillCommons})
	hoppers[9] = hoppers[8]
	hoppers[10] = &(Hopper{Refill: refillCommons})
	hoppers[11] = hoppers[10]
	hoppers[12] = &(Hopper{Refill: refillCommons})
	hoppers[13] = hoppers[12]

	for i, hopper := range hoppers {
		hopper.Refill(hopper)
	}

	var pack [14]Card

	for {
		for i, hopper := range hoppers {
			pack[i] = hopper.Pop()
		}

		if okPack(pack) {
			break
		}
		for _, card := range pack {
			log.Printf("%s\t%s", card.Rarity, card.Name)
		}
	}

	for _, card := range pack {
		log.Printf("%s\t%s", card.Rarity, card.Name)
	}

	return nil
}

func okPack(pack [14]Card) bool {
	h := make(map[Card]int)
	c := make(map[rune]int)
	for _, card := range pack {
		h[card]++
		if h[card] > 1 {
			log.Printf("bad")
			return false
		}
		if card.Rarity == "C" {
			for _, d := range card.Color {
				c[d]++
			}
		}
	}

	var total int
	for _, v := range c {
		total += v
	}
	average := float64(total) / 5.0
	var sd float64
	for _, v := range c {
		sd += math.Pow(float64(v)-average, 2)
	}

	sd = math.Sqrt(sd / 5.0)

	log.Printf("sd: %f", sd)
	log.Printf("%v", c)

	if sd > 1.2 {
		return false
	}

	log.Printf("good")
	return true
}

func violatesQuantityLimit(draftPool CardSet) bool {
	countsM := make(map[string]int)
	countsR := make(map[string]int)
	countsU := make(map[string]int)
	countsC := make(map[string]int)
	for _, list := range [][]Card{draftPool.Rares, draftPool.Uncommons, draftPool.Commons, draftPool.Foils} {
		for _, card := range list {
			switch card.Rarity {
			case "M":
				old := countsM[card.Name]
				if old >= MAX_M {
					return true
				}
				countsM[card.Name] = old + 1
			case "R":
				old := countsR[card.Name]
				if old >= MAX_R {
					return true
				}
				countsR[card.Name] = old + 1
			case "U":
				old := countsU[card.Name]
				if old >= MAX_U {
					return true
				}
				countsU[card.Name] = old + 1
			case "C":
				old := countsC[card.Name]
				if old >= MAX_C {
					return true
				}
				countsC[card.Name] = old + 1
			default:
				log.Printf("wtf")
				return true
			}
		}
	}

	return false
}

func shuffle(slice []Card) {
	for i := len(slice) - 1; i > 0; i-- {
		j := rand.Intn(i)
		slice[i], slice[j] = slice[j], slice[i]
	}
}
