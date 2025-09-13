package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/objectbox/objectbox-go/objectbox"
	"github.com/walkingeyerobot/r38/draftconfig"
	"github.com/walkingeyerobot/r38/schema"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/walkingeyerobot/r38/makedraft"
	"github.com/walkingeyerobot/r38/migrations"

	"golang.org/x/net/xsrftoken"

	"github.com/BurntSushi/migration"
	"github.com/bwmarrin/discordgo"
	"github.com/go-co-op/gocron"
	"github.com/google/shlex"
	"github.com/gorilla/sessions"
	_ "github.com/mattn/go-sqlite3"
)

type r38handler func(w http.ResponseWriter, r *http.Request, userId int64, tx *sql.Tx, ob *objectbox.ObjectBox) error

const ForestBearId = "700900270153924608"
const ForestBear = ":forestbear:" + ForestBearId
const DraftAlertsRole = "692079611680653442"
const DraftFriendRole = "692865288554938428"
const EveryoneRole = "685333271793500161"
const Boss = "176164707026206720"
const Henchman = "360954995866075136"
const Pink = 0xE50389

var secretKeyNoOneWillEverGuess = []byte(os.Getenv("SESSION_SECRET"))
var xsrfKey string
var store = sessions.NewCookieStore(secretKeyNoOneWillEverGuess)
var sock string
var dg *discordgo.Session

func main() {
	useAuthPtr := flag.Bool("auth", true, "bool")
	useObjectBox := flag.Bool("objectbox", false, "bool")
	dbFile := flag.String("dbfile", "draft.db", "string")
	dbDir := flag.String("dbdir", "objectbox", "string")
	flag.Parse()

	xsrfKey = os.Getenv("XSRF_KEY")
	if len(xsrfKey) == 0 {
		xsrfKeyBytes := make([]byte, 128)
		_, err := rand.Read(xsrfKeyBytes)
		if err != nil {
			log.Printf("error generating XSRF key: %s. set using XSRF_KEY env variable", err.Error())
		}
		chars := "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
		for i, b := range xsrfKeyBytes {
			xsrfKeyBytes[i] = chars[b%byte(len(chars))]
		}
		xsrfKey = string(xsrfKeyBytes)
	}

	var database *sql.DB
	var ob *objectbox.ObjectBox
	var err error
	if *useObjectBox {
		ob, err = objectbox.NewBuilder().Model(schema.ObjectBoxModel()).
			Directory(*dbDir).Build()
		if err != nil {
			log.Printf("error opening db: %s", err.Error())
			return
		}
		defer ob.Close()
	} else {
		database, err = migration.Open("sqlite3", *dbFile, migrations.Migrations)
		if err != nil {
			log.Printf("error opening db: %s", err.Error())
			return
		}
		err = database.Ping()
		if err != nil {
			return
		}
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
		Handler: NewHandler(database, ob, *useAuthPtr),
	}

	dg, err = discordgo.New("Bot " + os.Getenv("DISCORD_BOT_TOKEN"))
	if err != nil {
		log.Printf("Error initializing discord bot: %s", err.Error())
	} else {
		defer func() {
			log.Printf("Closing discord bot")
			err = dg.Close()
			if err != nil {
				log.Printf("%s", err.Error())
			}
		}()
		dg.AddHandler(DiscordReady)
		dg.AddHandler(DiscordMsgCreate(database, ob))
		dg.AddHandler(DiscordReactionAdd(database, ob))
		dg.AddHandler(DiscordReactionRemove(database, ob))
		err = dg.Open()
		if err != nil {
			log.Printf("Error initializing discord bot: %s", err.Error())
		} else {
			log.Printf("Discord bot initialized.")
		}
	}

	scheduler := gocron.NewScheduler(time.UTC)
	_, err = scheduler.Every(8).Hours().Do(ArchiveSpectatorChannels, database, ob)
	if err != nil {
		log.Printf("error setting up spectator channel archive task: %s", err.Error())
	}

	scheduler.StartAsync()

	log.Printf("Starting HTTP Server. Listening at %q", server.Addr)
	go func() {
		err = server.ListenAndServe() // this call blocks

		if err != nil {
			log.Printf("%s", err.Error())
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)

	<-stop

	err = server.Shutdown(context.Background())
}

var ZoneDraftError = fmt.Errorf("zone draft violation")
var MethodNotAllowedError = fmt.Errorf("invalid request method")

// NewHandler creates all server routes for serving the html.
func NewHandler(database *sql.DB, ob *objectbox.ObjectBox, useAuth bool) http.Handler {
	mux := http.NewServeMux()

	addHandler := func(route string, serveFunc r38handler, readonly bool) {
		isAuthRoute := strings.HasPrefix(route, "/auth/")
		isApiRoute := strings.HasPrefix(route, "/api/")
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var userID int64
			if isApiRoute {
				w.Header().Set("Content-Type", "application/json")
			}
			if useAuth {
				if isAuthRoute {
					userID = 0
				} else {
					session, err := store.Get(r, "session-name")
					if err != nil {
						userID = 0
					} else {
						userIDVal := session.Values["userid"]
						if userIDVal == nil {
							userID = 0
						} else {
							userIDStr, ok := userIDVal.(string)
							if ok {
								userIDInt, err := strconv.Atoi(userIDStr)
								if err != nil {
									userID = 0
								} else {
									userID = int64(userIDInt)
								}
							} else {
								userIDInt, ok := userIDVal.(uint64)
								if ok {
									userID = int64(userIDInt)
								} else {
									userID = 0
								}
							}
						}
					}
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

			var err error
			if database != nil {
				var tx *sql.Tx
				tx, err = database.BeginTx(ctx, &sql.TxOptions{ReadOnly: readonly})
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}

				err = serveFunc(w, r, userID, tx, nil)
				if err != nil {
					err = errors.Join(err, tx.Rollback())
				} else {
					err = tx.Commit()
				}
			} else if ob != nil {
				handle := func() error {
					return serveFunc(w, r, userID, nil, ob)
				}
				if readonly {
					err = ob.RunInReadTx(handle)
				} else {
					err = ob.RunInWriteTx(handle)
				}
			}
			if err != nil {
				if isApiRoute {
					if errors.Is(err, ZoneDraftError) {
						w.WriteHeader(http.StatusBadRequest)
					} else if errors.Is(err, MethodNotAllowedError) {
						w.WriteHeader(http.StatusMethodNotAllowed)
					} else {
						w.WriteHeader(http.StatusInternalServerError)
					}
					_ = json.NewEncoder(w).Encode(JSONError{Error: err.Error()})
				} else {
					http.Error(w, err.Error(), http.StatusInternalServerError)
				}
			}
		})
		mux.Handle(route, handler)
	}

	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	mux.Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(http.Dir("client-dist/assets"))))
	mux.Handle("/favicon.ico", http.StripPrefix("/", http.FileServer(http.Dir("client-dist"))))

	if useAuth {
		log.Printf("setting up auth routes...")
		addHandler("/auth/discord/login", oauthDiscordLogin, true) // don't actually need db at all
		addHandler("/auth/discord/callback", oauthDiscordCallback, false)
	}

	addHandler("/api/archive/", ServeAPIArchive, false)
	addHandler("/api/draft/", ServeAPIDraft, true)
	addHandler("/api/draftlist/", ServeAPIDraftList, true)
	addHandler("/api/pick/", ServeAPIPick, false)
	addHandler("/api/pickrfid/", ServeAPIPickRfid, false)
	addHandler("/api/join/", ServeAPIJoin, false)
	addHandler("/api/skip/", ServeAPISkip, false)
	addHandler("/api/prefs/", ServeAPIPrefs, true)
	addHandler("/api/setpref/", ServeAPISetPref, false)
	addHandler("/api/undopick/", ServeAPIUndoPick, false)
	addHandler("/api/userinfo/", ServeAPIUserInfo, true)
	addHandler("/api/getcardpack/", ServeAPIGetCardPack, true)
	addHandler("/api/set/", ServeAPISet, true)

	addHandler("/api/dev/forceEnd/", ServeAPIForceEnd, false)

	mux.Handle("/", http.HandlerFunc(HandleIndex))

	return mux
}

func HandleIndex(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "client-dist/index.html")
}

// ServeAPIArchive serves the /api/archive endpoint.
func ServeAPIArchive(w http.ResponseWriter, r *http.Request, userID int64, tx *sql.Tx, ob *objectbox.ObjectBox) error {
	if userID != 1 {
		return nil
	}

	re := regexp.MustCompile(`/api/archive/(\d+)`)
	parseResult := re.FindStringSubmatch(r.URL.Path)
	if parseResult == nil {
		return fmt.Errorf("bad api url")
	}
	draftID, err := strconv.ParseInt(parseResult[1], 10, 64)
	if err != nil {
		return fmt.Errorf("bad api url: %w", err)
	}

	if ob != nil {
		box := schema.BoxForDraft(ob)
		var draft *schema.Draft
		draft, err = box.Get(uint64(draftID))
		if err != nil {
			return fmt.Errorf("couldn't find draft to archive: %w", err)
		}
		draft.Archived = true
		_, err = box.Put(draft)
	}

	return err
}

// ServeAPIDraft serves the /api/draft endpoint.
func ServeAPIDraft(w http.ResponseWriter, r *http.Request, userID int64, tx *sql.Tx, ob *objectbox.ObjectBox) error {
	re := regexp.MustCompile(`/api/draft/(\d+)`)
	parseResult := re.FindStringSubmatch(r.URL.Path)
	if parseResult == nil {
		return fmt.Errorf("bad api url")
	}
	draftID, err := strconv.ParseInt(parseResult[1], 10, 64)
	if err != nil {
		return fmt.Errorf("bad api url: %w", err)
	}

	draftJSON, err := GetFilteredJSON(tx, ob, draftID, userID)
	if err != nil {
		return fmt.Errorf("error getting json: %w", err)
	}

	_, err = fmt.Fprint(w, draftJSON)
	return err
}

// ServeAPIDraftList serves the /api/draftlist endpoint.
func ServeAPIDraftList(w http.ResponseWriter, _ *http.Request, userId int64, tx *sql.Tx, ob *objectbox.ObjectBox) error {
	var drafts DraftList
	var err error
	if tx != nil {
		drafts, err = GetDraftList(userId, tx)
	} else if ob != nil {
		drafts, err = GetDraftListOb(userId, ob)
	}
	if err != nil {
		return err
	}
	return json.NewEncoder(w).Encode(drafts)
}

// ServeAPIPrefs serves the /api/prefs endpoint.
func ServeAPIPrefs(w http.ResponseWriter, _ *http.Request, userId int64, tx *sql.Tx, _ *objectbox.ObjectBox) error {
	prefs, err := GetUserPrefs(userId, tx)
	if err != nil {
		return err
	}
	return json.NewEncoder(w).Encode(prefs)
}

// ServeAPISetPref serves the /api/setpref endpoint.
func ServeAPISetPref(w http.ResponseWriter, r *http.Request, userId int64, tx *sql.Tx, ob *objectbox.ObjectBox) error {
	if r.Method != "POST" {
		return MethodNotAllowedError
	}

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		return fmt.Errorf("error reading post body: %w", err)
	}
	var pref PostedPref
	err = json.Unmarshal(bodyBytes, &pref)
	if err != nil {
		return fmt.Errorf("error parsing post body: %w", err)
	}

	if pref.FormatPref.Format != "" && tx != nil {
		var query string
		if dg != nil {
			query := `select discord_id from users where id = ?`
			row := tx.QueryRow(query, userId)
			var discordId sql.NullString
			err = row.Scan(&discordId)
			if err != nil {
				return err
			}
			if !discordId.Valid {
				return fmt.Errorf("user %d with no discord ID can't enable formats", userId)
			}
			member, err := dg.GuildMember(makedraft.GuildId, discordId.String)
			if err != nil {
				return err
			}
			isDraftFriend := false
			for _, role := range member.Roles {
				if role == DraftFriendRole {
					isDraftFriend = true
				}
			}
			if !isDraftFriend {
				return fmt.Errorf("user %d is not draft friend, can't enable formats", userId)
			}
		}

		var elig int
		if pref.FormatPref.Elig {
			elig = 1
		} else {
			elig = 0
		}
		query = `update userformats set elig = ? where user = ? and format = ?`
		_, err = tx.Exec(query, elig, userId, pref.FormatPref.Format)
		if err != nil {
			return fmt.Errorf("error updating user pref: %w", err)
		}
	}

	if pref.MtgoName != "" {
		if tx != nil {
			query := `update users set mtgo_name = ? where id = ?`
			_, err = tx.Exec(query, pref.MtgoName, userId)
			if err != nil {
				return fmt.Errorf("error updating user MTGO name: %w", err)
			}
		} else if ob != nil {
			userBox := schema.BoxForUser(ob)
			user, err := userBox.Get(uint64(userId))
			if err != nil {
				return fmt.Errorf("error updating user MTGO name: %w", err)
			}
			user.MtgoName = pref.MtgoName
			_, err = userBox.Put(user)
			if err != nil {
				return fmt.Errorf("error updating user MTGO name: %w", err)
			}
		}
	}

	return ServeAPIPrefs(w, r, userId, tx, ob)
}

// ServeAPIPick serves the /api/pick endpoint.
func ServeAPIPick(w http.ResponseWriter, r *http.Request, userID int64, tx *sql.Tx, ob *objectbox.ObjectBox) error {
	if r.Method != "POST" {
		return MethodNotAllowedError
	}

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		return fmt.Errorf("error reading post body: %w", err)
	}
	var pick PostedPick
	err = json.Unmarshal(bodyBytes, &pick)
	if err != nil {
		return fmt.Errorf("error parsing post body: %w", err)
	}

	err = doHandlePostedPick(w, pick, userID, false, tx, ob)
	if err != nil {
		return err
	}
	return nil
}

// ServeAPIPickRfid serves the /api/pickrfid endpoint.
func ServeAPIPickRfid(w http.ResponseWriter, r *http.Request, userId int64, tx *sql.Tx, ob *objectbox.ObjectBox) error {
	if r.Method != "POST" {
		return MethodNotAllowedError
	}

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		return fmt.Errorf("error reading post body: %w", err)
	}
	var rfidPick PostedRfidPick
	err = json.Unmarshal(bodyBytes, &rfidPick)
	if err != nil {
		return fmt.Errorf("error parsing post body: %w", err)
	}

	var cardIds []int64
	if tx != nil {
		for _, cardRfid := range rfidPick.CardRfids {
			var cardId, err = findCardByRfid(tx, cardRfid, userId, rfidPick.DraftId)
			if err != nil {
				return fmt.Errorf("error finding card in active draft: %w", err)
			}
			cardIds = append(cardIds, cardId)
		}
	} else if ob != nil {
		draft, err := schema.BoxForDraft(ob).Get(uint64(rfidPick.DraftId))
		if err != nil {
			return fmt.Errorf("error finding card in active draft: %w", err)
		}
		var seatIndex = slices.IndexFunc(draft.Seats, func(seat *schema.Seat) bool {
			return seat.User != nil && seat.User.Id == uint64(userId)
		})
		if seatIndex == -1 {
			return fmt.Errorf("user %d not in draft %d", userId, rfidPick.DraftId)
		}
		seat := draft.Seats[seatIndex]
	cards:
		for _, cardRfid := range rfidPick.CardRfids {
			for _, pack := range seat.Packs {
				for _, card := range pack.Cards {
					if card.CardId == cardRfid {
						cardIds = append(cardIds, int64(card.Id))
						continue cards
					}
				}
			}
			for _, pack := range draft.UnassignedPacks {
				for _, card := range pack.Cards {
					if card.CardId == cardRfid {
						cardIds = append(cardIds, int64(card.Id))
						draft.UnassignedPacks = slices.DeleteFunc(draft.UnassignedPacks, func(p *schema.Pack) bool {
							return p == pack
						})
						seat.Packs = append(seat.Packs, pack)
						seat.OriginalPacks = append(seat.Packs, pack)
						pack.Round = seat.Round
						_, err = schema.BoxForDraft(ob).Put(draft)
						if err != nil {
							return err
						}
						_, err = schema.BoxForSeat(ob).Put(seat)
						if err != nil {
							return err
						}
						_, err = schema.BoxForPack(ob).Put(pack)
						if err != nil {
							return err
						}
						continue cards
					}
				}
			}
			if slices.ContainsFunc(seat.PickedCards, func(pickedCard *schema.Card) bool {
				return pickedCard.CardId == cardRfid
			}) {
				// duplicate scan
				return nil
			}
			return fmt.Errorf("couldn't find card %s", cardRfid)
		}
	}

	var pick = PostedPick{
		DraftId:   rfidPick.DraftId,
		CardIds:   cardIds,
		XsrfToken: rfidPick.XsrfToken,
	}

	err = doHandlePostedPick(w, pick, userId, true, tx, ob)
	if err != nil {
		return err
	}
	return nil
}

func doHandlePostedPick(w http.ResponseWriter, pick PostedPick, userId int64, zoneDrafting bool, tx *sql.Tx, ob *objectbox.ObjectBox) error {
	var err error
	if len(pick.CardIds) == 1 {
		err = doSinglePick(tx, ob, userId, pick.DraftId, pick.CardIds[0], zoneDrafting)
		if err == nil && !xsrftoken.Valid(pick.XsrfToken, xsrfKey, strconv.FormatInt(userId, 16), fmt.Sprintf("pick%d", pick.DraftId)) {
			err = fmt.Errorf("invalid XSRF token")
		}
		if err != nil {
			// We can't send the actual error back to the client without leaking information about
			// where the card they tried to pick actually is.
			log.Printf("error making pick: %s", err.Error())
			if errors.Is(err, ZoneDraftError) {
				return fmt.Errorf("error making pick: %w", err)
			} else {
				return fmt.Errorf("error making pick")
			}
		}
	} else if len(pick.CardIds) == 2 {
		return fmt.Errorf("cogwork librarian power not implemented yet")
	} else {
		return fmt.Errorf("invalid number of picked cards: %d", len(pick.CardIds))
	}

	var draftJSON string
	draftJSON, err = GetFilteredJSON(tx, ob, pick.DraftId, userId)
	if err != nil {
		return fmt.Errorf("error getting json: %w", err)
	}

	_, err = fmt.Fprint(w, draftJSON)
	return err
}

// ServeAPIUndoPick serves the /api/undopick endpoint.
func ServeAPIUndoPick(_ http.ResponseWriter, r *http.Request, userID int64, tx *sql.Tx, ob *objectbox.ObjectBox) error {
	if r.Method != "POST" {
		return MethodNotAllowedError
	}

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		return fmt.Errorf("error reading post body: %w", err)
	}
	var undo PostedUndo
	err = json.Unmarshal(bodyBytes, &undo)
	if err != nil {
		return fmt.Errorf("error parsing post body: %w", err)
	}

	if !xsrftoken.Valid(undo.XsrfToken, xsrfKey, strconv.FormatInt(userID, 16), fmt.Sprintf("pick%d", undo.DraftId)) {
		err = fmt.Errorf("invalid XSRF token")
	}

	if tx != nil {
		query := `select
					events.id, events.round, events.position, events.card1, events.pack, events.seat
					from events
					join seats on seats.draft = events.draft and seats.position = events.position
					where events.draft = ?
					and seats.user = ?
					and events.round = seats.round
					order by events.id desc`
		row := tx.QueryRow(query, undo.DraftId, userID)
		var eventID int64
		var round int64
		var position int64
		var cardID int64
		var packID int64
		var seatID int64
		err = row.Scan(&eventID, &round, &position, &cardID, &packID, &seatID)
		if err != nil {
			return fmt.Errorf("couldn't undo pick: %w", err)
		}

		query = `delete from events where id = ?`
		_, err = tx.Exec(query, eventID)
		if err != nil {
			return fmt.Errorf("couldn't undo pick: %w", err)
		}

		// Put the picked card into the player's picks.
		query = `update cards set pack = ? where id = ?`

		_, err = tx.Exec(query, packID, cardID)
		if err != nil {
			return fmt.Errorf("couldn't undo pick: %w", err)
		}

		// Move the pack to the next Position.
		query = `update packs set seat = ? where id = ?`
		_, err = tx.Exec(query, seatID, packID)
		if err != nil {
			return fmt.Errorf("couldn't undo pick: %w", err)
		}
	} else if ob != nil {
		draftBox := schema.BoxForDraft(ob)
		draft, err := draftBox.Get(1)
		if err != nil {
			return err
		}
		seatIndex := slices.IndexFunc(draft.Seats, func(seat *schema.Seat) bool {
			return seat.User.Id == uint64(userID)
		})
		seat := draft.Seats[seatIndex]
		var lastEvent *schema.Event = nil
		for _, event := range draft.Events {
			if event.Position == seat.Position &&
				event.Round == seat.Round &&
				(lastEvent == nil || event.Id > lastEvent.Id) {
				lastEvent = event
			}
		}

		draft.Events = slices.DeleteFunc(draft.Events, func(e *schema.Event) bool {
			return e == lastEvent
		})
		_, err = draftBox.Put(draft)
		if err != nil {
			return err
		}

		card := lastEvent.Card1
		packBox := schema.BoxForPack(ob)
		pack, err := packBox.Get(lastEvent.Pack.Id)
		if err != nil {
			return err
		}
		pack.Cards = append(pack.Cards, card)
		_, err = packBox.Put(pack)
		if err != nil {
			return err
		}

		seatBox := schema.BoxForSeat(ob)
		newSeatIndex := slices.IndexFunc(draft.Seats, func(seat *schema.Seat) bool {
			return slices.IndexFunc(seat.Packs, func(pack *schema.Pack) bool {
				return pack.Id == lastEvent.Pack.Id
			}) != -1
		})
		newSeat := draft.Seats[newSeatIndex]
		newSeat.Packs = slices.DeleteFunc(newSeat.Packs, func(pack *schema.Pack) bool {
			return pack.Id == lastEvent.Pack.Id
		})
		_, err = seatBox.Put(newSeat)
		if err != nil {
			return err
		}

		seat.PickedCards = slices.DeleteFunc(seat.PickedCards, func(c *schema.Card) bool {
			return c.Id == card.Id
		})
		seat.Packs = append(seat.Packs, pack)
		_, err = seatBox.Put(seat)
		if err != nil {
			return err
		}
	}

	return nil
}

// ServeAPIJoin serves the /api/join endpoint.
func ServeAPIJoin(w http.ResponseWriter, r *http.Request, userID int64, tx *sql.Tx, ob *objectbox.ObjectBox) error {
	if r.Method != "POST" {
		return MethodNotAllowedError
	}

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		return fmt.Errorf("error reading post body: %w", err)
	}
	toJoin := PostedJoin{
		ID:       0,
		Position: -1,
	}
	err = json.Unmarshal(bodyBytes, &toJoin)
	if err != nil {
		return fmt.Errorf("error parsing post body: %w", err)
	}

	draftID := toJoin.ID

	if toJoin.Position < 0 {
		if tx != nil {
			err = doJoin(tx, userID, draftID)
		} else if ob != nil {
			err = doJoinOb(ob, userID, draftID)
		}
	} else if toJoin.Position > 7 {
		return fmt.Errorf("invalid position %d", toJoin.Position)
	} else {
		if tx != nil {
			err = doJoinSeat(tx, userID, draftID, toJoin.Position)
		} else if ob != nil {
			err = doJoinSeatPositionOb(ob, userID, draftID, toJoin.Position)
		}
	}
	if err != nil {
		return fmt.Errorf("error joining draft %d: %w", draftID, err)
	}

	draftJSON, err := GetFilteredJSON(tx, ob, draftID, userID)
	if err != nil {
		return fmt.Errorf("error getting json: %w", err)
	}

	_, err = fmt.Fprint(w, draftJSON)
	return err
}

// doJoin does the actual joining.
func doJoin(tx *sql.Tx, userID int64, draftID int64) error {
	query := `select
                    count(1)
                  from seats
                  where draft = ?
                    and user = ?
                    and position <> 8`
	row := tx.QueryRow(query, draftID, userID)
	var alreadyJoined int64
	err := row.Scan(&alreadyJoined)
	if err != nil {
		return err
	} else if alreadyJoined > 0 {
		return fmt.Errorf("user %d already joined %d", userID, draftID)
	}

	query = `select
                   id, seats.reserveduser
                 from seats
                 where draft = ?
                   and user is null
                   and position <> 8
                 order by seats.reserveduser = ? desc, random()
                 limit 1`
	row = tx.QueryRow(query, draftID, userID)
	var emptySeatID int64
	var reservedUser sql.NullInt64
	err = row.Scan(&emptySeatID, &reservedUser)
	if err != nil {
		return err
	}
	if reservedUser.Valid && reservedUser.Int64 != userID {
		return fmt.Errorf("no non-reserved seats available for user %d in draft %d", userID, draftID)
	}

	err = doJoinSeatId(tx, userID, draftID, query, emptySeatID, row)
	if err != nil {
		return err
	}

	return nil
}

// doJoinOb does the actual joining.
func doJoinOb(ob *objectbox.ObjectBox, userId int64, draftId int64) error {
	draft, err := schema.BoxForDraft(ob).Get(uint64(draftId))
	if err != nil {
		return err
	}

	var reservedSeat *schema.Seat
	var openSeat *schema.Seat
	for _, seat := range draft.Seats {
		if seat.User.Id == uint64(userId) {
			return fmt.Errorf("user %d already joined %d", userId, draftId)
		}
		if seat.ReservedUser.Id == uint64(userId) {
			reservedSeat = seat
		} else if openSeat == nil && seat.User == nil {
			openSeat = seat
		}
	}

	if reservedSeat != nil {
		err = doJoinSeatOb(ob, userId, reservedSeat)
	} else if openSeat != nil {
		err = doJoinSeatOb(ob, userId, openSeat)
	} else {
		return fmt.Errorf("no non-reserved seats available for user %d in draft %d", userId, draftId)
	}

	return err
}

// doJoinSeat joins at a specific seat Position.
func doJoinSeat(tx *sql.Tx, userID int64, draftID int64, position int64) error {
	query := `select
                    count(1)
                  from seats
                  where draft = ?
                    and user = ?
                    and position <> 8`
	row := tx.QueryRow(query, draftID, userID)
	var alreadyJoined int64
	err := row.Scan(&alreadyJoined)
	if err != nil {
		return err
	} else if alreadyJoined > 0 {
		return fmt.Errorf("user %d already joined %d", userID, draftID)
	}

	query = `select
                   id, seats.user, seats.reserveduser
                 from seats
                 where draft = ?
                   and position = ?
                 limit 1`
	row = tx.QueryRow(query, draftID, position)
	var seatID int64
	var user sql.NullInt64
	var reservedUser sql.NullInt64
	err = row.Scan(&seatID, &user, &reservedUser)
	if err != nil {
		return err
	}
	if user.Valid && user.Int64 != userID {
		return fmt.Errorf("seat %d (position %d) in draft %d already occupied by user %d", seatID, position, draftID, user.Int64)
	}
	if reservedUser.Valid && reservedUser.Int64 != userID {
		return fmt.Errorf("seat %d (position %d) in draft %d already reserved by user %d", seatID, position, draftID, reservedUser.Int64)
	}

	err = doJoinSeatId(tx, userID, draftID, query, seatID, row)
	if err != nil {
		return err
	}

	return nil
}

// doJoinSeatPositionOb joins at a specific seat Position.
func doJoinSeatPositionOb(ob *objectbox.ObjectBox, userId int64, draftId int64, position int64) error {
	draft, err := schema.BoxForDraft(ob).Get(uint64(draftId))
	if err != nil {
		return err
	}

	if slices.ContainsFunc(draft.Seats, func(seat *schema.Seat) bool {
		return seat.User != nil && seat.User.Id == uint64(userId)
	}) {
		return fmt.Errorf("user %d already joined %d", userId, draftId)
	}

	seatIndex := slices.IndexFunc(draft.Seats, func(seat *schema.Seat) bool {
		return seat.Position == int(position)
	})
	seat := draft.Seats[seatIndex]
	if seat.User != nil {
		return fmt.Errorf("seat %d (position %d) in draft %d already occupied by user %d", seat.Id, position, draftId, seat.User.Id)
	}
	if seat.ReservedUser != nil && seat.ReservedUser.Id != uint64(userId) {
		return fmt.Errorf("seat %d (position %d) in draft %d already reserved by user %d", seat.Id, position, draft.Id, seat.ReservedUser.Id)
	}

	err = doJoinSeatOb(ob, userId, seat)
	return err
}

func doJoinSeatId(tx *sql.Tx, userID int64, draftID int64, query string, emptySeatID int64, row *sql.Row) error {
	query = `update seats set user = ? where id = ?`
	_, err := tx.Exec(query, userID, emptySeatID)
	if err != nil {
		return err
	}

	if dg != nil {
		query = `select spectatorchannelid from drafts where id = ?`
		row = tx.QueryRow(query, draftID)
		var channelID string
		err = row.Scan(&channelID)
		if err != nil {
			log.Printf("no spectator channel found for draft %d", draftID)
		} else {
			query = `select discord_id from users where id = ?`
			row = tx.QueryRow(query, userID)
			var discordID string
			err = row.Scan(&discordID)
			if err != nil {
				log.Printf("no discord ID for user %d", userID)
			} else {
				err = dg.ChannelPermissionSet(channelID, discordID, 1, 0, discordgo.PermissionViewChannel)
				if err != nil {
					log.Printf("error locking spectator channel for user %s: %s", discordID, err.Error())
				}
			}
		}
	}
	return nil
}

func doJoinSeatOb(ob *objectbox.ObjectBox, userId int64, seat *schema.Seat) error {
	user, err := schema.BoxForUser(ob).Get(uint64(userId))
	if err != nil {
		return err
	}
	seat.User = user
	_, err = schema.BoxForSeat(ob).Put(seat)
	return err
}

// ServeAPISkip serves the /api/skip endpoint.
func ServeAPISkip(w http.ResponseWriter, r *http.Request, userId int64, tx *sql.Tx, ob *objectbox.ObjectBox) error {
	if r.Method != "POST" {
		return MethodNotAllowedError
	}

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		return fmt.Errorf("error reading post body: %w", err)
	}
	var toJoin PostedJoin
	err = json.Unmarshal(bodyBytes, &toJoin)
	if err != nil {
		return fmt.Errorf("error parsing post body: %w", err)
	}

	draftId := toJoin.ID

	if tx != nil {
		err = doSkip(tx, userId, draftId)
	} else if ob != nil {
		err = doSkipOb(ob, userId, draftId)
	}
	if err != nil {
		return fmt.Errorf("error skipping draft %d: %w", draftId, err)
	}

	draftJSON, err := GetFilteredJSON(tx, ob, draftId, userId)
	if err != nil {
		return fmt.Errorf("error getting json: %w", err)
	}

	_, err = fmt.Fprint(w, draftJSON)
	return err
}

// doSkip does the actual skipping.
func doSkip(tx *sql.Tx, userID int64, draftID int64) error {
	query := `select
                    count(1)
                  from seats
                  where draft = ?
                    and user = ?
                    and position <> 8`
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
                   and reserveduser = ?
                   and position <> 8
                 limit 1`
	row = tx.QueryRow(query, draftID, userID)
	var reservedSeatID int64
	err = row.Scan(&reservedSeatID)
	if err != nil {
		return err
	}

	query = `insert into skips (user, draft) values (?, ?)`
	_, err = tx.Exec(query, userID, draftID)
	if err != nil {
		return err
	}

	query = `select format from drafts where id = ?`
	row = tx.QueryRow(query, draftID)
	var format string
	err = row.Scan(&format)
	if err != nil {
		return err
	}
	query = `update userformats set epoch = epoch - 1 where user = ? and format = ?`
	_, err = tx.Exec(query, userID, format)
	if err != nil {
		return err
	}

	newUser, err := makedraft.AssignSeats(tx, draftID, format, 1)
	if err != nil {
		return err
	}
	if len(newUser) > 0 {
		query = `update seats set reserveduser = ? where id = ?`
		_, err = tx.Exec(query, newUser[0], reservedSeatID)
	} else {
		query = `update seats set reserveduser = null where id = ?`
		_, err = tx.Exec(query, reservedSeatID)
	}
	if err != nil {
		return err
	}

	return nil
}

// doSkipOb does the actual skipping.
func doSkipOb(ob *objectbox.ObjectBox, userId int64, draftId int64) error {
	draft, err := schema.BoxForDraft(ob).Get(uint64(draftId))
	if err != nil {
		return err
	}

	if slices.IndexFunc(draft.Seats, func(seat *schema.Seat) bool {
		return seat.User.Id == uint64(userId)
	}) != -1 {
		return fmt.Errorf("user %d already joined %d", userId, draftId)
	}

	reservedSeatIndex := slices.IndexFunc(draft.Seats, func(seat *schema.Seat) bool {
		return seat.ReservedUser.Id == uint64(userId)
	})
	if reservedSeatIndex == -1 {
		return fmt.Errorf("no seat reserved for user %d in draft %d", userId, draftId)
	}
	seat := draft.Seats[reservedSeatIndex]

	userBox := schema.BoxForUser(ob)
	user, err := userBox.Get(uint64(userId))
	if err != nil {
		return err
	}
	user.Skips = append(user.Skips, &schema.Skip{
		DraftId: uint64(draftId),
	})
	_, err = userBox.Put(user)
	if err != nil {
		return err
	}

	newUser, err := makedraft.AssignSeatsOb(ob, draftId, 1)
	if err != nil {
		return err
	}
	if len(newUser) > 0 {
		seat.ReservedUser = nil
	} else {
		reservedUser, err := userBox.Get(uint64(newUser[0]))
		if err != nil {
			return err
		}
		seat.ReservedUser = reservedUser
		_, err = schema.BoxForSeat(ob).Put(seat)
		if err != nil {
			return err
		}
	}

	return nil
}

// ServeAPIForceEnd serves the /api/dev/forceEnd testing endpoint.
func ServeAPIForceEnd(_ http.ResponseWriter, r *http.Request, userID int64, tx *sql.Tx, ob *objectbox.ObjectBox) error {
	if userID == 1 {
		bodyBytes, err := io.ReadAll(r.Body)
		if err != nil {
			return fmt.Errorf("error reading post body: %w", err)
		}
		var toJoin PostedJoin
		err = json.Unmarshal(bodyBytes, &toJoin)
		if err != nil {
			return fmt.Errorf("error parsing post body: %w", err)
		}

		draftID := toJoin.ID
		return NotifyEndOfDraft(tx, ob, draftID)
	} else {
		return http.ErrBodyNotAllowed
	}
}

// ServeAPIUserInfo serves the /api/userinfo endpoint.
func ServeAPIUserInfo(w http.ResponseWriter, _ *http.Request, userID int64, tx *sql.Tx, ob *objectbox.ObjectBox) error {
	var userInfoJSON []byte
	var err error
	if tx != nil {
		userInfoJSON, err = getUserJSON(userID, tx)
	} else if ob != nil {
		userInfoJSON, err = getUserJSONOb(userID, ob)
	}
	if err != nil {
		return err
	}

	_, err = w.Write(userInfoJSON)
	return err
}

// ServeAPIGetCardPack serves the /api/getcardpack endpoint.
func ServeAPIGetCardPack(w http.ResponseWriter, r *http.Request, _ int64, tx *sql.Tx, ob *objectbox.ObjectBox) error {
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		return fmt.Errorf("error reading post body: %w", err)
	}
	var getCardPack PostedGetCardPack
	err = json.Unmarshal(bodyBytes, &getCardPack)
	if err != nil {
		return fmt.Errorf("error parsing post body: %w", err)
	}

	if tx != nil {
		query := `select pack from cards
		join packs on cards.pack = packs.id
		join seats on packs.seat = seats.id
		where seats.draft = ?
		and cards.cardid = ?`
		var packId int64
		row := tx.QueryRow(query, getCardPack.DraftID, getCardPack.CardRfid)
		err = row.Scan(&packId)
		if errors.Is(err, sql.ErrNoRows) {
			_, err = io.WriteString(w, "{\"pack\": 0}")
			if err != nil {
				return fmt.Errorf("error writing response: %w", err)
			}
			return nil
		} else if err != nil {
			return fmt.Errorf("error finding pack for card: %w", err)
		}

		query = `select packs.id from packs
		join seats on packs.seat = seats.id
		where seats.draft = ?`

		var rows *sql.Rows
		rows, err = tx.Query(query, getCardPack.DraftID)
		if err != nil {
			return fmt.Errorf("error finding packs in draft: %w", err)
		}
		defer func() {
			_ = rows.Close()
		}()
		var packIndex = 1
		for rows.Next() {
			var packIdAtIndex int64
			err = rows.Scan(&packIdAtIndex)
			if err != nil {
				return fmt.Errorf("error finding packs in draft: %w", err)
			}
			if packIdAtIndex == packId {
				_, err = fmt.Fprintf(w, "{\"pack\": %d}", packIndex)
				if err != nil {
					return fmt.Errorf("error writing response: %w", err)
				}
				return nil
			}
			packIndex++
		}
	} else if ob != nil {
		draft, err := schema.BoxForDraft(ob).Get(uint64(getCardPack.DraftID))
		if err != nil {
			return err
		}

		packIndex := slices.IndexFunc(draft.UnassignedPacks, func(pack *schema.Pack) bool {
			return slices.IndexFunc(pack.Cards, func(card *schema.Card) bool {
				return card.CardId == getCardPack.CardRfid
			}) != -1
		})
		_, err = io.WriteString(w, fmt.Sprintf("{\"pack\": %d}", packIndex+1))
		if err != nil {
			return fmt.Errorf("error writing response: %w", err)
		}
		return nil
	}

	return errors.New("didn't find pack in draft")
}

// ServeAPISet serves the /api/set endpoint.
func ServeAPISet(w http.ResponseWriter, r *http.Request, _ int64, _ *sql.Tx, _ *objectbox.ObjectBox) error {
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		return fmt.Errorf("error reading post body: %w", err)
	}
	var setRequest PostedSet
	err = json.Unmarshal(bodyBytes, &setRequest)
	if err != nil {
		return fmt.Errorf("error parsing post body: %w", err)
	}

	jsonFile, err := os.Open("sets/" + setRequest.Set + ".json")
	if err != nil {
		return fmt.Errorf("error opening json file: %w", err)
	}
	defer func() {
		_ = jsonFile.Close()
	}()

	byteValue, err := io.ReadAll(jsonFile)
	if err != nil {
		return fmt.Errorf("error readalling: %w", err)
	}

	var cfg draftconfig.DraftConfig
	err = json.Unmarshal(byteValue, &cfg)
	if err != nil {
		return fmt.Errorf("error unmarshalling: %w", err)
	}

	var cards []SetCardData
	configCards, err := draftconfig.GetCards(cfg)
	if err != nil {
		return fmt.Errorf("error getting cards: %w", err)
	}
	for _, card := range configCards {
		var cardData R38CardData
		err = json.Unmarshal([]byte(card.Data), &cardData)
		if err != nil {
			return fmt.Errorf("error unmarshalling: %w", err)
		}
		cards = append(cards, SetCardData{
			Id:   card.ID,
			Data: cardData,
		})
	}

	err = json.NewEncoder(w).Encode(cards)
	if err != nil {
		return fmt.Errorf("error marshalling: %w", err)
	}

	return nil
}

func getUserJSON(userID int64, tx *sql.Tx) ([]byte, error) {
	var userInfo UserInfo

	if userID != 0 {
		query := `select
                            id,
                            discord_name,
       						mtgo_name,
                            picture
                          from users
                          where id = ?`
		row := tx.QueryRow(query, userID)
		var mtgoName sql.NullString
		err := row.Scan(&userInfo.ID, &userInfo.Name, &mtgoName, &userInfo.Picture)
		if err != nil {
			return nil, err
		}
		if mtgoName.Valid {
			userInfo.MtgoName = mtgoName.String
		}
	}

	userInfoJSON, err := json.Marshal(userInfo)
	if err != nil {
		return nil, err
	}
	return userInfoJSON, nil
}

func getUserJSONOb(userId int64, ob *objectbox.ObjectBox) ([]byte, error) {
	var userInfo UserInfo

	if userId != 0 {
		user, err := schema.BoxForUser(ob).Get(uint64(userId))
		if err != nil {
			return nil, err
		}
		userInfo.ID = int64(user.Id)
		userInfo.Name = user.DiscordName
		userInfo.Picture = user.Picture
		userInfo.MtgoName = user.MtgoName
	}

	userInfoJSON, err := json.Marshal(userInfo)
	if err != nil {
		return nil, err
	}
	return userInfoJSON, nil
}

func findCardByRfid(tx *sql.Tx, cardRfid string, userID int64, draftID int64) (int64, error) {
	// Find card in user's packs.
	query := `select cards.id
					from cards
					join packs on packs.id = cards.pack
					join seats on seats.id = packs.seat
					and seats.draft = ?
					and seats.user = ?
					and seats.round = packs.round
					where cards.cardid = ?`
	row := tx.QueryRow(query, draftID, userID, cardRfid)
	var cardId int64
	err := row.Scan(&cardId)
	if err == nil {
		return cardId, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return 0, err
	}

	// Find card in unclaimed packs.
	query = `select cards.id, packs.id
					from cards
					join packs on packs.id = cards.pack
					join seats on seats.id = packs.seat
					and seats.draft = ?
					and seats.position = 8
					where cards.cardid = ?`
	row = tx.QueryRow(query, draftID, cardRfid)
	var packId int64
	err = row.Scan(&cardId, &packId)
	if err != nil {
		return 0, err
	}

	query = `select seats.id, seats.round
					from seats
					where seats.user = ?
					and seats.draft = ?`
	row = tx.QueryRow(query, userID, draftID)
	var seatId int64
	var round int
	err = row.Scan(&seatId, &round)
	if err != nil {
		return 0, err
	}

	query = `update packs
					set seat = ?, original_seat = ?, round = ?
					where id = ?`
	_, err = tx.Exec(query, seatId, seatId, round, packId)
	if err != nil {
		return 0, err
	}

	return cardId, nil
}

// doSinglePick performs a normal pick based on a user id and a card id.
func doSinglePick(tx *sql.Tx, ob *objectbox.ObjectBox, userId int64, draftId int64, cardId int64, zoneDrafting bool) error {
	if tx != nil {
		packID, announcements, round, seatID, err := doPick(tx, userId, draftId, cardId, zoneDrafting)
		if err != nil {
			return err
		}
		err = doEvent(tx, draftId, userId, announcements, cardId, sql.NullInt64{}, packID, seatID, round)
		if err != nil {
			return err
		}
	} else if ob != nil {
		packID, announcements, round, seat, err := doPickOb(ob, userId, draftId, cardId, zoneDrafting)
		if err != nil {
			return err
		}
		err = doEventOb(ob, draftId, announcements, cardId, sql.NullInt64{}, packID, seat, round)
		if err != nil {
			return err
		}
	}
	return nil
}

// doPick actually performs a pick in the database.
// It returns the packID, announcements, round, and an error.
// Of those return values, packID and announcements are only really relevant for Cogwork Librarian,
// which is not currently fully implemented, but we leave them here anyway for when we want to do that.
func doPick(tx *sql.Tx, userId int64, draftId int64, cardId int64, zoneDrafting bool) (int64, []string, int64, int64, error) {
	var announcements []string

	// First we need information about the card. Determine which pack the card is in,
	// where that pack is at the table, who sits at that Position, which draft that
	// pack is a part of, and which round that card is in.
	query := `select
                   packs.id,
                   seats.position,
                   seats.user,
                   seats.round,
                   seats.id
                 from cards
                 join packs on cards.pack = packs.id
                 join seats on packs.seat = seats.id
                 where cards.id = ?`

	row := tx.QueryRow(query, cardId)
	var myPackID int64
	var position int64
	var userID2 sql.NullInt64
	var round int64
	var seatID int64
	err := row.Scan(&myPackID, &position, &userID2, &round, &seatID)

	if err != nil {
		return myPackID, announcements, round, seatID, err
	} else if userID2.Valid && userId != userID2.Int64 {
		return myPackID, announcements, round, seatID, fmt.Errorf("card does not belong to the user")
	} else if round == 0 {
		return myPackID, announcements, round, seatID, fmt.Errorf("card has already been picked")
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

	row = tx.QueryRow(query, userId, draftId)
	var myPackID2 int64
	err = row.Scan(&myPackID2)

	if err != nil {
		return myPackID, announcements, round, seatID, err
	} else if myPackID != myPackID2 {
		return myPackID, announcements, round, seatID,
			fmt.Errorf("card is not in the next available pack (in pack %d but expecting pack %d)", myPackID, myPackID2)
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

	row = tx.QueryRow(query, userId, draftId)
	var myPicksID int64
	var myCount int64
	err = row.Scan(&myPicksID, &myCount)

	if err != nil {
		return myPackID, announcements, round, seatID, err
	}

	// Get the Position Position that the pack will be passed to.
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

	// Now get the Position id that the pack will be passed to.
	query = `select
                           seats.id,
                           users.discord_id
                         from seats
                         left join users on seats.user = users.id
                         where seats.draft = ?
                           and seats.position = ?`

	row = tx.QueryRow(query, draftId, newPosition)
	var newPositionID int64
	var newPositionDiscordID sql.NullString
	err = row.Scan(&newPositionID, &newPositionDiscordID)
	if err != nil {
		return myPackID, announcements, round, seatID, err
	}

	if zoneDrafting {
		// Enforce zone drafting: can't pass yet if there are two packs
		// belonging to the new position.
		query = `select count(*) from packs where seat = ? and round = ?`
		row = tx.QueryRow(query, newPositionID, round)
		var packsCount int64
		err = row.Scan(&packsCount)
		if err != nil {
			return myPackID, announcements, round, seatID, err
		}
		log.Printf("zone draft check: seat %d has %d packs", newPosition+1, packsCount)
		if packsCount >= 2 {
			return myPackID, announcements, round, seatID,
				fmt.Errorf("%w: seat %d (position %d) already has %d packs",
					ZoneDraftError, newPositionID, newPosition, packsCount)
		}
		if packsCount == 1 {
			// Check to see if new position is still picking from their first pack
			query = `select count(*) from packs where original_seat = ? and round = ?`
			row = tx.QueryRow(query, newPositionID, round)
			err = row.Scan(&packsCount)
			if err != nil {
				return myPackID, announcements, round, seatID, err
			}
			if packsCount == 0 {
				return myPackID, announcements, round, seatID,
					fmt.Errorf("%w: seat %d (position %d) already has 2 packs (including unclaimed first pack)",
						ZoneDraftError, newPositionID, newPosition)
			}
		}
	}

	// Put the picked card into the player's picks.
	query = `update cards set pack = ? where id = ?`

	_, err = tx.Exec(query, myPicksID, cardId)
	if err != nil {
		return myPackID, announcements, round, seatID, err
	}

	// Move the pack to the next Position.
	query = `update packs set seat = ? where id = ?`
	_, err = tx.Exec(query, newPositionID, myPackID)
	if err != nil {
		return myPackID, announcements, round, seatID, err
	}
	log.Printf("will move pack %d to seat %d (position %d)", myPackID, newPositionID, newPosition)

	// Get the number of remaining packs in the Position.
	query = `select
                           count(1)
                         from v_packs
                         join seats on v_packs.seat = seats.id
                         where seats.user = ?
                           and v_packs.round = ?
                           and v_packs.count > 0
                           and seats.draft = ?`
	row = tx.QueryRow(query, userId, round, draftId)
	var packsLeftInSeat int64
	err = row.Scan(&packsLeftInSeat)
	if err != nil {
		return myPackID, announcements, round, seatID, err
	}

	if packsLeftInSeat == 0 {
		// If there are 0 packs left in the Position, check to see if the player we passed the pack to
		// is in the same round as us. If the rounds match, NotifyByDraftAndPosition.
		query = `select
                                   count(1)
                                 from seats a
                                 join seats b on a.draft = b.draft
                                 where a.user = ?
                                   and b.position = ?
                                   and a.draft = ?
                                   and a.round = b.round`
		row = tx.QueryRow(query, userId, newPosition, draftId)
		var roundsMatch int64
		err = row.Scan(&roundsMatch)
		if err != nil {
			log.Printf("cannot determine if rounds match for notify")
		} else if roundsMatch == 1 && newPositionDiscordID.Valid {
			log.Printf("attempting to notify Position %d draft %d", newPosition, draftId)
			err = NotifyByDraftAndDiscordID(draftId, newPositionDiscordID.String)
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

		_, err = tx.Exec(query, (myCount+1)/15+1, userId, draftId)
		if err != nil {
			return myPackID, announcements, round, seatID, err
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
                                         and position <> 8
                                         group by round
                                         order by round desc
                                         limit 1`

			row = tx.QueryRow(query, draftId)
			var nextRoundPlayers int64
			err = row.Scan(&nextRoundPlayers)
			if err != nil {
				log.Printf("error counting players and rounds")
			} else if nextRoundPlayers == 8 && myCount+1 == 45 {
				// The draft is over. Notify the admin.
				err = NotifyEndOfDraft(tx, nil, draftId)
				if err != nil {
					log.Printf("error notifying end of draft: %s", err.Error())
				}
			} else if nextRoundPlayers > 1 {
				// Now we know that we are not the only player in this round.
				// Get the Position of all players that currently have a pick.
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

				rows, err := tx.Query(query, draftId)
				if err != nil {
					log.Printf("error determining if there's a blocking player")
				} else {
					defer func() {
						_ = rows.Close()
					}()

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
						err = NotifyByDraftAndDiscordID(draftId, blockingDiscordID.String)
						if err != nil {
							log.Printf("error with blocking notify")
						}
					}
				}
			}
		}
	}

	log.Printf("player %d in draft %d (position %d) took card %d from pack %d",
		userId, draftId, position, cardId, myPackID)

	return myPackID, announcements, round, seatID, nil
}

// doPickOb actually performs a pick in the database.
// It returns the packID, announcements, round, and an error.
// Of those return values, packID and announcements are only really relevant for Cogwork Librarian,
// which is not currently fully implemented, but we leave them here anyway for when we want to do that.
func doPickOb(ob *objectbox.ObjectBox, userId int64, draftId int64, cardId int64, zoneDrafting bool) (int64, []string, int64, *schema.Seat, error) {
	var announcements []string

	var myPackID int64
	var round int64

	draft, err := schema.BoxForDraft(ob).Get(uint64(draftId))
	if err != nil {
		return myPackID, announcements, round, nil, err
	}

	numSeats, cardsPerPack := getNumSeatsAndCardsPerPack(draft)

	seatIndex := slices.IndexFunc(draft.Seats, func(seat *schema.Seat) bool {
		return seat.User != nil && seat.User.Id == uint64(userId)
	})
	if seatIndex == -1 {
		return myPackID, announcements, round, nil, fmt.Errorf("user %d not in draft %d", userId, draftId)
	}
	seat := draft.Seats[seatIndex]
	if len(seat.Packs) == 0 {
		return myPackID, announcements, round, seat, fmt.Errorf("seat %d has no current pack", seat.Id)
	}
	slices.SortFunc(seat.Packs, func(a, b *schema.Pack) int {
		if a.Round != b.Round {
			return a.Round - b.Round
		}
		return len(b.Cards) - len(a.Cards)
	})
	round = int64(seat.Round)
	pack := seat.Packs[0]
	myPackID = int64(pack.Id)
	cardIndex := slices.IndexFunc(pack.Cards, func(card *schema.Card) bool {
		return card.Id == uint64(cardId)
	})
	if cardIndex == -1 {
		return myPackID, announcements, round, seat, fmt.Errorf("card %d not in seat %d's current pack", cardId, seat.Id)
	}
	card := pack.Cards[cardIndex]

	// Put the picked card into the player's picks.
	pack.Cards = slices.DeleteFunc(pack.Cards, func(c *schema.Card) bool {
		return c == card
	})
	seat.PickedCards = append(seat.PickedCards, card)

	pass := !(draft.PickTwo && len(seat.PickedCards)%2 == 1)

	// Are we passing the pack after we've picked the card?
	if pass {
		seat.Packs = slices.DeleteFunc(seat.Packs, func(p *schema.Pack) bool {
			return p == pack
		})

		// Get the Position that the pack will be passed to.
		var newPosition int
		if round%2 == 0 {
			newPosition = seat.Position - 1
			if newPosition == -1 {
				newPosition = numSeats - 1
			}
		} else {
			newPosition = seat.Position + 1
			if newPosition == numSeats {
				newPosition = 0
			}
		}

		nextSeatIndex := slices.IndexFunc(draft.Seats, func(seat *schema.Seat) bool {
			return seat.Position == newPosition
		})
		nextSeat := draft.Seats[nextSeatIndex]

		if len(pack.Cards) > 0 {
			if len(pack.Cards) > 1 && zoneDrafting {
				// Enforce zone drafting: can't pass yet if there are two packs
				// belonging to the new position.
				packsCount := len(nextSeat.Packs)
				log.Printf("zone draft check: seat %d has %d packs", newPosition+1, packsCount)
				if packsCount >= 2 {
					return myPackID, announcements, round, seat,
						fmt.Errorf("%w: seat %d (position %d) already has %d packs",
							ZoneDraftError, nextSeat.Id, newPosition, packsCount)
				}
				if packsCount == 1 && len(nextSeat.OriginalPacks) == 0 {
					// New position is still picking from their first pack
					return myPackID, announcements, round, seat,
						fmt.Errorf("%w: seat %d (position %d) already has 2 packs (including unclaimed first pack)",
							ZoneDraftError, nextSeat.Id, newPosition)
				}
			}

			// Move the pack to the next Position.
			nextSeat.Packs = append(nextSeat.Packs, pack)
			log.Printf("will move pack %d to seat %d (position %d)", myPackID, nextSeat.Id, newPosition)
		}

		// Get the number of remaining packs in the Position.
		packsLeftInSeat := len(seat.Packs)

		if packsLeftInSeat == 0 {
			// If there are 0 packs left in the Position, check to see if the player we passed the pack to
			// is in the same round as us. If the rounds match, NotifyByDraftAndPosition.
			roundsMatch := seat.Round == nextSeat.Round
			if roundsMatch && nextSeat.User != nil && nextSeat.User.DiscordId != "" {
				log.Printf("attempting to notify Position %d draft %d", newPosition, draftId)
				err = NotifyByDraftAndDiscordID(draftId, nextSeat.User.DiscordId)
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
			seat.Round = len(seat.PickedCards)/cardsPerPack + 1
			round = int64(seat.Round)

			// If the rounds do NOT match from earlier, we have a situation where players are in different
			// rounds. Look for a blocking player.
			if !roundsMatch {
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
				nextRoundPlayers := 0
				nextRound := 0
				for _, s := range draft.Seats {
					nextRound = max(nextRound, s.Round)
				}
				for _, s := range draft.Seats {
					if s.Round == nextRound {
						nextRoundPlayers++
					}
				}
				if nextRoundPlayers == numSeats && len(seat.PickedCards) == cardsPerPack*3 {
					// The draft is over. Notify the admin.
					err = NotifyEndOfDraft(nil, ob, draftId)
					if err != nil {
						log.Printf("error notifying end of draft: %s", err.Error())
					}
				} else if nextRoundPlayers > 1 {
					// Now we know that we are not the only player in this round.
					blockingDiscordId := ""
					for _, s := range draft.Seats {
						if len(s.Packs) > 0 {
							if blockingDiscordId != "" || s.User == nil {
								blockingDiscordId = ""
								break
							}
							blockingDiscordId = s.User.DiscordId
						}
					}
					if blockingDiscordId != "" {
						err = NotifyByDraftAndDiscordID(draftId, blockingDiscordId)
					}
				}
			}
		}
		_, err = schema.BoxForSeat(ob).PutMany([]*schema.Seat{seat, nextSeat})
		if err != nil {
			return myPackID, announcements, round, seat, err
		}
	} else {
		_, err = schema.BoxForSeat(ob).Put(seat)
		if err != nil {
			return myPackID, announcements, round, seat, err
		}
	}

	_, err = schema.BoxForPack(ob).Put(pack)
	if err != nil {
		return myPackID, announcements, round, seat, err
	}

	log.Printf("player %d in draft %d (position %d) took card %d from pack %d",
		userId, draftId, seat.Position, cardId, myPackID)

	return myPackID, announcements, round, seat, nil
}

func getNumSeatsAndCardsPerPack(draft *schema.Draft) (int, int) {
	var numSeats int
	var cardsPerPack int
	if draft.PickTwo {
		numSeats = 4
		cardsPerPack = 14
	} else {
		numSeats = 8
		cardsPerPack = 15
	}
	return numSeats, cardsPerPack
}

// NotifyByDraftAndDiscordID sends a discord alert to a user.
func NotifyByDraftAndDiscordID(draftID int64, discordID string) error {
	return DiscordNotify(os.Getenv("PICK_ALERTS_CHANNEL_ID"),
		fmt.Sprintf(`<@%s> you have new picks <https://draftcu.be/draft/%d>`, discordID, draftID))
}

func NotifyEndOfDraft(tx *sql.Tx, ob *objectbox.ObjectBox, draftID int64) error {
	if tx != nil {
		draftName, err := GetDraftName(tx, draftID)
		if err != nil {
			return err
		}

		err = PostFirstRoundPairings(tx, draftID, draftName)
		if err != nil {
			return err
		}

		adminDiscordID, err := GetAdminDiscordId(tx)
		if err != nil {
			return err
		}
		err = NotifyAdminOfDraftCompletion(adminDiscordID, draftID)
		if err != nil {
			return err
		}

		result := tx.QueryRow("select spectatorchannelid from drafts where id=?", draftID)
		var channelID sql.NullString
		err = result.Scan(&channelID)
		if !channelID.Valid {
			// OK to not find channel
			return nil
		}
		err = UnlockSpectatorChannel(channelID.String)
		if err != nil {
			return err
		}
	} else if ob != nil {
		draft, err := schema.BoxForDraft(ob).Get(uint64(draftID))
		if err != nil {
			return err
		}

		err = PostFirstRoundPairingsOb(ob, draft)
		if err != nil {
			return err
		}

		admin, err := schema.BoxForUser(ob).Get(1)
		if err != nil {
			return err
		}
		err = NotifyAdminOfDraftCompletion(admin.DiscordId, int64(draft.Id))
		if err != nil {
			return err
		}

		if len(draft.SpectatorChannelId) > 0 {
			err = UnlockSpectatorChannel(draft.SpectatorChannelId)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func GetDraftName(tx *sql.Tx, draftID int64) (string, error) {
	query := `select
					name
				  from drafts
				  where drafts.id = ?`
	row := tx.QueryRow(query, draftID)
	var draftName string
	err := row.Scan(&draftName)
	if err != nil {
		log.Print(err.Error())
	}
	return draftName, err
}

func PostFirstRoundPairings(tx *sql.Tx, draftID int64, draftName string) error {
	query := `select
					discord_id, discord_name
				  from users
				  inner join seats on users.id = seats.user
				  where seats.draft = ?
				  order by seats.position`
	drafters, err := tx.Query(query, draftID)
	if err != nil {
		log.Print(err.Error())
		return err
	}
	var drafterIds []string
	for drafters.Next() {
		var drafterId sql.NullString
		var drafterName string
		err = drafters.Scan(&drafterId, &drafterName)
		if err != nil {
			log.Print(err.Error())
			return err
		}
		if drafterId.Valid {
			drafterIds = append(drafterIds, fmt.Sprintf("<@%s>", drafterId.String))
		} else {
			drafterIds = append(drafterIds, drafterName)
		}
	}

	if len(drafterIds) == 8 {
		pairings := fmt.Sprintf(`%s vs %s
%s vs %s
%s vs %s
%s vs %s`,
			drafterIds[0], drafterIds[4],
			drafterIds[1], drafterIds[5],
			drafterIds[2], drafterIds[6],
			drafterIds[3], drafterIds[7])
		round := 1
		err2 := PostPairings(tx, draftID, draftName, round, pairings)
		if err2 != nil {
			return err2
		}
	}
	return nil
}

func PostFirstRoundPairingsOb(ob *objectbox.ObjectBox, draft *schema.Draft) error {
	drafterIds := [8]string{}
	for _, seat := range draft.Seats {
		if len(seat.User.DiscordId) > 0 {
			drafterIds[seat.Position] = seat.User.DiscordId
		} else {
			drafterIds[seat.Position] = seat.User.DiscordName
		}
	}

	var pairings string
	if len(draft.Seats) == 8 {
		pairings = fmt.Sprintf(`%s vs %s
%s vs %s
%s vs %s
%s vs %s`,
			drafterIds[0], drafterIds[4],
			drafterIds[1], drafterIds[5],
			drafterIds[2], drafterIds[6],
			drafterIds[3], drafterIds[7])
	} else {
		pairings = fmt.Sprintf(`%s vs %s
%s vs %s`,
			drafterIds[0], drafterIds[2],
			drafterIds[1], drafterIds[3])
	}
	round := 1
	err := PostPairingsOb(ob, draft, round, pairings)
	return err
}

func PostPairings(tx *sql.Tx, draftID int64, draftName string, round int, pairings string) error {
	msg, err := DiscordNotifyEmbed(
		os.Getenv("DRAFT_ANNOUNCEMENTS_CHANNEL_ID"),
		&discordgo.MessageEmbed{
			Title:       fmt.Sprintf("%s, Round %d", draftName, round),
			Description: pairings,
			Color:       Pink,
			Footer: &discordgo.MessageEmbedFooter{
				Text: "React with 🏆 if you win and 💀 if you lose.",
			},
		})
	if err != nil {
		log.Print(err.Error())
		return err
	}
	if msg == nil {
		return nil
	}
	err = dg.MessageReactionAdd(msg.ChannelID, msg.ID, "🏆")
	if err != nil {
		log.Printf("%s", err.Error())
	}
	err = dg.MessageReactionAdd(msg.ChannelID, msg.ID, "💀")
	if err != nil {
		log.Printf("%s", err.Error())
	}
	_, err = tx.Exec("insert into pairingmsgs (msgid, draft, round) values (?, ?, ?)",
		msg.ID, draftID, round)
	if err != nil {
		log.Print(err.Error())
		return err
	}
	return nil
}

func PostPairingsOb(ob *objectbox.ObjectBox, draft *schema.Draft, round int, pairings string) error {
	msg, err := DiscordNotifyEmbed(
		os.Getenv("DRAFT_ANNOUNCEMENTS_CHANNEL_ID"),
		&discordgo.MessageEmbed{
			Title:       fmt.Sprintf("%s, Round %d", draft.Name, round),
			Description: pairings,
			Color:       Pink,
			Footer: &discordgo.MessageEmbedFooter{
				Text: "React with 🏆 if you win and 💀 if you lose.",
			},
		})
	if err != nil {
		log.Print(err.Error())
		return err
	}
	if msg == nil {
		return nil
	}
	err = dg.MessageReactionAdd(msg.ChannelID, msg.ID, "🏆")
	if err != nil {
		log.Printf("%s", err.Error())
	}
	err = dg.MessageReactionAdd(msg.ChannelID, msg.ID, "💀")
	if err != nil {
		log.Printf("%s", err.Error())
	}
	_, err = schema.BoxForPairingMsg(ob).Put(&schema.PairingMsg{
		MsgId: msg.ID,
		Draft: draft,
		Round: round,
	})
	if err != nil {
		log.Print(err.Error())
		return err
	}
	return nil
}

func NotifyAdminOfDraftCompletion(adminDiscordId string, draftID int64) error {
	return DiscordNotify(os.Getenv("PICK_ALERTS_CHANNEL_ID"),
		fmt.Sprintf(`<@%s> draft %d is finished!`, adminDiscordId, draftID))
}

func GetAdminDiscordId(tx *sql.Tx) (string, error) {
	query := `select
                    discord_id
                  from users
                  where id = 1`
	row := tx.QueryRow(query)
	var adminDiscordID string
	err := row.Scan(&adminDiscordID)
	return adminDiscordID, err
}

func GetAdminDiscordIdOb(ob *objectbox.ObjectBox) (string, error) {
	user, err := schema.BoxForUser(ob).Get(1)
	if err != nil {
		return "", err
	}
	return user.DiscordId, nil
}

func UnlockSpectatorChannel(channelId string) error {
	if dg != nil {
		channel, err := dg.Channel(channelId)
		if err != nil {
			return err
		}
		for _, perm := range channel.PermissionOverwrites {
			err = dg.ChannelPermissionDelete(channelId, perm.ID)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// DiscordNotify posts a message to discord.
func DiscordNotify(channelId string, message string) error {
	if dg != nil {
		_, err := dg.ChannelMessageSend(channelId, message)
		return err
	}
	return nil
}

// DiscordNotifyEmbed posts a message to discord.
func DiscordNotifyEmbed(channelId string, message *discordgo.MessageEmbed) (*discordgo.Message, error) {
	if dg != nil {
		msg, err := dg.ChannelMessageSendEmbed(channelId, message)
		return msg, err
	}
	return nil, nil
}

// GetJSONObject returns a better DraftJSON object. May be filtered.
func GetJSONObject(tx *sql.Tx, draftID int64) (DraftJSON, error) {
	var draft DraftJSON

	query := `select
                    drafts.id,
                    drafts.name,
                    drafts.inperson,
                    seats.position,
                    packs.round,
                    users.discord_name,
                    users.mtgo_name,
                    cards.id,
                    users.id,
                    cards.data,
                    users.picture,
                    seats.scansound,
                    seats.errorsound
                  from seats
                  left join users on users.id = seats.user
                  join drafts on drafts.id = seats.draft
                  left join packs on packs.original_seat = seats.id
                  left join cards on cards.original_pack = packs.id
                  where drafts.id = ?`

	rows, err := tx.Query(query, draftID)
	if err != nil {
		return draft, fmt.Errorf("error getting draft details: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()
	var indices [8][3]int64
	for rows.Next() {
		var position int64
		var packRound sql.NullInt64
		var cardID sql.NullInt64
		var nullableDiscordName sql.NullString
		var nullableMtgoName sql.NullString
		var draftUserID sql.NullInt64
		var cardData sql.NullString
		var nullablePicture sql.NullString
		var scanSound int64
		var errorSound int64
		err = rows.Scan(&draft.DraftID,
			&draft.DraftName,
			&draft.InPerson,
			&position,
			&packRound,
			&nullableDiscordName,
			&nullableMtgoName,
			&cardID,
			&draftUserID,
			&cardData,
			&nullablePicture,
			&scanSound,
			&errorSound)
		if err != nil {
			return draft, err
		}

		if position == 8 {
			continue
		}

		if packRound.Valid && cardID.Valid && cardData.Valid {
			dataObj := make(map[string]interface{})
			err = json.Unmarshal([]byte(cardData.String), &dataObj)
			if err != nil {
				log.Printf("making nil card data because of error %s", err.Error())
				dataObj = nil
			}
			dataObj["id"] = cardID.Int64

			nextIndex := indices[position][packRound.Int64-1]
			if nextIndex >= 15 {
				return draft, fmt.Errorf("too many cards for seat %d round %d", position, packRound.Int64)
			}

			draft.Seats[position].Packs[packRound.Int64-1][nextIndex] = dataObj

			indices[position][packRound.Int64-1]++
		}

		draft.Seats[position].PlayerName = nullableDiscordName.String
		if nullableMtgoName.Valid {
			draft.Seats[position].MtgoName = nullableMtgoName.String
		}
		draft.Seats[position].PlayerID = draftUserID.Int64
		draft.Seats[position].PlayerImage = nullablePicture.String
		draft.Seats[position].ScanSound = scanSound
		draft.Seats[position].ErrorSound = errorSound
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
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return draft, fmt.Errorf("error getting draft events: %s", err)
	}
	defer func() {
		_ = rows.Close()
	}()
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

// GetJSONObjectOb returns a better DraftJSON object. May be filtered.
func GetJSONObjectOb(ob *objectbox.ObjectBox, draftId int64) (DraftJSON, error) {
	var draftJson = DraftJSON{DraftID: draftId}

	draft, err := schema.BoxForDraft(ob).Get(uint64(draftId))
	if err != nil {
		return draftJson, err
	}
	draftJson.DraftName = draft.Name
	draftJson.InPerson = draft.InPerson

	for _, seat := range draft.Seats {
		if seat.User != nil {
			draftJson.Seats[seat.Position].PlayerID = int64(seat.User.Id)
			draftJson.Seats[seat.Position].PlayerName = seat.User.DiscordName
			draftJson.Seats[seat.Position].PlayerImage = seat.User.Picture
			draftJson.Seats[seat.Position].MtgoName = seat.User.MtgoName
		}
		draftJson.Seats[seat.Position].ScanSound = int64(seat.ScanSound)
		draftJson.Seats[seat.Position].ErrorSound = int64(seat.ErrorSound)
		for _, pack := range seat.OriginalPacks {
			for k, card := range pack.OriginalCards {
				dataObj := make(map[string]interface{})
				err = json.Unmarshal([]byte(card.Data), &dataObj)
				if err != nil {
					log.Printf("making nil card data because of error %s", err.Error())
					dataObj = nil
				}
				dataObj["id"] = card.Id
				draftJson.Seats[seat.Position].Packs[pack.Round-1][k] = dataObj
			}
		}
	}

	for _, event := range draft.Events {
		var eventJson DraftEvent
		eventJson.Round = int64(event.Round)
		eventJson.Position = int64(event.Position)
		eventJson.Cards = append(eventJson.Cards, int64(event.Card1.Id))
		if event.Card2 != nil {
			eventJson.Cards = append(eventJson.Cards, int64(event.Card2.Id))
			eventJson.Librarian = true
		}
		if event.Announcement != "" {
			eventJson.Announcements = strings.Split(event.Announcement, "\n")
		}
		eventJson.Type = "Pick"
		draftJson.Events = append(draftJson.Events, eventJson)
	}

	return draftJson, err
}

// GetFilteredJSON returns a filtered json object of replay data.
func GetFilteredJSON(tx *sql.Tx, ob *objectbox.ObjectBox, draftId int64, userId int64) (string, error) {
	draftInfo, err := GetDraftListEntry(userId, tx, ob, draftId)
	if err != nil {
		return "", fmt.Errorf("error getting draft list entry: %w", err)
	}

	var draft DraftJSON
	if tx != nil {
		draft, err = GetJSONObject(tx, draftId)
	} else if ob != nil {
		draft, err = GetJSONObjectOb(ob, draftId)
	}
	if err != nil {
		return "", fmt.Errorf("error getting draft details: %w", err)
	}

	draft.PickXsrf = xsrftoken.Generate(xsrfKey, strconv.FormatInt(userId, 16), fmt.Sprintf("pick%d", draftId))

	var returnFullReplay bool
	if draftInfo.Finished {
		// If the draft is over, everyone can see the full replay.
		returnFullReplay = true
	} else if draftInfo.Joined {
		// If we're a member of the draft and it's NOT over,
		// we need to see if we're done with the draft. If we are,
		// we can see the full replay. Otherwise, we need to
		// filter.
		if tx != nil {
			query := `select
                            round
                          from seats
                          where draft = ?
                            and user = ?`
			row := tx.QueryRow(query, draftId, userId)
			var myRound sql.NullInt64
			err = row.Scan(&myRound)
			if err != nil {
				return "", fmt.Errorf("error detecting end of draft %d for user %d: %w", draftId, userId, err)
			}
			if myRound.Valid && myRound.Int64 >= 4 {
				returnFullReplay = true
			}
		} else if ob != nil {
			draft, err := schema.BoxForDraft(ob).Get(uint64(draftId))
			if err != nil {
				return "", fmt.Errorf("error detecting end of draft %d for user %d: %w", draftId, userId, err)
			}
			for _, seat := range draft.Seats {
				if seat.User != nil && seat.User.Id == uint64(userId) {
					returnFullReplay = seat.Round >= 4
					break
				}
			}
		}
	} else if userId != 0 && draftInfo.AvailableSeats == 0 && draftInfo.ReservedSeats == 0 {
		// If we're logged in AND the draft is full,
		// we can see the full replay.
		returnFullReplay = true
	}

	if returnFullReplay {
		ret, err := json.Marshal(draft)
		if err != nil {
			return "", fmt.Errorf("error marshalling full draft replay: %w", err)
		}
		return string(ret), nil
	}

	var buff bytes.Buffer
	if sock != "" {
		conn, err := net.Dial("unix", sock)
		if err != nil {
			return "", fmt.Errorf("error connecting to filter service: %w", err)
		}
		defer func() {
			_ = conn.Close()
		}()

		ret, err := json.Marshal(Perspective{User: userId, Draft: draft})
		if err != nil {
			return "", fmt.Errorf("error marshalling filter service request: %w", err)
		}

		stop := "\r\n\r\n"

		_, err = conn.Write(ret)
		if err != nil {
			return "", err
		}
		_, err = conn.Write([]byte(stop))
		if err != nil {
			return "", err
		}

		_, err = io.Copy(&buff, conn)
		if err != nil {
			return "", err
		}
	}

	return buff.String(), nil
}

// doEvent records an event (pick) into the database.
func doEvent(tx *sql.Tx, draftID int64, userID int64, announcements []string, cardID1 int64, cardID2 sql.NullInt64, packID int64, seatID int64, round int64) error {
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
		query = `insert into events (round, draft, position, announcement, card1, card2, pack, seat, modified) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`
		_, err = tx.Exec(query, round, draftID, position, strings.Join(announcements, "\n"), cardID1, cardID2.Int64, packID, seatID, count)
	} else {
		query = `insert into events (round, draft, position, announcement, card1, card2, pack, seat, modified) VALUES (?, ?, ?, ?, ?, null, ?, ?, ?)`
		_, err = tx.Exec(query, round, draftID, position, strings.Join(announcements, "\n"), cardID1, packID, seatID, count)
	}

	return err
}

// doEventOb records an event (pick) into the database.
func doEventOb(ob *objectbox.ObjectBox, draftId int64, announcements []string, cardId1 int64, cardId2 sql.NullInt64, packId int64, seat *schema.Seat, round int64) error {
	draftBox := schema.BoxForDraft(ob)
	draft, err := draftBox.Get(uint64(draftId))
	if err != nil {
		return err
	}
	card1, err := schema.BoxForCard(ob).Get(uint64(cardId1))
	if err != nil {
		return err
	}
	pack, err := schema.BoxForPack(ob).Get(uint64(packId))
	if cardId2.Valid {
		card2, err := schema.BoxForCard(ob).Get(uint64(cardId2.Int64))
		if err != nil {
			return err
		}
		draft.Events = append(draft.Events, &schema.Event{
			Position:     seat.Position,
			Announcement: strings.Join(announcements, "\n"),
			Card1:        card1,
			Card2:        card2,
			Pack:         pack,
			Modified:     len(seat.PickedCards),
			Round:        int(round),
		})
	} else {
		draft.Events = append(draft.Events, &schema.Event{
			Position:     seat.Position,
			Announcement: strings.Join(announcements, "\n"),
			Card1:        card1,
			Card2:        nil,
			Pack:         pack,
			Modified:     len(seat.PickedCards),
			Round:        int(round),
		})
	}

	_, err = draftBox.Put(draft)
	return err
}

func GetDraftList(userID int64, tx *sql.Tx) (DraftList, error) {
	drafts := DraftList{Drafts: make([]DraftListEntry, 0)}

	query := `select
                    drafts.id,
                    drafts.name,
                    sum(seats.user is null and seats.reserveduser is null and seats.position is not null) as empty_seats,
                    sum(seats.reserveduser not null and seats.user is null) as reserved_seats,
                    coalesce(sum(seats.user = ?), 0) as joined,
                    coalesce(sum(seats.reserveduser = ?), 0) as reserved,
                    coalesce(skips.user = ?, 0) as skipped,
                    min(seats.round) > 3 as finished,
                    drafts.inperson
                  from drafts
                  left join seats on drafts.id = seats.draft
                  left join skips on drafts.id = skips.draft and skips.user = ?
                  where seats.position <> 8
                  group by drafts.id
                  order by drafts.id`

	rows, err := tx.Query(query, userID, userID, userID, userID)
	if err != nil {
		return drafts, fmt.Errorf("can't get draft list: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()
	for rows.Next() {
		var d DraftListEntry
		err = rows.Scan(&d.ID,
			&d.Name,
			&d.AvailableSeats,
			&d.ReservedSeats,
			&d.Joined,
			&d.Reserved,
			&d.Skipped,
			&d.Finished,
			&d.InPerson)
		if err != nil {
			return drafts, fmt.Errorf("can't get draft list: %w", err)
		}

		d = AddStatus(d, userID)

		drafts.Drafts = append(drafts.Drafts, d)
	}
	return drafts, nil
}

func GetDraftListOb(userId int64, ob *objectbox.ObjectBox) (DraftList, error) {
	draftList := DraftList{
		Drafts: []DraftListEntry{},
	}
	drafts, err := schema.BoxForDraft(ob).Query(schema.Draft_.Archived.Equals(false)).Find()
	if err != nil {
		return draftList, err
	}
	var user *schema.User
	if userId == 0 {
		user = &schema.User{}
	} else {
		user, err = schema.BoxForUser(ob).Get(uint64(userId))
		if err != nil {
			return draftList, err
		}
	}
	for _, draft := range drafts {
		draftList.Drafts = append(draftList.Drafts, draftToDraftListEntry(draft, user))
	}
	return draftList, nil
}

func AddStatus(d DraftListEntry, userId int64) DraftListEntry {
	if d.Joined {
		d.Status = "member"
	} else if d.Reserved {
		d.Status = "reserved"
	} else if d.Finished {
		d.Status = "spectator"
	} else if userId == 0 {
		d.Status = "closed"
	} else if d.AvailableSeats == 0 && d.ReservedSeats == 0 {
		d.Status = "spectator"
	} else if d.AvailableSeats == 0 || d.Skipped {
		d.Status = "closed"
	} else {
		d.Status = "joinable"
	}
	return d
}

func GetDraftListEntry(userId int64, tx *sql.Tx, ob *objectbox.ObjectBox, draftId int64) (DraftListEntry, error) {
	var ret DraftListEntry

	if tx != nil {
		drafts, err := GetDraftList(userId, tx)
		if err != nil {
			return ret, err
		}

		draftCount := len(drafts.Drafts)
		i := sort.Search(draftCount, func(i int) bool { return draftId <= drafts.Drafts[i].ID })
		if i < draftCount && i >= 0 {
			return drafts.Drafts[i], nil
		}

		return ret, fmt.Errorf("could not find draft id %d", draftId)
	} else if ob != nil {
		draft, err := schema.BoxForDraft(ob).Get(uint64(draftId))
		if err != nil {
			return ret, err
		}

		var user *schema.User
		if userId == 0 {
			user = &schema.User{}
		} else {
			user, err = schema.BoxForUser(ob).Get(uint64(userId))
			if err != nil {
				return ret, err
			}
		}

		return draftToDraftListEntry(draft, user), nil
	}

	return ret, nil
}

func draftToDraftListEntry(draft *schema.Draft, user *schema.User) DraftListEntry {
	numAvailable := int64(0)
	numReserved := int64(0)
	finished := true
	joined := false
	reserved := false

	for _, seat := range draft.Seats {
		if seat.User != nil {
			if seat.User.Id == user.Id {
				joined = true
			}
		} else if seat.ReservedUser != nil {
			numReserved++
			if seat.ReservedUser.Id == user.Id {
				reserved = true
			}
		} else {
			numAvailable++
		}
		if seat.Round <= 3 {
			finished = false
		}
	}

	skipped := slices.ContainsFunc(user.Skips, func(skip *schema.Skip) bool {
		return skip.DraftId == draft.Id
	})

	return AddStatus(DraftListEntry{
		AvailableSeats: numAvailable,
		ReservedSeats:  numReserved,
		Finished:       finished,
		ID:             int64(draft.Id),
		Joined:         joined,
		Reserved:       reserved,
		Skipped:        skipped,
		Name:           draft.Name,
		InPerson:       draft.InPerson,
	}, int64(user.Id))
}

func GetUserPrefs(userID int64, tx *sql.Tx) (UserFormatPrefs, error) {
	var prefs UserFormatPrefs

	if tx != nil {
		query := `select
				format, elig
				from userformats
				where user = ?`
		result, err := tx.Query(query, userID)
		if err != nil {
			return prefs, err
		}
		for result.Next() {
			var pref UserFormatPref
			err = result.Scan(&pref.Format, &pref.Elig)
			prefs.Prefs = append(prefs.Prefs, pref)
		}
	}

	return prefs, nil
}

func DiscordReady(s *discordgo.Session, _ *discordgo.Ready) {
	err := s.UpdateCustomStatus("Tier 5 Wolf Combo")
	if err != nil {
		log.Printf("Error readying discord bot: %s", err.Error())
	}
}

func DiscordMsgCreate(database *sql.DB, ob *objectbox.ObjectBox) func(s *discordgo.Session, msg *discordgo.MessageCreate) {
	return func(s *discordgo.Session, msg *discordgo.MessageCreate) {
		if msg.Author.ID == Boss || msg.Author.ID == Henchman {
			if msg.GuildID == "" {
				args, err := shlex.Split(msg.Content)
				if err != nil {
					_, _ = dg.ChannelMessageSend(msg.ChannelID, err.Error())
					return
				}
				if strings.HasPrefix(msg.Content, "makedraft") {
					settings, err := makedraft.ParseSettings(args)
					if msg.Author.ID == Henchman {
						*settings.Simulate = true
					}

					var resp string
					if err != nil {
						resp = fmt.Sprintf("%s", err.Error())
					} else if database != nil {
						tx, err := database.Begin()
						if err != nil {
							log.Printf("%s", err.Error())
							return
						}
						defer func() {
							_ = tx.Rollback()
						}()

						err = errors.Join(err, makedraft.MakeDraft(settings, tx, nil))

						if err != nil {
							resp = fmt.Sprintf("%s", err.Error())
						} else {
							if !*settings.Simulate {
								err = tx.Commit()
							} else {
								err = nil
							}

							if err != nil {
								resp = fmt.Sprintf("can't commit :( %s", err.Error())
							} else {
								resp = fmt.Sprintf("done!")
							}
						}
					} else if ob != nil {
						err = ob.RunInWriteTx(func() error {
							return makedraft.MakeDraft(settings, nil, ob)
						})
						if err != nil {
							resp = fmt.Sprintf("can't commit :( %s", err.Error())
						} else {
							resp = fmt.Sprintf("done!")
						}
					}
					_, err = dg.ChannelMessageSend(msg.ChannelID, resp)
					if err != nil {
						log.Printf("Error responding to discord bot DM: %s", err)
					}
				}
			} else if msg.Content == "!alerts" {
				DiscordSendRoleReactionMessage(s, database, ob, msg.ChannelID,
					ForestBear, ForestBearId, DraftAlertsRole,
					"Draft alerts", "if you would like notifications for games being played")
			}
		}
	}
}

func DiscordSendRoleReactionMessage(s *discordgo.Session, database *sql.DB, ob *objectbox.ObjectBox, channelID string, emoji string, emojiId string, roleId string, title string, description string) {
	sent, err := DiscordNotifyEmbed(
		channelID,
		&discordgo.MessageEmbed{
			Title: title,
			Description: "\nReact with <" + emoji + "> " + description + ".\n\n" +
				"If you would like to remove the role, simply remove your reaction.\n",
			Color: Pink,
		})
	if err != nil {
		log.Printf("Error responding to discord bot !alerts: %s", err.Error())
	} else if sent != nil {
		if database != nil {
			_, err = database.Exec(
				`insert into rolemsgs (msgid, emoji, roleid) values (?, ?, ?)`,
				sent.ID, emojiId, roleId)
		} else if ob != nil {
			roleMsg := schema.RoleMsg{
				MsgId:  sent.ID,
				Emoji:  emojiId,
				RoleId: roleId,
			}
			_, err = schema.BoxForRoleMsg(ob).Put(&roleMsg)
		}
		if err != nil {
			log.Printf("Error responding to discord bot !alerts: %s", err.Error())
		}
		err = s.MessageReactionAdd(sent.ChannelID, sent.ID, emoji)
		if err != nil {
			log.Printf("Error responding to discord bot !alerts: %s", err.Error())
		}
	}
}

func DiscordReactionAdd(database *sql.DB, ob *objectbox.ObjectBox) func(s *discordgo.Session, msg *discordgo.MessageReactionAdd) {
	return func(s *discordgo.Session, msg *discordgo.MessageReactionAdd) {
		if database != nil {
			tx, err := database.Begin()
			if err != nil {
				log.Printf("%s", err.Error())
				return
			}
			defer func() {
				_ = tx.Rollback()
			}()
			row := tx.QueryRow(`select emoji, roleid from rolemsgs where msgid = ?`, msg.MessageID)
			var emojiID string
			var roleID string
			err = row.Scan(&emojiID, &roleID)
			if err == nil {
				if emojiID == msg.Emoji.ID {
					err = s.GuildMemberRoleAdd(msg.GuildID, msg.UserID, roleID)
					if err != nil {
						log.Printf("%s", err.Error())
					}
				}
			} else if errors.Is(err, sql.ErrNoRows) {
				row = tx.QueryRow(`select draft, round from pairingmsgs where msgid = ?`, msg.MessageID)
				var draftID int64
				var round int
				err = row.Scan(&draftID, &round)
				if err == nil {
					row = tx.QueryRow(`select id from users where discord_id = ?`, msg.UserID)
					var user int
					err = row.Scan(&user)
					if err == nil {
						if msg.Emoji.Name == "🏆" {
							_, err = tx.Exec(`insert into results (draft, round, user, win, timestamp) values (?, ?, ?, 1, datetime('now'))`,
								draftID, round, user)
						} else if msg.Emoji.Name == "💀" {
							_, err = tx.Exec(`insert into results (draft, round, user, win, timestamp) values (?, ?, ?, 0, datetime('now'))`,
								draftID, round, user)
						}
						if err != nil {
							log.Printf("%s", err.Error())
						}
						CheckNextRoundPairings(tx, draftID, round)
					} else {
						log.Printf("%s", err.Error())
					}
				} else if !errors.Is(err, sql.ErrNoRows) {
					log.Printf("%s", err.Error())
				}
			} else {
				log.Printf("%s", err.Error())
			}
			err = tx.Commit()
			if err != nil {
				log.Printf("%s", err.Error())
			}
		} else if ob != nil {
			err := ob.RunInWriteTx(func() error {
				roleMsgs, err := schema.BoxForRoleMsg(ob).Query(schema.RoleMsg_.MsgId.Equals(msg.MessageID, true)).Find()
				if err != nil {
					log.Printf("%s", err.Error())
					return err
				}
				if len(roleMsgs) > 0 {
					if roleMsgs[0].Emoji == msg.Emoji.ID {
						err = s.GuildMemberRoleAdd(msg.GuildID, msg.UserID, roleMsgs[0].RoleId)
						if err != nil {
							log.Printf("%s", err.Error())
							return err
						}
					}
				} else {
					pairingMsgs, err := schema.BoxForPairingMsg(ob).Query(schema.PairingMsg_.MsgId.Equals(msg.MessageID, true)).Find()
					if err != nil {
						log.Printf("%s", err.Error())
						return err
					}
					if len(pairingMsgs) == 0 {
						return nil
					}
					users, err := schema.BoxForUser(ob).Query(schema.User_.DiscordId.Equals(msg.UserID, true)).Find()
					if err != nil {
						log.Printf("%s", err.Error())
						return err
					}
					if len(users) == 0 {
						return fmt.Errorf("couldn't find user %s", msg.UserID)
					}
					if msg.Emoji.Name == "🏆" {
						_, err = schema.BoxForResult(ob).Put(&schema.Result{
							Draft:     pairingMsgs[0].Draft,
							Round:     pairingMsgs[0].Round,
							User:      users[0],
							Win:       true,
							Timestamp: time.Now(),
						})
					} else if msg.Emoji.Name == "💀" {
						_, err = schema.BoxForResult(ob).Put(&schema.Result{
							Draft:     pairingMsgs[0].Draft,
							Round:     pairingMsgs[0].Round,
							User:      users[0],
							Win:       false,
							Timestamp: time.Now(),
						})
					}
					if err != nil {
						log.Printf("%s", err.Error())
						return err
					}
					CheckNextRoundPairingsOb(ob, pairingMsgs[0].Draft, pairingMsgs[0].Round)
				}
				return nil
			})
			if err != nil {
				log.Printf("Error handling discord bot reaction add: %s", err.Error())
			}
		}
	}
}

func DiscordReactionRemove(database *sql.DB, ob *objectbox.ObjectBox) func(s *discordgo.Session, msg *discordgo.MessageReactionRemove) {
	return func(s *discordgo.Session, msg *discordgo.MessageReactionRemove) {
		if database != nil {
			tx, err := database.Begin()
			if err != nil {
				log.Printf("%s", err.Error())
				return
			}
			defer func() {
				_ = tx.Rollback()
			}()
			row := tx.QueryRow(`select emoji, roleid from rolemsgs where msgid = ?`, msg.MessageID)
			var emojiID string
			var roleID string
			err = row.Scan(&emojiID, &roleID)
			if err == nil {
				if emojiID == msg.Emoji.ID {
					err = s.GuildMemberRoleRemove(msg.GuildID, msg.UserID, roleID)
					if err != nil {
						log.Printf("%s", err.Error())
					}
				}
			} else if errors.Is(err, sql.ErrNoRows) {
				row = tx.QueryRow(`select draft, round from pairingmsgs where msgid = ?`, msg.MessageID)
				var draftID int
				var round int
				err = row.Scan(&draftID, &round)
				if err == nil {
					row = tx.QueryRow(`select id from users where discord_id = ?`, msg.UserID)
					var user int
					err = row.Scan(&user)
					if err == nil {
						if msg.Emoji.Name == "🏆" {
							_, err = tx.Exec(`delete from results where draft = ? and round = ? and user = ? and win = 1`,
								draftID, round, user)
						} else if msg.Emoji.Name == "💀" {
							_, err = tx.Exec(`delete from results where draft = ? and round = ? and user = ? and win = 0`,
								draftID, round, user)
						}
					}
					if err != nil {
						log.Printf("%s", err.Error())
					}
				} else if !errors.Is(err, sql.ErrNoRows) {
					log.Printf("%s", err.Error())
				}
			} else {
				log.Printf("%s", err.Error())
			}
			err = tx.Commit()
			if err != nil {
				log.Printf("%s", err.Error())
			}
		} else if ob != nil {
			err := ob.RunInWriteTx(func() error {
				roleMsgs, err := schema.BoxForRoleMsg(ob).Query(schema.RoleMsg_.MsgId.Equals(msg.MessageID, true)).Find()
				if err != nil {
					return err
				}
				if len(roleMsgs) > 0 {
					if roleMsgs[0].Emoji == msg.Emoji.ID {
						return s.GuildMemberRoleRemove(msg.GuildID, msg.UserID, roleMsgs[0].RoleId)
					}
					return nil
				}
				pairingMsgs, err := schema.BoxForPairingMsg(ob).Query(schema.PairingMsg_.MsgId.Equals(msg.MessageID, true)).Find()
				if err != nil {
					return err
				}
				if len(pairingMsgs) > 0 {
					users, err := schema.BoxForUser(ob).Query(schema.User_.DiscordId.Equals(msg.UserID, true)).Find()
					if err != nil {
						return err
					}
					if len(users) == 0 {
						return fmt.Errorf("couldn't find user %s", msg.UserID)
					}
					resultBox := schema.BoxForResult(ob)
					results, err := resultBox.Query(schema.Result_.Draft.Equals(pairingMsgs[0].Draft.Id)).Find()
					if err != nil {
						return err
					}
					if len(results) > 0 {
						var win bool
						if msg.Emoji.Name == "🏆" {
							win = true
						} else if msg.Emoji.Name == "💀" {
							win = false
						} else {
							return nil
						}
						for _, result := range results {
							if result.User.Id == users[0].Id &&
								result.Round == pairingMsgs[0].Round &&
								result.Win == win {
								err = resultBox.Remove(result)
								if err != nil {
									return err
								}
							}
						}
					}
				}
				return nil
			})
			if err != nil {
				log.Printf("Error handling discord bot reaction add: %s", err.Error())
			}
		}
	}
}

func CheckNextRoundPairings(tx *sql.Tx, draftID int64, round int) {
	row := tx.QueryRow(`select count(1) from results where draft = ? and round <= ?`, draftID, round)
	var numResults int
	err := row.Scan(&numResults)
	if err != nil {
		log.Printf("%s", err.Error())
		return
	}
	if numResults == round*8 {
		var table1 []string
		var table2 []string
		var table3 []string
		var table4 []string
		if round == 1 {
			// Pair round 2
			result, err := tx.Query(
				`select
							users.discord_name, users.discord_id, win, seats.position
							from results
							join seats on seats.draft = results.draft and seats.user = results.user
							join users on users.id = results.user
							where results.draft = ? and results.round = ?`,
				draftID, round)
			if err != nil {
				log.Printf("%s", err.Error())
				return
			}
			for result.Next() {
				var discordName string
				var discordId sql.NullString
				var win int
				var seat int
				err = result.Scan(&discordName, &discordId, &win, &seat)
				if err != nil {
					log.Printf("%s", err.Error())
					return
				}
				var player string
				if discordId.Valid {
					player = fmt.Sprintf("<@%s>", discordId.String)
				} else {
					player = discordName
				}
				if win == 1 {
					if seat%2 == 0 {
						table1 = append(table1, player)
					} else {
						table2 = append(table2, player)
					}
				} else {
					if seat%2 == 0 {
						table3 = append(table3, player)
					} else {
						table4 = append(table4, player)
					}
				}
			}
		} else if round == 2 {
			// Pair round 3
			result, err := tx.Query(`select
						users.discord_name, users.discord_id, sum(win), seats.position
						from results
						join seats on seats.draft = results.draft and seats.user = results.user
						join users on users.id = results.user
						where results.draft = ? and results.round <= ?
						group by results.user`,
				draftID, round)
			if err != nil {
				log.Printf("%s", err.Error())
				return
			}
			for result.Next() {
				var discordName string
				var discordId sql.NullString
				var wins int
				var seat int
				err = result.Scan(&discordName, &discordId, &wins, &seat)
				if err != nil {
					log.Printf("%s", err.Error())
					return
				}
				var player string
				if discordId.Valid {
					player = fmt.Sprintf("<@%s>", discordId.String)
				} else {
					player = discordName
				}
				if wins == 2 {
					table1 = append(table1, player)
				} else if wins == 0 {
					table4 = append(table4, player)
				} else {
					if seat < 4 {
						if len(table2) < 2 {
							table2 = append(table2, player)
						} else {
							table3 = append(table3, player)
						}
					} else {
						if len(table3) < 2 {
							table3 = append(table3, player)
						} else {
							table2 = append(table2, player)
						}
					}
				}
			}
		} else if round == 3 {
			if dg != nil {
				row := tx.QueryRow(`select
							users.discord_name, users.discord_id
							from results
							join seats on seats.draft = results.draft and seats.user = results.user
							join users on users.id = results.user
							where results.draft = ?
							group by results.user
							having sum(win) = 3`,
					draftID, round)
				var discordName string
				var discordId sql.NullString
				err = row.Scan(&discordName, &discordId)
				if err != nil {
					log.Printf("%s", err.Error())
					return
				}
				var player string
				if discordId.Valid {
					player = fmt.Sprintf("<@%s>", discordId.String)
				} else {
					player = discordName
				}
				adminDiscordID, err := GetAdminDiscordId(tx)
				if err != nil {
					log.Printf("%s", err.Error())
					return
				}
				draftName, err := GetDraftName(tx, draftID)
				if err != nil {
					log.Printf("%s", err.Error())
					return
				}
				_, err = dg.ChannelMessageSend(os.Getenv("DRAFT_ANNOUNCEMENTS_CHANNEL_ID"),
					fmt.Sprintf("Congratulations to %s, winner of *%s*!\n\n"+
						"All players, please ping <@%s> directly when you're ready to return cards.",
						player, draftName, adminDiscordID))
				if err != nil {
					log.Printf("%s", err.Error())
					return
				}
			}
		}
		if len(table1) == 2 && len(table2) == 2 && len(table3) == 2 && len(table4) == 2 {
			pairings := fmt.Sprintf(`%s vs %s
%s vs %s
%s vs %s
%s vs %s`,
				table1[0], table1[1],
				table2[0], table2[1],
				table3[0], table3[1],
				table4[0], table4[1])
			draftName, err := GetDraftName(tx, draftID)
			if err != nil {
				log.Printf("%s", err.Error())
				return
			}
			err = PostPairings(tx, draftID, draftName, round+1, pairings)
			if err != nil {
				log.Printf("%s", err.Error())
				return
			}
			err = tx.Commit()
			if err != nil {
				log.Printf("%s", err.Error())
				return
			}
		}
	}
}

func CheckNextRoundPairingsOb(ob *objectbox.ObjectBox, draft *schema.Draft, round int) {
	results, err := schema.BoxForResult(ob).Query(schema.Result_.Draft.Equals(draft.Id), schema.Result_.Round.LessOrEqual(round)).Find()
	if err != nil {
		log.Printf("%s", err.Error())
		return
	}
	numSeats, _ := getNumSeatsAndCardsPerPack(draft)
	if len(results) == round*numSeats {
		var users []*schema.User
		var wins [8]int
		for _, result := range results {
			userIndex := slices.IndexFunc(users, func(user *schema.User) bool {
				return user.Id == result.User.Id
			})
			if userIndex <= -1 {
				userIndex = len(users)
				users = append(users, result.User)
			}
			if result.Win {
				wins[userIndex]++
			}
		}
		var table1 []string
		var table2 []string
		var table3 []string
		var table4 []string
		if round == 1 {
			// Pair round 2
			for i, user := range users {
				seat := slices.IndexFunc(draft.Seats, func(seat *schema.Seat) bool {
					return seat.User.Id == user.Id
				})
				if seat == -1 {
					log.Printf("found error for user %d who does not have a seat in draft %d", user.Id, draft.Id)
					return
				}
				var player string
				if len(user.DiscordId) > 0 {
					player = fmt.Sprintf("<@%s>", user.DiscordId)
				} else {
					player = user.DiscordName
				}
				if wins[i] == 1 {
					if seat%2 == 0 {
						table1 = append(table1, player)
					} else {
						table2 = append(table2, player)
					}
				} else {
					if seat%2 == 0 {
						table3 = append(table3, player)
					} else {
						table4 = append(table4, player)
					}
				}
			}
		} else if round == 2 {
			var users []*schema.User
			var wins [8]int
			for _, result := range results {
				userIndex := slices.IndexFunc(users, func(user *schema.User) bool {
					return user.Id == result.User.Id
				})
				if userIndex <= -1 {
					userIndex = len(users)
					users = append(users, result.User)
				}
				if result.Win {
					wins[userIndex]++
				}
			}
			for i, user := range users {
				seat := slices.IndexFunc(draft.Seats, func(seat *schema.Seat) bool {
					return seat.User.Id == user.Id
				})
				if seat == -1 {
					log.Printf("found error for user %d who does not have a seat in draft %d", user.Id, draft.Id)
					return
				}
				wins := wins[i]
				var player string
				if len(user.DiscordId) > 0 {
					player = fmt.Sprintf("<@%s>", user.DiscordId)
				} else {
					player = user.DiscordName
				}
				if wins == 2 {
					table1 = append(table1, player)
				} else if wins == 0 {
					table4 = append(table4, player)
				} else {
					if seat < 4 {
						if len(table2) < 2 {
							table2 = append(table2, player)
						} else {
							table3 = append(table3, player)
						}
					} else {
						if len(table3) < 2 {
							table3 = append(table3, player)
						} else {
							table2 = append(table2, player)
						}
					}
				}
			}
		} else if round == 3 {
			if dg != nil {
				winnerIndex := slices.Index(wins[:], 3)
				if winnerIndex == -1 {
					log.Printf("no winner found in draft %d", draft.Id)
					return
				}
				winner := users[winnerIndex]
				var player string
				if len(winner.DiscordId) > 0 {
					player = fmt.Sprintf("<@%s>", winner.DiscordId)
				} else {
					player = winner.DiscordName
				}
				adminDiscordID, err := GetAdminDiscordIdOb(ob)
				_, err = dg.ChannelMessageSend(os.Getenv("DRAFT_ANNOUNCEMENTS_CHANNEL_ID"),
					fmt.Sprintf("Congratulations to %s, winner of *%s*!\n\n"+
						"All players, please ping <@%s> directly when you're ready to return cards.",
						player, draft.Name, adminDiscordID))
				if err != nil {
					log.Printf("%s", err.Error())
					return
				}
			}
		}
		if len(table1) == 2 && len(table2) == 2 && len(table3) == 2 && len(table4) == 2 {
			pairings := fmt.Sprintf(`%s vs %s
%s vs %s
%s vs %s
%s vs %s`,
				table1[0], table1[1],
				table2[0], table2[1],
				table3[0], table3[1],
				table4[0], table4[1])
			err = PostPairingsOb(ob, draft, round+1, pairings)
			if err != nil {
				log.Printf("%s", err.Error())
				return
			}
		}
	}
}

func ArchiveSpectatorChannels(db *sql.DB, ob *objectbox.ObjectBox) error {
	if dg != nil {
		log.Printf("archiving spectator channels")
		if db != nil {
			tx, err := db.Begin()
			if err != nil {
				log.Printf("error archiving spectator channels: %s", err.Error())
				return err
			}
			defer func() {
				_ = tx.Rollback()
			}()

			query := `select
					spectatorchannelid
					from drafts
					join results on results.draft = drafts.id
					group by results.draft
					having count(results.id) = 24 and max(results.timestamp) < datetime('now', '-3 days')`
			result, err := tx.Query(query)
			if err != nil {
				log.Printf("error archiving spectator channels: %s", err.Error())
				return err
			}
			var channels []interface{}
			for result.Next() {
				var channelID string
				err = result.Scan(&channelID)
				if err != nil {
					log.Printf("error archiving spectator channels: %s", err.Error())
					return err
				}
				log.Printf("locking channel %s", channelID)
				err = dg.ChannelPermissionSet(channelID, EveryoneRole, 0, 0, discordgo.PermissionViewChannel)
				if err != nil {
					log.Printf("error archiving spectator channels: %s", err.Error())
					return err
				}
				channels = append(channels, channelID)
			}

			if len(channels) > 0 {
				query = `update drafts set spectatorchannelid = null where spectatorchannelid in (?` + strings.Repeat(",?", len(channels)-1) + `)`
				_, err = tx.Exec(query, channels...)
				if err != nil {
					log.Printf("error archiving spectator channels: %s", err.Error())
					return err
				}
			}

			err = tx.Commit()
			if err != nil {
				log.Printf("error archiving spectator channels: %s", err.Error())
				return err
			}
		} else if ob != nil {
			err := ob.RunInWriteTx(func() error {
				draftBox := schema.BoxForDraft(ob)
				drafts, err := draftBox.Query(schema.Draft_.SpectatorChannelId.NotEquals("", false)).Find()
				if err != nil {
					return err
				}
				for _, draft := range drafts {
					threeDaysAgo, err := objectbox.TimeInt64ConvertToDatabaseValue(time.Now().Add(time.Duration(-36) * time.Hour))
					if err != nil {
						return err
					}
					resultsCount, err := schema.BoxForResult(ob).Query(schema.Result_.Draft.Equals(draft.Id),
						schema.Result_.Timestamp.LessOrEqual(threeDaysAgo)).Count()
					if resultsCount == 24 {
						channelId := draft.SpectatorChannelId
						draft.SpectatorChannelId = ""
						log.Printf("locking channel %s", channelId)
						err = dg.ChannelPermissionSet(channelId, EveryoneRole, 0, 0, discordgo.PermissionViewChannel)
						if err != nil {
							log.Printf("error archiving spectator channels: %s", err.Error())
							return err
						}
					}
				}
				_, err = draftBox.PutMany(drafts)
				if err != nil {
					return err
				}
				return nil
			})
			if err != nil {
				return fmt.Errorf("error archiving spectator channels: %w", err)
			}
		}
	}

	log.Printf("done archiving spectator channels")
	return nil
}
