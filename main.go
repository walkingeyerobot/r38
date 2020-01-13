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
	Id      int64
	Name    string
	Tags    string
	Number  string
	Edition string
}

type QuestionPageData struct {
	QuestionId int64
	DraftId    int64
	Message    string
	Answers    []string
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

	// MakeDraft("test draft two")

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

	librarianHandler := http.HandlerFunc(ServeLibrarian)
	mux.Handle("/librarian/", AuthMiddleware(librarianHandler))
	powerHandler := http.HandlerFunc(ServePower)
	mux.Handle("/power/", AuthMiddleware(powerHandler))
	draftHandler := http.HandlerFunc(ServeDraft)
	mux.Handle("/draft/", AuthMiddleware(draftHandler))
	pickHandler := http.HandlerFunc(ServePick)
	mux.Handle("/pick/", AuthMiddleware(pickHandler))
	joinHandler := http.HandlerFunc(ServeJoin)
	mux.Handle("/join/", AuthMiddleware(joinHandler))
	answerHandler := http.HandlerFunc(ServeAnswer)
	mux.Handle("/answer/", AuthMiddleware(answerHandler))
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

	draftId1, packId1, err := doPick(userId, cardId1, false)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	draftId2, packId2, err := doPick(userId, cardId2, true)

	if packId1 != packId2 {
		http.Error(w, "pack ids somehow don't match.", http.StatusInternalServerError)
		return
	}

	if draftId1 != draftId2 {
		http.Error(w, "draft ids somehow don't match.", http.StatusInternalServerError)
		return
	}

	query = `select cards.id from cards join packs join seats where cards.pack=packs.id and packs.seat=seats.id and seats.draft=? and cards.name="Cogwork Librarian" and seats.user=?`

	row := database.QueryRow(query, draftId1, userId)
	var librarianId int64
	err = row.Scan(&librarianId)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	query = `update cards set pack=? where id=?`
	log.Printf("%s\t%d,%d", query, packId1, librarianId)
	_, err = database.Exec(query, packId1, librarianId)
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

	query := `select cards.name, seats.draft from cards join packs join seats where cards.pack=packs.id and packs.seat=seats.id and seats.user=? and cards.id=? and cards.faceup=true and cards.name<>"Aether Searcher"`

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

func ServeAnswer(w http.ResponseWriter, r *http.Request) {
	session, err := store.Get(r, "session-name")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	userId := session.Values["userid"]

	re := regexp.MustCompile(`/answer/(\d+)/(\w+)`)
	parseResult := re.FindStringSubmatch(r.URL.Path)

	if parseResult == nil {
		http.Error(w, "bad url", http.StatusInternalServerError)
		return
	}

	questionId := parseResult[1]
	answer := parseResult[2]

	query := `select questions.message, questions.answers, questions.seat, seats.draft, seats.position from questions join seats where questions.id=? and questions.seat=seats.id`

	row := database.QueryRow(query, questionId)
	var message string
	var rawAnswers string
	var seatId int64
	var draftId int64
	var position int64
	err = row.Scan(&message, &rawAnswers, &seatId, &draftId, &position)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	answers := strings.Split(rawAnswers, ",")
	var newAnswers []string

	for _, v := range answers {
		if v != answer {
			newAnswers = append(newAnswers, v)
		}
	}
	if len(answers) == len(newAnswers) {
		http.Error(w, "invalid answer.", http.StatusInternalServerError)
		return
	}

	query = `insert into revealed (draft, message) VALUES (?, "Seat " || ? || " answered '" || ? || "' to the question '" || ? || "'")`
	log.Printf("%s\t%d,%d,%s,%s", query, draftId, position, answer, message)
	_, err = database.Exec(query, draftId, position, answer, message)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	query = `update questions set answered=true where id=?`
	log.Printf("%s\t%s", query, questionId)
	_, err = database.Exec(query, questionId)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if strings.Contains(message, "Regicide") {
		query = `select count(1) from questions join seats where questions.seat=seats.id and questions.message like "%Regicide%" and seats.draft=?`

		row = database.QueryRow(query, draftId)
		var regicideQuestions int64
		err = row.Scan(&regicideQuestions)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if regicideQuestions < 3 {
			query = `select position from seats where user=? and draft=?`
			row = database.QueryRow(query, userId, draftId)
			var position int64
			err = row.Scan(&position)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			newPosition := position + 1
			if newPosition == 8 {
				newPosition = 0
			}

			query = `insert into questions (seat, message, answers) values ((select id from seats where draft=? and position=?), ?, ?)`
			log.Printf("%s\t%d,%d,%s,%s", query, draftId, newPosition, message, strings.Join(newAnswers, ","))
			_, err = database.Exec(query, draftId, newPosition, message, strings.Join(newAnswers, ",")) // should possibly be newAnswers[:]
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
	} else {
		http.Error(w, "unknown question type.", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, fmt.Sprintf("/draft/%d", draftId), http.StatusTemporaryRedirect)
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

	query := `select questions.id,questions.message,questions.answers from questions join seats where questions.seat=seats.id and seats.draft=? and seats.user=? and answered=false`
	row := database.QueryRow(query, draftId, userId)

	var questionId int64
	var message string
	var rawAnswers string
	err = row.Scan(&questionId, &message, &rawAnswers)
	if err == nil {
		answers := strings.Split(rawAnswers, ",")
		data := QuestionPageData{DraftId: draftId, QuestionId: questionId, Message: message, Answers: answers}
		t := template.Must(template.ParseFiles("question.tmpl"))

		t.Execute(w, data)
		return
	} else if err != nil && err != sql.ErrNoRows {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	myPack, myPicks, powers2, err := getPackPicksAndPowers(draftId, userId)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	query = `select seats.position, drafts.name from seats join drafts where seats.draft=? and seats.user=? and seats.draft=drafts.id`
	row = database.QueryRow(query, draftId, userId)
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
	//-----------------------------------------------------------------
	draftId, _, err := doPick(userId, cardId, true)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	//-----------------------------------------------------------------

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

	res, err = database.Exec(`INSERT INTO packs (seat, modified, round) VALUES (?, 0, NULL)`, seatIds[8])
	if err != nil {
		// error
		return
	}
	packIds[24], err = res.LastInsertId()
	if err != nil {
		// error
		return
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
	for i := 539; i > 164; i-- {
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

func doPick(userId int64, cardId int64, pass bool) (int64, int64, error) {
	query := `select drafts.id, seats.round, seats.position, cards.name from drafts join seats join packs join cards where drafts.id=seats.draft and seats.id=packs.seat and packs.id=cards.pack and cards.id=? and packs.round <> 0`

	row := database.QueryRow(query, cardId)
	var draftId int64
	var round int64
	var position int64
	var cardName string
	err := row.Scan(&draftId, &round, &position, &cardName)
	if err != nil {
		return -1, -1, err
	}

	query = `select packs.id, packs.modified from drafts join seats join packs join cards where cards.id=? and drafts.id=? and drafts.id=seats.draft and seats.id=packs.seat and packs.id=cards.pack and seats.user=? and (packs.round=0 or (packs.round=seats.round and packs.modified in (select min(packs.modified) from packs join seats join drafts where seats.draft=? and seats.user=? and seats.id=packs.seat and drafts.id=seats.draft and packs.round=seats.round)))`

	row = database.QueryRow(query, cardId, draftId, userId, draftId, userId)
	var oldPackId int64
	var modified int64
	err = row.Scan(&oldPackId, &modified)
	if err != nil {
		return draftId, -1, err
	}

	query = `select packs.id, seats.id from packs join seats join users where seats.user=? and seats.id=packs.seat and packs.round=0 and seats.draft=? and users.id=seats.user`

	row = database.QueryRow(query, userId, draftId)
	var pickId int64
	var seatId int64
	err = row.Scan(&pickId, &seatId)
	if err != nil {
		return draftId, oldPackId, err
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
		return draftId, oldPackId, err
	}

	if pass {
		query = `begin transaction;update cards set pack=? where id=?;update packs set seat=?, modified=modified+10 where id=?;commit`
		log.Printf("%s\t%d,%d,%d,%d", query, pickId, cardId, newPositionId, oldPackId)

		_, err = database.Exec(query, pickId, cardId, newPositionId, oldPackId)
		if err != nil {
			return draftId, oldPackId, err
		}

		err = CleanupEmptyPacks()
		if err != nil {
			return draftId, oldPackId, err
		}

		query = `select count(1) from packs join seats where packs.seat=seats.id and packs.round=?`
		row = database.QueryRow(query, round)
		var packsLeftInRound int64
		err = row.Scan(&packsLeftInRound)
		if err != nil {
			return draftId, oldPackId, err
		}

		if packsLeftInRound == 0 {
			query = `update seats set round=round+1 where draft=?`
			log.Printf("%s\t%d", query, draftId)

			_, err = database.Exec(query, draftId)
			if err != nil {
				return draftId, oldPackId, err
			}

			// TODO: ping all users. maybe.
		} else {
			query = `select count(1) from packs join seats where packs.seat=seats.id and seats.user=? and packs.round=?`
			row = database.QueryRow(query, userId, round)
			var packsLeftInSeat int64
			err = row.Scan(&packsLeftInSeat)
			if err != nil {
				return draftId, oldPackId, err
			}

			if packsLeftInSeat == 0 {
				err = NotifyByDraftAndPosition(draftId, newPosition)
				if err != nil {
					return draftId, oldPackId, err
				}
			}
		}
	} else {
		query = `update cards set pack=? where id=?`
		log.Printf("%s\t%d,%s,%d,%d", query, pickId, cardId)

		_, err = database.Exec(query, pickId, cardId)
		if err != nil {
			return draftId, oldPackId, err
		}
	}

	query = `select cards.id from cards join packs join seats where cards.faceup=true and cards.name="Aether Searcher" and cards.pack=packs.id and packs.seat=seats.id and seats.user=?`

	row = database.QueryRow(query, userId)
	var faceupAetherSearcherId int64
	err = row.Scan(&faceupAetherSearcherId)
	if err != nil && err != sql.ErrNoRows {
		return draftId, oldPackId, err
	} else if err == nil {
		query = `INSERT INTO revealed (draft, message) VALUES (?, "Seat " || ? || " revealed " || ? || " to Aether Searcher")`
		log.Printf("%s\t%d,%d,%s", query, draftId, position, cardName)
		_, err = database.Exec(query, draftId, position, cardName)
		if err != nil {
			return draftId, oldPackId, err
		}

		query = `update cards set faceup=FALSE where id=?`
		log.Printf("%s\t%d", query, faceupAetherSearcherId)
		_, err = database.Exec(query, faceupAetherSearcherId)
		if err != nil {
			return draftId, oldPackId, err
		}
	}

	switch cardName {
	case "Lore Seeker":
		query = `select packs.id from packs join seats where packs.seat=seats.id and seats.draft=? and seats.position is null`
		row = database.QueryRow(query, draftId)
		var extraPackId int64
		err = row.Scan(&extraPackId)
		if err != nil {
			return draftId, oldPackId, err
		}

		query = `update packs set seat = ?, round = ?, modified = ? where id = ?`
		log.Printf("%s\t%d,%d,%d,%d", query, seatId, round, modified+5, extraPackId)
		_, err = database.Exec(query, seatId, round, modified+5, extraPackId)
		if err != nil {
			return draftId, oldPackId, err
		}

		query = `INSERT INTO revealed (draft, message) VALUES (?, "Seat " || ? || " revealed Lore Seeker.")`
		log.Printf("%s\t%d,%d", query, draftId, position)
		_, err = database.Exec(query, draftId, position)
		if err != nil {
			return draftId, oldPackId, err
		}
	case "Aether Searcher":
		query = `update cards set faceup=TRUE where id=?`
		log.Printf("%s\t%s", query, cardId)
		_, err = database.Exec(query, cardId)
		if err != nil {
			return draftId, oldPackId, err
		}

		query = `INSERT INTO revealed (draft, message) VALUES (?, "Seat " || ? || " revealed Aether Searcher.")`
		log.Printf("%s\t%d,%d", query, draftId, position)
		_, err = database.Exec(query, draftId, position)
		if err != nil {
			return draftId, oldPackId, err
		}
	case "Regicide":
		query = `INSERT INTO revealed (draft, message) VALUES (?, "Seat " || ? || " revealed Regicide.")`
		log.Printf("%s\t%d,%d", query, draftId, position)
		_, err = database.Exec(query, draftId, position)
		if err != nil {
			return draftId, oldPackId, err
		}

		positionToAsk := position - 1
		if positionToAsk == -1 {
			positionToAsk = 7
		}
		query = `INSERT INTO questions (seat, message, answers) VALUES ((SELECT id FROM seats where draft=? and position=?), "Name a color for Regicide.", "White,Blue,Black,Red,Green")`
		log.Printf("%s\t%d,%d", query, draftId, positionToAsk)
		_, err = database.Exec(query, draftId, positionToAsk)
		if err != nil {
			return draftId, oldPackId, err
		}
	case "Cogwork Librarian":
		query = `update cards set faceup=true where id=?`
		log.Printf("%s\t%d", query, cardId)
		_, err = database.Exec(query, cardId)
		if err != nil {
			return draftId, oldPackId, err
		}

		query = `INSERT INTO revealed (draft, message) VALUES (?, "Seat " || ? || " revealed Cogwork Librarian.")`
		log.Printf("%s\t%d,%d", query, draftId, position)
		_, err = database.Exec(query, draftId, position)
		if err != nil {
			return draftId, oldPackId, err
		}
	}

	return draftId, oldPackId, nil
}

func CleanupEmptyPacks() error {
	query := `delete from packs where id in (select packs.id from packs left join cards on packs.id=cards.pack group by packs.id having count(cards.id)=0)`
	log.Printf("%s", query)
	_, err := database.Exec(query)
	if err != nil {
		return err
	}

	return nil
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
