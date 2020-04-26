package main

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"golang.org/x/oauth2"
)

const (
	discordURLAPI = "https://discordapp.com/api/users/@me"
)

// DiscordUserInfo contains user account info from Discord.
type DiscordUserInfo struct {
	ID      string `json:"id"`
	Email   string `json:"email"`
	Picture string `json:"avatar"`
	Name    string `json:"username"`
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

func oauthDiscordLogin(w http.ResponseWriter, r *http.Request) {
	oauthLogin(w, r, discordOauthConfig)
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

	statement, _ := database.Prepare(`INSERT INTO users (discord_id, discord_name, picture) VALUES (?, ?, ?)`)
	statement.Exec(p.ID, p.Name, p.Picture)

	// BEGIN MIGRATION BLOCK
	// delete this block when migration to discord oauth is deemed complete

	row := database.QueryRow(`SELECT id FROM users_old WHERE slack="<@" || ? || ">"`, p.ID)
	var oldUserId int64
	err = row.Scan(&oldUserId)
	if err == sql.ErrNoRows {
		// everything is fine, new user
	} else if err != nil {
		// something bad happened
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	} else {
		log.Printf("found old user")
		// we found the old user
		_, err = database.Exec(`delete from users where id=?; UPDATE users set id=? where discord_id=?`, oldUserId, oldUserId, p.ID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	// END MIGRATION BLOCK

	row = database.QueryRow(`SELECT id FROM users WHERE discord_id = ?`, p.ID)
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
