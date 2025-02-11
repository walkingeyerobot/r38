package main

import (
	"context"
	"database/sql"
	"flag"
	"github.com/walkingeyerobot/r38/makedraft"
	"log"
)
import "os"

func main() {
	flagSet := flag.NewFlagSet(os.Args[0], flag.ContinueOnError)

	settings := makedraft.Settings{}
	settings.Set = flagSet.String(
		"set", "sets/cube.json",
		"A .json file containing relevant set data.")
	settings.Database = flagSet.String(
		"database", "draft.db",
		"The sqlite3 database to insert to.")
	settings.Seed = flagSet.Int(
		"seed", 0,
		"The random seed to use to generate the draft. If 0, time.Now().UnixNano() will be used.")
	settings.Verbose = flagSet.Bool(
		"v", false,
		"If true, will enable verbose output.")
	settings.Simulate = flagSet.Bool(
		"simulate", false,
		"If true, won't commit to the database.")
	settings.Name = flagSet.String(
		"name", "untitled draft",
		"The name of the draft.")

	flagSet.Parse(os.Args[1:])

	database, err := sql.Open("sqlite3", *settings.Database)
	if err != nil {
		log.Printf("error opening database %s: %s", *settings.Database, err.Error())
		return
	}
	err = database.Ping()
	if err != nil {
		log.Printf("error pinging database: %s", err.Error())
		return
	}

	tx, err := database.BeginTx(context.Background(), &sql.TxOptions{ReadOnly: false})
	if err != nil {
		log.Printf("can't create a context: %s", err.Error())
		return
	}
	defer tx.Rollback()

	err = makedraft.MakeDraft(settings, tx)
	if err != nil {
		log.Printf("%s", err.Error())
		return
	}

	if !*settings.Simulate {
		err = tx.Commit()
	} else {
		err = nil
	}

	if err != nil {
		log.Printf("can't commit :( %s", err.Error())
	} else {
		log.Printf("done!")
	}
}
