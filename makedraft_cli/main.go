package main

import (
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

	ob, err := objectbox.NewBuilder().Model(schema.ObjectBoxModel()).Directory(*settings.DatabaseDir).Build()
	if err != nil {
		log.Printf("error opening database %s: %s", *settings.DatabaseDir, err.Error())
		os.Exit(1)
	}
	defer ob.Close()

	err = ob.RunInWriteTx(func() error {
		return makedraft.MakeDraft(settings, ob)
	})
	if err != nil {
		log.Printf("%s", err.Error())
		os.Exit(1)
	}

	log.Printf("done!")
	os.Exit(0)
}
