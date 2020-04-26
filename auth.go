package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

const (
	oauthGoogleURLAPI = "https://www.googleapis.com/oauth2/v2/userinfo?access_token="
	discordURLAPI     = "https://discordapp.com/api/users/@me"
)

//GoogleUserInfo contains user account info from Google.
type GoogleUserInfo struct {
	ID      string `json:"id"`
	Email   string `json:"email"`
	Picture string `json:"picture"`
}

// DiscordUserInfo contains user account info from Discord.
type DiscordUserInfo struct {
	ID       string `json:"id"`
	Email    string `json:"email"`
	Picture  string `json:"avatar"`
	Discord  string `json:"discriminator"`
	Username string `json:"username"`
}

// TODO(walkingeyerobot): Tear out google oauth after migration,
// or make environment variables / callback URIs / generic and consistent?
// We may also want to rename google_id to oauth_id in sql schema; right now
// I'm overloading that field with discord_id but don't know if it matters.

var googleOauthConfig = &oauth2.Config{
	RedirectURL:  os.Getenv("GOOGLE_REDIRECT_URL"),
	ClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
	ClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
	Scopes:       []string{"https://www.googleapis.com/auth/userinfo.email"},
	Endpoint:     google.Endpoint,
}

var discordOauthConfig = &oauth2.Config{
	RedirectURL:  os.Getenv("DISCORD_REDIRECT_URL"),
	ClientID:     os.Getenv("DISCORD_CLIENT_ID"),
	ClientSecret: os.Getenv("DISCORD_CLIENT_SECRET"),
	Scopes:       []string{"email", "identify"},
	Endpoint: oauth2.Endpoint{
		AuthURL:  "https://discordapp.com/api/oauth2/authorize",
		TokenURL: "https://discordapp.com/api/oauth2/token",
	},
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

// TODO(jpgleg): Can we pipe this in from main:mux.HandleFunc directly somehow?
func oauthLogin(w http.ResponseWriter, r *http.Request, config *oauth2.Config) {
	oauthState := generateStateOauthCookie(w)
	u := config.AuthCodeURL(oauthState)
	http.Redirect(w, r, u, http.StatusTemporaryRedirect)
}

func oauthGoogleLogin(w http.ResponseWriter, r *http.Request) {
	oauthLogin(w, r, googleOauthConfig)
}

func oauthDiscordLogin(w http.ResponseWriter, r *http.Request) {
	oauthLogin(w, r, discordOauthConfig)
}

// TODO(jpgleg): See about deduping/refactoring the below, if we're keeping both.
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
	statement.Exec(p.ID, p.Email, p.Picture)
	row := database.QueryRow(`SELECT id FROM users WHERE google_id = ?`, p.ID)
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

func oauthDiscordCallback(w http.ResponseWriter, r *http.Request) {
	oauthState, _ := r.Cookie("oauthstate")

	if r.FormValue("state") != oauthState.Value {
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}

	data, err := getUserDataFromDiscord(r.FormValue("code"))
	if err != nil {
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}

	session, err := store.Get(r, "session-name")
	var p DiscordUserInfo
	err = json.Unmarshal(data, &p)
	if err != nil {
		fmt.Fprintf(w, err.Error(), http.StatusInternalServerError)
		return
	}
	p.Picture = fmt.Sprintf("https://cdn.discordapp.com/avatars/%v/%s.png", p.ID, p.Picture)
	p.Discord = fmt.Sprintf("%s#%s", p.Username, p.Discord)

	statement, _ := database.Prepare(`INSERT INTO users (google_id, email, picture, slack, discord) VALUES (?, ?, ?, "", ?)`)
	statement.Exec(p.ID, p.Email, p.Picture, p.Discord)
	row := database.QueryRow(`SELECT id FROM users WHERE google_id = ?`, p.ID)
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

func getUserDataFromGoogle(code string) ([]byte, error) {
	token, err := googleOauthConfig.Exchange(context.Background(), code)
	if err != nil {
		return nil, fmt.Errorf("code exchange wrong: %s", err.Error())
	}
	response, err := http.Get(oauthGoogleURLAPI + token.AccessToken)
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

func getUserDataFromDiscord(code string) ([]byte, error) {
	token, err := discordOauthConfig.Exchange(context.Background(), code)
	if err != nil {
		return nil, fmt.Errorf("code exchange wrong: %s", err.Error())
	}
	req, err := http.NewRequest("GET", discordURLAPI, nil)
	if err != nil {
		return nil, fmt.Errorf("could not initialize Discord API request: %s", err.Error())
	}
	req.Header.Set("Authorization", "Bearer "+token.AccessToken)
	client := &http.Client{}
	response, err := client.Do(req)
	if err != nil {
		if response != nil {
			response.Body.Close()
		}
		return nil, fmt.Errorf("failed getting user info: %s", err.Error())
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("received status code %v getting user info: %s", response.StatusCode, err.Error())
	}
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %s", err.Error())
	}
	return body, nil
}
