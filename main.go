package main

import (
	"bufio"
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/binary"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"github.com/gorilla/sessions"
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
	Name   string
	Id     int64
	Seats  int64
	Joined bool
}

type IndexPageData struct {
	Drafts []Draft
}

type DraftPageData struct {
	DraftId   string
	DraftName string
	Picks     []Card
	Pack      []Card
}

type Card struct {
	Id      int64
	Name    string
	Tags    string
	Number  string
	Edition string
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

	// MakeDraft("test draft")

	server := &http.Server{
		Addr:    fmt.Sprintf(":12264"),
		Handler: Newww(),
	}

	log.Printf("Starting HTTP Server. Listening at %q", server.Addr)
	err = server.ListenAndServe()
	if err != nil {
		log.Printf("%v", err)
	}
}

func Newww() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/auth/google/login", oauthGoogleLogin)
	mux.HandleFunc("/auth/google/callback", oauthGoogleCallback)

	draftHandler := http.HandlerFunc(ServeDraft)
	mux.Handle("/draft/", AuthMiddleware(draftHandler))
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
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if session.Values["userid"] != nil {
			log.Printf("%s %s", session.Values["userid"], r.URL.Path)
			next.ServeHTTP(w, r)
		} else {
			fmt.Fprintf(w, `<html><body><a href="/auth/google/login">login</a></body></html>`)
		}
	})
}

func ServeIndex(w http.ResponseWriter, r *http.Request) {
	session, err := store.Get(r, "session-name")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	userId := session.Values["userid"]

	query := `select drafts.id, drafts.name, sum(seats.user is null) as empty_seats, coalesce(sum(seats.user = ?), 0) as joined from drafts left join seats on drafts.id = seats.draft group by drafts.id`

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
		Drafts = append(Drafts, d)
	}

	t := template.Must(template.ParseFiles("index.tmpl"))

	data := IndexPageData{Drafts: Drafts}
	t.Execute(w, data)
}

func ServeDraft(w http.ResponseWriter, r *http.Request) {
	session, err := store.Get(r, "session-name")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	userId := session.Values["userid"]

	re := regexp.MustCompile(`/draft/(\d+)`)
	parseResult := re.FindStringSubmatch(r.URL.Path)

	if parseResult == nil {
		http.Error(w, "bad url", http.StatusInternalServerError)
		return
	}

	draftId := parseResult[1]

	query := `select packs.round, cards.id, cards.name, cards.tags, cards.number, cards.edition from drafts join seats join packs join cards where drafts.id=? and drafts.id=seats.draft and seats.id=packs.seat and packs.id=cards.pack and seats.user=? and (packs.round=0 or (packs.round=drafts.round and packs.modified in (select min(packs.modified) from packs join seats join drafts where seats.draft=? and seats.user=? and seats.id=packs.seat and drafts.id=seats.draft and packs.round=drafts.round)))`

	rows, err := database.Query(query, draftId, userId, draftId, userId)
	if err == sql.ErrNoRows {
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	} else if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	var count int
	var myPicks []Card
	var myPack []Card
	for rows.Next() {
		var id int64
		var round int64
		var name string
		var tags string
		var number string
		var edition string
		err = rows.Scan(&round, &id, &name, &tags, &number, &edition)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		if round == 0 {
			myPicks = append(myPicks, Card{Name: name, Tags: tags, Number: number, Edition: edition, Id: id})
		} else {
			myPack = append(myPack, Card{Name: name, Tags: tags, Number: number, Edition: edition, Id: id})
		}
		count++
	}

	if count == 0 {
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}

	t := template.Must(template.ParseFiles("draft.tmpl"))

	data := DraftPageData{Picks: myPicks, Pack: myPack, DraftId: draftId, DraftName: "todo: pass draft name"}
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

	query = `update seats set user=? where id=(select id from seats where draft=? and user is null order by random() limit 1)`
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
	userId := session.Values["userid"]

	re := regexp.MustCompile(`/pick/(\d+)`)
	parseResult := re.FindStringSubmatch(r.URL.Path)

	if parseResult == nil {
		http.Error(w, "bad url", http.StatusInternalServerError)
		return
	}

	cardId := parseResult[1]

	query := `select drafts.id, drafts.round, seats.position, cards.name from drafts join seats join packs join cards where drafts.id=seats.draft and seats.id=packs.seat and packs.id=cards.pack and cards.id=? and packs.round <> 0`

	row := database.QueryRow(query, cardId)
	var draftId int64
	var round int64
	var position int64
	var cardName string
	err = row.Scan(&draftId, &round, &position, &cardName)
	if err != nil {
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}

	query = `select packs.id from drafts join seats join packs join cards where cards.id=? and drafts.id=? and drafts.id=seats.draft and seats.id=packs.seat and packs.id=cards.pack and seats.user=? and (packs.round=0 or (packs.round=drafts.round and packs.modified in (select min(packs.modified) from packs join seats join drafts where seats.draft=? and seats.user=? and seats.id=packs.seat and drafts.id=seats.draft and packs.round=drafts.round)))`

	row = database.QueryRow(query, cardId, draftId, userId, draftId, userId)
	var oldPackId int64
	err = row.Scan(&oldPackId)
	if err != nil {
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}

	query = `select packs.id from packs join seats where seats.user=? and seats.id=packs.seat and packs.round=0 and seats.draft=?`

	row = database.QueryRow(query, userId, draftId)
	var pickId int64
	err = row.Scan(&pickId)
	if err != nil {
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}

	query = `select id from seats where draft=? and position=?`

	var newPosition int64
	if round%2 == 0 {
		newPosition = position + 1
		if newPosition == 8 {
			newPosition = 0
		}
	} else {
		newPosition = position - 1
		if newPosition == -1 {
			newPosition = 7
		}
	}
	row = database.QueryRow(query, draftId, newPosition)
	var newPositionId int64
	err = row.Scan(&newPositionId)
	if err != nil {
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}

	query = `begin transaction;update cards set pack=? where id=?;update packs set seat=?, modified=modified+1 where id=?;commit`
	log.Printf("%s\t%d,%s,%d,%d", query, pickId, cardId, newPositionId, oldPackId)

	_, err = database.Exec(query, pickId, cardId, newPositionId, oldPackId)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	query = `select count(1) from cards join packs join seats where seats.draft=? and seats.id=packs.seat and cards.pack=packs.id and packs.round=?`

	row = database.QueryRow(query, draftId, round)
	var picksLeft int64
	err = row.Scan(&picksLeft)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if picksLeft == 0 {
		query = `update drafts set round=round+1 where id=?`
		log.Printf("%s\t%d", query, draftId)
		_, err = database.Exec(query, draftId)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	if cardName == "Lore Seeker" {
	} else if cardName == "Cogwork Librarian" {
	} else if cardName == "Paliano Vanguard" {
	} else if cardName == "Aether Searcher" {
	} else if cardName == "Regicide" {
	} else if cardName == "Whispergear Sneak" {
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
	var seatIds [8]int64
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

	query = `INSERT INTO packs (seat, modified, round) VALUES (?, 0, ?)`
	var packIds [25]int64
	for i := 0; i < 8; i++ {
		for j := 0; j < 4; j++ {
			res, err = database.Exec(query, seatIds[i], j)
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

	query = `INSERT INTO cards (pack, edition, number, tags, name) VALUES (?, ?, ?, ?, ?)`
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
	for i := 539; i > 179; i-- {
		j := rnd.Intn(i)
		lines[i], lines[j] = lines[j], lines[i]
		database.Exec(query, packIds[(539-i)/15], lines[i][4], lines[i][5], lines[i][7], lines[i][0])
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
	statement, _ := database.Prepare("INSERT INTO users (google_id, email, picture) VALUES (?, ?, ?)")
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
