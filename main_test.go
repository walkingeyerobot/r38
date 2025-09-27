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

var SEED = 677483

func doSetup(t *testing.T, seed int) (*objectbox.ObjectBox, error) {
	xsrfKey = "test"
	filterSocket = ""
	ignoredDiscordCalls = nil

	ob, err := objectbox.NewBuilder().Model(schema.ObjectBoxModel()).
		Directory(fmt.Sprintf("memory:test-db-%d-%d", time.Now().UnixNano(), os.Getpid())).Build()
	if err != nil {
		t.Error(err)
	}

	_, err = schema.BoxForUser(ob).PutMany([]*schema.User{
		{DiscordName: "Ashiok", DiscordId: "1"},
		{DiscordName: "Chandra", DiscordId: "2"},
		{DiscordName: "Elspeth", DiscordId: "3"},
		{DiscordName: "Jaya", DiscordId: "4"},
		{DiscordName: "Kaya", DiscordId: "5"},
		{DiscordName: "Liliana", DiscordId: "6"},
		{DiscordName: "Nahiri", DiscordId: "7"},
		{DiscordName: "Serra", DiscordId: "8"},
		{DiscordName: "Jace", DiscordId: "9"},
		{DiscordName: "Basri", DiscordId: "10"},
		{DiscordName: "Ajani", DiscordId: "11"},
		{DiscordName: "Gideon", DiscordId: "12"},
	})
	if err != nil {
		t.Error(err)
	}

	rand.Seed(int64(seed))
	return ob, err
}

func makeDraft(t *testing.T, handlers http.Handler, seed int, inPerson bool, pickTwo bool) {
	makedraft.FakeSpectatorChannelID = "spectator-channel"
	w := httptest.NewRecorder()
	handlers.ServeHTTP(w,
		httptest.NewRequest("POST", "/api/makedraft/?as=1",
			strings.NewReader(fmt.Sprintf(`{
				"name": "test draft",
				"seed": %d,
				"inPerson": %t,
				"pickTwo": %t
			}`, seed, inPerson, pickTwo))))

	res := w.Result()
	if res.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(res.Body)
		t.Errorf("error making draft: %s", body)
		t.FailNow()
	}
}

func populateDraft(t *testing.T, handlers http.Handler, numSeats int) (players []int, seats []int) {
	players = rand.Perm(12)
	seats = rand.Perm(numSeats)
	for seat := range numSeats {
		t.Logf("joining seat %d as player %d", seats[seat]+1, players[seat]+1)
		w := httptest.NewRecorder()
		handlers.ServeHTTP(w,
			httptest.NewRequest("POST", fmt.Sprintf("/api/join/?as=%d", players[seat]+1),
				strings.NewReader(fmt.Sprintf(`{"id": 1, "position": %d}`, seats[seat]))))
		res := w.Result()
		if res.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(res.Body)
			t.Errorf("player joining draft failed: %s", body)
		}
	}
	return
}

func findCardToPick(t *testing.T, ob *objectbox.ObjectBox, seat int, round int, card int, inPerson bool) *schema.Card {
	draft, err := schema.BoxForDraft(ob).Get(1)
	if err != nil {
		t.Errorf("error reading draft %s", err.Error())
		t.FailNow()
	}
	if inPerson && card == 0 {
		pack := draft.UnassignedPacks[rand.Intn(len(draft.UnassignedPacks))]
		return pack.Cards[rand.Intn(len(pack.Cards))]
	} else {
		seatIndex := slices.IndexFunc(draft.Seats, func(s *schema.Seat) bool {
			return s.Position == seat
		})
		packs := draft.Seats[seatIndex].Packs
		var cardsPerPack int
		if draft.PickTwo {
			cardsPerPack = 14
		} else {
			cardsPerPack = 15
		}
		for _, pack := range packs {
			if pack.Round == round+1 && len(pack.Cards) == cardsPerPack-card {
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

func TestOnlineDraft(t *testing.T) {
	ob, err := doSetup(t, SEED)
	if err != nil {
		t.Errorf("error in setup: %s", err.Error())
		t.FailNow()
	}
	defer ob.Close()

	handlers := NewHandler(ob, false)

	makeDraft(t, handlers, SEED, false, false)

	players, seats := populateDraft(t, handlers, 8)

	for round := range 3 {
		for card := range 15 {
			for _, seat := range rand.Perm(8) {
				player := players[seat] + 1

				cardId := findCardToPick(t, ob, seats[seat], round, card, false).Id

				token := xsrftoken.Generate(xsrfKey, strconv.FormatInt(int64(player), 16), "pick1")

				t.Logf("pack %d pick %d: player %d (position %d) picking card %d", round+1, card+1, player, seats[seat]+1, cardId)
				w := httptest.NewRecorder()
				handlers.ServeHTTP(w,
					httptest.NewRequest("POST", fmt.Sprintf("/api/pick/?as=%d", player),
						strings.NewReader(fmt.Sprintf(`{"draftId": 1, "cards": [%d], "xsrfToken": "%s"}`, cardId, token))))
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

func TestInPersonDraft(t *testing.T) {
	ob, err := doSetup(t, SEED)
	if err != nil {
		t.Errorf("error in setup: %s", err.Error())
		t.FailNow()
	}
	defer ob.Close()

	handlers := NewHandler(ob, false)

	makeDraft(t, handlers, SEED, true, false)

	players, seats := populateDraft(t, handlers, 8)

	for round := range 3 {
		for card := range 15 {
			for _, seat := range rand.Perm(8) {
				player := players[seat] + 1

				cardId := findCardToPick(t, ob, seats[seat], round, card, true).CardId

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

func FuzzInPersonDraft(f *testing.F) {
	f.Add(SEED)
	f.Fuzz(func(t *testing.T, seed int) {
		ob, err := doSetup(t, seed)
		if err != nil {
			t.Errorf("error in setup: %s", err.Error())
			t.FailNow()
		}
		defer ob.Close()

		handlers := NewHandler(ob, false)

		makeDraft(t, handlers, SEED, true, false)

		players, seats := populateDraft(t, handlers, 8)

		for round := range 3 {
			for card := range 15 {
				for _, seat := range rand.Perm(8) {
					player := players[seat] + 1

					cardId := findCardToPick(t, ob, seats[seat], round, card, true).CardId

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
	})
}

func TestInPersonPickTwoDraft(t *testing.T) {
	ob, err := doSetup(t, SEED)
	if err != nil {
		t.Errorf("error in setup: %s", err.Error())
		t.FailNow()
	}
	defer ob.Close()

	handlers := NewHandler(ob, false)

	makeDraft(t, handlers, SEED, true, true)

	players, seats := populateDraft(t, handlers, 4)

	for round := range 3 {
		for cardPair := range 7 {
			var pickedFirst [4]bool
			for _, seat := range rand.Perm(8) {
				seat /= 2
				card := cardPair * 2
				if pickedFirst[seat] {
					card++
				} else {
					pickedFirst[seat] = true
				}
				player := players[seat] + 1

				cardId := findCardToPick(t, ob, seats[seat], round, card, true).CardId

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

func TestInPersonDraftUndoFirstPick(t *testing.T) {
	ob, err := doSetup(t, SEED)
	defer ob.Close()

	handlers := NewHandler(ob, false)

	makeDraft(t, handlers, SEED, true, false)

	players, seats := populateDraft(t, handlers, 8)

	player := players[0] + 1
	card := findCardToPick(t, ob, seats[0], 0, 0, true)

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
		t.FailNow()
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
	ob, err := doSetup(t, SEED)
	defer ob.Close()

	handlers := NewHandler(ob, false)

	makeDraft(t, handlers, SEED, true, false)

	players, seats := populateDraft(t, handlers, 8)

	player := players[0] + 1
	card := findCardToPick(t, ob, seats[0], 0, 0, true)

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
		t.FailNow()
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

func TestInPersonDraftUndoSubsequentPick(t *testing.T) {
	ob, err := doSetup(t, SEED)
	defer ob.Close()

	handlers := NewHandler(ob, false)

	makeDraft(t, handlers, SEED, true, false)

	players, seats := populateDraft(t, handlers, 8)

	player := players[0] + 1

	for card := range 5 {
		for _, seat := range rand.Perm(8) {
			player := players[seat] + 1

			card := findCardToPick(t, ob, seats[seat], 0, card, true)

			token := xsrftoken.Generate(xsrfKey, strconv.FormatInt(int64(player), 16), "pick1")

			handlers.ServeHTTP(httptest.NewRecorder(),
				httptest.NewRequest("POST", fmt.Sprintf("/api/pickrfid/?as=%d", player),
					strings.NewReader(fmt.Sprintf(`{"draftId": 1, "cardRfids": ["%s"], "xsrfToken": "%s"}`, card.CardId, token))))
		}
	}

	card := findCardToPick(t, ob, seats[0], 0, 5, true)

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

func TestInPersonDraftPickAfterUndo(t *testing.T) {
	ob, err := doSetup(t, SEED)
	defer ob.Close()

	handlers := NewHandler(ob, false)

	makeDraft(t, handlers, SEED, true, false)

	players, seats := populateDraft(t, handlers, 8)

	player := players[0] + 1

	card := findCardToPick(t, ob, seats[0], 0, 0, true)

	token := xsrftoken.Generate(xsrfKey, strconv.FormatInt(int64(player), 16), "pick1")

	handlers.ServeHTTP(httptest.NewRecorder(),
		httptest.NewRequest("POST", fmt.Sprintf("/api/pickrfid/?as=%d", player),
			strings.NewReader(fmt.Sprintf(`{"draftId": 1, "cardRfids": ["%s"], "xsrfToken": "%s"}`, card.CardId, token))))

	for _, seat := range rand.Perm(3) {
		player := players[seat+1] + 1

		card := findCardToPick(t, ob, seats[seat+1], 0, 0, true)

		token := xsrftoken.Generate(xsrfKey, strconv.FormatInt(int64(player), 16), "pick1")

		handlers.ServeHTTP(httptest.NewRecorder(),
			httptest.NewRequest("POST", fmt.Sprintf("/api/pickrfid/?as=%d", player),
				strings.NewReader(fmt.Sprintf(`{"draftId": 1, "cardRfids": ["%s"], "xsrfToken": "%s"}`, card.CardId, token))))
	}

	w := httptest.NewRecorder()
	handlers.ServeHTTP(w,
		httptest.NewRequest("POST", fmt.Sprintf("/api/undopick/?as=%d", player),
			strings.NewReader(fmt.Sprintf(`{"draftId": 1, "xsrfToken": "%s"}`, token))))
	res := w.Result()
	if res.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(res.Body)
		t.Errorf("undo failed: %s", body)
	}

	card = findCardToPick(t, ob, seats[0], 0, 0, true)

	handlers.ServeHTTP(httptest.NewRecorder(),
		httptest.NewRequest("POST", fmt.Sprintf("/api/pickrfid/?as=%d", player),
			strings.NewReader(fmt.Sprintf(`{"draftId": 1, "cardRfids": ["%s"], "xsrfToken": "%s"}`, card.CardId, token))))

	draft, err := schema.BoxForDraft(ob).Get(1)
	if err != nil {
		t.Errorf("couldn't find draft: %s", err.Error())
		t.FailNow()
	}

	for _, event := range draft.Events {
		for _, otherEvent := range draft.Events {
			if event.Id != otherEvent.Id && event.Modified == otherEvent.Modified {
				t.Errorf("duplicate events %+v %+v", event, otherEvent)
				break
			}
		}
	}
}

func TestInPersonDraftEnforceZoneDraftingNextPlayerMakingFirstPick(t *testing.T) {
	ob, err := doSetup(t, SEED)
	if err != nil {
		t.Errorf("setup error: %s", err.Error())
	}
	defer ob.Close()

	handlers := NewHandler(ob, false)

	makeDraft(t, handlers, SEED, true, false)

	players, seats := populateDraft(t, handlers, 8)

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
	card := findCardToPick(t, ob, seat, 0, 0, true)
	handlers.ServeHTTP(httptest.NewRecorder(),
		httptest.NewRequest("POST", fmt.Sprintf("/api/pickrfid/?as=%d", player),
			strings.NewReader(fmt.Sprintf(`{"draftId": 1, "cardRfids": ["%s"], "xsrfToken": "%s"}`, card.CardId, token))))

	// player before test player makes first pick
	card = findCardToPick(t, ob, previousSeat, 0, 0, true)
	prevToken := xsrftoken.Generate(xsrfKey, strconv.FormatInt(int64(previousPlayer), 16), "pick1")
	handlers.ServeHTTP(httptest.NewRecorder(),
		httptest.NewRequest("POST", fmt.Sprintf("/api/pickrfid/?as=%d", previousPlayer),
			strings.NewReader(fmt.Sprintf(`{"draftId": 1, "cardRfids": ["%s"], "xsrfToken": "%s"}`, card.CardId, prevToken))))

	// test player makes second pick in violation
	card = findCardToPick(t, ob, seat, 0, 1, true)
	w := httptest.NewRecorder()
	handlers.ServeHTTP(w,
		httptest.NewRequest("POST", fmt.Sprintf("/api/pickrfid/?as=%d", player),
			strings.NewReader(fmt.Sprintf(`{"draftId": 1, "cardRfids": ["%s"], "xsrfToken": "%s"}`, card.CardId, token))))

	res := w.Result()
	if res.StatusCode != http.StatusBadRequest {
		t.Errorf("expected pick to fail due to zone drafting violation, but status code was %d", res.StatusCode)
	}
}

func TestInPersonDraftEnforceZoneDraftingNextPlayerMakingSubsequentPick(t *testing.T) {
	ob, err := doSetup(t, SEED)
	if err != nil {
		t.Errorf("setup error: %s", err.Error())
	}
	defer ob.Close()

	handlers := NewHandler(ob, false)

	makeDraft(t, handlers, SEED, true, false)

	players, seats := populateDraft(t, handlers, 8)

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
	card := findCardToPick(t, ob, seat, 0, 0, true)
	handlers.ServeHTTP(httptest.NewRecorder(),
		httptest.NewRequest("POST", fmt.Sprintf("/api/pickrfid/?as=%d", player),
			strings.NewReader(fmt.Sprintf(`{"draftId": 1, "cardRfids": ["%s"], "xsrfToken": "%s"}`, card.CardId, token))))

	t.Logf("left player %d making first pick", previousPlayer)
	card = findCardToPick(t, ob, previousSeat, 0, 0, true)
	handlers.ServeHTTP(httptest.NewRecorder(),
		httptest.NewRequest("POST", fmt.Sprintf("/api/pickrfid/?as=%d", previousPlayer),
			strings.NewReader(fmt.Sprintf(`{"draftId": 1, "cardRfids": ["%s"], "xsrfToken": "%s"}`, card.CardId, prevToken))))

	t.Logf("right player %d making first pick", nextPlayer)
	card = findCardToPick(t, ob, nextSeat, 0, 0, true)
	handlers.ServeHTTP(httptest.NewRecorder(),
		httptest.NewRequest("POST", fmt.Sprintf("/api/pickrfid/?as=%d", nextPlayer),
			strings.NewReader(fmt.Sprintf(`{"draftId": 1, "cardRfids": ["%s"], "xsrfToken": "%s"}`, card.CardId, nextToken))))

	t.Logf("player %d making second pick", player)
	card = findCardToPick(t, ob, seat, 0, 1, true)
	handlers.ServeHTTP(httptest.NewRecorder(),
		httptest.NewRequest("POST", fmt.Sprintf("/api/pickrfid/?as=%d", player),
			strings.NewReader(fmt.Sprintf(`{"draftId": 1, "cardRfids": ["%s"], "xsrfToken": "%s"}`, card.CardId, token))))

	t.Logf("2 left player %d making first pick", prevPrevPlayer)
	card = findCardToPick(t, ob, prevPrevSeat, 0, 0, true)
	handlers.ServeHTTP(httptest.NewRecorder(),
		httptest.NewRequest("POST", fmt.Sprintf("/api/pickrfid/?as=%d", prevPrevPlayer),
			strings.NewReader(fmt.Sprintf(`{"draftId": 1, "cardRfids": ["%s"], "xsrfToken": "%s"}`, card.CardId, prevPrevToken))))

	t.Logf("left player %d making second pick", previousPlayer)
	card = findCardToPick(t, ob, previousSeat, 0, 1, true)
	handlers.ServeHTTP(httptest.NewRecorder(),
		httptest.NewRequest("POST", fmt.Sprintf("/api/pickrfid/?as=%d", previousPlayer),
			strings.NewReader(fmt.Sprintf(`{"draftId": 1, "cardRfids": ["%s"], "xsrfToken": "%s"}`, card.CardId, prevToken))))

	t.Logf("player %d making third pick", player)
	card = findCardToPick(t, ob, seat, 0, 2, true)
	w := httptest.NewRecorder()
	handlers.ServeHTTP(w,
		httptest.NewRequest("POST", fmt.Sprintf("/api/pickrfid/?as=%d", player),
			strings.NewReader(fmt.Sprintf(`{"draftId": 1, "cardRfids": ["%s"], "xsrfToken": "%s"}`, card.CardId, token))))

	res := w.Result()
	if res.StatusCode != http.StatusBadRequest {
		t.Errorf("expected pick to fail due to zone drafting violation, but status code was %d", res.StatusCode)
	}
}

func TestOnlineDraftNotifiesPlayerOfPicksAvailable(t *testing.T) {
	ob, err := doSetup(t, SEED)
	if err != nil {
		t.Errorf("error in setup: %s", err.Error())
		t.FailNow()
	}
	defer ob.Close()

	handlers := NewHandler(ob, false)

	makeDraft(t, handlers, SEED, false, false)

	players, seats := populateDraft(t, handlers, 8)

	player := players[0] + 1
	seat := seats[0]

	token := xsrftoken.Generate(xsrfKey, strconv.FormatInt(int64(player), 16), "pick1")

	// test player makes first pick
	cardId := findCardToPick(t, ob, seat, 0, 0, false).Id
	handlers.ServeHTTP(httptest.NewRecorder(),
		httptest.NewRequest("POST", fmt.Sprintf("/api/pick/?as=%d", player),
			strings.NewReader(fmt.Sprintf(`{"draftId": 1, "cards": [%d], "xsrfToken": "%s"}`, cardId, token))))

	if !slices.ContainsFunc(ignoredDiscordCalls, func(call DiscordCall) bool {
		return call.Type == "notify" && strings.Contains(call.Message, fmt.Sprintf("<@%d>", players[7]+1))
	}) {
		t.Error("didn't notify player of picks")
	}
}

func TestOnlineDraftDoesNotNotifyPlayerOfPicksWhenPassingPlayerHasMorePacks(t *testing.T) {
	ob, err := doSetup(t, SEED)
	if err != nil {
		t.Errorf("error in setup: %s", err.Error())
		t.FailNow()
	}
	defer ob.Close()

	handlers := NewHandler(ob, false)

	makeDraft(t, handlers, SEED, false, false)

	players, seats := populateDraft(t, handlers, 8)

	player := players[0] + 1
	seat := seats[0]
	nextSeat := seat + 1
	if nextSeat >= 8 {
		nextSeat -= 8
	}
	nextSeatIndex := slices.Index(seats, nextSeat)
	nextPlayer := players[nextSeatIndex] + 1
	followingSeat := seat + 2
	if followingSeat >= 8 {
		followingSeat -= 8
	}
	followingSeatIndex := slices.Index(seats, followingSeat)
	followingPlayer := players[followingSeatIndex] + 1

	token := xsrftoken.Generate(xsrfKey, strconv.FormatInt(int64(player), 16), "pick1")

	// test player makes first pick
	cardId := findCardToPick(t, ob, seat, 0, 0, false).Id
	handlers.ServeHTTP(httptest.NewRecorder(),
		httptest.NewRequest("POST", fmt.Sprintf("/api/pick/?as=%d", player),
			strings.NewReader(fmt.Sprintf(`{"draftId": 1, "cards": [%d], "xsrfToken": "%s"}`, cardId, token))))

	// next player first pick
	cardId = findCardToPick(t, ob, nextSeat, 0, 0, false).Id
	nextToken := xsrftoken.Generate(xsrfKey, strconv.FormatInt(int64(nextPlayer), 16), "pick1")
	handlers.ServeHTTP(httptest.NewRecorder(),
		httptest.NewRequest("POST", fmt.Sprintf("/api/pick/?as=%d", nextPlayer),
			strings.NewReader(fmt.Sprintf(`{"draftId": 1, "cards": [%d], "xsrfToken": "%s"}`, cardId, nextToken))))

	if slices.ContainsFunc(ignoredDiscordCalls, func(call DiscordCall) bool {
		return call.Type == "notify" && strings.Contains(call.Message, fmt.Sprintf("<@%d>", followingPlayer))
	}) {
		t.Error("notified player of picks prematurely")
	}
}

func TestOnlineDraftLocksSpectatorChannelOnJoin(t *testing.T) {
	ob, err := doSetup(t, SEED)
	if err != nil {
		t.Errorf("error in setup: %s", err.Error())
		t.FailNow()
	}
	defer ob.Close()

	handlers := NewHandler(ob, false)

	makeDraft(t, handlers, SEED, false, false)

	w := httptest.NewRecorder()
	handlers.ServeHTTP(w,
		httptest.NewRequest("POST", fmt.Sprintf("/api/join/?as=3"),
			strings.NewReader("{\"id\": 1}")))

	if !slices.ContainsFunc(ignoredDiscordCalls, func(call DiscordCall) bool {
		return call.Type == "lockChannel" && call.Message == "3" && call.ChannelId == "spectator-channel"
	}) {
		t.Error("didn't lock channel")
	}
}
