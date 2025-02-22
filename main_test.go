package main

import (
	"database/sql"
	"fmt"
	"github.com/BurntSushi/migration"
	"github.com/walkingeyerobot/r38/makedraft"
	"github.com/walkingeyerobot/r38/migrations"
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
)

func ptr[T any](v T) *T {
	return &v
}

var SEED = 677483

func doSetup(t *testing.T, seed int) (*sql.DB, error) {
	xsrfKey = "test"
	sock = ""

	db, err := migration.Open("sqlite3", "file::memory:?cache=shared", migrations.Migrations)
	if err != nil {
		t.Error(err)
	}

	var testDataQuery []byte
	testDataQuery, err = os.ReadFile("testdata/InitQuery.sql")
	if err != nil {
		t.Error(err)
	}

	_, err = db.Exec(string(testDataQuery))
	if err != nil {
		t.Error(err)
	}

	rand.Seed(int64(seed))
	return db, err
}

func makeDraft(t *testing.T, err error, db *sql.DB, seed int) {
	var tx *sql.Tx

	tx, err = db.Begin()
	if err != nil {
		t.Error(err)
	}
	err = makedraft.MakeDraft(makedraft.Settings{
		Set:                              ptr("sets/cube.json"),
		Database:                         ptr(""),
		Seed:                             &seed,
		InPerson:                         ptr(true),
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
	}, tx)
	if err != nil {
		t.Error(err)
	}
	err = tx.Commit()
	if err != nil {
		t.Error(err)
	}
}

func populateDraft(t *testing.T, handlers http.Handler) (players []int, seats []int) {
	players = rand.Perm(12)
	seats = rand.Perm(8)
	for seat := range 8 {
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

func findCardToPick(t *testing.T, db *sql.DB, player int, round int, card int) string {
	var rows *sql.Rows
	var err error
	if card == 0 {
		query := `select cards.cardId from cards 
								join packs on packs.id = cards.pack
								join seats on seats.id = packs.seat
								where seats.position = 8`
		rows, err = db.Query(query)
	} else {
		query := `select cards.cardId from cards
								join v_packs on v_packs.id = cards.pack
								join seats on seats.id = v_packs.seat
								where seats.user = ?
								and v_packs.round = ?
								and v_packs.count = ?`
		rows, err = db.Query(query, player, round+1, 15-card)
	}
	if err != nil {
		t.Errorf("error finding card to pick for player %d, pack %d, pick %d: %s",
			player, round+1, card+1, err.Error())
	}

	var cardId string
	var cards []string
	for rows.Next() {
		err = rows.Scan(&cardId)
		if err != nil {
			t.Errorf("error finding card to pick for player %d, pack %d, pick %d: %s",
				player, round+1, card+1, err.Error())
		}
		cards = append(cards, cardId)
	}
	if len(cards) == 0 {
		t.Errorf("no cards in pack for player %d, pack %d, pick %d", player, round+1, card+1)
	}
	cardId = cards[rand.Intn(len(cards))]
	return cardId
}

func FuzzInPersonDraft(f *testing.F) {
	f.Add(SEED)
	f.Fuzz(func(t *testing.T, seed int) {
		db, err := doSetup(t, seed)
		defer db.Close()

		makeDraft(t, err, db, seed)

		handlers := NewHandler(db, false)

		players, seats := populateDraft(t, handlers)

		for round := range 3 {
			for card := range 15 {
				for _, seat := range rand.Perm(8) {
					player := players[seat] + 1

					cardId := findCardToPick(t, db, player, round, card)

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

func TestInPersonDraftUndoFirstPick(t *testing.T) {
	db, err := doSetup(t, SEED)
	defer db.Close()

	makeDraft(t, err, db, SEED)

	handlers := NewHandler(db, false)

	players, seats := populateDraft(t, handlers)

	player := players[0] + 1
	cardRfid := findCardToPick(t, db, player, 0, 0)
	row := db.QueryRow(`select id, pack from cards where cardid = ?`, cardRfid)
	var cardId int64
	var origPackID int64
	err = row.Scan(&cardId, &origPackID)
	if err != nil {
		t.Errorf("couldn't find picked card ID: %s", err.Error())
	}
	row = db.QueryRow(`select id from seats where position = ?`, seats[0])
	var origSeatID int64
	err = row.Scan(&origSeatID)
	if err != nil {
		t.Errorf("couldn't find pack seat ID: %s", err.Error())
	}

	token := xsrftoken.Generate(xsrfKey, strconv.FormatInt(int64(player), 16), "pick1")

	handlers.ServeHTTP(httptest.NewRecorder(),
		httptest.NewRequest("POST", fmt.Sprintf("/api/pickrfid/?as=%d", player),
			strings.NewReader(fmt.Sprintf(`{"draftId": 1, "cardRfids": ["%s"], "xsrfToken": "%s"}`, cardRfid, token))))

	w := httptest.NewRecorder()
	handlers.ServeHTTP(w,
		httptest.NewRequest("POST", fmt.Sprintf("/api/undopick/?as=%d", player),
			strings.NewReader(fmt.Sprintf(`{"draftId": 1, "xsrfToken": "%s"}`, token))))
	res := w.Result()
	if res.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(res.Body)
		t.Errorf("undo failed: %s", body)
	}

	row = db.QueryRow(`select count(*) from events where draft = 1 and position = ? and card1 = ?`,
		seats[0], cardId)
	var eventsCount int64
	err = row.Scan(&eventsCount)
	if err != nil {
		t.Errorf("couldn't execute events count query: %s", err.Error())
	}
	if eventsCount != 0 {
		t.Error("didn't delete event")
	}

	row = db.QueryRow(`select pack from cards where id = ?`, cardId)
	var packID int64
	err = row.Scan(&packID)
	if err != nil {
		t.Errorf("couldn't execute pack query: %s", err.Error())
	}
	if packID != origPackID {
		t.Errorf("didn't reset card's pack (was %d, expected %d)", packID, origPackID)
	}

	row = db.QueryRow(`select seat from packs where id = ?`, origPackID)
	var seatID int64
	err = row.Scan(&seatID)
	if err != nil {
		t.Errorf("couldn't execute seat query: %s", err.Error())
	}
	if seatID != origSeatID {
		t.Errorf("didn't reset pack's seat (was %d, expected %d)", seatID, origSeatID)
	}
}

func TestInPersonDraftUndoSubsequentPick(t *testing.T) {
	db, err := doSetup(t, SEED)
	defer db.Close()

	makeDraft(t, err, db, SEED)

	handlers := NewHandler(db, false)

	players, seats := populateDraft(t, handlers)

	player := players[0] + 1

	for card := range 5 {
		for _, seat := range rand.Perm(8) {
			player := players[seat] + 1

			cardId := findCardToPick(t, db, player, 0, card)

			token := xsrftoken.Generate(xsrfKey, strconv.FormatInt(int64(player), 16), "pick1")

			handlers.ServeHTTP(httptest.NewRecorder(),
				httptest.NewRequest("POST", fmt.Sprintf("/api/pickrfid/?as=%d", player),
					strings.NewReader(fmt.Sprintf(`{"draftId": 1, "cardRfids": ["%s"], "xsrfToken": "%s"}`, cardId, token))))
		}
	}

	cardRfid := findCardToPick(t, db, player, 0, 5)
	row := db.QueryRow(`select id, pack from cards where cardid = ?`, cardRfid)
	var cardId int64
	var origPackID int64
	err = row.Scan(&cardId, &origPackID)
	if err != nil {
		t.Errorf("couldn't find picked card ID: %s", err.Error())
	}
	row = db.QueryRow(`select id from seats where position = ?`, seats[0])
	var origSeatID int64
	err = row.Scan(&origSeatID)
	if err != nil {
		t.Errorf("couldn't find pack seat ID: %s", err.Error())
	}

	token := xsrftoken.Generate(xsrfKey, strconv.FormatInt(int64(player), 16), "pick1")

	handlers.ServeHTTP(httptest.NewRecorder(),
		httptest.NewRequest("POST", fmt.Sprintf("/api/pickrfid/?as=%d", player),
			strings.NewReader(fmt.Sprintf(`{"draftId": 1, "cardRfids": ["%s"], "xsrfToken": "%s"}`, cardRfid, token))))

	w := httptest.NewRecorder()
	handlers.ServeHTTP(w,
		httptest.NewRequest("POST", fmt.Sprintf("/api/undopick/?as=%d", player),
			strings.NewReader(fmt.Sprintf(`{"draftId": 1, "xsrfToken": "%s"}`, token))))
	res := w.Result()
	if res.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(res.Body)
		t.Errorf("undo failed: %s", body)
	}

	row = db.QueryRow(`select count(*) from events where draft = 1 and position = ? and card1 = ?`,
		seats[0], cardId)
	var eventsCount int64
	err = row.Scan(&eventsCount)
	if err != nil {
		t.Errorf("couldn't execute events count query: %s", err.Error())
	}
	if eventsCount != 0 {
		t.Error("didn't delete event")
	}

	row = db.QueryRow(`select pack from cards where id = ?`, cardId)
	var packID int64
	err = row.Scan(&packID)
	if err != nil {
		t.Errorf("couldn't execute pack query: %s", err.Error())
	}
	if packID != origPackID {
		t.Errorf("didn't reset card's pack (was %d, expected %d)", packID, origPackID)
	}

	row = db.QueryRow(`select seat from packs where id = ?`, origPackID)
	var seatID int64
	err = row.Scan(&seatID)
	if err != nil {
		t.Errorf("couldn't execute seat query: %s", err.Error())
	}
	if seatID != origSeatID {
		t.Errorf("didn't reset pack's seat (was %d, expected %d)", seatID, origSeatID)
	}
}

func TestInPersonDraftEnforceZoneDraftingNextPlayerMakingFirstPick(t *testing.T) {
	db, err := doSetup(t, SEED)
	defer db.Close()

	makeDraft(t, err, db, SEED)

	handlers := NewHandler(db, false)

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
	cardId := findCardToPick(t, db, player, 0, 0)
	handlers.ServeHTTP(httptest.NewRecorder(),
		httptest.NewRequest("POST", fmt.Sprintf("/api/pickrfid/?as=%d", player),
			strings.NewReader(fmt.Sprintf(`{"draftId": 1, "cardRfids": ["%s"], "xsrfToken": "%s"}`, cardId, token))))

	// player before test player makes first pick
	cardId = findCardToPick(t, db, previousPlayer, 0, 0)
	prevToken := xsrftoken.Generate(xsrfKey, strconv.FormatInt(int64(previousPlayer), 16), "pick1")
	handlers.ServeHTTP(httptest.NewRecorder(),
		httptest.NewRequest("POST", fmt.Sprintf("/api/pickrfid/?as=%d", previousPlayer),
			strings.NewReader(fmt.Sprintf(`{"draftId": 1, "cardRfids": ["%s"], "xsrfToken": "%s"}`, cardId, prevToken))))

	// test player makes second pick in violation
	cardId = findCardToPick(t, db, player, 0, 1)
	w := httptest.NewRecorder()
	handlers.ServeHTTP(w,
		httptest.NewRequest("POST", fmt.Sprintf("/api/pickrfid/?as=%d", player),
			strings.NewReader(fmt.Sprintf(`{"draftId": 1, "cardRfids": ["%s"], "xsrfToken": "%s"}`, cardId, token))))

	res := w.Result()
	if res.StatusCode == http.StatusOK {
		t.Error("expected pick to fail due to zone drafting violation, but pick succeeded")
	}
}

func TestInPersonDraftEnforceZoneDraftingNextPlayerMakingSubsequentPick(t *testing.T) {
	db, err := doSetup(t, SEED)
	defer db.Close()

	makeDraft(t, err, db, SEED)

	handlers := NewHandler(db, false)

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
	cardId := findCardToPick(t, db, player, 0, 0)
	handlers.ServeHTTP(httptest.NewRecorder(),
		httptest.NewRequest("POST", fmt.Sprintf("/api/pickrfid/?as=%d", player),
			strings.NewReader(fmt.Sprintf(`{"draftId": 1, "cardRfids": ["%s"], "xsrfToken": "%s"}`, cardId, token))))

	t.Logf("left player %d making first pick", previousPlayer)
	cardId = findCardToPick(t, db, previousPlayer, 0, 0)
	handlers.ServeHTTP(httptest.NewRecorder(),
		httptest.NewRequest("POST", fmt.Sprintf("/api/pickrfid/?as=%d", previousPlayer),
			strings.NewReader(fmt.Sprintf(`{"draftId": 1, "cardRfids": ["%s"], "xsrfToken": "%s"}`, cardId, prevToken))))

	t.Logf("right player %d making first pick", nextPlayer)
	cardId = findCardToPick(t, db, nextPlayer, 0, 0)
	handlers.ServeHTTP(httptest.NewRecorder(),
		httptest.NewRequest("POST", fmt.Sprintf("/api/pickrfid/?as=%d", nextPlayer),
			strings.NewReader(fmt.Sprintf(`{"draftId": 1, "cardRfids": ["%s"], "xsrfToken": "%s"}`, cardId, nextToken))))

	t.Logf("player %d making second pick", player)
	cardId = findCardToPick(t, db, player, 0, 1)
	handlers.ServeHTTP(httptest.NewRecorder(),
		httptest.NewRequest("POST", fmt.Sprintf("/api/pickrfid/?as=%d", player),
			strings.NewReader(fmt.Sprintf(`{"draftId": 1, "cardRfids": ["%s"], "xsrfToken": "%s"}`, cardId, token))))

	t.Logf("2 left player %d making first pick", prevPrevPlayer)
	cardId = findCardToPick(t, db, prevPrevPlayer, 0, 0)
	handlers.ServeHTTP(httptest.NewRecorder(),
		httptest.NewRequest("POST", fmt.Sprintf("/api/pickrfid/?as=%d", prevPrevPlayer),
			strings.NewReader(fmt.Sprintf(`{"draftId": 1, "cardRfids": ["%s"], "xsrfToken": "%s"}`, cardId, prevPrevToken))))

	t.Logf("left player %d making second pick", previousPlayer)
	cardId = findCardToPick(t, db, previousPlayer, 0, 1)
	handlers.ServeHTTP(httptest.NewRecorder(),
		httptest.NewRequest("POST", fmt.Sprintf("/api/pickrfid/?as=%d", previousPlayer),
			strings.NewReader(fmt.Sprintf(`{"draftId": 1, "cardRfids": ["%s"], "xsrfToken": "%s"}`, cardId, prevToken))))

	t.Logf("player %d making third pick", player)
	cardId = findCardToPick(t, db, player, 0, 2)
	w := httptest.NewRecorder()
	handlers.ServeHTTP(w,
		httptest.NewRequest("POST", fmt.Sprintf("/api/pickrfid/?as=%d", player),
			strings.NewReader(fmt.Sprintf(`{"draftId": 1, "cardRfids": ["%s"], "xsrfToken": "%s"}`, cardId, token))))

	res := w.Result()
	if res.StatusCode == http.StatusOK {
		t.Error("expected pick to fail due to zone drafting violation, but pick succeeded")
	}
}
