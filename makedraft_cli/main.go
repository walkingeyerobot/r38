package main

import (
	"context"
	"database/sql"
	"github.com/objectbox/objectbox-go/objectbox"
	"github.com/walkingeyerobot/r38/schema"
	"log"
	"os"

	"github.com/walkingeyerobot/r38/makedraft"
)

func main() {
	settings, err := makedraft.ParseSettings(os.Args)
	if err != nil {
		log.Printf("error parsing flags %s", err.Error())
		os.Exit(1)
	}

	if *settings.DatabaseDir == "" {
		database, err := sql.Open("sqlite3", *settings.Database)
		if err != nil {
			log.Printf("error opening database %s: %s", *settings.Database, err.Error())
			os.Exit(1)
		}
		err = database.Ping()
		if err != nil {
			log.Printf("error pinging database: %s", err.Error())
			os.Exit(1)
		}

		tx, err := database.BeginTx(context.Background(), &sql.TxOptions{ReadOnly: false})
		if err != nil {
			log.Printf("can't create a context: %s", err.Error())
			os.Exit(1)
		}
		defer tx.Rollback()

		err = makedraft.MakeDraft(settings, tx, nil)
		if err != nil {
			log.Printf("%s", err.Error())
			os.Exit(1)
		}

		if !*settings.Simulate {
			err = tx.Commit()
		} else {
			err = nil
		}

		if err != nil {
			log.Printf("can't commit :( %s", err.Error())
			os.Exit(1)
		}
	} else {
		ob, err := objectbox.NewBuilder().Model(schema.ObjectBoxModel()).Directory(*settings.DatabaseDir).Build()
		if err != nil {
			log.Printf("error opening database %s: %s", *settings.DatabaseDir, err.Error())
			os.Exit(1)
		}
		defer ob.Close()

		err = ob.RunInWriteTx(func() error {
			return makedraft.MakeDraft(settings, nil, ob)
		})
		if err != nil {
			log.Printf("%s", err.Error())
			os.Exit(1)
		}
	}

	log.Printf("done!")
	os.Exit(0)
}
