package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/objectbox/objectbox-go/objectbox"
	"github.com/walkingeyerobot/r38/schema"
	"io"
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
	_, _ = rand.Read(b)
	state := base64.URLEncoding.EncodeToString(b)
	cookie := http.Cookie{Name: "oauthstate", Value: state, Expires: expiration}
	http.SetCookie(w, &cookie)
	return state
}

func oauthDiscordLogin(w http.ResponseWriter, r *http.Request, _ int64, _ *objectbox.ObjectBox) error {
	oauthState := generateStateOauthCookie(w)
	u := discordOauthConfig.AuthCodeURL(oauthState)
	http.Redirect(w, r, u, http.StatusTemporaryRedirect)
	return nil
}

func oauthDiscordCallback(w http.ResponseWriter, r *http.Request, _ int64, ob *objectbox.ObjectBox) error {
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
		p.Picture = "https://cdn.discordapp.com/embed/avatars/0.png"
	}

	userBox := schema.BoxForUser(ob)
	users, err := userBox.Query(schema.User_.DiscordId.Equals(p.ID, true)).Find()
	if err != nil {
		return err
	}

	var user *schema.User
	if len(users) > 0 {
		user = users[0]
	} else {
		user = &schema.User{
			DiscordId:   p.ID,
			DiscordName: p.Name,
			Picture:     p.Picture,
		}
		_, err := userBox.Put(user)
		if err != nil {
			return err
		}
	}
	session.Values["userid"] = user.Id

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
		return nil, fmt.Errorf("code exchange wrong: %w", err)
	}
	req, err := http.NewRequest("GET", discordURLAPI, nil)
	if err != nil {
		return nil, fmt.Errorf("could not initialize Discord API request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token.AccessToken)
	client := &http.Client{}
	response, err := client.Do(req)
	defer func() {
		if response != nil {
			_ = response.Body.Close()
		}
	}()
	if err != nil {
		return nil, fmt.Errorf("failed getting user info: %w", err)
	}
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("received status code %v getting user info: %w", response.StatusCode, err)
	}
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}
	return body, nil
}
