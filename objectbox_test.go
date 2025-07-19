package main

import (
	"fmt"
	"github.com/davecgh/go-spew/spew"
	"github.com/objectbox/objectbox-go/objectbox"
	"github.com/walkingeyerobot/r38/makedraft"
	"github.com/walkingeyerobot/r38/schema"
	"golang.org/x/net/xsrftoken"
	"io"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"os"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"
)

func doSetupOB(t *testing.T, seed int) (*objectbox.ObjectBox, error) {
	xsrfKey = "test"
	sock = ""

	ob, err := objectbox.NewBuilder().Model(schema.ObjectBoxModel()).
		Directory(fmt.Sprintf("memory:test-db-%d-%d", time.Now().UnixNano(), os.Getpid())).Build()
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

func findCardToPickOB(t *testing.T, ob *objectbox.ObjectBox, seat int, round int, card int) *schema.Card {
	draft, err := schema.BoxForDraft(ob).Get(1)
	if err != nil {
		t.Errorf("error reading draft %s", err.Error())
		t.FailNow()
	}
	if card == 0 {
		pack := draft.UnassignedPacks[rand.Intn(len(draft.UnassignedPacks))]
		return pack.Cards[rand.Intn(len(pack.Cards))]
	} else {
		seatIndex := slices.IndexFunc(draft.Seats, func(s *schema.Seat) bool {
			return s.Position == seat
		})
		packs := draft.Seats[seatIndex].Packs
		for _, pack := range packs {
			if pack.Round == round+1 && len(pack.Cards) == 15-card {
				if len(pack.Cards) == 0 {
					t.Errorf("no cards in pack for seat %d, pack %d, pick %d", seat, round+1, card+1)
				}
				return pack.Cards[rand.Intn(len(pack.Cards))]
			}
		}
		spew.Dump(packs)
	}
	t.Errorf("no pack found for seat %d, pack %d, pick %d", seat, round+1, card+1)
	t.FailNow()
	return nil
}

func TestInPersonDraftOB(t *testing.T) {
	ob, err := doSetupOB(t, SEED)
	if err != nil {
		t.Errorf("error in setup: %s", err.Error())
		t.FailNow()
	}
	defer ob.Close()

	makeDraftOB(t, ob, SEED)

	handlers := NewHandler(nil, ob, false)

	players, seats := populateDraft(t, handlers)

	for round := range 3 {
		for card := range 15 {
			for _, seat := range rand.Perm(8) {
				player := players[seat] + 1

				cardId := findCardToPickOB(t, ob, seats[seat], round, card).CardId

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
					spew.Dump(schema.BoxForDraft(ob).Get(1))
					t.FailNow()
				}
			}

		}
	}
}

func TestInPersonDraftUndoFirstPickOB(t *testing.T) {
	ob, err := doSetupOB(t, SEED)
	defer ob.Close()

	makeDraftOB(t, ob, SEED)

	handlers := NewHandler(nil, ob, false)

	players, seats := populateDraft(t, handlers)

	player := players[0] + 1
	card := findCardToPickOB(t, ob, seats[0], 0, 0)

	token := xsrftoken.Generate(xsrfKey, strconv.FormatInt(int64(player), 16), "pick1")

	handlers.ServeHTTP(httptest.NewRecorder(),
		httptest.NewRequest("POST", fmt.Sprintf("/api/pickrfid/?as=%d", player),
			strings.NewReader(fmt.Sprintf(`{"draftId": 1, "cardRfids": ["%s"], "xsrfToken": "%s"}`, card.CardId, token))))

	w := httptest.NewRecorder()
	handlers.ServeHTTP(w,
		httptest.NewRequest("POST", fmt.Sprintf("/api/undopick/?as=%d", player),
			strings.NewReader(fmt.Sprintf(`{"draftId": 1, "xsrfToken": "%s"}`, token))))
	res := w.Result()
	if res.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(res.Body)
		t.Errorf("undo failed: %s", body)
	}

	draft, err := schema.BoxForDraft(ob).Get(1)
	if err != nil {
		t.Errorf("couldn't get draft: %s", err.Error())
	}
	if len(draft.Events) != 0 {
		t.Error("didn't delete event")
	}

	for _, seat := range draft.Seats {
		if seat.Position == seats[0] {
			if len(seat.PickedCards) != 0 {
				t.Errorf("didn't remove card from picked cards")
			}
			if len(seat.Packs) != 1 {
				t.Errorf("removed pack from seat")
			}
			cardIndex := slices.IndexFunc(seat.Packs[0].Cards, func(c *schema.Card) bool {
				return c.Id == card.Id
			})
			if cardIndex == -1 {
				t.Errorf("didn't put card back in pack")
			}
		}
	}
}

func TestInPersonDraftIgnoresDuplicatePick(t *testing.T) {
	ob, err := doSetupOB(t, SEED)
	defer ob.Close()

	makeDraftOB(t, ob, SEED)

	handlers := NewHandler(nil, ob, false)

	players, seats := populateDraft(t, handlers)

	player := players[0] + 1
	card := findCardToPickOB(t, ob, seats[0], 0, 0)

	token := xsrftoken.Generate(xsrfKey, strconv.FormatInt(int64(player), 16), "pick1")

	handlers.ServeHTTP(httptest.NewRecorder(),
		httptest.NewRequest("POST", fmt.Sprintf("/api/pickrfid/?as=%d", player),
			strings.NewReader(fmt.Sprintf(`{"draftId": 1, "cardRfids": ["%s"], "xsrfToken": "%s"}`, card.CardId, token))))

	w := httptest.NewRecorder()
	handlers.ServeHTTP(w,
		httptest.NewRequest("POST", fmt.Sprintf("/api/pickrfid/?as=%d", player),
			strings.NewReader(fmt.Sprintf(`{"draftId": 1, "cardRfids": ["%s"], "xsrfToken": "%s"}`, card.CardId, token))))
	res := w.Result()
	if res.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(res.Body)
		t.Errorf("error on duplicate scan: %s", body)
	}

	draft, err := schema.BoxForDraft(ob).Get(1)
	if err != nil {
		t.Errorf("couldn't get draft: %s", err.Error())
	}
	if len(draft.Events) != 1 {
		t.Errorf("wrong number of events (%d)", len(draft.Events))
	}

	for _, seat := range draft.Seats {
		if seat.Position == seats[0] {
			if len(seat.PickedCards) != 1 {
				t.Errorf("wrong number of picked cards in seat (%d)", len(seat.PickedCards))
			}
		}
	}
}

func TestInPersonDraftUndoSubsequentPickOB(t *testing.T) {
	ob, err := doSetupOB(t, SEED)
	defer ob.Close()

	makeDraftOB(t, ob, SEED)

	handlers := NewHandler(nil, ob, false)

	players, seats := populateDraft(t, handlers)

	player := players[0] + 1

	for card := range 5 {
		for _, seat := range rand.Perm(8) {
			player := players[seat] + 1

			card := findCardToPickOB(t, ob, seats[seat], 0, card)

			token := xsrftoken.Generate(xsrfKey, strconv.FormatInt(int64(player), 16), "pick1")

			handlers.ServeHTTP(httptest.NewRecorder(),
				httptest.NewRequest("POST", fmt.Sprintf("/api/pickrfid/?as=%d", player),
					strings.NewReader(fmt.Sprintf(`{"draftId": 1, "cardRfids": ["%s"], "xsrfToken": "%s"}`, card.CardId, token))))
		}
	}

	card := findCardToPickOB(t, ob, seats[0], 0, 5)

	token := xsrftoken.Generate(xsrfKey, strconv.FormatInt(int64(player), 16), "pick1")

	handlers.ServeHTTP(httptest.NewRecorder(),
		httptest.NewRequest("POST", fmt.Sprintf("/api/pickrfid/?as=%d", player),
			strings.NewReader(fmt.Sprintf(`{"draftId": 1, "cardRfids": ["%s"], "xsrfToken": "%s"}`, card.CardId, token))))

	w := httptest.NewRecorder()
	handlers.ServeHTTP(w,
		httptest.NewRequest("POST", fmt.Sprintf("/api/undopick/?as=%d", player),
			strings.NewReader(fmt.Sprintf(`{"draftId": 1, "xsrfToken": "%s"}`, token))))
	res := w.Result()
	if res.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(res.Body)
		t.Errorf("undo failed: %s", body)
	}

	draft, err := schema.BoxForDraft(ob).Get(1)
	if err != nil {
		t.Errorf("couldn't find draft: %s", err.Error())
		t.FailNow()
	}

	for _, event := range draft.Events {
		if event.Card1.Id == card.Id {
			t.Errorf("didn't delete event")
			break
		}
	}

	for _, seat := range draft.Seats {
		if seat.Position == seats[0] {
			if slices.IndexFunc(seat.PickedCards, func(c *schema.Card) bool {
				return c.Id == card.Id
			}) != -1 {
				t.Errorf("didn't remove card from picked cards")
			}
			if slices.IndexFunc(seat.Packs, func(pack *schema.Pack) bool {
				return slices.IndexFunc(pack.Cards, func(c *schema.Card) bool {
					return c.Id == card.Id
				}) != -1
			}) == -1 {
				t.Errorf("didn't put card back in pack or pack back in seat")
			}
		}
	}
}

func TestInPersonDraftEnforceZoneDraftingNextPlayerMakingFirstPickOB(t *testing.T) {
	ob, err := doSetupOB(t, SEED)
	if err != nil {
		t.Errorf("setup error: %s", err.Error())
	}
	defer ob.Close()

	makeDraftOB(t, ob, SEED)

	handlers := NewHandler(nil, ob, false)

	players, seats := populateDraft(t, handlers)

	player := players[0] + 1
	seat := seats[0]
	previousSeat := seat - 1
	if previousSeat < 0 {
		previousSeat += 8
	}
	previousSeatIndex := slices.Index(seats, previousSeat)
	previousPlayer := players[previousSeatIndex] + 1

	token := xsrftoken.Generate(xsrfKey, strconv.FormatInt(int64(player), 16), "pick1")

	// test player makes first pick
	card := findCardToPickOB(t, ob, seat, 0, 0)
	handlers.ServeHTTP(httptest.NewRecorder(),
		httptest.NewRequest("POST", fmt.Sprintf("/api/pickrfid/?as=%d", player),
			strings.NewReader(fmt.Sprintf(`{"draftId": 1, "cardRfids": ["%s"], "xsrfToken": "%s"}`, card.CardId, token))))

	// player before test player makes first pick
	card = findCardToPickOB(t, ob, previousSeat, 0, 0)
	prevToken := xsrftoken.Generate(xsrfKey, strconv.FormatInt(int64(previousPlayer), 16), "pick1")
	handlers.ServeHTTP(httptest.NewRecorder(),
		httptest.NewRequest("POST", fmt.Sprintf("/api/pickrfid/?as=%d", previousPlayer),
			strings.NewReader(fmt.Sprintf(`{"draftId": 1, "cardRfids": ["%s"], "xsrfToken": "%s"}`, card.CardId, prevToken))))

	// test player makes second pick in violation
	card = findCardToPickOB(t, ob, seat, 0, 1)
	w := httptest.NewRecorder()
	handlers.ServeHTTP(w,
		httptest.NewRequest("POST", fmt.Sprintf("/api/pickrfid/?as=%d", player),
			strings.NewReader(fmt.Sprintf(`{"draftId": 1, "cardRfids": ["%s"], "xsrfToken": "%s"}`, card.CardId, token))))

	res := w.Result()
	if res.StatusCode != http.StatusBadRequest {
		t.Errorf("expected pick to fail due to zone drafting violation, but status code was %d", res.StatusCode)
	}
}

func TestInPersonDraftEnforceZoneDraftingNextPlayerMakingSubsequentPickOB(t *testing.T) {
	ob, err := doSetupOB(t, SEED)
	if err != nil {
		t.Errorf("setup error: %s", err.Error())
	}
	defer ob.Close()

	makeDraftOB(t, ob, SEED)

	handlers := NewHandler(nil, ob, false)

	players, seats := populateDraft(t, handlers)

	player := players[0] + 1
	seat := seats[0]

	previousSeat := seat - 1
	if previousSeat < 0 {
		previousSeat += 8
	}
	previousSeatIndex := slices.Index(seats, previousSeat)
	previousPlayer := players[previousSeatIndex] + 1
	prevToken := xsrftoken.Generate(xsrfKey, strconv.FormatInt(int64(previousPlayer), 16), "pick1")

	prevPrevSeat := seat - 2
	if prevPrevSeat < 0 {
		prevPrevSeat += 8
	}
	prevPrevSeatIndex := slices.Index(seats, prevPrevSeat)
	prevPrevPlayer := players[prevPrevSeatIndex] + 1
	prevPrevToken := xsrftoken.Generate(xsrfKey, strconv.FormatInt(int64(prevPrevPlayer), 16), "pick1")

	nextSeat := seat + 1
	if nextSeat > 7 {
		nextSeat -= 8
	}
	nextSeatIndex := slices.Index(seats, nextSeat)
	nextPlayer := players[nextSeatIndex] + 1
	nextToken := xsrftoken.Generate(xsrfKey, strconv.FormatInt(int64(nextPlayer), 16), "pick1")

	token := xsrftoken.Generate(xsrfKey, strconv.FormatInt(int64(player), 16), "pick1")

	t.Logf("player %d making first pick", player)
	card := findCardToPickOB(t, ob, seat, 0, 0)
	handlers.ServeHTTP(httptest.NewRecorder(),
		httptest.NewRequest("POST", fmt.Sprintf("/api/pickrfid/?as=%d", player),
			strings.NewReader(fmt.Sprintf(`{"draftId": 1, "cardRfids": ["%s"], "xsrfToken": "%s"}`, card.CardId, token))))

	t.Logf("left player %d making first pick", previousPlayer)
	card = findCardToPickOB(t, ob, previousSeat, 0, 0)
	handlers.ServeHTTP(httptest.NewRecorder(),
		httptest.NewRequest("POST", fmt.Sprintf("/api/pickrfid/?as=%d", previousPlayer),
			strings.NewReader(fmt.Sprintf(`{"draftId": 1, "cardRfids": ["%s"], "xsrfToken": "%s"}`, card.CardId, prevToken))))

	t.Logf("right player %d making first pick", nextPlayer)
	card = findCardToPickOB(t, ob, nextSeat, 0, 0)
	handlers.ServeHTTP(httptest.NewRecorder(),
		httptest.NewRequest("POST", fmt.Sprintf("/api/pickrfid/?as=%d", nextPlayer),
			strings.NewReader(fmt.Sprintf(`{"draftId": 1, "cardRfids": ["%s"], "xsrfToken": "%s"}`, card.CardId, nextToken))))

	t.Logf("player %d making second pick", player)
	card = findCardToPickOB(t, ob, seat, 0, 1)
	handlers.ServeHTTP(httptest.NewRecorder(),
		httptest.NewRequest("POST", fmt.Sprintf("/api/pickrfid/?as=%d", player),
			strings.NewReader(fmt.Sprintf(`{"draftId": 1, "cardRfids": ["%s"], "xsrfToken": "%s"}`, card.CardId, token))))

	t.Logf("2 left player %d making first pick", prevPrevPlayer)
	card = findCardToPickOB(t, ob, prevPrevSeat, 0, 0)
	handlers.ServeHTTP(httptest.NewRecorder(),
		httptest.NewRequest("POST", fmt.Sprintf("/api/pickrfid/?as=%d", prevPrevPlayer),
			strings.NewReader(fmt.Sprintf(`{"draftId": 1, "cardRfids": ["%s"], "xsrfToken": "%s"}`, card.CardId, prevPrevToken))))

	t.Logf("left player %d making second pick", previousPlayer)
	card = findCardToPickOB(t, ob, previousSeat, 0, 1)
	handlers.ServeHTTP(httptest.NewRecorder(),
		httptest.NewRequest("POST", fmt.Sprintf("/api/pickrfid/?as=%d", previousPlayer),
			strings.NewReader(fmt.Sprintf(`{"draftId": 1, "cardRfids": ["%s"], "xsrfToken": "%s"}`, card.CardId, prevToken))))

	t.Logf("player %d making third pick", player)
	card = findCardToPickOB(t, ob, seat, 0, 2)
	w := httptest.NewRecorder()
	handlers.ServeHTTP(w,
		httptest.NewRequest("POST", fmt.Sprintf("/api/pickrfid/?as=%d", player),
			strings.NewReader(fmt.Sprintf(`{"draftId": 1, "cardRfids": ["%s"], "xsrfToken": "%s"}`, card.CardId, token))))

	res := w.Result()
	if res.StatusCode != http.StatusBadRequest {
		t.Errorf("expected pick to fail due to zone drafting violation, but status code was %d", res.StatusCode)
	}
}
