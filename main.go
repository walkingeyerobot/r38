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

type Draft struct {
	Name       string
	ID         int64
	Seats      int64
	Joined     bool
	Joinable   bool
	Replayable bool
}

type IndexPageData struct {
	Drafts  []Draft
	ViewUrl string
	UserId  int64
}

type DraftPageData struct {
	DraftId   int64
	DraftName string
	Picks     []Card
	Pack      []Card
	Powers    []Card
	Position  int64
	Revealed  []string
	ViewUrl   string
}

type Card struct {
	Id      int64       `json:"r38Id"`
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

type Seat struct {
	Rounds []Round `json:"rounds"`
	Name   string  `json:"name"`
	Id     int64   `json:"id"`
}

type Round struct {
	Packs []Pack `json:"packs"`
	Round int64  `json:"round"`
}

type Pack struct {
	Cards []Card `json:"cards"`
}

type DraftJson struct {
	Seats  []Seat       `json:"seats"`
	Name   string       `json:"name"`
	Id     int64        `json:id"`
	Events []DraftEvent `json:"events"`
}

type Perspective struct {
	User  int64      `json:"user"`
	Draft DraftJson2 `json:"draft"`
}

type DraftJson2 struct {
	DraftId   int64         `json:"draftId"`
	DraftName string        `json:"draftName"`
	Seats     [8]Seat2      `json:"seats"`
	Events    []DraftEvent2 `json:"events"`
}

type Seat2 struct {
	Packs       [3][15]interface{} `json:"packs"`
	PlayerName  string             `json:"playerName"`
	PlayerId    int64              `json:"playerId"`
	PlayerImage string             `json:"playerImage"`
}

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

type ReplayPageData struct {
	Json     string
	UserJson string
	Json2    string
}

type bulkMTGOExport struct {
	PlayerID int64
	Username string
	Deck     string
}

type NameAndQuantity struct {
	Name     string
	Quantity int64
}

type JsonError struct {
	Error string `json:"error"`
}

type DraftList struct {
	Drafts []DraftListEntry `json:"drafts"`
}

type DraftListEntry struct {
	Id             int64  `json:"id"`
	Name           string `json:"name"`
	AvailableSeats int64  `json:"availableSeats"`
	Status         string `json:"status"`
}

type PostedPick struct {
	CardIds []int64 `json:"cards"`
}

type UserInfo struct {
	Name    string `json:"name"`
	Picture string `json:"picture"`
	Id      int64  `json:"userId"`
}

type r38handler func(w http.ResponseWriter, r *http.Request, userId int64)
type viewingFunc func(r *http.Request, userId int64) (bool, error)

var secret_key_no_one_will_ever_guess = []byte(os.Getenv("SESSION_SECRET"))
var store = sessions.NewCookieStore(secret_key_no_one_will_ever_guess)
var database *sql.DB
var useAuth bool
var IsViewing viewingFunc
var sock string

func main() {
	useAuthPtr := flag.Bool("auth", true, "bool")
	flag.Parse()

	useAuth = *useAuthPtr

	if useAuth {
		IsViewing = AuthIsViewing
	} else {
		IsViewing = NonAuthIsViewing
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
			var userId int64
			if useAuth {
				session, err := store.Get(r, "session-name")
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				userIdInt, err := strconv.Atoi(session.Values["userid"].(string))
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				userId = int64(userIdInt)
			} else {
				userId = 1
			}

			/*
				if userId != 1 {
					http.Error(w, "server down for maintaince", http.StatusInternalServerError)
					return
				}
			*/

			if userId == 1 {
				q := r.URL.Query()
				val := q.Get("as")
				if val != "" {
					userIdInt, err := strconv.Atoi(val)
					if err != nil {
						http.Error(w, err.Error(), http.StatusInternalServerError)
						return
					}
					userId = int64(userIdInt)
				}
			}

			serveFunc(w, r, userId)
		})
		mux.Handle(route, middleware(handler))
	}

	addHandler("/proxy/", ServeProxy)
	addHandler("/replay/", ServeReplay)
	addHandler("/deckbuilder/", ServeDeckbuilder)
	addHandler("/librarian/", ServeLibrarian)
	addHandler("/power/", ServePower)
	addHandler("/draft/", ServeDraft)
	addHandler("/pdf/", ServePdf)
	addHandler("/pick/", ServePick)
	addHandler("/join/", ServeJoin)
	addHandler("/mtgo/", ServeMtgo)
	addHandler("/bulk_mtgo/", ServeBulkMTGO)
	addHandler("/index/", ServeIndex)

	addHandler("/api/draft/", ServeApiDraft)
	addHandler("/api/draftlist/", ServeApiDraftList)
	addHandler("/api/pick/", ServeApiPick)

	addHandler("/", ServeIndex)

	return mux
}

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

func NonAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/proxy/") {
			log.Printf(r.URL.Path)
		}
		next.ServeHTTP(w, r)
		return
	})
}

func AuthIsViewing(r *http.Request, userId int64) (bool, error) {
	session, err := store.Get(r, "session-name")
	if err != nil {
		return false, err
	}
	realUserIdInt, err := strconv.Atoi(session.Values["userid"].(string))
	if err != nil {
		return false, err
	}
	realUserId := int64(realUserIdInt)

	return userId != realUserId, nil
}

func NonAuthIsViewing(r *http.Request, userId int64) (bool, error) {
	return userId != 1, nil
}

func GetViewParam(r *http.Request, userId int64) string {
	param := ""
	viewing, err := IsViewing(r, userId)
	if err == nil && viewing {
		param = fmt.Sprintf("?as=%d", userId)
	}
	return param
}

func ServeApiDraft(w http.ResponseWriter, r *http.Request, userID int64) {
	re := regexp.MustCompile(`/api/draft/(\d+)`)
	parseResult := re.FindStringSubmatch(r.URL.Path)
	if parseResult == nil {
		json.NewEncoder(w).Encode(JsonError{Error: "bad api url"})
		return
	}
	draftID, err := strconv.ParseInt(parseResult[1], 10, 64)
	if err != nil {
		json.NewEncoder(w).Encode(JsonError{Error: fmt.Sprintf("bad api url: %s", err.Error())})
		return
	}

	draftJson, err := GetFilteredJson(draftID, userID)
	if err != nil {
		json.NewEncoder(w).Encode(JsonError{Error: fmt.Sprintf("error getting json: %s", err.Error())})
		return
	}

	fmt.Fprint(w, draftJson)
}

func ServeApiDraftList(w http.ResponseWriter, r *http.Request, userID int64) {
	query := `select drafts.id, drafts.name, sum(seats.user is null and seats.position is not null) as empty_seats, coalesce(sum(seats.user = ?), 0) as joined from drafts left join seats on drafts.id = seats.draft group by drafts.id`

	rows, err := database.Query(query, userID)
	if err != nil {
		json.NewEncoder(w).Encode(JsonError{Error: fmt.Sprintf("can't get draft list: %s", err.Error())})
		return
	}
	defer rows.Close()
	var drafts DraftList
	for rows.Next() {
		var d DraftListEntry
		var joined int64
		err = rows.Scan(&d.Id, &d.Name, &d.AvailableSeats, &joined)
		if err != nil {
			json.NewEncoder(w).Encode(JsonError{Error: fmt.Sprintf("can't get draft list: %s", err.Error())})
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

func ServeApiPick(w http.ResponseWriter, r *http.Request, userID int64) {
	if r.Method != "POST" {
		http.Error(w, "invalid request method", http.StatusMethodNotAllowed)
		return
	}

	bodyBytes, err := ioutil.ReadAll(r.Body)
	if err != nil {
		json.NewEncoder(w).Encode(JsonError{Error: fmt.Sprintf("error reading post body: %s", err.Error())})
		return
	}
	var pick PostedPick
	err = json.Unmarshal(bodyBytes, &pick)
	if err != nil {
		json.NewEncoder(w).Encode(JsonError{Error: fmt.Sprintf("error parsing post body: %s", err.Error())})
		return
	}
	var draftID int64
	if len(pick.CardIds) == 1 {
		draftID, _, announcements, round, err := doPick(userID, pick.CardIds[0], true)
		if err != nil {
			json.NewEncoder(w).Encode(JsonError{Error: fmt.Sprintf("error making pick: %s", err.Error())})
			return
		}

		err = DoEvent(draftID, userID, announcements, pick.CardIds[0], sql.NullInt64{Valid: false}, round)
		if err != nil {
			json.NewEncoder(w).Encode(JsonError{Error: fmt.Sprintf("error recording event: %s", err.Error())})
			return
		}
	} else if len(pick.CardIds) == 2 {
		json.NewEncoder(w).Encode(JsonError{Error: "cogwork librarian power not implemented yet"})
		return
	} else {
		json.NewEncoder(w).Encode(JsonError{Error: fmt.Sprintf("invalid number of cards: %d", len(pick.CardIds))})
		return
	}

	draftJson, err := GetFilteredJson(draftID, userID)
	if err != nil {
		json.NewEncoder(w).Encode(JsonError{Error: fmt.Sprintf("error getting json: %s", err.Error())})
		return
	}

	fmt.Fprint(w, draftJson)
}

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
	exports := []bulkMTGOExport{}
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
		exports = append(exports, bulkMTGOExport{PlayerID: playerID, Username: re.ReplaceAllString(username, "_"), Deck: export})
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

func ServeMtgo(w http.ResponseWriter, r *http.Request, userId int64) {
	re := regexp.MustCompile(`/mtgo/(\d+)`)
	parseResult := re.FindStringSubmatch(r.URL.Path)

	if parseResult == nil {
		http.Error(w, "bad url", http.StatusInternalServerError)
		return
	}

	draftIdInt, err := strconv.Atoi(parseResult[1])
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	draftId := int64(draftIdInt)

	doServeMtgo(w, r, userId, draftId)
}

func doServeMtgo(w http.ResponseWriter, r *http.Request, userId int64, draftId int64) {
	export, err := exportToMTGO(userId, draftId)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Disposition", "attachment; filename=r38export.dek")
	w.Header().Set("Content-Type", r.Header.Get("Content-Type"))
	io.WriteString(w, export)
}

// createZipExport creates a .zip file containing decks from a bulk MTGO export.
func createZipExport(exports []bulkMTGOExport) ([]byte, error) {
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
func exportToMTGO(userId int64, draftId int64) (string, error) {
	_, picks, _, err := getPackPicksAndPowers(draftId, userId)
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
func ServeProxy(w http.ResponseWriter, r *http.Request, userId int64) {
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

func ServeReplay(w http.ResponseWriter, r *http.Request, userId int64) {
	re := regexp.MustCompile(`/replay/(\d+)`)
	parseResult := re.FindStringSubmatch(r.URL.Path)

	ServeVueApp(parseResult, w, userId)
}

func ServeDeckbuilder(w http.ResponseWriter, r *http.Request, userId int64) {
	re := regexp.MustCompile(`/deckbuilder/(\d+)`)
	parseResult := re.FindStringSubmatch(r.URL.Path)

	ServeVueApp(parseResult, w, userId)
}

func ServeVueApp(parseResult []string, w http.ResponseWriter, userId int64) {
	if parseResult == nil {
		http.Error(w, "bad url", http.StatusInternalServerError)
		return
	}

	draftIdInt, err := strconv.Atoi(parseResult[1])
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	draftId := int64(draftIdInt)

	canViewReplay, err := CanViewReplay(draftId, userId)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if !canViewReplay {
		http.Error(w, "lol no", http.StatusInternalServerError)
		return
	}

	draftJson, err := GetJson(draftId)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	query := `select id,discord_name,picture from users where id=?`
	row := database.QueryRow(query, userId)
	var userInfo UserInfo
	err = row.Scan(&userInfo.Id, &userInfo.Name, &userInfo.Picture)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	userInfoJson, err := json.Marshal(userInfo)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	draftJson2, err := GetJson2(draftId)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := ReplayPageData{Json: draftJson, UserJson: string(userInfoJson), Json2: draftJson2}

	t := template.Must(template.ParseFiles("replay.tmpl"))

	t.Execute(w, data)
}

func CanViewReplay(draftId int64, userId int64) (bool, error) {
	query := `select min(round) from seats where draft=?`
	row := database.QueryRow(query, draftId)
	var round int64
	err := row.Scan(&round)
	if err != nil {
		return false, err
	}

	if round != 4 && userId != 1 && draftId != 9 {
		query = `select user from seats where draft=? and position is not null`
		rows, err := database.Query(query, draftId)
		if err != nil {
			return false, err
		}
		defer rows.Close()

		valid := true
		for rows.Next() {
			var playerId sql.NullInt64
			err = rows.Scan(&playerId)
			if err != nil {
				return false, err
			}
			if !playerId.Valid || playerId.Int64 == userId {
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

func ServeLibrarian(w http.ResponseWriter, r *http.Request, userId int64) {
	re := regexp.MustCompile(`/librarian/(\d+)/(\d+)`)
	parseResult := re.FindStringSubmatch(r.URL.Path)

	if parseResult == nil {
		http.Error(w, "bad url", http.StatusInternalServerError)
		return
	}

	cardId1Int, err := strconv.Atoi(parseResult[1])
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	cardId2Int, err := strconv.Atoi(parseResult[2])
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	cardId1 := int64(cardId1Int)
	cardId2 := int64(cardId2Int)

	query := `select seats.draft from cards join packs join seats where cards.pack=packs.id and packs.seat=seats.id and cards.id IN (?,?)`

	rows, err := database.Query(query, cardId1, cardId2)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	var checkId int64
	checkId = 0
	rowCount := 0
	for rows.Next() {
		rowCount++
		var checkId2 int64
		err = rows.Scan(&checkId2)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if checkId == 0 {
			checkId = checkId2
		} else if checkId != checkId2 {
			http.Error(w, "woah there!", http.StatusInternalServerError)
			return
		}
	}

	if checkId == 0 || rowCount != 2 {
		http.Error(w, "woah there", http.StatusInternalServerError)
		return
	}

	draftId1, packId1, announcements1, round1, err := doPick(userId, cardId1, false)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	draftId2, packId2, announcements2, round2, err := doPick(userId, cardId2, true)

	if packId1 != packId2 {
		http.Error(w, "pack ids somehow don't match.", http.StatusInternalServerError)
		return
	}

	if draftId1 != draftId2 {
		http.Error(w, "draft ids somehow don't match.", http.StatusInternalServerError)
		return
	}

	if round1 != round2 {
		http.Error(w, "rounds somehow don't match.", http.StatusInternalServerError)
		return
	}

	query = `select position from seats where draft=? and user=?`
	row := database.QueryRow(query, draftId1, userId)
	var position int64
	err = row.Scan(&position)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	announcements := append(announcements1, fmt.Sprintf("Seat %d used Cogwork Librarian's ability", position))
	announcements = append(announcements, announcements2...)

	query = `select cards.id from cards join packs join seats where cards.pack=packs.id and packs.seat=seats.id and seats.draft=? and cards.name="Cogwork Librarian" and seats.user=?`

	row = database.QueryRow(query, draftId1, userId)
	var librarianId int64
	err = row.Scan(&librarianId)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	query = `update cards set pack=?, faceup=false where id=?`
	log.Printf("%s\t%d,%d", query, packId1, librarianId)
	_, err = database.Exec(query, packId1, librarianId)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = DoEvent(draftId1, userId, announcements, cardId1, sql.NullInt64{Int64: cardId2, Valid: true}, round1)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/draft/%d", draftId1), http.StatusTemporaryRedirect)
}

func ServePower(w http.ResponseWriter, r *http.Request, userId int64) {
	re := regexp.MustCompile(`/power/(\d+)`)
	parseResult := re.FindStringSubmatch(r.URL.Path)

	if parseResult == nil {
		http.Error(w, "bad url", http.StatusInternalServerError)
		return
	}

	cardIdInt, err := strconv.Atoi(parseResult[1])
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	cardId := int64(cardIdInt)

	query := `select cards.name, seats.draft from cards join packs join seats where cards.pack=packs.id and packs.seat=seats.id and seats.user=? and cards.id=? and cards.faceup=true`

	row := database.QueryRow(query, userId, cardId)
	var cardName string
	var draftId int64
	err = row.Scan(&cardName, &draftId)
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
		myPack, myPicks, _, err := getPackPicksAndPowers(draftId, userId)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if len(myPack) < 2 {
			http.Redirect(w, r, fmt.Sprintf("/draft/%d", draftId), http.StatusTemporaryRedirect)
			return
		}

		t := template.Must(template.ParseFiles("librarian.tmpl"))

		data := DraftPageData{Pack: myPack, Picks: myPicks, DraftId: draftId}
		t.Execute(w, data)
		// use some js to construct the url /librarian/cogworklibrarianid/pick1id/pick2id
		// redirect to the draft url
	}
}

func ServeIndex(w http.ResponseWriter, r *http.Request, userId int64) {
	query := `select drafts.id, drafts.name, sum(seats.user is null and seats.position is not null) as empty_seats, coalesce(sum(seats.user = ?), 0) as joined from drafts left join seats on drafts.id = seats.draft group by drafts.id`

	rows, err := database.Query(query, userId)
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
		d.Replayable, err = CanViewReplay(d.ID, userId)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		Drafts = append(Drafts, d)
	}

	viewParam := GetViewParam(r, userId)
	data := IndexPageData{Drafts: Drafts, ViewUrl: viewParam, UserId: userId}
	t := template.Must(template.ParseFiles("index.tmpl"))
	t.Execute(w, data)
}

func ServePdf(w http.ResponseWriter, r *http.Request, userId int64) {
	re := regexp.MustCompile(`/pdf/(\d+)`)
	parseResult := re.FindStringSubmatch(r.URL.Path)

	if parseResult == nil {
		http.Error(w, "bad url", http.StatusInternalServerError)
		return
	}

	draftIdInt, err := strconv.Atoi(parseResult[1])
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	draftId := int64(draftIdInt)

	doServePdf(w, r, userId, draftId)
}

func doServePdf(w http.ResponseWriter, r *http.Request, userId int64, draftId int64) {
	_, myPicks, _, err := getPackPicksAndPowers(draftId, userId)
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

func ServeDraft(w http.ResponseWriter, r *http.Request, userId int64) {
	re := regexp.MustCompile(`/draft/(\d+)`)
	parseResult := re.FindStringSubmatch(r.URL.Path)

	if parseResult == nil {
		http.Error(w, "bad url", http.StatusInternalServerError)
		return
	}

	draftIdInt, err := strconv.Atoi(parseResult[1])
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	draftId := int64(draftIdInt)

	myPack, myPicks, powers2, err := getPackPicksAndPowers(draftId, userId)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	query := `select seats.position, drafts.name from seats join drafts where seats.draft=? and seats.user=? and seats.draft=drafts.id`
	row := database.QueryRow(query, draftId, userId)
	var position int64
	var draftName string
	err = row.Scan(&position, &draftName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	query = `select message from revealed where draft=?`
	rows, err := database.Query(query, draftId)
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
	viewParam := GetViewParam(r, userId)
	t := template.Must(template.ParseFiles("draft.tmpl"))

	data := DraftPageData{Picks: myPicks, Pack: myPack, DraftId: draftId, DraftName: draftName, Powers: powers2, Position: position, Revealed: revealed, ViewUrl: viewParam}

	t.Execute(w, data)
}

func ServeJoin(w http.ResponseWriter, r *http.Request, userId int64) {
	if userId == 19 {
		http.Error(w, "bad feinberg", http.StatusInternalServerError)
		return
	}

	re := regexp.MustCompile(`/join/(\d+)`)
	parseResult := re.FindStringSubmatch(r.URL.Path)

	if parseResult == nil {
		http.Error(w, "bad url", http.StatusInternalServerError)
		return
	}

	draftIdInt, err := strconv.Atoi(parseResult[1])
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	draftId := int64(draftIdInt)

	query := `select true from seats where draft=? and user=?`

	row := database.QueryRow(query, draftId, userId)
	var alreadyJoined bool
	err = row.Scan(&alreadyJoined)

	if err != nil && err != sql.ErrNoRows {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	} else if err != sql.ErrNoRows || alreadyJoined {
		http.Redirect(w, r, fmt.Sprintf("/draft/%s", draftId), http.StatusTemporaryRedirect)
		return
	}

	query = `update seats set user=? where id=(select id from seats where draft=? and user is null and position is not null order by random() limit 1)`
	log.Printf("%s\t%d,%d", query, userId, draftId)

	_, err = database.Exec(query, userId, draftId)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	viewParam := GetViewParam(r, userId)
	http.Redirect(w, r, fmt.Sprintf("/draft/%d%s", draftId, viewParam), http.StatusTemporaryRedirect)
}

func ServePick(w http.ResponseWriter, r *http.Request, userId int64) {
	re := regexp.MustCompile(`/pick/(\d+)`)
	parseResult := re.FindStringSubmatch(r.URL.Path)

	if parseResult == nil {
		http.Error(w, "bad url", http.StatusInternalServerError)
		return
	}

	cardIdInt, err := strconv.Atoi(parseResult[1])
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	cardId := int64(cardIdInt)
	draftId, _, announcements, round, err := doPick(userId, cardId, true)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = DoEvent(draftId, userId, announcements, cardId, sql.NullInt64{Valid: false}, round)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	viewParam := GetViewParam(r, userId)
	http.Redirect(w, r, fmt.Sprintf("/draft/%d%s", draftId, viewParam), http.StatusTemporaryRedirect)
}

func getPackPicksAndPowers(draftId int64, userId int64) ([]Card, []Card, []Card, error) {
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

	rows, err := database.Query(query, draftId, userId, draftId, userId)
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
			myPicks = append(myPicks, Card{Name: name, Tags: tags, Number: number, Edition: edition, Id: id, Mtgo: mtgoString})
			if faceup == true {
				switch name {
				case "Cogwork Librarian":
					powers = append(powers, Card{Name: name, Tags: tags, Number: number, Edition: edition, Id: id, Mtgo: mtgoString})
				}
			}
		} else {
			myPack = append(myPack, Card{Name: name, Tags: tags, Number: number, Edition: edition, Id: id, Mtgo: mtgoString})
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

func doPick(userId int64, cardId int64, pass bool) (int64, int64, []string, int64, error) {
	announcements := []string{}

	query := `select drafts.id, seats.round, seats.position, cards.name from drafts join seats join packs join cards where drafts.id=seats.draft and seats.id=packs.seat and packs.id=cards.pack and cards.id=? and packs.round <> 0`

	row := database.QueryRow(query, cardId)
	var draftId int64
	var round int64
	var position int64
	var cardName string
	err := row.Scan(&draftId, &round, &position, &cardName)
	if err != nil {
		return -1, -1, announcements, -1, err
	}

	query = `select packs.id, packs.modified from drafts join seats join packs join cards where cards.id=? and drafts.id=? and drafts.id=seats.draft and seats.id=packs.seat and packs.id=cards.pack and seats.user=? and (packs.round=0 or (packs.round=seats.round and packs.modified in (select min(packs.modified) from packs join seats join drafts where seats.draft=? and seats.user=? and seats.id=packs.seat and drafts.id=seats.draft and packs.round=seats.round)))`

	row = database.QueryRow(query, cardId, draftId, userId, draftId, userId)
	var oldPackId int64
	var modified int64
	err = row.Scan(&oldPackId, &modified)
	if err != nil {
		return draftId, -1, announcements, round, err
	}

	query = `select packs.id, seats.id from packs join seats where seats.user=? and seats.id=packs.seat and packs.round=0 and seats.draft=?`

	row = database.QueryRow(query, userId, draftId)
	var pickId int64
	var seatId int64
	err = row.Scan(&pickId, &seatId)
	if err != nil {
		return draftId, oldPackId, announcements, round, err
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
	row = database.QueryRow(query, draftId, newPosition)
	var newPositionId int64
	err = row.Scan(&newPositionId)
	if err != nil {
		return draftId, oldPackId, announcements, round, err
	}

	var cardModified int64
	cardModified = 0
	query = `select max(cards.modified) from cards join packs on cards.pack=packs.id join seats on seats.id=packs.seat where seats.draft=? and seats.user=?`
	row = database.QueryRow(query, draftId, userId)

	err = row.Scan(&cardModified)

	if err != nil {
		cardModified = 0
	} else {
		cardModified += 1
	}

	if pass {
		query = `begin transaction;update cards set pack=?, modified=? where id=?;update packs set seat=?, modified=modified+10 where id=?;commit`
		log.Printf("%s\t%d,%d,%d,%d,%d", query, pickId, cardModified, cardId, newPositionId, oldPackId)

		_, err = database.Exec(query, pickId, cardModified, cardId, newPositionId, oldPackId)
		if err != nil {
			return draftId, oldPackId, announcements, round, err
		}

		query = `select count(1) from v_packs join seats where v_packs.seat=seats.id and seats.user=? and v_packs.round=? and v_packs.count>0 and seats.draft=?`
		row = database.QueryRow(query, userId, round, draftId)
		var packsLeftInSeat int64
		err = row.Scan(&packsLeftInSeat)
		if err != nil {
			return draftId, oldPackId, announcements, round, err
		}

		if packsLeftInSeat == 0 {
			query = `select count(1) from seats a join seats b on a.draft=b.draft where a.user=? and b.position=? and a.draft=? and a.round=b.round`
			row = database.QueryRow(query, userId, newPosition, draftId)
			var roundsMatch int64
			err = row.Scan(&roundsMatch)
			if err != nil {
				log.Printf("cannot determine if rounds match for notify")
			} else if roundsMatch == 1 {
				err = NotifyByDraftAndPosition(draftId, newPosition)
				if err != nil {
					log.Printf("error with notify")
				}
			}

			// WARNING: if you ever have a draft with anything other than 8 players, 15 cards per pack, and 3 rounds, this will break horribly.
			query = `update seats set round=round+1 where user=? and draft=? and (select count(1) in (15, 30, 45) from packs join cards on cards.pack=packs.id join seats on seats.id=packs.seat where seats.draft=? and packs.round=0 and seats.user=?)`
			log.Printf("%s\t%d,%d,%d,%d", query, userId, draftId, draftId, userId)
			_, err = database.Exec(query, userId, draftId, draftId, userId)
			if err != nil {
				log.Printf("error with possibly updating round")
			}

			if roundsMatch == 0 {
				query = `select count(1) from seats where draft=? group by round order by round desc limit 1`

				row = database.QueryRow(query, draftId)
				var nextRoundPlayers int64
				err = row.Scan(&nextRoundPlayers)
				if err != nil {
					log.Printf("error counting players and rounds")
				} else if nextRoundPlayers > 1 {
					query = `select seats.position from seats left join packs on seats.id=packs.seat join v_packs on v_packs.id=packs.id where v_packs.count>0 and packs.round=seats.round and seats.draft=? group by seats.id`

					rows, err := database.Query(query, draftId)
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
							err = NotifyByDraftAndPosition(draftId, blockingPosition)
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
		log.Printf("%s\t%d,%s,%d,%d", query, pickId, cardModified, cardId)

		_, err = database.Exec(query, pickId, cardModified, cardId)
		if err != nil {
			return draftId, oldPackId, announcements, round, err
		}
	}

	switch cardName {
	case "Cogwork Librarian":
		query = `update cards set faceup=true where id=?`
		log.Printf("%s\t%d", query, cardId)
		_, err = database.Exec(query, cardId)
		if err != nil {
			return draftId, oldPackId, announcements, round, err
		}

		query = `INSERT INTO revealed (draft, message) VALUES (?, "Seat " || ? || " revealed Cogwork Librarian.")`
		log.Printf("%s\t%d,%d", query, draftId, position)
		_, err = database.Exec(query, draftId, position)
		if err != nil {
			return draftId, oldPackId, announcements, round, err
		}

		announcements = append(announcements, fmt.Sprintf("Seat %d revealed Cogwork Librarian", position))
	}

	return draftId, oldPackId, announcements, round, nil
}

func NotifyByDraftAndPosition(draftId int64, position int64) error {
	log.Printf("Attempting to notify %d %d", draftId, position)

	query := `select users.discord_id from users join seats where users.id=seats.user and seats.draft=? and seats.position=?`

	row := database.QueryRow(query, draftId, position)
	var discordId string
	err := row.Scan(&discordId)
	if err != nil {
		return err
	}

	var jsonStr = []byte(fmt.Sprintf(`{"content": "<@%s> you have new picks <http://draft.thefoley.net/draft/%d>"}`, discordId, draftId))
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

func GetJsonObject(draftId int64) (DraftJson, error) {
	var draft DraftJson

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

	rows, err := database.Query(query, draftId)
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
		var nullableDiscordId sql.NullString
		var nullableCmc sql.NullInt64
		var nullableType sql.NullString
		var nullableColor sql.NullString
		var nullableMtgo sql.NullString
		var draftUserId sql.NullInt64
		var nullableData sql.NullString
		err = rows.Scan(&draft.Name, &nullablePosition, &packSeat, &nullableRound, &card.Name, &card.Edition, &card.Number, &card.Tags, &nullableDiscordId, &nullableCmc, &nullableType, &nullableColor, &nullableMtgo, &card.Id, &draftUserId, &nullableData, &draft.Id)
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
			dataObj["r38_data"].(map[string]interface{})["id"] = card.Id
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
		draft.Seats[position].Name = nullableDiscordId.String
		draft.Seats[position].Id = draftUserId.Int64
	}

	query = `select seats.position, events.announcement, cards1.name, cards2.name, events.id, events.modified, events.round, cards1.id, cards2.id from events join seats on events.draft=seats.draft and events.user=seats.user left join cards as cards1 on events.card1=cards1.id left join cards as cards2 on events.card2=cards2.id where events.draft=?`
	rows, err = database.Query(query, draftId)
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

func GetJsonObject2(draftId int64) (DraftJson2, error) {
	var draft DraftJson2

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

	rows, err := database.Query(query, draftId)
	if err != nil {
		return draft, err
	}
	defer rows.Close()
	var indices [8][3]int64
	for rows.Next() {
		var position int64
		var packRound int64
		var cardId int64
		var nullableDiscordId sql.NullString
		var draftUserId sql.NullInt64
		var cardData string
		var nullablePicture sql.NullString
		err = rows.Scan(&draft.DraftId, &draft.DraftName, &position, &packRound, &nullableDiscordId, &cardId, &draftUserId, &cardData, &nullablePicture)
		if err != nil {
			return draft, err
		}

		dataObj := make(map[string]interface{})
		err = json.Unmarshal([]byte(cardData), &dataObj)
		if err != nil {
			log.Printf("making nil card data because of error %s", err.Error())
			dataObj = nil
		}
		dataObj["r38_data"].(map[string]interface{})["id"] = cardId

		packRound--

		nextIndex := indices[position][packRound]

		draft.Seats[position].Packs[packRound][nextIndex] = dataObj
		draft.Seats[position].PlayerName = nullableDiscordId.String
		draft.Seats[position].PlayerId = draftUserId.Int64
		draft.Seats[position].PlayerImage = nullablePicture.String

		indices[position][packRound]++
	}

	query = `select
                   seats.position,
                   events.announcement,
                   cards1.name,
                   cards2.name,
                   events.id,
                   events.modified,
                   events.round,
                   cards1.id,
                   cards2.id
                 from events
                 join seats on events.draft=seats.draft and events.user=seats.user
                 left join cards as cards1 on events.card1=cards1.id
                 left join cards as cards2 on events.card2=cards2.id
                 where events.draft=?`
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
	rows, err = database.Query(query, draftId)
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

func GetJson(draftId int64) (string, error) {
	draft, err := GetJsonObject(draftId)
	if err != nil {
		return "", err
	}
	ret, err := json.Marshal(draft)
	if err != nil {
		return "", err
	}

	return string(ret), nil
}

func GetJson2(draftId int64) (string, error) {
	draft, err := GetJsonObject2(draftId)
	if err != nil {
		return "", err
	}
	ret, err := json.Marshal(draft)
	if err != nil {
		return "", err
	}

	return string(ret), nil
}

func GetFilteredJson(draftId int64, userId int64) (string, error) {
	draft, err := GetJsonObject2(draftId)
	if err != nil {
		return "", err
	}

	query := `select (select round from seats where draft=? and user=?), (select count(1) from seats where draft=? and user is null)`
	var myRound sql.NullInt64
	var emptySeats int64
	row := database.QueryRow(query, draftId, userId, draftId)
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

	ret, err := json.Marshal(Perspective{User: userId, Draft: draft})
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

func DoEvent(draftId int64, userId int64, announcements []string, cardId1 int64, cardId2 sql.NullInt64, round int64) error {
	var query string
	var err error
	if cardId2.Valid {
		query = `insert into events (round, draft, user, announcement, card1, card2, modified) VALUES (?, ?, ?, ?, ?, ?, (select count(1) from seats join packs join cards where seats.user=? and seats.draft=? and packs.seat=seats.id and cards.pack=packs.id and packs.round=0))`
		log.Printf("%s\t%d,%d,%d,%s,%d,%d,%d,%d", query, round, draftId, userId, strings.Join(announcements, "\n"), cardId1, cardId2.Int64, userId, draftId)
		_, err = database.Exec(query, round, draftId, userId, strings.Join(announcements, "\n"), cardId1, cardId2.Int64, userId, draftId)
	} else {
		query := `insert into events (round, draft, user, announcement, card1, card2, modified) VALUES (?, ?, ?, ?, ?, null, (select count(1) from seats join packs join cards where seats.user=? and seats.draft=? and packs.seat=seats.id and cards.pack=packs.id and packs.round=0))`
		log.Printf("%s\t%d,%d,%d,%s,%d,%d,%d", query, round, draftId, userId, strings.Join(announcements, "\n"), cardId1, userId, draftId)
		_, err = database.Exec(query, round, draftId, userId, strings.Join(announcements, "\n"), cardId1, userId, draftId)
	}

	return err
}
