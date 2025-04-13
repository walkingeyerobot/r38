package main

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
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
	Picture string `json:"avatar"`
	Name    string `json:"username"`
}

var discordOauthConfig = &oauth2.Config{
	RedirectURL:  os.Getenv("DISCORD_REDIRECT_URL"),
	ClientID:     os.Getenv("DISCORD_CLIENT_ID"),
	ClientSecret: os.Getenv("DISCORD_CLIENT_SECRET"),
	Scopes:       []string{"identify"},
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

func oauthDiscordLogin(w http.ResponseWriter, r *http.Request, userID int64, tx *sql.Tx) error {
	oauthState := generateStateOauthCookie(w)
	u := discordOauthConfig.AuthCodeURL(oauthState)
	http.Redirect(w, r, u, http.StatusTemporaryRedirect)
	return nil
}

func oauthDiscordCallback(w http.ResponseWriter, r *http.Request, userID int64, tx *sql.Tx) error {
	oauthState, _ := r.Cookie("oauthstate")

	if r.FormValue("state") != oauthState.Value {
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return nil
	}

	data, err := getUserDataFromDiscord(r.FormValue("code"))
	if err != nil {
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return nil
	}

	session, err := store.Get(r, "session-name")
	var p DiscordUserInfo
	err = json.Unmarshal(data, &p)
	if err != nil {
		return err
	}
	if p.Picture != "" {
		p.Picture = fmt.Sprintf("https://cdn.discordapp.com/avatars/%v/%s.png", p.ID, p.Picture)
	} else {
		p.Picture = "http://draftcu.be/static/favicon.png"
	}

	statement, err := tx.Prepare(`INSERT INTO users (discord_id, discord_name, picture) VALUES (?, ?, ?)`)
	if err != nil {
		return err
	}
	statement.Exec(p.ID, p.Name, p.Picture)

	query := `update users set discord_name = ?, picture = ? where discord_id = ?`
	_, err = tx.Exec(query, p.Name, p.Picture, p.ID)
	if err != nil {
		return err
	}

	row := tx.QueryRow(`SELECT id FROM users WHERE discord_id = ?`, p.ID)
	var rowid string
	err = row.Scan(&rowid)
	if err != nil {
		return err
	}
	session.Values["userid"] = rowid
	err = session.Save(r, w)
	if err != nil {
		return err
	}

	http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
	return nil
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
