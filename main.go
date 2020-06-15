package main

import (
	"archive/zip"
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
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

	"github.com/gorilla/sessions"
	"github.com/jung-kurt/gofpdf"
	_ "github.com/mattn/go-sqlite3"
)

// Draft describes a draft for the purposes of the index page.
type Draft struct {
	Name       string
	ID         int64
	Seats      int64
	Joined     bool
	Joinable   bool
	Replayable bool
}

// IndexPageData is the input to index.tmpl.
type IndexPageData struct {
	Drafts  []Draft
	ViewURL string
	UserID  int64
}

// DraftPageData is the input to draft.tmpl.
type DraftPageData struct {
	DraftID   int64
	DraftName string
	Picks     []Card
	Pack      []Card
	Powers    []Card
	Position  int64
	Revealed  []string
	ViewURL   string
}

// Card is exported to the client to describe cards.
type Card struct {
	ID      int64       `json:"r38Id"`
	Name    string      `json:"name"`
	Tags    string      `json:"tags"`
	Number  string      `json:"number"`
	Edition string      `json:"edition"`
	Mtgo    string      `json:"mtgo"`
	Cmc     int64       `json:"cmc"`
	Type    string      `json:"type"`
	Color   string      `json:"color"`
	Data    interface{} `json:"data"`
}

// Seat is exported to the client to help describe the draft.
type Seat struct {
	Rounds []Round `json:"rounds"`
	Name   string  `json:"name"`
	ID     int64   `json:"id"`
}

// Round is exported to the client to help describe the draft.
type Round struct {
	Packs []Pack `json:"packs"`
	Round int64  `json:"round"`
}

// Pack is exported to the client to help describe the draft.
type Pack struct {
	Cards []Card `json:"cards"`
}

// DraftJSON is the old shitty way to tell the replay client about the draft.
type DraftJSON struct {
	Seats  []Seat       `json:"seats"`
	Name   string       `json:"name"`
	ID     int64        `json:id"`
	Events []DraftEvent `json:"events"`
}

// Perspective tells the client from which user's perspective the replay data is from.
type Perspective struct {
	User  int64      `json:"user"`
	Draft DraftJSON2 `json:"draft"`
}

// DraftJSON2 describes the draft to the replay viewer.
type DraftJSON2 struct {
	DraftID   int64         `json:"draftId"`
	DraftName string        `json:"draftName"`
	Seats     [8]Seat2      `json:"seats"`
	Events    []DraftEvent2 `json:"events"`
}

// Seat2 is part of DraftJSON2.
type Seat2 struct {
	Packs       [3][15]interface{} `json:"packs"`
	PlayerName  string             `json:"playerName"`
	PlayerID    int64              `json:"playerId"`
	PlayerImage string             `json:"playerImage"`
}

// DraftEvent2 is part of DraftJSON2.
type DraftEvent2 struct {
	Position       int64    `json:"position"`
	Announcements  []string `json:"announcements"`
	Cards          []int64  `json:"cards"`
	PlayerModified int64    `json:"playerModified"`
	DraftModified  int64    `json:"draftModified"`
	Round          int64    `json:"round"`
	Librarian      bool     `json:"librarian"`
	Type           string   `json:"type"`
}

// DraftEvent describes draft events to the replay viewer.
type DraftEvent struct {
	Player         int64    `json:"player"`
	Announcements  []string `json:"announcements"`
	Card1          string
	Card2          string
	Cards          []int64 `json:"cards"`
	PlayerModified int64   `json:"playerModified"`
	DraftModified  int64   `json:"draftModified"`
	Round          int64   `json:"round"`
	Librarian      bool    `json:"librarian"`
}

// ReplayPageData is the input to replay.tmpl.
type ReplayPageData struct {
	JSON     string
	UserJSON string
	JSON2    string
}

// BulkMTGOExport is used to bulk export .dek files for the admin.
type BulkMTGOExport struct {
	PlayerID int64
	Username string
	Deck     string
}

// NameAndQuantity is used in MTGO .dek exports.
type NameAndQuantity struct {
	Name     string
	Quantity int64
}

// JSONError helps to pass an error to the client when something breaks.
type JSONError struct {
	Error string `json:"error"`
}

// DraftList is turned into JSON and used for the REST API.
type DraftList struct {
	Drafts []DraftListEntry `json:"drafts"`
}

// DraftListEntry is turned into JSON and used for the REST API.
type DraftListEntry struct {
	ID             int64  `json:"id"`
	Name           string `json:"name"`
	AvailableSeats int64  `json:"availableSeats"`
	Status         string `json:"status"`
}

// PostedPick is JSON accepted from the client when a user makes a pick.
type PostedPick struct {
	CardIds []int64 `json:"cards"`
}

// UserInfo is JSON passed to the client.
type UserInfo struct {
	Name    string `json:"name"`
	Picture string `json:"picture"`
	ID      int64  `json:"userId"`
}

type r38handler func(w http.ResponseWriter, r *http.Request, userId int64)
type viewingFunc func(r *http.Request, userId int64) (bool, error)

var secretKeyNoOneWillEverGuess = []byte(os.Getenv("SESSION_SECRET"))
var store = sessions.NewCookieStore(secretKeyNoOneWillEverGuess)
var database *sql.DB
var useAuth bool
var isViewing viewingFunc
var sock string

func main() {
	useAuthPtr := flag.Bool("auth", true, "bool")
	flag.Parse()

	useAuth = *useAuthPtr

	if useAuth {
		isViewing = AuthIsViewing
	} else {
		isViewing = NonAuthIsViewing
	}

	var err error

	database, err = sql.Open("sqlite3", "draft.db")
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
		Handler: NewHandler(*useAuthPtr),
	}

	log.Printf("Starting HTTP Server. Listening at %q", server.Addr)
	err = server.ListenAndServe() // this call blocks

	if err != nil {
		log.Printf("%v", err)
	}
}

// NewHandler creates all server routes for serving the html.
func NewHandler(useAuth bool) http.Handler {
	mux := http.NewServeMux()

	middleware := AuthMiddleware

	if !useAuth {
		middleware = NonAuthMiddleware
	}

	mux.HandleFunc("/auth/discord/login", oauthDiscordLogin)
	mux.HandleFunc("/auth/discord/callback", oauthDiscordCallback)

	fs := http.FileServer(http.Dir("static"))
	mux.Handle("/static/", http.StripPrefix("/static/", fs))

	addHandler := func(route string, serveFunc r38handler) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var userID int64
			if useAuth {
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

			serveFunc(w, r, userID)
		})
		mux.Handle(route, middleware(handler))
	}

	addHandler("/proxy/", ServeProxy)
	addHandler("/replay/", ServeReplay)
	addHandler("/deckbuilder/", ServeDeckbuilder)
	addHandler("/librarian/", ServeLibrarian)
	addHandler("/power/", ServePower)
	addHandler("/draft/", ServeDraft)
	addHandler("/pdf/", ServePDF)
	addHandler("/pick/", ServePick)
	addHandler("/join/", ServeJoin)
	addHandler("/mtgo/", ServeMTGO)
	addHandler("/bulk_mtgo/", ServeBulkMTGO)
	addHandler("/index/", ServeIndex)

	addHandler("/api/draft/", ServeAPIDraft)
	addHandler("/api/draftlist/", ServeAPIDraftList)
	addHandler("/api/pick/", ServeAPIPick)

	addHandler("/", ServeIndex)

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
			if !strings.HasPrefix(r.URL.Path, "/proxy/") {
				log.Printf("%s %s", session.Values["userid"], r.URL.Path)
			}
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
		if !strings.HasPrefix(r.URL.Path, "/proxy/") {
			log.Printf(r.URL.Path)
		}
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
func ServeAPIDraft(w http.ResponseWriter, r *http.Request, userID int64) {
	re := regexp.MustCompile(`/api/draft/(\d+)`)
	parseResult := re.FindStringSubmatch(r.URL.Path)
	if parseResult == nil {
		json.NewEncoder(w).Encode(JSONError{Error: "bad api url"})
		return
	}
	draftID, err := strconv.ParseInt(parseResult[1], 10, 64)
	if err != nil {
		json.NewEncoder(w).Encode(JSONError{Error: fmt.Sprintf("bad api url: %s", err.Error())})
		return
	}

	draftJSON, err := GetFilteredJSON(draftID, userID)
	if err != nil {
		json.NewEncoder(w).Encode(JSONError{Error: fmt.Sprintf("error getting json: %s", err.Error())})
		return
	}

	fmt.Fprint(w, draftJSON)
}

// ServeAPIDraftList serves the /api/draftlist endpoint.
func ServeAPIDraftList(w http.ResponseWriter, r *http.Request, userID int64) {
	query := `select drafts.id, drafts.name, sum(seats.user is null and seats.position is not null) as empty_seats, coalesce(sum(seats.user = ?), 0) as joined from drafts left join seats on drafts.id = seats.draft group by drafts.id`

	rows, err := database.Query(query, userID)
	if err != nil {
		json.NewEncoder(w).Encode(JSONError{Error: fmt.Sprintf("can't get draft list: %s", err.Error())})
		return
	}
	defer rows.Close()
	var drafts DraftList
	for rows.Next() {
		var d DraftListEntry
		var joined int64
		err = rows.Scan(&d.ID, &d.Name, &d.AvailableSeats, &joined)
		if err != nil {
			json.NewEncoder(w).Encode(JSONError{Error: fmt.Sprintf("can't get draft list: %s", err.Error())})
			return
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
}

// ServeAPIPick serves the /api/pick endpoint.
func ServeAPIPick(w http.ResponseWriter, r *http.Request, userID int64) {
	if r.Method != "POST" {
		http.Error(w, "invalid request method", http.StatusMethodNotAllowed)
		return
	}

	bodyBytes, err := ioutil.ReadAll(r.Body)
	if err != nil {
		json.NewEncoder(w).Encode(JSONError{Error: fmt.Sprintf("error reading post body: %s", err.Error())})
		return
	}
	var pick PostedPick
	err = json.Unmarshal(bodyBytes, &pick)
	if err != nil {
		json.NewEncoder(w).Encode(JSONError{Error: fmt.Sprintf("error parsing post body: %s", err.Error())})
		return
	}
	var draftID int64
	if len(pick.CardIds) == 1 {
		draftID, _, announcements, round, err := doPick(userID, pick.CardIds[0], true)
		if err != nil {
			json.NewEncoder(w).Encode(JSONError{Error: fmt.Sprintf("error making pick: %s", err.Error())})
			return
		}

		err = DoEvent(draftID, userID, announcements, pick.CardIds[0], sql.NullInt64{Valid: false}, round)
		if err != nil {
			json.NewEncoder(w).Encode(JSONError{Error: fmt.Sprintf("error recording event: %s", err.Error())})
			return
		}
	} else if len(pick.CardIds) == 2 {
		json.NewEncoder(w).Encode(JSONError{Error: "cogwork librarian power not implemented yet"})
		return
	} else {
		json.NewEncoder(w).Encode(JSONError{Error: fmt.Sprintf("invalid number of cards: %d", len(pick.CardIds))})
		return
	}

	draftJSON, err := GetFilteredJSON(draftID, userID)
	if err != nil {
		json.NewEncoder(w).Encode(JSONError{Error: fmt.Sprintf("error getting json: %s", err.Error())})
		return
	}

	fmt.Fprint(w, draftJSON)
}

// ServeBulkMTGO serves a .zip file of all .dek files for a draft. Only useful to the admin.
func ServeBulkMTGO(w http.ResponseWriter, r *http.Request, userID int64) {
	if userID != 1 {
		http.Error(w, "auth error in bulk export", http.StatusForbidden)
		return
	}
	re := regexp.MustCompile(`/bulk_mtgo/(\d+)`)
	parseResult := re.FindStringSubmatch(r.URL.Path)
	if parseResult == nil {
		http.Error(w, "draft not found", http.StatusInternalServerError)
		return
	}
	draftID, err := strconv.ParseInt(parseResult[1], 10, 64)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	query := `select seats.user, users.discord_name from seats join users where seats.user=users.id and seats.draft=?`
	rows, err := database.Query(query, draftID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
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
			log.Printf("error reading player in draft %s, skipping: %s", draftID, err)
			break
		}
		export, err := exportToMTGO(playerID, draftID)
		if err != nil {
			log.Printf("could not export to MTGO for player %s in draft %s: %s", playerID, draftID, err)
			break
		}
		exports = append(exports, BulkMTGOExport{PlayerID: playerID, Username: re.ReplaceAllString(username, "_"), Deck: export})
	}

	// Generate the ZIP file for all exported decks.
	archive, err := createZipExport(exports)
	if err != nil {
		log.Printf("error creating zip file: %s", err)
		return
	}
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%d-r38-bulk.zip", draftID))
	w.Header().Set("Content-Type", r.Header.Get("Content-Type"))
	io.WriteString(w, string(archive))
}

// ServeMTGO exports a draft to a .dek file for the user.
func ServeMTGO(w http.ResponseWriter, r *http.Request, userID int64) {
	re := regexp.MustCompile(`/mtgo/(\d+)`)
	parseResult := re.FindStringSubmatch(r.URL.Path)

	if parseResult == nil {
		http.Error(w, "bad url", http.StatusInternalServerError)
		return
	}

	draftIDInt, err := strconv.Atoi(parseResult[1])
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	draftID := int64(draftIDInt)

	export, err := exportToMTGO(userID, draftID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Disposition", "attachment; filename=r38export.dek")
	w.Header().Set("Content-Type", r.Header.Get("Content-Type"))
	io.WriteString(w, export)
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
func exportToMTGO(userID int64, draftID int64) (string, error) {
	_, picks, _, err := getPackPicksAndPowers(draftID, userID)
	if err != nil {
		return "", err
	}
	export := "<?xml version=\"1.0\" encoding=\"utf-8\"?>\n<Deck xmlns:xsd=\"http://www.w3.org/2001/XMLSchema\" xmlns:xsi=\"http://www.w3.org/2001/XMLSchema-instance\">\n<NetDeckID>0</NetDeckID>\n<PreconstructedDeckID>0</PreconstructedDeckID>\n"
	var nq map[string]NameAndQuantity
	nq = make(map[string]NameAndQuantity)
	for _, pick := range picks {
		if pick.Mtgo != "" {
			o := nq[pick.Mtgo]
			o.Name = pick.Name
			o.Quantity++
			nq[pick.Mtgo] = o
		}
	}
	for mtgo, info := range nq {
		export = export + fmt.Sprintf("<Cards CatID=\"%s\" Quantity=\"%d\" Sideboard=\"false\" Name=\"%s\" />\n", mtgo, info.Quantity, info.Name)
	}
	export = export + "</Deck>"
	return export, nil
}

// proxyCard is a wrapper around Scryfall's REST API to follow redirects and grab image contents.
func proxyCard(edition, number string) ([]byte, error) {
	scryfall := "http://api.scryfall.com/cards/%s/%s?format=image&version=normal"
	response, err := http.Get(fmt.Sprintf(scryfall, edition, number))
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	contents, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	return contents, nil
}

// ServeProxy repackages proxied image content with a Cache-Control header.
func ServeProxy(w http.ResponseWriter, r *http.Request, userID int64) {
	re := regexp.MustCompile(`/proxy/([a-zA-Z0-9]+)/([a-zA-Z0-9]+)/?`)
	parseResult := re.FindStringSubmatch(r.URL.Path)
	if parseResult == nil {
		http.Error(w, "bad url", http.StatusInternalServerError)
		return
	}
	img, err := proxyCard(parseResult[1], parseResult[2])
	if err != nil {
		fmt.Fprintf(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Cache-Control", "max-age=86400,public")
	w.Header().Set("Content-Type", "image/jpeg")
	w.Write(img)
	return
}

// ServeReplay serves the replay.
func ServeReplay(w http.ResponseWriter, r *http.Request, userID int64) {
	re := regexp.MustCompile(`/replay/(\d+)`)
	parseResult := re.FindStringSubmatch(r.URL.Path)

	ServeVueApp(parseResult, w, userID)
}

// ServeDeckbuilder serves the deckbuilder.
func ServeDeckbuilder(w http.ResponseWriter, r *http.Request, userID int64) {
	re := regexp.MustCompile(`/deckbuilder/(\d+)`)
	parseResult := re.FindStringSubmatch(r.URL.Path)

	ServeVueApp(parseResult, w, userID)
}

// ServeVueApp serves to vue.
func ServeVueApp(parseResult []string, w http.ResponseWriter, userID int64) {
	if parseResult == nil {
		http.Error(w, "bad url", http.StatusInternalServerError)
		return
	}

	draftIDInt, err := strconv.Atoi(parseResult[1])
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	draftID := int64(draftIDInt)

	canViewReplay, err := CanViewReplay(draftID, userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if !canViewReplay {
		http.Error(w, "lol no", http.StatusInternalServerError)
		return
	}

	draftJSON, err := GetJSON(draftID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	query := `select id,discord_name,picture from users where id=?`
	row := database.QueryRow(query, userID)
	var userInfo UserInfo
	err = row.Scan(&userInfo.ID, &userInfo.Name, &userInfo.Picture)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	userInfoJSON, err := json.Marshal(userInfo)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	draftJSON2, err := GetJSON2(draftID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := ReplayPageData{JSON: draftJSON, UserJSON: string(userInfoJSON), JSON2: draftJSON2}

	t := template.Must(template.ParseFiles("replay.tmpl"))

	t.Execute(w, data)
}

// CanViewReplay returns true if the given user is able to view the full replay of the given draft.
func CanViewReplay(draftID int64, userID int64) (bool, error) {
	query := `select min(round) from seats where draft=?`
	row := database.QueryRow(query, draftID)
	var round int64
	err := row.Scan(&round)
	if err != nil {
		return false, err
	}

	if round != 4 && userID != 1 && draftID != 9 {
		query = `select user from seats where draft=? and position is not null`
		rows, err := database.Query(query, draftID)
		if err != nil {
			return false, err
		}
		defer rows.Close()

		valid := true
		for rows.Next() {
			var playerID sql.NullInt64
			err = rows.Scan(&playerID)
			if err != nil {
				return false, err
			}
			if !playerID.Valid || playerID.Int64 == userID {
				valid = false
				break
			}
		}

		if !valid {
			return false, nil
		}
	}

	return true, nil
}

// ServeLibrarian handles cogwork librarian picks.
func ServeLibrarian(w http.ResponseWriter, r *http.Request, userID int64) {
	re := regexp.MustCompile(`/librarian/(\d+)/(\d+)`)
	parseResult := re.FindStringSubmatch(r.URL.Path)

	if parseResult == nil {
		http.Error(w, "bad url", http.StatusInternalServerError)
		return
	}

	cardID1Int, err := strconv.Atoi(parseResult[1])
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	cardID2Int, err := strconv.Atoi(parseResult[2])
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	cardID1 := int64(cardID1Int)
	cardID2 := int64(cardID2Int)

	query := `select seats.draft from cards join packs join seats where cards.pack=packs.id and packs.seat=seats.id and cards.id IN (?,?)`

	rows, err := database.Query(query, cardID1, cardID2)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	var checkID int64
	checkID = 0
	rowCount := 0
	for rows.Next() {
		rowCount++
		var checkID2 int64
		err = rows.Scan(&checkID2)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if checkID == 0 {
			checkID = checkID2
		} else if checkID != checkID2 {
			http.Error(w, "woah there!", http.StatusInternalServerError)
			return
		}
	}

	if checkID == 0 || rowCount != 2 {
		http.Error(w, "woah there", http.StatusInternalServerError)
		return
	}

	draftID1, packID1, announcements1, round1, err := doPick(userID, cardID1, false)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	draftID2, packID2, announcements2, round2, err := doPick(userID, cardID2, true)

	if packID1 != packID2 {
		http.Error(w, "pack ids somehow don't match.", http.StatusInternalServerError)
		return
	}

	if draftID1 != draftID2 {
		http.Error(w, "draft ids somehow don't match.", http.StatusInternalServerError)
		return
	}

	if round1 != round2 {
		http.Error(w, "rounds somehow don't match.", http.StatusInternalServerError)
		return
	}

	query = `select position from seats where draft=? and user=?`
	row := database.QueryRow(query, draftID1, userID)
	var position int64
	err = row.Scan(&position)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	announcements := append(announcements1, fmt.Sprintf("Seat %d used Cogwork Librarian's ability", position))
	announcements = append(announcements, announcements2...)

	query = `select cards.id from cards join packs join seats where cards.pack=packs.id and packs.seat=seats.id and seats.draft=? and cards.name="Cogwork Librarian" and seats.user=?`

	row = database.QueryRow(query, draftID1, userID)
	var librarianID int64
	err = row.Scan(&librarianID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	query = `update cards set pack=?, faceup=false where id=?`
	log.Printf("%s\t%d,%d", query, packID1, librarianID)
	_, err = database.Exec(query, packID1, librarianID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = DoEvent(draftID1, userID, announcements, cardID1, sql.NullInt64{Int64: cardID2, Valid: true}, round1)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/draft/%d", draftID1), http.StatusTemporaryRedirect)
}

// ServePower serves pages to the user when they try to use a power (right now just Cogwork Librarian).
func ServePower(w http.ResponseWriter, r *http.Request, userID int64) {
	re := regexp.MustCompile(`/power/(\d+)`)
	parseResult := re.FindStringSubmatch(r.URL.Path)

	if parseResult == nil {
		http.Error(w, "bad url", http.StatusInternalServerError)
		return
	}

	cardIDInt, err := strconv.Atoi(parseResult[1])
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	cardID := int64(cardIDInt)

	query := `select cards.name, seats.draft from cards join packs join seats where cards.pack=packs.id and packs.seat=seats.id and seats.user=? and cards.id=? and cards.faceup=true`

	row := database.QueryRow(query, userID, cardID)
	var cardName string
	var draftID int64
	err = row.Scan(&cardName, &draftID)
	if err == sql.ErrNoRows {
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	} else if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	switch cardName {
	case "Cogwork Librarian":
		// show the current pack and current picks
		myPack, myPicks, _, err := getPackPicksAndPowers(draftID, userID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if len(myPack) < 2 {
			http.Redirect(w, r, fmt.Sprintf("/draft/%d", draftID), http.StatusTemporaryRedirect)
			return
		}

		t := template.Must(template.ParseFiles("librarian.tmpl"))

		data := DraftPageData{Pack: myPack, Picks: myPicks, DraftID: draftID}
		t.Execute(w, data)
		// use some js to construct the url /librarian/cogworklibrarianid/pick1id/pick2id
		// redirect to the draft url
	}
}

// ServeIndex serves the index page.
func ServeIndex(w http.ResponseWriter, r *http.Request, userID int64) {
	query := `select drafts.id, drafts.name, sum(seats.user is null and seats.position is not null) as empty_seats, coalesce(sum(seats.user = ?), 0) as joined from drafts left join seats on drafts.id = seats.draft group by drafts.id`

	rows, err := database.Query(query, userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	var Drafts []Draft
	for rows.Next() {
		var d Draft
		err = rows.Scan(&d.ID, &d.Name, &d.Seats, &d.Joined)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		d.Joinable = d.Seats > 0 && !d.Joined
		d.Replayable, err = CanViewReplay(d.ID, userID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		Drafts = append(Drafts, d)
	}

	viewParam := GetViewParam(r, userID)
	data := IndexPageData{Drafts: Drafts, ViewURL: viewParam, UserID: userID}
	t := template.Must(template.ParseFiles("index.tmpl"))
	t.Execute(w, data)
}

// ServePDF serves a pdf suitable for printing proxies.
func ServePDF(w http.ResponseWriter, r *http.Request, userID int64) {
	re := regexp.MustCompile(`/pdf/(\d+)`)
	parseResult := re.FindStringSubmatch(r.URL.Path)

	if parseResult == nil {
		http.Error(w, "bad url", http.StatusInternalServerError)
		return
	}

	draftIDInt, err := strconv.Atoi(parseResult[1])
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	draftID := int64(draftIDInt)

	_, myPicks, _, err := getPackPicksAndPowers(draftID, userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/pdf")

	pdf := gofpdf.New("P", "in", "Letter", "")
	pdf.AddPage()
	pdf.SetFont("Arial", "B", float64(12))
	pdf.SetTextColor(255, 255, 0)
	cardsOnLine := 0
	linesOnPage := 0
	options := gofpdf.ImageOptions{
		ImageType:             "JPG",
		ReadDpi:               false,
		AllowNegativePosition: false,
	}

	for idx, pick := range myPicks {
		imgResp, err := http.Get(
			fmt.Sprintf("http://api.scryfall.com/cards/%s/%s?format=image&version=normal",
				pick.Edition, pick.Number))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		pdf.RegisterImageOptionsReader(strconv.Itoa(idx), options, imgResp.Body)
		x := 0.25 + float64(cardsOnLine)*2.4
		y := 0.25 + float64(linesOnPage)*3.35
		pdf.ImageOptions(strconv.Itoa(idx), x, y, 2.4, 0, false, options, 0, "")

		pdf.SetXY(x, y+3.15)
		pdf.Cell(0.1, 0.1, pick.Tags)

		cardsOnLine++
		if cardsOnLine == 3 {
			cardsOnLine = 0
			linesOnPage++
			if linesOnPage == 3 {
				linesOnPage = 0
				pdf.AddPage()
			}
		}

		err = imgResp.Body.Close()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	err = pdf.Output(w)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// ServeDraft serves the draft page.
func ServeDraft(w http.ResponseWriter, r *http.Request, userID int64) {
	re := regexp.MustCompile(`/draft/(\d+)`)
	parseResult := re.FindStringSubmatch(r.URL.Path)

	if parseResult == nil {
		http.Error(w, "bad url", http.StatusInternalServerError)
		return
	}

	draftIDInt, err := strconv.Atoi(parseResult[1])
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	draftID := int64(draftIDInt)

	myPack, myPicks, powers2, err := getPackPicksAndPowers(draftID, userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	query := `select seats.position, drafts.name from seats join drafts where seats.draft=? and seats.user=? and seats.draft=drafts.id`
	row := database.QueryRow(query, draftID, userID)
	var position int64
	var draftName string
	err = row.Scan(&position, &draftName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	query = `select message from revealed where draft=?`
	rows, err := database.Query(query, draftID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	var revealed []string
	for rows.Next() {
		var msg string
		err = rows.Scan(&msg)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		revealed = append(revealed, msg)
	}
	viewParam := GetViewParam(r, userID)
	t := template.Must(template.ParseFiles("draft.tmpl"))

	data := DraftPageData{Picks: myPicks, Pack: myPack, DraftID: draftID, DraftName: draftName, Powers: powers2, Position: position, Revealed: revealed, ViewURL: viewParam}

	t.Execute(w, data)
}

// ServeJoin allows the user to join a draft, if possible.
func ServeJoin(w http.ResponseWriter, r *http.Request, userID int64) {
	re := regexp.MustCompile(`/join/(\d+)`)
	parseResult := re.FindStringSubmatch(r.URL.Path)

	if parseResult == nil {
		http.Error(w, "bad url", http.StatusInternalServerError)
		return
	}

	draftIDInt, err := strconv.Atoi(parseResult[1])
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	draftID := int64(draftIDInt)

	query := `select true from seats where draft=? and user=?`

	row := database.QueryRow(query, draftID, userID)
	var alreadyJoined bool
	err = row.Scan(&alreadyJoined)

	if err != nil && err != sql.ErrNoRows {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	} else if err != sql.ErrNoRows || alreadyJoined {
		http.Redirect(w, r, fmt.Sprintf("/draft/%s", draftID), http.StatusTemporaryRedirect)
		return
	}

	query = `update seats set user=? where id=(select id from seats where draft=? and user is null and position is not null order by random() limit 1)`
	log.Printf("%s\t%d,%d", query, userID, draftID)

	_, err = database.Exec(query, userID, draftID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	viewParam := GetViewParam(r, userID)
	http.Redirect(w, r, fmt.Sprintf("/draft/%d%s", draftID, viewParam), http.StatusTemporaryRedirect)
}

// ServePick handles the user picking cards.
func ServePick(w http.ResponseWriter, r *http.Request, userID int64) {
	re := regexp.MustCompile(`/pick/(\d+)`)
	parseResult := re.FindStringSubmatch(r.URL.Path)

	if parseResult == nil {
		http.Error(w, "bad url", http.StatusInternalServerError)
		return
	}

	cardIDInt, err := strconv.Atoi(parseResult[1])
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	cardID := int64(cardIDInt)
	draftID, _, announcements, round, err := doPick(userID, cardID, true)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = DoEvent(draftID, userID, announcements, cardID, sql.NullInt64{Valid: false}, round)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	viewParam := GetViewParam(r, userID)
	http.Redirect(w, r, fmt.Sprintf("/draft/%d%s", draftID, viewParam), http.StatusTemporaryRedirect)
}

// getPackPicksAndPowers returns data useful to active drafters.
func getPackPicksAndPowers(draftID int64, userID int64) ([]Card, []Card, []Card, error) {
	query := `select
                    packs.round,
                    cards.id,
                    cards.name,
                    cards.tags,
                    cards.number,
                    cards.edition,
                    cards.faceup,
                    cards.mtgo
                  from drafts
                  join seats on drafts.id=seats.draft
                  join packs on seats.id=packs.seat
                  join cards on packs.id=cards.pack
                  where seats.draft=?
                    and seats.user=?
                    and (packs.round=0
                         or (packs.round=seats.round
                             and packs.id=
                               (select v_packs.id
                                from v_packs
                                join seats on seats.id=v_packs.seat
                                where seats.draft=?
                                  and seats.user=?
                                  and v_packs.round=seats.round
                                order by v_packs.count desc
                                limit 1)))
                  order by cards.modified`

	rows, err := database.Query(query, draftID, userID, draftID, userID)
	if err == sql.ErrNoRows {
		return nil, nil, nil, errors.New("no cards")
	} else if err != nil {
		return nil, nil, nil, err
	}
	defer rows.Close()
	var count int
	var myPicks []Card
	var myPack []Card
	var powers []Card
	for rows.Next() {
		var id int64
		var round int64
		var name string
		var tags string
		var number string
		var edition string
		var faceup bool
		var mtgo sql.NullString
		err = rows.Scan(&round, &id, &name, &tags, &number, &edition, &faceup, &mtgo)
		if err != nil {
			return nil, nil, nil, err
		}

		var mtgoString string
		if mtgo.Valid {
			mtgoString = mtgo.String
		}

		if round == 0 {
			myPicks = append(myPicks, Card{Name: name, Tags: tags, Number: number, Edition: edition, ID: id, Mtgo: mtgoString})
			if faceup == true {
				switch name {
				case "Cogwork Librarian":
					powers = append(powers, Card{Name: name, Tags: tags, Number: number, Edition: edition, ID: id, Mtgo: mtgoString})
				}
			}
		} else {
			myPack = append(myPack, Card{Name: name, Tags: tags, Number: number, Edition: edition, ID: id, Mtgo: mtgoString})
		}
		count++
	}

	if count == 0 {
		return nil, nil, nil, errors.New("no cards")
	}

	var powers2 []Card
	for _, powerCard := range powers {
		switch powerCard.Name {
		case "Cogwork Librarian":
			if len(myPack) >= 2 {
				powers2 = append(powers2, powerCard)
			}
		}
	}

	return myPack, myPicks, powers2, nil
}

// doPick actually performs a pick in the database.
func doPick(userID int64, cardID int64, pass bool) (int64, int64, []string, int64, error) {
	announcements := []string{}

	query := `select drafts.id, seats.round, seats.position, cards.name from drafts join seats join packs join cards where drafts.id=seats.draft and seats.id=packs.seat and packs.id=cards.pack and cards.id=? and packs.round <> 0`

	row := database.QueryRow(query, cardID)
	var draftID int64
	var round int64
	var position int64
	var cardName string
	err := row.Scan(&draftID, &round, &position, &cardName)
	if err != nil {
		return -1, -1, announcements, -1, err
	}

	query = `select packs.id, packs.modified from drafts join seats join packs join cards where cards.id=? and drafts.id=? and drafts.id=seats.draft and seats.id=packs.seat and packs.id=cards.pack and seats.user=? and (packs.round=0 or (packs.round=seats.round and packs.modified in (select min(packs.modified) from packs join seats join drafts where seats.draft=? and seats.user=? and seats.id=packs.seat and drafts.id=seats.draft and packs.round=seats.round)))`

	row = database.QueryRow(query, cardID, draftID, userID, draftID, userID)
	var oldPackID int64
	var modified int64
	err = row.Scan(&oldPackID, &modified)
	if err != nil {
		return draftID, -1, announcements, round, err
	}

	query = `select packs.id, seats.id from packs join seats where seats.user=? and seats.id=packs.seat and packs.round=0 and seats.draft=?`

	row = database.QueryRow(query, userID, draftID)
	var pickID int64
	var seatID int64
	err = row.Scan(&pickID, &seatID)
	if err != nil {
		return draftID, oldPackID, announcements, round, err
	}

	query = `select id from seats where draft=? and position=?`

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
	row = database.QueryRow(query, draftID, newPosition)
	var newPositionID int64
	err = row.Scan(&newPositionID)
	if err != nil {
		return draftID, oldPackID, announcements, round, err
	}

	var cardModified int64
	cardModified = 0
	query = `select max(cards.modified) from cards join packs on cards.pack=packs.id join seats on seats.id=packs.seat where seats.draft=? and seats.user=?`
	row = database.QueryRow(query, draftID, userID)

	err = row.Scan(&cardModified)

	if err != nil {
		cardModified = 0
	} else {
		cardModified++
	}

	if pass {
		query = `begin transaction;update cards set pack=?, modified=? where id=?;update packs set seat=?, modified=modified+10 where id=?;commit`
		log.Printf("%s\t%d,%d,%d,%d,%d", query, pickID, cardModified, cardID, newPositionID, oldPackID)

		_, err = database.Exec(query, pickID, cardModified, cardID, newPositionID, oldPackID)
		if err != nil {
			return draftID, oldPackID, announcements, round, err
		}

		query = `select count(1) from v_packs join seats where v_packs.seat=seats.id and seats.user=? and v_packs.round=? and v_packs.count>0 and seats.draft=?`
		row = database.QueryRow(query, userID, round, draftID)
		var packsLeftInSeat int64
		err = row.Scan(&packsLeftInSeat)
		if err != nil {
			return draftID, oldPackID, announcements, round, err
		}

		if packsLeftInSeat == 0 {
			query = `select count(1) from seats a join seats b on a.draft=b.draft where a.user=? and b.position=? and a.draft=? and a.round=b.round`
			row = database.QueryRow(query, userID, newPosition, draftID)
			var roundsMatch int64
			err = row.Scan(&roundsMatch)
			if err != nil {
				log.Printf("cannot determine if rounds match for notify")
			} else if roundsMatch == 1 {
				err = NotifyByDraftAndPosition(draftID, newPosition)
				if err != nil {
					log.Printf("error with notify")
				}
			}

			// WARNING: if you ever have a draft with anything other than 8 players, 15 cards per pack, and 3 rounds, this will break horribly.
			query = `update seats set round=round+1 where user=? and draft=? and (select count(1) in (15, 30, 45) from packs join cards on cards.pack=packs.id join seats on seats.id=packs.seat where seats.draft=? and packs.round=0 and seats.user=?)`
			log.Printf("%s\t%d,%d,%d,%d", query, userID, draftID, draftID, userID)
			_, err = database.Exec(query, userID, draftID, draftID, userID)
			if err != nil {
				log.Printf("error with possibly updating round")
			}

			if roundsMatch == 0 {
				query = `select count(1) from seats where draft=? group by round order by round desc limit 1`

				row = database.QueryRow(query, draftID)
				var nextRoundPlayers int64
				err = row.Scan(&nextRoundPlayers)
				if err != nil {
					log.Printf("error counting players and rounds")
				} else if nextRoundPlayers > 1 {
					query = `select seats.position from seats left join packs on seats.id=packs.seat join v_packs on v_packs.id=packs.id where v_packs.count>0 and packs.round=seats.round and seats.draft=? group by seats.id`

					rows, err := database.Query(query, draftID)
					if err != nil {
						log.Printf("error determining if there's a blocking player")
					} else {
						defer rows.Close()

						rowCount := 0
						var blockingPosition int64
						for rows.Next() {
							rowCount++
							err = rows.Scan(&blockingPosition)
							if err != nil {
								log.Printf("some kind of error with scanning: %s", err.Error())
								rowCount = 2
								break
							}
						}
						if rowCount == 1 {
							err = NotifyByDraftAndPosition(draftID, blockingPosition)
							if err != nil {
								log.Printf("error with blocking notify")
							}
						}
					}
				}
			}
		}
	} else {
		query = `update cards set pack=?, modified=? where id=?`
		log.Printf("%s\t%d,%s,%d,%d", query, pickID, cardModified, cardID)

		_, err = database.Exec(query, pickID, cardModified, cardID)
		if err != nil {
			return draftID, oldPackID, announcements, round, err
		}
	}

	switch cardName {
	case "Cogwork Librarian":
		query = `update cards set faceup=true where id=?`
		log.Printf("%s\t%d", query, cardID)
		_, err = database.Exec(query, cardID)
		if err != nil {
			return draftID, oldPackID, announcements, round, err
		}

		query = `INSERT INTO revealed (draft, message) VALUES (?, "Seat " || ? || " revealed Cogwork Librarian.")`
		log.Printf("%s\t%d,%d", query, draftID, position)
		_, err = database.Exec(query, draftID, position)
		if err != nil {
			return draftID, oldPackID, announcements, round, err
		}

		announcements = append(announcements, fmt.Sprintf("Seat %d revealed Cogwork Librarian", position))
	}

	return draftID, oldPackID, announcements, round, nil
}

// NotifyByDraftAndPosition sends a discord alert to a user.
func NotifyByDraftAndPosition(draftID int64, position int64) error {
	log.Printf("Attempting to notify %d %d", draftID, position)

	query := `select users.discord_id from users join seats where users.id=seats.user and seats.draft=? and seats.position=?`

	row := database.QueryRow(query, draftID, position)
	var discordID string
	err := row.Scan(&discordID)
	if err != nil {
		return err
	}

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

// GetJSONObject returns a DraftJSON object, which describes a draft replay.
func GetJSONObject(draftID int64) (DraftJSON, error) {
	var draft DraftJSON

	query := `select
                    drafts.name,
                    seats.position,
                    packs.original_seat,
                    packs.round,
                    cards.name,
                    cards.edition,
                    cards.number,
                    cards.tags,
                    users.discord_name,
                    cards.cmc,
                    cards.type,
                    cards.color,
                    cards.mtgo,
                    cards.id,
                    users.id,
                    cards.data,
                    drafts.id
                  from seats
                  left join users on users.id=seats.user
                  join drafts on drafts.id=seats.draft
                  join packs on packs.original_seat=seats.id
                  join cards on cards.original_pack=packs.id
                  where drafts.id=?`

	rows, err := database.Query(query, draftID)
	if err != nil {
		return draft, err
	}
	defer rows.Close()
	var i int64
	var j int64
	for i = 0; i < 8; i++ {
		draft.Seats = append(draft.Seats, Seat{Rounds: []Round{}})
		for j = 0; j < 4; j++ {
			draft.Seats[i].Rounds = append(draft.Seats[i].Rounds, Round{Round: j, Packs: []Pack{Pack{Cards: []Card{}}}})
		}
	}

	for rows.Next() {
		var nullablePosition sql.NullInt64
		var packSeat int64
		var nullableRound sql.NullInt64
		var card Card
		var nullableDiscordID sql.NullString
		var nullableCmc sql.NullInt64
		var nullableType sql.NullString
		var nullableColor sql.NullString
		var nullableMtgo sql.NullString
		var draftUserID sql.NullInt64
		var nullableData sql.NullString
		err = rows.Scan(&draft.Name, &nullablePosition, &packSeat, &nullableRound, &card.Name, &card.Edition, &card.Number, &card.Tags, &nullableDiscordID, &nullableCmc, &nullableType, &nullableColor, &nullableMtgo, &card.ID, &draftUserID, &nullableData, &draft.ID)
		if err != nil {
			return draft, err
		}

		if nullableCmc.Valid {
			card.Cmc = nullableCmc.Int64
		} else {
			card.Cmc = -1
		}
		card.Type = nullableType.String
		card.Color = nullableColor.String
		if !nullableData.Valid || nullableData.String == "" {
			card.Data = nil
		} else {
			dataObj := make(map[string]interface{})
			err = json.Unmarshal([]byte(nullableData.String), &dataObj)
			if err != nil {
				log.Printf("making nil card data because of error %s", err.Error())
				card.Data = nil
			} else {
				card.Data = dataObj
			}
			dataObj["r38_data"].(map[string]interface{})["id"] = card.ID
		}
		if nullableMtgo.Valid {
			card.Mtgo = nullableMtgo.String
		} else {
			card.Mtgo = ""
		}

		if !nullablePosition.Valid || !nullableRound.Valid {
			continue
		}
		position := nullablePosition.Int64
		packRound := nullableRound.Int64

		draft.Seats[position].Rounds[packRound].Packs[0].Cards = append(draft.Seats[position].Rounds[packRound].Packs[0].Cards, card)
		draft.Seats[position].Name = nullableDiscordID.String
		draft.Seats[position].ID = draftUserID.Int64
	}

	query = `select seats.position, events.announcement, cards1.name, cards2.name, events.id, events.modified, events.round, cards1.id, cards2.id from events join seats on events.draft=seats.draft and events.user=seats.user left join cards as cards1 on events.card1=cards1.id left join cards as cards2 on events.card2=cards2.id where events.draft=?`
	rows, err = database.Query(query, draftID)
	if err != nil && err != sql.ErrNoRows {
		return draft, err
	}
	defer rows.Close()
	for rows.Next() {
		var event DraftEvent
		var announcements string
		var card2 sql.NullString
		var card1id int64
		var card2id sql.NullInt64
		err = rows.Scan(&event.Player, &announcements, &event.Card1, &card2, &event.DraftModified, &event.PlayerModified, &event.Round, &card1id, &card2id)
		event.Cards = append(event.Cards, card1id)
		if err != nil {
			return draft, err
		}
		if card2.Valid {
			event.Card2 = card2.String
			event.Cards = append(event.Cards, card2id.Int64)
			event.Librarian = true
		}
		if announcements != "" {
			event.Announcements = strings.Split(announcements, "\n")
		} else {
			event.Announcements = []string{}
		}
		draft.Events = append(draft.Events, event)
	}

	if len(draft.Events) == 0 {
		draft.Events = []DraftEvent{}
	}

	return draft, nil
}

// GetJSONObject2 returns a better DraftJSON2 object. May be filtered.
func GetJSONObject2(draftID int64) (DraftJSON2, error) {
	var draft DraftJSON2

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
                  left join users on users.id=seats.user
                  join drafts on drafts.id=seats.draft
                  join packs on packs.original_seat=seats.id
                  join cards on cards.original_pack=packs.id
                  where drafts.id=?`

	rows, err := database.Query(query, draftID)
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
		dataObj["r38_data"].(map[string]interface{})["id"] = cardID

		packRound--

		nextIndex := indices[position][packRound]

		draft.Seats[position].Packs[packRound][nextIndex] = dataObj
		draft.Seats[position].PlayerName = nullableDiscordID.String
		draft.Seats[position].PlayerID = draftUserID.Int64
		draft.Seats[position].PlayerImage = nullablePicture.String

		indices[position][packRound]++
	}

	query = `select
                   seats.position,
                   events.announcement,
                   events.card1,
                   events.card2,
                   events.id,
                   events.modified,
                   events.round
                 from events
                 join seats on events.draft=seats.draft and events.user=seats.user
                 where events.draft=?`
	rows, err = database.Query(query, draftID)
	if err != nil && err != sql.ErrNoRows {
		return draft, err
	}
	defer rows.Close()
	for rows.Next() {
		var event DraftEvent2
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
		draft.Events = []DraftEvent2{}
	}

	return draft, nil
}

// GetJSON returns a json string of replay data.
func GetJSON(draftID int64) (string, error) {
	draft, err := GetJSONObject(draftID)
	if err != nil {
		return "", err
	}
	ret, err := json.Marshal(draft)
	if err != nil {
		return "", err
	}

	return string(ret), nil
}

// GetJSON2 returns a better json string of replay data. May be filtered.
func GetJSON2(draftID int64) (string, error) {
	draft, err := GetJSONObject2(draftID)
	if err != nil {
		return "", err
	}
	ret, err := json.Marshal(draft)
	if err != nil {
		return "", err
	}

	return string(ret), nil
}

// GetFilteredJSON returns a filtered json object of replay data.
func GetFilteredJSON(draftID int64, userID int64) (string, error) {
	draft, err := GetJSONObject2(draftID)
	if err != nil {
		return "", err
	}

	query := `select (select round from seats where draft=? and user=?), (select count(1) from seats where draft=? and user is null)`
	var myRound sql.NullInt64
	var emptySeats int64
	row := database.QueryRow(query, draftID, userID, draftID)
	err = row.Scan(&myRound, &emptySeats)
	if err != nil {
		return "", err
	} else if (myRound.Valid && myRound.Int64 == 4) || (!myRound.Valid && emptySeats == 0) {
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

// DoEvent records an event (pick) into the database.
func DoEvent(draftID int64, userID int64, announcements []string, cardID1 int64, cardID2 sql.NullInt64, round int64) error {
	var query string
	var err error
	if cardID2.Valid {
		query = `insert into events (round, draft, user, announcement, card1, card2, modified) VALUES (?, ?, ?, ?, ?, ?, (select count(1) from seats join packs join cards where seats.user=? and seats.draft=? and packs.seat=seats.id and cards.pack=packs.id and packs.round=0))`
		log.Printf("%s\t%d,%d,%d,%s,%d,%d,%d,%d", query, round, draftID, userID, strings.Join(announcements, "\n"), cardID1, cardID2.Int64, userID, draftID)
		_, err = database.Exec(query, round, draftID, userID, strings.Join(announcements, "\n"), cardID1, cardID2.Int64, userID, draftID)
	} else {
		query := `insert into events (round, draft, user, announcement, card1, card2, modified) VALUES (?, ?, ?, ?, ?, null, (select count(1) from seats join packs join cards where seats.user=? and seats.draft=? and packs.seat=seats.id and cards.pack=packs.id and packs.round=0))`
		log.Printf("%s\t%d,%d,%d,%s,%d,%d,%d", query, round, draftID, userID, strings.Join(announcements, "\n"), cardID1, userID, draftID)
		_, err = database.Exec(query, round, draftID, userID, strings.Join(announcements, "\n"), cardID1, userID, draftID)
	}

	return err
}
