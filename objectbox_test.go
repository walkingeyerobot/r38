package main

import (
	"fmt"
	"github.com/objectbox/objectbox-go/objectbox"
	"github.com/walkingeyerobot/r38/makedraft"
	"github.com/walkingeyerobot/r38/schema"
	"golang.org/x/net/xsrftoken"
	"io"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
)

func doSetupOB(t *testing.T, seed int) (*objectbox.ObjectBox, error) {
	t.SkipNow()
	xsrfKey = "test"
	sock = ""

	ob, err := objectbox.NewBuilder().Model(schema.ObjectBoxModel()).Directory("memory:test-db").Build()
	if err != nil {
		t.Error(err)
	}

	_, err = schema.BoxForUser(ob).PutMany([]*schema.User{
		{DiscordName: "Ashiok"},
		{DiscordName: "Chandra"},
		{DiscordName: "Elspeth"},
		{DiscordName: "Jaya"},
		{DiscordName: "Kaya"},
		{DiscordName: "Liliana"},
		{DiscordName: "Nahiri"},
		{DiscordName: "Serra"},
		{DiscordName: "Jace"},
		{DiscordName: "Basri"},
		{DiscordName: "Ajani"},
		{DiscordName: "Gideon"},
	})
	if err != nil {
		t.Error(err)
	}

	rand.Seed(int64(seed))
	return ob, err
}

func makeDraftOB(t *testing.T, ob *objectbox.ObjectBox, seed int) {
	err := ob.RunInWriteTx(func() error {
		err := makedraft.MakeDraft(makedraft.Settings{
			Set:                              ptr("sets/cube.json"),
			Database:                         ptr(""),
			Seed:                             &seed,
			InPerson:                         ptr(true),
			AssignSeats:                      ptr(false),
			AssignPacks:                      ptr(false),
			Verbose:                          ptr(false),
			Simulate:                         ptr(false),
			Name:                             ptr("test draft"),
			MaxMythic:                        ptr(0),
			MaxRare:                          ptr(0),
			MaxUncommon:                      ptr(0),
			MaxCommon:                        ptr(0),
			PackCommonColorStdevMax:          ptr(0.0),
			PackCommonRatingMin:              ptr(0.0),
			PackCommonRatingMax:              ptr(0.0),
			DraftCommonColorStdevMax:         ptr(0.0),
			PackCommonColorIdentityStdevMax:  ptr(0.0),
			DraftCommonColorIdentityStdevMax: ptr(0.0),
			DfcMode:                          ptr(false),
			AbortMissingCommonColor:          ptr(false),
			AbortMissingCommonColorIdentity:  ptr(false),
			AbortDuplicateThreeColorIdentityUncommons: ptr(false),
		}, nil, ob)
		return err
	})
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
}

func findCardToPickOB(t *testing.T, ob *objectbox.ObjectBox, seat int, round int, card int) string {
	draft, err := schema.BoxForDraft(ob).Get(1)
	if err != nil {
		t.Errorf("error reading draft %s", err.Error())
		t.FailNow()
	}
	if card == 0 {
		pack := draft.UnassignedPacks[rand.Intn(len(draft.UnassignedPacks))]
		return pack.Cards[rand.Intn(len(pack.Cards))].CardId
	} else {
		packs := draft.Seats[seat].Packs
		for _, pack := range packs {
			if pack.Round == round && len(pack.Cards) == 15-card {
				if len(pack.Cards) == 0 {
					t.Errorf("no cards in pack for seat %d, pack %d, pick %d", seat, round+1, card+1)
				}
				return pack.Cards[rand.Intn(len(pack.Cards))].CardId
			}
		}
	}
	t.Errorf("no pack found for seat %d, pack %d, pick %d", seat, round+1, card+1)
	t.FailNow()
	return ""
}

func FuzzInPersonDraftOB(f *testing.F) {
	f.Add(SEED)
	f.Fuzz(func(t *testing.T, seed int) {
		ob, err := doSetupOB(t, seed)
		if err != nil {
			t.Errorf("error in setup: %s", err.Error())
			t.FailNow()
		}
		defer ob.Close()

		makeDraftOB(t, ob, seed)

		handlers := NewHandler(nil, ob, false)

		players, seats := populateDraft(t, handlers)

		for round := range 3 {
			for card := range 15 {
				for _, seat := range rand.Perm(8) {
					player := players[seat] + 1

					cardId := findCardToPickOB(t, ob, player, round, card)

					token := xsrftoken.Generate(xsrfKey, strconv.FormatInt(int64(player), 16), "pick1")

					t.Logf("pack %d pick %d: player %d (position %d) picking card %s", round+1, card+1, player, seats[seat]+1, cardId)
					w := httptest.NewRecorder()
					handlers.ServeHTTP(w,
						httptest.NewRequest("POST", fmt.Sprintf("/api/pickrfid/?as=%d", player),
							strings.NewReader(fmt.Sprintf(`{"draftId": 1, "cardRfids": ["%s"], "xsrfToken": "%s"}`, cardId, token))))
					res := w.Result()
					if res.StatusCode != http.StatusOK {
						body, _ := io.ReadAll(res.Body)
						t.Errorf("pick failed: %s", body)
					}
				}

			}
		}
	})
}
