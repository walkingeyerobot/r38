package main

import (
	"bufio"
	"bytes"
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/binary"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gorilla/sessions"
	"github.com/jung-kurt/gofpdf"
	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"html/template"
	"io/ioutil"
	"log"
	badrand "math/rand"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var secret_key_no_one_will_ever_guess = []byte(os.Getenv("SESSION_SECRET"))
var store = sessions.NewCookieStore(secret_key_no_one_will_ever_guess)
var database *sql.DB

type GoogleUserInfo struct {
	Id      string `json:"id"`
	Email   string `json:"email"`
	Picture string `json:"picture"`
}

type cryptoSource struct{}

func (s cryptoSource) Seed(seed int64) {}

func (s cryptoSource) Int63() int64 {
	return int64(s.Uint64() & ^uint64(1<<63))
}

func (s cryptoSource) Uint64() (v uint64) {
	err := binary.Read(rand.Reader, binary.BigEndian, &v)
	if err != nil {
		log.Fatal(err)
	}
	return v
}

type Draft struct {
	Name     string
	Id       int64
	Seats    int64
	Joined   bool
	Joinable bool
}

type IndexPageData struct {
	Drafts []Draft
}

type DraftPageData struct {
	DraftId   int64
	DraftName string
	Picks     []Card
	Pack      []Card
	Powers    []Card
	Position  int64
	Revealed  []string
}

type Card struct {
	Id      int64  `json:"-"`
	Name    string `json:"name"`
	Tags    string `json:"tags"`
	Number  string `json:"number"`
	Edition string `json:"edition"`
}

type Seat struct {
	Rounds []Round `json:"rounds"`
}

type Round struct {
	Packs []Pack `json:"packs"`
	Round int64  `json:"round"`
}

type Pack struct {
	Cards []Card `json:"cards"`
}

type DraftJson struct {
	Seats     []Seat       `json:"seats"`
	Name      string       `json:"name"`
	ExtraPack []Card       `json:"extraPack"`
	Events    []DraftEvent `json:"events"`
}

type DraftEvent struct {
	Player         int64    `json:"player"`
	Announcements  []string `json:"announcements"`
	Card1          string   `json:"card1"`
	Card2          string   `json:"card2"`
	PlayerModified int64    `json:"playerModified"`
	DraftModified  int64    `json:"draftModified"`
	Round          int64    `json:"round"`
}

type ReplayPageData struct {
	Json string
}

func main() {
	var err error
	database, err = sql.Open("sqlite3", "draft.db")
	if err != nil {
		return
	}
	err = database.Ping()
	if err != nil {
		return
	}

	// MakeDraft("test draft v2.0")

	server := &http.Server{
		Addr:    fmt.Sprintf(":12264"),
		Handler: NewHandler(),
	}

	log.Printf("Starting HTTP Server. Listening at %q", server.Addr)
	err = server.ListenAndServe()
	if err != nil {
		log.Printf("%v", err)
	}
}

func NewHandler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/auth/google/login", oauthGoogleLogin)
	mux.HandleFunc("/auth/google/callback", oauthGoogleCallback)

	fs := http.FileServer(http.Dir("static"))
	mux.Handle("/static/", AuthMiddleware(http.StripPrefix("/static/", fs)))

	replayHandler := http.HandlerFunc(ServeReplay)
	mux.Handle("/replay/", AuthMiddleware(replayHandler))
	viewHandler := http.HandlerFunc(ServeView)
	mux.Handle("/view/", AuthMiddleware(viewHandler))
	librarianHandler := http.HandlerFunc(ServeLibrarian)
	mux.Handle("/librarian/", AuthMiddleware(librarianHandler))
	powerHandler := http.HandlerFunc(ServePower)
	mux.Handle("/power/", AuthMiddleware(powerHandler))
	draftHandler := http.HandlerFunc(ServeDraft)
	mux.Handle("/draft/", AuthMiddleware(draftHandler))
	pdfHandler := http.HandlerFunc(ServePdf)
	mux.Handle("/pdf/", AuthMiddleware(pdfHandler))
	pickHandler := http.HandlerFunc(ServePick)
	mux.Handle("/pick/", AuthMiddleware(pickHandler))
	joinHandler := http.HandlerFunc(ServeJoin)
	mux.Handle("/join/", AuthMiddleware(joinHandler))
	indexHandler := http.HandlerFunc(ServeIndex)
	mux.Handle("/", AuthMiddleware(indexHandler))

	return mux
}

func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session, err := store.Get(r, "session-name")
		if err != nil {
			fmt.Fprintf(w, `<html><body><a href="/auth/google/login">login</a></body></html>`)
			return
		}
		if session.Values["userid"] != nil {
			log.Printf("%s %s", session.Values["userid"], r.URL.Path)
			next.ServeHTTP(w, r)
			return
		} else {
			fmt.Fprintf(w, `<html><body><a href="/auth/google/login">login</a></body></html>`)
			return
		}
	})
}

func ServeReplay(w http.ResponseWriter, r *http.Request) {
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
	userId := int64(userIdInt)

	if userId != 1 {
		http.Error(w, "lol no", http.StatusInternalServerError)
		return
	}

	re := regexp.MustCompile(`/replay/(\d+)`)
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

	json, err := GetJson(draftId)

	t := template.Must(template.ParseFiles("replay.tmpl"))

	data := ReplayPageData{Json: json}

	t.Execute(w, data)
}
func ServeLibrarian(w http.ResponseWriter, r *http.Request) {
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
	userId := int64(userIdInt)

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

func ServePower(w http.ResponseWriter, r *http.Request) {
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
	userId := int64(userIdInt)

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

func ServeIndex(w http.ResponseWriter, r *http.Request) {
	session, err := store.Get(r, "session-name")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	userId := session.Values["userid"]

	query := `select drafts.id, drafts.name, sum(seats.user is null and seats.position is not null) as empty_seats, coalesce(sum(seats.user = ?), 0) as joined from drafts left join seats on drafts.id = seats.draft group by drafts.id`

	rows, err := database.Query(query, userId, userId)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	var Drafts []Draft
	for rows.Next() {
		var d Draft
		err = rows.Scan(&d.Id, &d.Name, &d.Seats, &d.Joined)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		d.Joinable = d.Seats > 0 && !d.Joined
		Drafts = append(Drafts, d)
	}

	t := template.Must(template.ParseFiles("index.tmpl"))

	data := IndexPageData{Drafts: Drafts}
	t.Execute(w, data)
}

func ServePdf(w http.ResponseWriter, r *http.Request) {
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
	userId := int64(userIdInt)

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

func ServeDraft(w http.ResponseWriter, r *http.Request) {
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
	userId := int64(userIdInt)

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

	doServeDraft(w, r, userId, draftId)
}

func ServeView(w http.ResponseWriter, r *http.Request) {
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
	userId := int64(userIdInt)

	if userId != 1 {
		http.Error(w, "lol no", http.StatusInternalServerError)
		return
	}

	re := regexp.MustCompile(`/view/(\d+)`)
	parseResult := re.FindStringSubmatch(r.URL.Path)

	if parseResult == nil {
		http.Error(w, "bad url", http.StatusInternalServerError)
		return
	}

	positionInt, err := strconv.Atoi(parseResult[1])
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	position := int64(positionInt)

	query := `select user from seats where position=? and draft=8`
	row := database.QueryRow(query, position)
	var user2Id int64
	err = row.Scan(&user2Id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	doServeDraft(w, r, user2Id, 8)
}

func doServeDraft(w http.ResponseWriter, r *http.Request, userId int64, draftId int64) {
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

	t := template.Must(template.ParseFiles("draft.tmpl"))

	data := DraftPageData{Picks: myPicks, Pack: myPack, DraftId: draftId, DraftName: draftName, Powers: powers2, Position: position, Revealed: revealed}
	t.Execute(w, data)
}

func ServeJoin(w http.ResponseWriter, r *http.Request) {
	session, err := store.Get(r, "session-name")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	userId := session.Values["userid"]

	re := regexp.MustCompile(`/join/(\d+)`)
	parseResult := re.FindStringSubmatch(r.URL.Path)

	if parseResult == nil {
		http.Error(w, "bad url", http.StatusInternalServerError)
		return
	}

	draftId := parseResult[1]

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
	log.Printf("%s\t%s,%s", query, userId, draftId)

	_, err = database.Exec(query, userId, draftId)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/draft/%s", draftId), http.StatusTemporaryRedirect)
}

func ServePick(w http.ResponseWriter, r *http.Request) {
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
	userId := int64(userIdInt)

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

	http.Redirect(w, r, fmt.Sprintf("/draft/%d", draftId), http.StatusTemporaryRedirect)
}

func MakeDraft(name string) {
	query := `INSERT INTO drafts (name) VALUES (?);`
	res, err := database.Exec(query, name)
	if err != nil {
		// error
		return
	}

	draftId, err := res.LastInsertId()
	if err != nil {
		// error
		return
	}
	query = `INSERT INTO seats (position, draft) VALUES (?, ?)`
	var seatIds [9]int64
	for i := 0; i < 8; i++ {
		res, err = database.Exec(query, i, draftId)
		if err != nil {
			// error
			return
		}
		seatIds[i], err = res.LastInsertId()
		if err != nil {
			// error
			return
		}
	}

	res, err = database.Exec(`INSERT INTO seats (position, draft) VALUES(NULL, ?)`, draftId)
	if err != nil {
		// error
		return
	}
	seatIds[8], err = res.LastInsertId()
	if err != nil {
		// error
		return
	}

	query = `INSERT INTO packs (seat, original_seat, modified, round) VALUES (?, ?, 0, ?)`
	var packIds [25]int64
	for i := 0; i < 8; i++ {
		for j := 0; j < 4; j++ {
			res, err = database.Exec(query, seatIds[i], seatIds[i], j)
			if err != nil {
				// error
				return
			}
			if j != 0 {
				packIds[(3*i)+(j-1)], err = res.LastInsertId()
				if err != nil {
					// error
					return
				}
			}
		}
	}

	res, err = database.Exec(`INSERT INTO packs (seat, original_seat, modified, round) VALUES (?, ?, 0, NULL)`, seatIds[8], seatIds[8])
	if err != nil {
		// error
		return
	}
	packIds[24], err = res.LastInsertId()
	if err != nil {
		// error
		return
	}

	query = `INSERT INTO cards (pack, original_pack, edition, number, tags, name) VALUES (?, ?, ?, ?, ?, ?)`
	file, err := os.Open("cube.csv")
	if err != nil {
		// error
		return
	}
	defer file.Close()
	reader := csv.NewReader(bufio.NewReader(file))
	lines, err := reader.ReadAll()
	if err != nil {
		// error
		return
	}

	var src cryptoSource
	rnd := badrand.New(src)
	for i := 539; i > 164; i-- {
		j := rnd.Intn(i)
		lines[i], lines[j] = lines[j], lines[i]
		packId := packIds[(539-i)/15]
		database.Exec(query, packId, packId, lines[i][4], lines[i][5], lines[i][10], lines[i][0])
	}
	fmt.Printf("done generating new draft\n")
}

var googleOauthConfig = &oauth2.Config{
	RedirectURL:  os.Getenv("GOOGLE_REDIRECT_URL"),
	ClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
	ClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
	Scopes:       []string{"https://www.googleapis.com/auth/userinfo.email"},
	Endpoint:     google.Endpoint,
}

const oauthGoogleUrlAPI = "https://www.googleapis.com/oauth2/v2/userinfo?access_token="

func oauthGoogleLogin(w http.ResponseWriter, r *http.Request) {
	oauthState := generateStateOauthCookie(w)
	u := googleOauthConfig.AuthCodeURL(oauthState)
	http.Redirect(w, r, u, http.StatusTemporaryRedirect)
}

func oauthGoogleCallback(w http.ResponseWriter, r *http.Request) {
	oauthState, _ := r.Cookie("oauthstate")

	if r.FormValue("state") != oauthState.Value {
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}

	data, err := getUserDataFromGoogle(r.FormValue("code"))
	if err != nil {
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}

	session, err := store.Get(r, "session-name")
	var p GoogleUserInfo
	err = json.Unmarshal(data, &p)
	if err != nil {
		fmt.Fprintf(w, err.Error(), http.StatusInternalServerError)
		return
	}
	statement, _ := database.Prepare(`INSERT INTO users (google_id, email, picture, slack, discord) VALUES (?, ?, ?, "", "")`)
	statement.Exec(p.Id, p.Email, p.Picture)
	row := database.QueryRow(`SELECT id FROM users WHERE google_id = ?`, p.Id)
	var rowid string
	err = row.Scan(&rowid)
	if err != nil {
		fmt.Fprintf(w, err.Error(), http.StatusInternalServerError)
		return
	}
	session.Values["userid"] = rowid
	err = session.Save(r, w)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
}

func generateStateOauthCookie(w http.ResponseWriter) string {
	var expiration = time.Now().Add(365 * 24 * time.Hour)

	b := make([]byte, 16)
	rand.Read(b)
	state := base64.URLEncoding.EncodeToString(b)
	cookie := http.Cookie{Name: "oauthstate", Value: state, Expires: expiration}
	http.SetCookie(w, &cookie)

	return state
}

func getUserDataFromGoogle(code string) ([]byte, error) {
	token, err := googleOauthConfig.Exchange(context.Background(), code)
	if err != nil {
		return nil, fmt.Errorf("code exchange wrong: %s", err.Error())
	}
	response, err := http.Get(oauthGoogleUrlAPI + token.AccessToken)
	if err != nil {
		return nil, fmt.Errorf("failed getting user info: %s", err.Error())
	}
	defer response.Body.Close()
	contents, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("failed read response: %s", err.Error())
	}
	return contents, nil
}

func getPackPicksAndPowers(draftId int64, userId int64) ([]Card, []Card, []Card, error) {
	query := `select packs.round, cards.id, cards.name, cards.tags, cards.number, cards.edition, cards.faceup from drafts join seats join packs join cards where drafts.id=? and drafts.id=seats.draft and seats.id=packs.seat and packs.id=cards.pack and seats.user=? and (packs.round=0 or (packs.round=seats.round and packs.modified in (select min(packs.modified) from packs join seats join drafts where seats.draft=? and seats.user=? and seats.id=packs.seat and drafts.id=seats.draft and packs.round=seats.round)))`

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
		err = rows.Scan(&round, &id, &name, &tags, &number, &edition, &faceup)
		if err != nil {
			return nil, nil, nil, err
		}
		if round == 0 {
			myPicks = append(myPicks, Card{Name: name, Tags: tags, Number: number, Edition: edition, Id: id})
			if faceup == true {
				switch name {
				case "Cogwork Librarian":
					powers = append(powers, Card{Name: name, Tags: tags, Number: number, Edition: edition, Id: id})
				}
			}
		} else {
			myPack = append(myPack, Card{Name: name, Tags: tags, Number: number, Edition: edition, Id: id})
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

	query = `select packs.id, seats.id from packs join seats join users where seats.user=? and seats.id=packs.seat and packs.round=0 and seats.draft=? and users.id=seats.user`

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

	if pass {
		query = `begin transaction;update cards set pack=? where id=?;update packs set seat=?, modified=modified+10 where id=?;commit`
		log.Printf("%s\t%d,%d,%d,%d", query, pickId, cardId, newPositionId, oldPackId)

		_, err = database.Exec(query, pickId, cardId, newPositionId, oldPackId)
		if err != nil {
			return draftId, oldPackId, announcements, round, err
		}

		query = `select count(1) from v_packs join seats where v_packs.seat=seats.id and v_packs.round=? and v_packs.count>0`
		row = database.QueryRow(query, round)
		var packsLeftInRound int64
		err = row.Scan(&packsLeftInRound)
		if err != nil {
			return draftId, oldPackId, announcements, round, err
		}

		if packsLeftInRound == 0 {
			query = `update seats set round=round+1 where draft=?`
			log.Printf("%s\t%d", query, draftId)

			_, err = database.Exec(query, draftId)
			if err != nil {
				return draftId, oldPackId, announcements, round, err
			}
		} else {
			query = `select count(1) from v_packs join seats where v_packs.seat=seats.id and seats.user=? and v_packs.round=? and v_packs.count>0`
			row = database.QueryRow(query, userId, round)
			var packsLeftInSeat int64
			err = row.Scan(&packsLeftInSeat)
			if err != nil {
				return draftId, oldPackId, announcements, round, err
			}

			if packsLeftInSeat == 0 {
				err = NotifyByDraftAndPosition(draftId, newPosition)
				if err != nil {
					return draftId, oldPackId, announcements, round, err
				}
			}
		}
	} else {
		query = `update cards set pack=? where id=?`
		log.Printf("%s\t%d,%s,%d,%d", query, pickId, cardId)

		_, err = database.Exec(query, pickId, cardId)
		if err != nil {
			return draftId, oldPackId, announcements, round, err
		}
	}

	switch cardName {
	case "Lore Seeker":
		query = `select packs.id from packs join seats where packs.seat=seats.id and seats.draft=? and seats.position is null`
		row = database.QueryRow(query, draftId)
		var extraPackId int64
		err = row.Scan(&extraPackId)
		if err != nil {
			return draftId, oldPackId, announcements, round, err
		}

		query = `update packs set seat = ?, round = ?, modified = ? where id = ?`
		log.Printf("%s\t%d,%d,%d,%d", query, seatId, round, modified+5, extraPackId)
		_, err = database.Exec(query, seatId, round, modified+5, extraPackId)
		if err != nil {
			return draftId, oldPackId, announcements, round, err
		}

		query = `INSERT INTO revealed (draft, message) VALUES (?, "Seat " || ? || " revealed Lore Seeker.")`
		log.Printf("%s\t%d,%d", query, draftId, position)
		_, err = database.Exec(query, draftId, position)
		if err != nil {
			return draftId, oldPackId, announcements, round, err
		}

		announcements = append(announcements, fmt.Sprintf("Seat %d revealed Lore Seeker", position))
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

	query := `select users.slack,users.discord from users join seats where users.id=seats.user and seats.draft=? and seats.position=?`

	row := database.QueryRow(query, draftId, position)
	slack := ""
	discord := ""
	err := row.Scan(&slack, &discord)
	if err == sql.ErrNoRows {
		return nil
	} else if err != nil {
		return err
	}

	if slack != "" {
		var jsonStr = []byte(fmt.Sprintf(`{"text": "%s you have new picks <http://draft.thefoley.net/draft/%d>"}`, slack, draftId))
		req, err := http.NewRequest("POST", os.Getenv("SLACK_WEBHOOK_URL"), bytes.NewBuffer(jsonStr))
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
	}
	return nil
}

func GetJson(draftId int64) (string, error) {
	query := `select drafts.name, seats.position, packs.original_seat, packs.round, cards.name, cards.edition, cards.number, cards.tags from drafts join seats join packs join cards where drafts.id=seats.draft and seats.id=packs.original_seat and packs.id=cards.original_pack and drafts.id=?`

	rows, err := database.Query(query, draftId)
	if err != nil {
		return "", err
	}
	defer rows.Close()
	var draft DraftJson
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
		err = rows.Scan(&draft.Name, &nullablePosition, &packSeat, &nullableRound, &card.Name, &card.Edition, &card.Number, &card.Tags)
		if err != nil {
			return "", err
		}
		if !nullablePosition.Valid || !nullableRound.Valid {
			draft.ExtraPack = append(draft.ExtraPack, card)
			continue
		}
		position := nullablePosition.Int64
		packRound := nullableRound.Int64

		draft.Seats[position].Rounds[packRound].Packs[0].Cards = append(draft.Seats[position].Rounds[packRound].Packs[0].Cards, card)
	}

	query = `select seats.position, events.announcement, cards1.name, cards2.name, events.id, events.modified, events.round from events join seats on events.draft=seats.draft and events.user=seats.user left join cards as cards1 on events.card1=cards1.id left join cards as cards2 on events.card2=cards2.id where events.draft=?`
	rows, err = database.Query(query, draftId)
	if err != nil && err != sql.ErrNoRows {
		return "", err
	}
	defer rows.Close()
	for rows.Next() {
		var event DraftEvent
		var announcements string
		var card2 sql.NullString
		err = rows.Scan(&event.Player, &announcements, &event.Card1, &card2, &event.DraftModified, &event.PlayerModified, &event.Round)
		if err != nil {
			return "", err
		}
		if card2.Valid {
			event.Card2 = card2.String
		}
		if announcements != "" {
			event.Announcements = strings.Split(announcements, "\n")
		} else {
			event.Announcements = []string{}
		}
		draft.Events = append(draft.Events, event)
	}

	b, err := json.Marshal(draft)
	if err != nil {
		return "", err
	}

	return string(b), nil
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
