package main

import (
	"archive/zip"
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/sessions"
	_ "github.com/mattn/go-sqlite3"
)

type r38handler func(w http.ResponseWriter, r *http.Request, userId int64, tx *sql.Tx) error
type viewingFunc func(r *http.Request, userId int64) (bool, error)

var secretKeyNoOneWillEverGuess = []byte(os.Getenv("SESSION_SECRET"))
var store = sessions.NewCookieStore(secretKeyNoOneWillEverGuess)
var isViewing viewingFunc
var sock string

func main() {
	useAuthPtr := flag.Bool("auth", true, "bool")
	flag.Parse()

	useAuth := *useAuthPtr

	if useAuth {
		isViewing = AuthIsViewing
	} else {
		isViewing = NonAuthIsViewing
	}

	var err error

	database, err := sql.Open("sqlite3", "draft.db")
	if err != nil {
		return
	}
	err = database.Ping()
	if err != nil {
		return
	}

	port, valid := os.LookupEnv("R38_PORT")
	if !valid {
		port = "12264"
	}

	sock, valid = os.LookupEnv("R38_SOCK")
	if !valid {
		sock = "./r38.sock"
	}

	server := &http.Server{
		Addr:    fmt.Sprintf(":%s", port),
		Handler: NewHandler(database, useAuth),
	}

	log.Printf("Starting HTTP Server. Listening at %q", server.Addr)
	err = server.ListenAndServe() // this call blocks

	if err != nil {
		log.Printf("%s", err.Error())
	}
}

// NewHandler creates all server routes for serving the html.
func NewHandler(database *sql.DB, useAuth bool) http.Handler {
	mux := http.NewServeMux()

	middleware := AuthMiddleware
	if !useAuth {
		middleware = NonAuthMiddleware
	}

	addHandler := func(route string, serveFunc r38handler, readonly bool) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var userID int64
			if useAuth {
				if strings.HasPrefix(route, "/auth/") {
					userID = 0
				} else {
					session, err := store.Get(r, "session-name")
					if err != nil {
						http.Error(w, err.Error(), http.StatusInternalServerError)
						return
					}
					userIDInt, err := strconv.Atoi(session.Values["userid"].(string))
					if err != nil {
						http.Error(w, err.Error(), http.StatusInternalServerError)
						return
					}
					userID = int64(userIDInt)
				}
			} else {
				userID = 1
			}

			if userID == 1 {
				q := r.URL.Query()
				val := q.Get("as")
				if val != "" {
					userIDInt, err := strconv.Atoi(val)
					if err != nil {
						http.Error(w, err.Error(), http.StatusInternalServerError)
						return
					}
					userID = int64(userIDInt)
				}
			}

			ctx := r.Context()
			ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
			defer cancel()

			tx, err := database.BeginTx(ctx, &sql.TxOptions{ReadOnly: readonly})
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			err = serveFunc(w, r, userID, tx)
			if err != nil {
				tx.Rollback()
				if strings.HasPrefix(route, "/api/") {
					w.WriteHeader(http.StatusInternalServerError)
					json.NewEncoder(w).Encode(JSONError{Error: err.Error()})
				} else {
					http.Error(w, err.Error(), http.StatusInternalServerError)
				}
			} else {
				tx.Commit()
			}
		})
		mux.Handle(route, middleware(handler))
	}

	fs := http.FileServer(http.Dir("static"))
	mux.Handle("/static/", http.StripPrefix("/static/", fs))

	if useAuth {
		addHandler("/auth/discord/login", oauthDiscordLogin, true) // don't actually need db at all
		addHandler("/auth/discord/callback", oauthDiscordCallback, false)
	}

	addHandler("/replay/", ServeVueApp, true)
	addHandler("/home/", ServeVueApp, true)
	addHandler("/deckbuilder/", ServeVueApp, true)

	addHandler("/join/", ServeJoin, false)
	addHandler("/index/", ServeIndex, true)

	addHandler("/bulk_mtgo/", ServeBulkMTGO, true)

	addHandler("/api/draft/", ServeAPIDraft, true)
	addHandler("/api/draftlist/", ServeAPIDraftList, true)
	addHandler("/api/pick/", ServeAPIPick, false)
	addHandler("/api/join/", ServeAPIJoin, false)

	addHandler("/", ServeIndex, true)

	return mux
}

// AuthMiddleware makes sure users are logged in if auth is enabled.
func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session, err := store.Get(r, "session-name")
		if err != nil {
			t := template.Must(template.ParseFiles("login.tmpl"))
			t.Execute(w, nil)
			return
		}
		if session.Values["userid"] != nil {
			log.Printf("%s %s", session.Values["userid"], r.URL.Path)
			next.ServeHTTP(w, r)
			return
		}
		t := template.Must(template.ParseFiles("login.tmpl"))
		t.Execute(w, nil)
		return
	})
}

// NonAuthMiddleware just passes the request along in non-auth mode.
func NonAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf(r.URL.Path)
		next.ServeHTTP(w, r)
		return
	})
}

// AuthIsViewing determines if ?as= is being used in auth mode.
func AuthIsViewing(r *http.Request, userID int64) (bool, error) {
	session, err := store.Get(r, "session-name")
	if err != nil {
		return false, err
	}
	realUserIDInt, err := strconv.Atoi(session.Values["userid"].(string))
	if err != nil {
		return false, err
	}
	realUserID := int64(realUserIDInt)

	return userID != realUserID, nil
}

// NonAuthIsViewing determines if ?as= is being used in non-auth mode.
func NonAuthIsViewing(r *http.Request, userID int64) (bool, error) {
	return userID != 1, nil
}

// GetViewParam gets the view parameter that needs to be passed along if there is one.
func GetViewParam(r *http.Request, userID int64) string {
	param := ""
	viewing, err := isViewing(r, userID)
	if err == nil && viewing {
		param = fmt.Sprintf("?as=%d", userID)
	}
	return param
}

// ServeAPIDraft serves the /api/draft endpoint.
func ServeAPIDraft(w http.ResponseWriter, r *http.Request, userID int64, tx *sql.Tx) error {
	re := regexp.MustCompile(`/api/draft/(\d+)`)
	parseResult := re.FindStringSubmatch(r.URL.Path)
	if parseResult == nil {
		return fmt.Errorf("bad api url")
	}
	draftID, err := strconv.ParseInt(parseResult[1], 10, 64)
	if err != nil {
		return fmt.Errorf("bad api url: %s", err.Error())
	}

	draftJSON, err := GetFilteredJSON(tx, draftID, userID)
	if err != nil {
		return fmt.Errorf("error getting json: %s", err.Error())
	}

	fmt.Fprint(w, draftJSON)
	return nil
}

// ServeAPIDraftList serves the /api/draftlist endpoint.
func ServeAPIDraftList(w http.ResponseWriter, r *http.Request, userID int64, tx *sql.Tx) error {
	query := `select
                    drafts.id,
                    drafts.name,
                    sum(seats.user is null and seats.position is not null) as empty_seats,
                    coalesce(sum(seats.user = ?), 0) as joined
                  from drafts
                  left join seats on drafts.id = seats.draft
                  group by drafts.id`

	rows, err := tx.Query(query, userID)
	if err != nil {
		return fmt.Errorf("can't get draft list: %s", err.Error())
	}
	defer rows.Close()
	var drafts DraftList
	for rows.Next() {
		var d DraftListEntry
		var joined int64
		err = rows.Scan(&d.ID, &d.Name, &d.AvailableSeats, &joined)
		if err != nil {
			return fmt.Errorf("can't get draft list: %s", err.Error())
		}
		if joined == 1 {
			d.Status = "member"
		} else if d.AvailableSeats == 0 {
			d.Status = "spectator"
		} else {
			d.Status = "joinable"
		}

		drafts.Drafts = append(drafts.Drafts, d)
	}

	json.NewEncoder(w).Encode(drafts)
	return nil
}

// ServeAPIPick serves the /api/pick endpoint.
func ServeAPIPick(w http.ResponseWriter, r *http.Request, userID int64, tx *sql.Tx) error {
	if r.Method != "POST" {
		// we have to return an error manually here because we want to return
		// a different http status code.
		tx.Rollback()
		http.Error(w, "invalid request method", http.StatusMethodNotAllowed)
		return nil
	}

	bodyBytes, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return fmt.Errorf("error reading post body: %s", err.Error())
	}
	var pick PostedPick
	err = json.Unmarshal(bodyBytes, &pick)
	if err != nil {
		return fmt.Errorf("error parsing post body: %s", err.Error())
	}
	var draftID int64
	if len(pick.CardIds) == 1 {
		draftID, err = doSinglePick(tx, userID, pick.CardIds[0])
		if err != nil {
			// We can't send the actual error back to the client without leaking information about
			// where the card they tried to pick actually is.
			log.Printf("error making pick: %s", err.Error())
			return fmt.Errorf("error making pick")
		}
	} else if len(pick.CardIds) == 2 {
		return fmt.Errorf("cogwork librarian power not implemented yet")
	} else {
		return fmt.Errorf("invalid number of picked cards: %d", len(pick.CardIds))
	}

	draftJSON, err := GetFilteredJSON(tx, draftID, userID)
	if err != nil {
		return fmt.Errorf("error getting json: %s", err.Error())
	}

	fmt.Fprint(w, draftJSON)
	return nil
}

// ServeAPIJoin serves the /api/join endpoint.
func ServeAPIJoin(w http.ResponseWriter, r *http.Request, userID int64, tx *sql.Tx) error {
	if r.Method != "POST" {
		// we have to return an error manually here because we want to return
		// a different http status code.
		tx.Rollback()
		http.Error(w, "invalid request method", http.StatusMethodNotAllowed)
		return nil
	}

	bodyBytes, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return fmt.Errorf("error reading post body: %s", err.Error())
	}
	var toJoin PostedJoin
	err = json.Unmarshal(bodyBytes, &toJoin)
	if err != nil {
		return fmt.Errorf("error parsing post body: %s", err.Error())
	}

	draftID := toJoin.ID

	err = doJoin(tx, userID, draftID)
	if err != nil {
		return fmt.Errorf("error joining draft %d: %s", draftID, err.Error())
	}

	draftJSON, err := GetFilteredJSON(tx, draftID, userID)
	if err != nil {
		return fmt.Errorf("error getting json: %s", err.Error())
	}

	fmt.Fprint(w, draftJSON)
	return nil
}

// ServeBulkMTGO serves a .zip file of all .dek files for a draft. Only useful to the admin.
func ServeBulkMTGO(w http.ResponseWriter, r *http.Request, userID int64, tx *sql.Tx) error {
	if userID != 1 {
		return fmt.Errorf("auth error in bulk export")
	}
	re := regexp.MustCompile(`/bulk_mtgo/(\d+)`)
	parseResult := re.FindStringSubmatch(r.URL.Path)
	if parseResult == nil {
		return fmt.Errorf("draft not found")
	}
	draftID, err := strconv.ParseInt(parseResult[1], 10, 64)
	if err != nil {
		return err
	}
	query := `select
                    seats.user,
                    users.discord_name
                  from seats
                  join users on users.id = seats.user
                  where seats.draft = ?`
	rows, err := tx.Query(query, draftID)
	if err != nil {
		return err
	}
	defer rows.Close()

	re = regexp.MustCompile(`[/\\]`) // this could be more complete

	// Generate the MTGO export for each player.
	exports := []BulkMTGOExport{}
	for rows.Next() {
		var playerID int64
		var username string
		err := rows.Scan(&playerID, &username)
		if err != nil {
			log.Printf("error reading player in draft %d, skipping: %s", draftID, err)
			break
		}
		export, err := exportToMTGO(tx, playerID, draftID)
		if err != nil {
			log.Printf("could not export to MTGO for player %d in draft %d: %s", playerID, draftID, err)
			break
		}
		exports = append(exports, BulkMTGOExport{PlayerID: playerID, Username: re.ReplaceAllString(username, "_"), Deck: export})
	}

	// Generate the ZIP file for all exported decks.
	archive, err := createZipExport(exports)
	if err != nil {
		return fmt.Errorf("error creating zip file: %s", err.Error())
	}
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%d-r38-bulk.zip", draftID))
	w.Header().Set("Content-Type", r.Header.Get("Content-Type"))
	io.WriteString(w, string(archive))
	return nil
}

// createZipExport creates a .zip file containing decks from a bulk MTGO export.
func createZipExport(exports []BulkMTGOExport) ([]byte, error) {
	buf := new(bytes.Buffer)
	zipWriter := zip.NewWriter(buf)
	for _, export := range exports {
		zipFile, err := zipWriter.Create(fmt.Sprintf("%s.dek", export.Username))
		if err != nil {
			return nil, err
		}
		_, err = zipFile.Write([]byte(export.Deck))
		if err != nil {
			return nil, err
		}
	}
	err := zipWriter.Close()
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// exportToMTGO creates an MTGO compatible .dek string for given user and draft.
func exportToMTGO(tx *sql.Tx, userID int64, draftID int64) (string, error) {
	query := `select
                    cards.data
                  from cards
                  join packs on cards.pack = packs.id
                  join seats on packs.seat = seats.id
                  where seats.user = ?
                    and seats.draft = ?
                    and packs.round = 0`
	rows, err := tx.Query(query, userID, draftID)
	if err != nil {
		return "", err
	}
	defer rows.Close()

	export := "<?xml version=\"1.0\" encoding=\"utf-8\"?>\n<Deck xmlns:xsd=\"http://www.w3.org/2001/XMLSchema\" xmlns:xsi=\"http://www.w3.org/2001/XMLSchema-instance\">\n<NetDeckID>0</NetDeckID>\n<PreconstructedDeckID>0</PreconstructedDeckID>\n"
	var nq map[int64]NameAndQuantity
	nq = make(map[int64]NameAndQuantity)
	for rows.Next() {
		var dataString string
		var data R38CardData
		err = rows.Scan(&dataString)
		if err != nil {
			return "", err
		}
		err = json.Unmarshal([]byte(dataString), &data)
		if err != nil {
			return "", err
		}
		if data.MTGO != 0 {
			o := nq[data.MTGO]
			o.Name = data.Scryfall.Name
			o.Quantity++
			nq[data.MTGO] = o
		}
	}
	for mtgo, info := range nq {
		export = export + fmt.Sprintf("<Cards CatID=\"%d\" Quantity=\"%d\" Sideboard=\"false\" Name=\"%s\" />\n", mtgo, info.Quantity, info.Name)
	}
	export = export + "</Deck>"
	return export, nil
}

// ServeVueApp serves to vue.
func ServeVueApp(w http.ResponseWriter, r *http.Request, userID int64, tx *sql.Tx) error {
	query := `select
                    id,
                    discord_name,
                    picture
                  from users
                  where id = ?`
	row := tx.QueryRow(query, userID)
	var userInfo UserInfo
	err := row.Scan(&userInfo.ID, &userInfo.Name, &userInfo.Picture)
	if err != nil {
		return err
	}

	userInfoJSON, err := json.Marshal(userInfo)
	if err != nil {
		return err
	}

	data := VuePageData{UserJSON: string(userInfoJSON)}

	t := template.Must(template.ParseFiles("vue.tmpl"))

	t.Execute(w, data)
	return nil
}

// ServeIndex serves the index page.
func ServeIndex(w http.ResponseWriter, r *http.Request, userID int64, tx *sql.Tx) error {
	query := `select drafts.id, drafts.name, sum(seats.user is null and seats.position is not null) as empty_seats, coalesce(sum(seats.user = ?), 0) as joined from drafts left join seats on drafts.id = seats.draft group by drafts.id`

	rows, err := tx.Query(query, userID)
	if err != nil {
		return err
	}
	defer rows.Close()
	var Drafts []Draft
	for rows.Next() {
		var d Draft
		err = rows.Scan(&d.ID, &d.Name, &d.Seats, &d.Joined)
		if err != nil {
			return err
		}
		d.Joinable = d.Seats > 0 && !d.Joined
		d.Replayable = true

		Drafts = append(Drafts, d)
	}

	viewParam := GetViewParam(r, userID)
	data := IndexPageData{Drafts: Drafts, ViewURL: viewParam, UserID: userID}
	t := template.Must(template.ParseFiles("index.tmpl"))
	t.Execute(w, data)
	return nil
}

// ServeJoin allows the user to join a draft, if possible.
func ServeJoin(w http.ResponseWriter, r *http.Request, userID int64, tx *sql.Tx) error {
	re := regexp.MustCompile(`/join/(\d+)`)
	parseResult := re.FindStringSubmatch(r.URL.Path)

	if parseResult == nil {
		return fmt.Errorf("bad url")
	}

	draftIDInt, err := strconv.Atoi(parseResult[1])
	if err != nil {
		return err
	}
	draftID := int64(draftIDInt)

	err = doJoin(tx, userID, draftID)
	if err != nil {
		return err
	}

	viewParam := GetViewParam(r, userID)
	http.Redirect(w, r, fmt.Sprintf("/replay/%d%s", draftID, viewParam), http.StatusTemporaryRedirect)
	return nil
}

// doJoin does the actual joining.
func doJoin(tx *sql.Tx, userID int64, draftID int64) error {
	query := `select
                    count(1)
                  from seats
                  where draft = ?
                    and user = ?`
	row := tx.QueryRow(query, draftID, userID)
	var alreadyJoined int64
	err := row.Scan(&alreadyJoined)
	if err != nil {
		return err
	} else if alreadyJoined > 0 {
		return fmt.Errorf("user %d already joined %d", userID, draftID)
	}

	query = `select
                   id
                 from seats
                 where draft = ?
                   and user is null
                 order by random()
                 limit 1`
	row = tx.QueryRow(query, draftID)
	var emptySeatID int64
	err = row.Scan(&emptySeatID)
	if err != nil {
		return err
	}

	query = `update seats set user = ? where id = ?`
	_, err = tx.Exec(query, userID, emptySeatID)
	if err != nil {
		return err
	}

	return nil
}

// doSinglePick performs a normal pick based on a user id and a card id. It returns the draft id and an error.
func doSinglePick(tx *sql.Tx, userID int64, cardID int64) (int64, error) {
	draftID, _, announcements, round, err := doPick(tx, userID, cardID, true)
	if err != nil {
		return draftID, err
	}
	err = doEvent(tx, draftID, userID, announcements, cardID, sql.NullInt64{}, round)
	if err != nil {
		return draftID, err
	}
	return draftID, nil
}

// doPick actually performs a pick in the database.
// It returns the draftID, packID, announcements, round, and an error.
// Of those return values, packID and announcements are only really relevant for Cogwork Librarian,
// which is not currently fully implemented, but we leave them here anyway for when we want to do that.
func doPick(tx *sql.Tx, userID int64, cardID int64, pass bool) (int64, int64, []string, int64, error) {
	announcements := []string{}

	// First we need information about the card. Determine which pack the card is in,
	// where that pack is at the table, who sits at that position, which draft that
	// pack is a part of, and which round that card is in.
	query := `select
                   packs.id,
                   seats.position,
                   seats.draft,
                   seats.user,
                   seats.round
                 from cards
                 join packs on cards.pack = packs.id
                 join seats on packs.seat = seats.id
                 where cards.id = ?`

	row := tx.QueryRow(query, cardID)
	var myPackID int64
	var position int64
	var draftID int64
	var userID2 int64
	var round int64
	err := row.Scan(&myPackID, &position, &draftID, &userID2, &round)

	if err != nil {
		return draftID, myPackID, announcements, round, err
	} else if userID != userID2 {
		return draftID, myPackID, announcements, round, fmt.Errorf("card does not belong to the user.")
	} else if round == 0 {
		return draftID, myPackID, announcements, round, fmt.Errorf("card has already been picked.")
	}

	// Now get the pack id that the user is allowed to pick from in the draft that the
	// card is from. Note that there might be no such pack.
	query = `select
                    v_packs.id
                  from seats
                  join v_packs on seats.id = v_packs.seat
                  where seats.user = ?
                    and seats.draft = ?
                    and seats.round = v_packs.round
                  order by v_packs.count desc
                  limit 1`

	row = tx.QueryRow(query, userID, draftID)
	var myPackID2 int64
	err = row.Scan(&myPackID2)

	if err != nil {
		return draftID, myPackID, announcements, round, err
	} else if myPackID != myPackID2 {
		return draftID, myPackID, announcements, round, fmt.Errorf("card is not in the next available pack.")
	}

	// once we're here, we know the pick is valid

	// Determine which pack we're putting the drafted card into.
	query = `select
                   v_packs.id,
                   v_packs.count
                 from v_packs
                 join seats on seats.id = v_packs.seat
                 where v_packs.round = 0
                   and seats.user = ?
                   and seats.draft = ?`

	row = tx.QueryRow(query, userID, draftID)
	var myPicksID int64
	var myCount int64
	err = row.Scan(&myPicksID, &myCount)

	if err != nil {
		return draftID, myPackID, announcements, round, err
	}

	// Are we passing the pack after we've picked the card?
	if pass {
		// Get the seat position that the pack will be passed to.
		var newPosition int64
		if round%2 == 0 {
			newPosition = position - 1
			if newPosition == -1 {
				newPosition = 7
			}
		} else {
			newPosition = position + 1
			if newPosition == 8 {
				newPosition = 0
			}
		}

		// Now get the seat id that the pack will be passed to.
		query = `select
                           seats.id,
                           users.discord_id
                         from seats
                         left join users on seats.user = users.id
                         where seats.draft = ?
                           and seats.position = ?`

		row = tx.QueryRow(query, draftID, newPosition)
		var newPositionID int64
		var newPositionDiscordID sql.NullString
		err = row.Scan(&newPositionID, &newPositionDiscordID)
		if err != nil {
			return draftID, myPackID, announcements, round, err
		}

		// Put the picked card into the player's picks.
		query = `update cards set pack = ? where id = ?`

		_, err = tx.Exec(query, myPicksID, cardID)
		if err != nil {
			return draftID, myPackID, announcements, round, err
		}

		// Move the pack to the next seat.
		query = `update packs set seat = ? where id = ?`
		_, err = tx.Exec(query, newPositionID, myPackID)
		if err != nil {
			return draftID, myPackID, announcements, round, err
		}

		// Get the number of remaining packs in the seat.
		query = `select
                           count(1)
                         from v_packs
                         join seats on v_packs.seat = seats.id
                         where seats.user = ?
                           and v_packs.round = ?
                           and v_packs.count > 0
                           and seats.draft = ?`
		row = tx.QueryRow(query, userID, round, draftID)
		var packsLeftInSeat int64
		err = row.Scan(&packsLeftInSeat)
		if err != nil {
			return draftID, myPackID, announcements, round, err
		}

		if packsLeftInSeat == 0 {
			// If there are 0 packs left in the seat, check to see if the player we passed the pack to
			// is in the same round as us. If the rounds match, NotifyByDraftAndPosition.
			query = `select
                                   count(1)
                                 from seats a
                                 join seats b on a.draft = b.draft
                                 where a.user = ?
                                   and b.position = ?
                                   and a.draft = ?
                                   and a.round = b.round`
			row = tx.QueryRow(query, userID, newPosition, draftID)
			var roundsMatch int64
			err = row.Scan(&roundsMatch)
			if err != nil {
				log.Printf("cannot determine if rounds match for notify")
			} else if roundsMatch == 1 && newPositionDiscordID.Valid {
				log.Printf("attempting to notify position %d draft %d", newPosition, draftID)
				err = NotifyByDraftAndDiscordID(draftID, newPositionDiscordID.String)
				if err != nil {
					log.Printf("error with notify")
				}
			}

			// Now that we've passed the pack, check to see if we should advance to the next round.
			// Update our round.

			// WARNING: if you ever have a draft with anything other than 15 cards per pack, or you have
			// something like Lore Seeker in your draft, this is going to break horribly.
			// If we're only doing normal drafts, round is effectively something that can be calculated,
			// but by explicitly storing it, we allow ourselves the possibility of expanding support to
			// weirder formats.
			query = `update seats set round = ? where user = ? and draft = ?`

			_, err = tx.Exec(query, (myCount+1)/15+1, userID, draftID)
			if err != nil {
				return draftID, myPackID, announcements, round, err
			}

			// If the rounds do NOT match from earlier, we have a situation where players are in different
			// rounds. Look for a blocking player.
			if roundsMatch == 0 {
				// We now know that we've passed a pack to someone in a different round.
				// We know that player is necessarily in a round earlier than ours because
				// we couldn't pass them a pack from a round they're already finished with.
				// That means we did not send a notification, because that player can't yet
				// pick from that pack.
				// That means there is a chance someone else is blocking the draft and needs
				// a friendly reminder to make their picks.
				// Before we find the blocking player, we need to make sure we're not the
				// only ones in this round.
				// If we are the only ones in this round, we very likely just passed the
				// blocking player their last pick of their round, so they are very likely
				// the most recent ping. We don't want to double ping.
				query = `select
                                           count(1)
                                         from seats
                                         where draft = ?
                                         group by round
                                         order by round desc
                                         limit 1`

				row = tx.QueryRow(query, draftID)
				var nextRoundPlayers int64
				err = row.Scan(&nextRoundPlayers)
				if err != nil {
					log.Printf("error counting players and rounds")
				} else if nextRoundPlayers > 1 {
					// Now we know that we are not the only player in this round.
					// Get the position of all players that currently have a pick.
					query = `select
                                                   seats.position,
                                                   users.discord_id
                                                 from seats
                                                 left join v_packs on seats.id = v_packs.seat
                                                 join users on seats.user = users.id
                                                 where v_packs.count > 0
                                                   and v_packs.round = seats.round
                                                   and seats.draft = ?
                                                 group by seats.id`

					rows, err := tx.Query(query, draftID)
					if err != nil {
						log.Printf("error determining if there's a blocking player")
					} else {
						defer rows.Close()

						rowCount := 0
						var blockingPosition int64
						var blockingDiscordID sql.NullString
						for rows.Next() {
							rowCount++
							err = rows.Scan(&blockingPosition, &blockingDiscordID)
							if err != nil {
								log.Printf("some kind of error with scanning: %s", err.Error())
								rowCount = 2
								break
							}
						}
						if rowCount == 1 && blockingDiscordID.Valid {
							err = NotifyByDraftAndDiscordID(draftID, blockingDiscordID.String)
							if err != nil {
								log.Printf("error with blocking notify")
							}
						}
					}
				}
			}
		}
	} else {
		// we're in some sort of non-working cogwork librarian situation
		// just take the card from the pack.
		query = `update cards set pack = ? where id = ?`

		_, err = tx.Exec(query, myPicksID, cardID)
		if err != nil {
			return draftID, myPackID, announcements, round, err
		}
	}

	log.Printf("player %d in draft %d took card %d", userID, draftID, cardID)

	return draftID, myPackID, announcements, round, nil
}

// NotifyByDraftAndDiscordID sends a discord alert to a user.
func NotifyByDraftAndDiscordID(draftID int64, discordID string) error {
	var jsonStr = []byte(fmt.Sprintf(`{"content": "<@%s> you have new picks <http://draft.thefoley.net/draft/%d>"}`, discordID, draftID))
	req, err := http.NewRequest("POST", os.Getenv("DISCORD_WEBHOOK_URL"), bytes.NewBuffer(jsonStr))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("Error sending msg. Status: %v", resp.Status)
	}

	return nil
}

// GetJSONObject returns a better DraftJSON object. May be filtered.
func GetJSONObject(tx *sql.Tx, draftID int64) (DraftJSON, error) {
	var draft DraftJSON

	query := `select
                    drafts.id,
                    drafts.name,
                    seats.position,
                    packs.round,
                    users.discord_name,
                    cards.id,
                    users.id,
                    cards.data,
                    users.picture
                  from seats
                  left join users on users.id = seats.user
                  join drafts on drafts.id = seats.draft
                  join packs on packs.original_seat = seats.id
                  join cards on cards.original_pack = packs.id
                  where drafts.id = ?`

	rows, err := tx.Query(query, draftID)
	if err != nil {
		return draft, err
	}
	defer rows.Close()
	var indices [8][3]int64
	for rows.Next() {
		var position int64
		var packRound int64
		var cardID int64
		var nullableDiscordID sql.NullString
		var draftUserID sql.NullInt64
		var cardData string
		var nullablePicture sql.NullString
		err = rows.Scan(&draft.DraftID, &draft.DraftName, &position, &packRound, &nullableDiscordID, &cardID, &draftUserID, &cardData, &nullablePicture)
		if err != nil {
			return draft, err
		}

		dataObj := make(map[string]interface{})
		err = json.Unmarshal([]byte(cardData), &dataObj)
		if err != nil {
			log.Printf("making nil card data because of error %s", err.Error())
			dataObj = nil
		}
		dataObj["id"] = cardID

		packRound--

		nextIndex := indices[position][packRound]

		draft.Seats[position].Packs[packRound][nextIndex] = dataObj
		draft.Seats[position].PlayerName = nullableDiscordID.String
		draft.Seats[position].PlayerID = draftUserID.Int64
		draft.Seats[position].PlayerImage = nullablePicture.String

		indices[position][packRound]++
	}

	query = `select
                   position,
                   announcement,
                   card1,
                   card2,
                   id,
                   modified,
                   round
                 from events
                 where draft = ?`
	rows, err = tx.Query(query, draftID)
	if err != nil && err != sql.ErrNoRows {
		return draft, err
	}
	defer rows.Close()
	for rows.Next() {
		var event DraftEvent
		var announcements string
		var card1id int64
		var card2id sql.NullInt64
		err = rows.Scan(&event.Position, &announcements, &card1id, &card2id, &event.DraftModified, &event.PlayerModified, &event.Round)
		if err != nil {
			return draft, err
		}
		event.Cards = append(event.Cards, card1id)
		if card2id.Valid {
			event.Cards = append(event.Cards, card2id.Int64)
			event.Librarian = true
		}
		if announcements != "" {
			event.Announcements = strings.Split(announcements, "\n")
		} else {
			event.Announcements = []string{}
		}
		event.Type = "Pick"
		draft.Events = append(draft.Events, event)
	}

	if len(draft.Events) == 0 {
		draft.Events = []DraftEvent{}
	}

	return draft, nil
}

// GetFilteredJSON returns a filtered json object of replay data.
func GetFilteredJSON(tx *sql.Tx, draftID int64, userID int64) (string, error) {
	draft, err := GetJSONObject(tx, draftID)
	if err != nil {
		return "", err
	}

	query := `select (
                    select
                      round
                    from seats
                    where draft = ?
                      and user = ?), (
                    select
                      count(1)
                    from seats
                    where draft = ?
                      and user is null)`
	var myRound sql.NullInt64
	var emptySeats int64
	row := tx.QueryRow(query, draftID, userID, draftID)
	err = row.Scan(&myRound, &emptySeats)
	if err != nil {
		return "", err
	} else if (myRound.Valid && myRound.Int64 >= 4) || (!myRound.Valid && emptySeats == 0) {
		// either we're not in the draft, or the draft is over for us
		// therefore, we can see the whole draft.
		ret, err := json.Marshal(draft)
		if err != nil {
			return "", err
		}
		return string(ret), nil
	}

	// this is an ongoing draft that we're a member of. filter the json.
	conn, err := net.Dial("unix", sock)
	if err != nil {
		return "", err
	}
	defer conn.Close()

	ret, err := json.Marshal(Perspective{User: userID, Draft: draft})
	if err != nil {
		return "", err
	}

	stop := "\r\n\r\n"

	conn.Write([]byte(ret))
	conn.Write([]byte(stop))

	var buff bytes.Buffer
	io.Copy(&buff, conn)

	return buff.String(), nil
}

// doEvent records an event (pick) into the database.
func doEvent(tx *sql.Tx, draftID int64, userID int64, announcements []string, cardID1 int64, cardID2 sql.NullInt64, round int64) error {
	query := `select
                    v_packs.count,
                    seats.position
                  from v_packs
                  join seats on v_packs.seat = seats.id
                  where v_packs.round = 0
                    and seats.draft = ?
                    and seats.user = ?`
	row := tx.QueryRow(query, draftID, userID)
	var count int64
	var position int64
	err := row.Scan(&count, &position)
	if err != nil {
		return err
	}

	if cardID2.Valid {
		query = `insert into events (round, draft, position, announcement, card1, card2, modified) VALUES (?, ?, ?, ?, ?, ?, ?)`
		_, err = tx.Exec(query, round, draftID, position, strings.Join(announcements, "\n"), cardID1, cardID2.Int64, count)
	} else {
		query = `insert into events (round, draft, position, announcement, card1, card2, modified) VALUES (?, ?, ?, ?, ?, null, ?)`
		_, err = tx.Exec(query, round, draftID, position, strings.Join(announcements, "\n"), cardID1, count)
	}

	return err
}
