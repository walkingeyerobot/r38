package main

import (
	"bufio"
	"crypto/rand"
	"database/sql"
	"encoding/binary"
	"encoding/csv"
	"flag"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"log"
	badrand "math/rand"
	"os"
	"strconv"
)

type cryptoSource struct{}

func (s cryptoSource) Seed(seed int64) {}

func (s cryptoSource) Int63() int64 {
	return int64(s.Uint64() & ^uint64(1<<63))
}

func (s cryptoSource) Uint64() (v uint64) {
	err := binary.Read(rand.Reader, binary.BigEndian, &v)
	if err != nil {
		log.Fatal(err)
	}
	return v
}

type Card struct {
	Mtgo string
	Number string
	Rarity string
	Name string
	Color string
	Cmc int64
	Type string
	Rating float64
}

type CardSet struct {
	Mythics []Card
	Rares []Card
	Uncommons []Card
	Commons []Card
	Foils []Card
}

var database *sql.DB

func main() {
	draftNamePtr := flag.String("name", "untitled draft", "string")
	filenamePtr := flag.String("filename", "ktk.csv", "string")
	databasePtr := flag.String("database", "draft2.db", "string")
	flag.Parse()

	name := *draftNamePtr

	var err error
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

	err = generateStandardDraft(packIds, *filenamePtr)
	if err != nil {
		return
	}

	/*
	err = generateCubeDraft(packIds, *filenamePtr)
	if err != nil {
		return
	}
*/
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

func generateCubeDraft(packIds [24]int64, filename string) (error) {
	lines, err := readCsv(filename)
	if err != nil {
		return err
	}

	query := `INSERT INTO cards (pack, original_pack, edition, number, tags, name, cmc, type, color, mtgo) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	var src cryptoSource
	rnd := badrand.New(src)
	for i := 539; i > 179; i-- {
		j := rnd.Intn(i)
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

func generateStandardDraft(packIds [24]int64, filename string) (error) {
	lines, err := readCsv(filename)
	if err != nil {
		return err
	}

	var src cryptoSource
	rnd := badrand.New(src)

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

	mythicCount := len(allCards.Mythics)
	rareCount := len(allCards.Rares)
	uncommonCount := len(allCards.Uncommons)
	commonCount := len(allCards.Commons)

	var draftPool CardSet

	for {
		for i := 0; i < 24; i++ {
			r := rnd.Intn(rareCount * 2 + mythicCount)
			if r < mythicCount {
				draftPool.Rares = append(draftPool.Rares, allCards.Mythics[rnd.Intn(mythicCount)])
			} else {
				draftPool.Rares = append(draftPool.Rares, allCards.Rares[rnd.Intn(rareCount)])
			}
		}

		qq := violatesQuantityLimit(draftPool)
		log.Printf("%v", qq)

		if qq == true {
			log.Printf("violated quantity limit test 1")
			draftPool.Rares = []Card{}
			continue
		}

		// if we reach here, everything is good so far
		break
	}

	// rares and mythics are all set at this point
	// move on to uncommons

	for {
		for i := 0; i < 72; i++ {
			r := rnd.Intn(uncommonCount)
			draftPool.Uncommons = append(draftPool.Uncommons, allCards.Uncommons[r])
		}

		if violatesQuantityLimit(draftPool) {
			log.Printf("violated quantity limit test 2")
			draftPool.Uncommons = []Card{}
			continue
		}
		
		break
	}

	// uncommons are all set at this point
	// do foils next

	for {
		for i := 0; i < 24; i++ {
			r := rnd.Intn(4)
			if r > 0 {
				// add nothing
				continue
			}

			r = rnd.Intn(7)
			var card Card
			if r < 4 {
				// add a foil common
				card = allCards.Commons[rnd.Intn(commonCount)]
			} else if r < 6 {
				// add a foil uncommon
				card = allCards.Uncommons[rnd.Intn(uncommonCount)]
			} else {
				r = rnd.Intn(rareCount * 2 + mythicCount)
				if r < mythicCount {
					// add a foil mythic
					card = allCards.Mythics[rnd.Intn(mythicCount)]
				} else {
					// add a foil rare
					card = allCards.Rares[rnd.Intn(rareCount)]
				}
			}

			draftPool.Foils = append(draftPool.Foils, card)
		}

		if violatesQuantityLimit(draftPool) {
			log.Printf("violated quantity limit test 3")
			draftPool.Foils = []Card{}
			continue
		}
		
		break
	}

	log.Printf("%v", draftPool)
	
	return nil
}

func violatesQuantityLimit(draftPool CardSet) (bool) {
	log.Printf("~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~")
	MAX_M := 2
	MAX_R := 3
	MAX_U := 5
	MAX_C := 8

	fails := false
	
	countsM := make(map[string]int)
	countsR := make(map[string]int)
	countsU := make(map[string]int)
	countsC := make(map[string]int)
	for _, list := range [][]Card{draftPool.Rares, draftPool.Uncommons, draftPool.Commons, draftPool.Foils} {
		for _, card := range list {
			log.Printf("%s", card.Name)
			switch card.Rarity {
			case "M":
				old := countsM[card.Name]
				if old >= MAX_M {
					log.Printf("failed: %d >= %d", old, MAX_M)
					fails = true
				}
				countsM[card.Name] = old + 1
			case "R":
				old := countsR[card.Name]
				if old >= MAX_R {
					log.Printf("failed: %d >= %d", old, MAX_M)
					fails = true
				}
				countsR[card.Name] = old + 1
			case "U":
				old := countsU[card.Name]
				if old >= MAX_U {
					log.Printf("failed: %d >= %d", old, MAX_M)
					fails = true
				}
				countsU[card.Name] = old + 1
			case "C":
				old := countsC[card.Name]
				if old >= MAX_C {
					log.Printf("failed: %d >= %d", old, MAX_M)
					fails = true
				}
				countsC[card.Name] = old + 1
			default:
				log.Printf("wtf")
				fails = true
			}
		}
	}

	if fails {
		log.Printf("%v", countsR)
	} else {
		log.Printf("passes")
		log.Printf("%v", fails)
	}
	log.Printf("~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~")
	return fails
}
