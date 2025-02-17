package main

import (
	"database/sql"
	"fmt"
	"github.com/BurntSushi/migration"
	"github.com/walkingeyerobot/r38/makedraft"
	"github.com/walkingeyerobot/r38/migrations"
	"golang.org/x/net/xsrftoken"
	"golang.org/x/sys/unix"
	"io"
	"math/rand"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"
)

func ptr[T any](v T) *T {
	return &v
}

var SEED = 677483

func TestInPersonDraft(t *testing.T) {
	xsrfKey = "test"
	sock = "test.sock"

	unix.Unlink(sock)
	filterListener, err := net.Listen("unix", sock)
	defer func() { unix.Unlink(sock) }()
	if err != nil {
		t.Error(err)
	}
	defer filterListener.Close()
	go func() {
		for {
			conn, filterErr := filterListener.Accept()
			if filterErr != nil {
				break
			}
			filterErr = conn.Close()
			if filterErr != nil {
				break
			}
		}
	}()

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

	var tx *sql.Tx

	rand.Seed(int64(SEED))

	tx, err = db.Begin()
	if err != nil {
		t.Error(err)
	}
	err = makedraft.MakeDraft(makedraft.Settings{
		Set:                              ptr("sets/cube.json"),
		Database:                         ptr(""),
		Seed:                             &SEED,
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

	handlers := NewHandler(db, false)

	players := rand.Perm(12)
	seats := rand.Perm(8)
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

	for round := range 3 {
		for card := range 15 {
			for _, seat := range rand.Perm(8) {
				player := players[seat] + 1

				var rows *sql.Rows
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
}
