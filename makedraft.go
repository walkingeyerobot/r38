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

func main() {
	draftNamePtr := flag.String("name", "untitled draft", "string")
	flag.Parse()

	name := *draftNamePtr

	var database *sql.DB

	var err error
	database, err = sql.Open("sqlite3", "draft.db")
	if err != nil {
		return
	}
	err = database.Ping()
	if err != nil {
		return
	}

	query := `INSERT INTO drafts (name) VALUES (?);`
	res, err := database.Exec(query, name)
	if err != nil {
		// error
		return
	}

	draftId, err := res.LastInsertId()
	if err != nil {
		// error
		return
	}
	query = `INSERT INTO seats (position, draft) VALUES (?, ?)`
	var seatIds [9]int64
	for i := 0; i < 8; i++ {
		res, err = database.Exec(query, i, draftId)
		if err != nil {
			// error
			return
		}
		seatIds[i], err = res.LastInsertId()
		if err != nil {
			// error
			return
		}
	}

	res, err = database.Exec(`INSERT INTO seats (position, draft) VALUES(NULL, ?)`, draftId)
	if err != nil {
		// error
		return
	}
	seatIds[8], err = res.LastInsertId()
	if err != nil {
		// error
		return
	}

	query = `INSERT INTO packs (seat, original_seat, modified, round) VALUES (?, ?, 0, ?)`
	var packIds [25]int64
	for i := 0; i < 8; i++ {
		for j := 0; j < 4; j++ {
			res, err = database.Exec(query, seatIds[i], seatIds[i], j)
			if err != nil {
				// error
				return
			}
			if j != 0 {
				packIds[(3*i)+(j-1)], err = res.LastInsertId()
				if err != nil {
					// error
					return
				}
			}
		}
	}

	res, err = database.Exec(`INSERT INTO packs (seat, original_seat, modified, round) VALUES (?, ?, 0, NULL)`, seatIds[8], seatIds[8])
	if err != nil {
		// error
		return
	}
	packIds[24], err = res.LastInsertId()
	if err != nil {
		// error
		return
	}

	query = `INSERT INTO cards (pack, original_pack, edition, number, tags, name) VALUES (?, ?, ?, ?, ?, ?)`
	file, err := os.Open("vintagecube.csv")
	if err != nil {
		// error
		return
	}
	defer file.Close()
	reader := csv.NewReader(bufio.NewReader(file))
	_, err = reader.Read() // throw away the first line
	if err != nil {
		// error
		return
	}
	lines, err := reader.ReadAll()
	if err != nil {
		// error
		return
	}

	var src cryptoSource
	rnd := badrand.New(src)
	for i := 539; i > 164; i-- {
		j := rnd.Intn(i)
		lines[i], lines[j] = lines[j], lines[i]
		packId := packIds[(539-i)/15]
		database.Exec(query, packId, packId, lines[i][4], lines[i][5], lines[i][10], lines[i][0])
	}
	fmt.Printf("done generating new draft\n")
}
