package main

import (
	"flag"
	"log"
	"os"

	"github.com/davecgh/go-spew/spew"
	"github.com/objectbox/objectbox-go/objectbox"

	"github.com/walkingeyerobot/r38/schema"
)

func main() {
	flagSet := flag.NewFlagSet(os.Args[0], flag.ContinueOnError)

	draftId := flagSet.Uint64("id", 0, "ID of the draft to dump.")

	err := flagSet.Parse(os.Args[1:])
	if err != nil {
		log.Printf("error parsing flags: %s", err.Error())
		os.Exit(1)
	}

	ob, err := objectbox.NewBuilder().Model(schema.ObjectBoxModel()).Build()
	if err != nil {
		log.Printf("error opening database: %s", err.Error())
		os.Exit(1)
	}
	defer ob.Close()

	box := schema.BoxForDraft(ob)

	draft, err := box.Get(*draftId)
	if err != nil {
		log.Printf("error reading draft: %s", err.Error())
		os.Exit(1)
	}
	if draft == nil {
		log.Printf("draft not found")
		os.Exit(1)
	}

	spew.Dump(draft)
}
