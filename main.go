package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/objectbox/objectbox-go/objectbox"
	"github.com/walkingeyerobot/r38/draftconfig"
	"github.com/walkingeyerobot/r38/schema"
	"io"
	"log"
	mathrand "math/rand/v2"
	"net"
	"net/http"
	"os"
	"os/signal"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/walkingeyerobot/r38/makedraft"

	"golang.org/x/net/xsrftoken"

	"github.com/bwmarrin/discordgo"
	"github.com/go-co-op/gocron"
	"github.com/google/shlex"
	"github.com/gorilla/sessions"
)

type r38handler func(w http.ResponseWriter, r *http.Request, userId int64, ob *objectbox.ObjectBox) error

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

type DiscordCall struct {
	Type      string
	ChannelId string
	Message   string
}

var ignoredDiscordCalls []DiscordCall

func main() {
	useAuthPtr := flag.Bool("auth", true, "bool")
	flag.Bool("objectbox", false, "bool")
	flag.String("dbfile", "draft.db", "string")
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

	var ob *objectbox.ObjectBox
	var err error
	ob, err = objectbox.NewBuilder().Model(schema.ObjectBoxModel()).
		Directory(*dbDir).Build()
	if err != nil {
		log.Printf("error opening db: %s", err.Error())
		return
	}
	defer ob.Close()

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
		Handler: NewHandler(ob, *useAuthPtr),
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
		dg.AddHandler(DiscordMsgCreate(ob))
		dg.AddHandler(DiscordReactionAdd(ob))
		dg.AddHandler(DiscordReactionRemove(ob))
		err = dg.Open()
		if err != nil {
			log.Printf("Error initializing discord bot: %s", err.Error())
		} else {
			log.Printf("Discord bot initialized.")
		}
	}

	scheduler := gocron.NewScheduler(time.UTC)
	_, err = scheduler.Every(8).Hours().Do(ArchiveSpectatorChannels, ob)
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
func NewHandler(ob *objectbox.ObjectBox, useAuth bool) http.Handler {
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
			handle := func() error {
				err := serveFunc(w, r, userID, ob)
				if err != nil {
					log.Printf("error serving %s: %s", route, err.Error())
				}
				return err
			}
			if readonly {
				err = ob.RunInReadTx(handle)
			} else {
				err = ob.RunInWriteTx(handle)
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
	addHandler("/api/draftpacks/", ServeAPIDraftPacks, true)
	addHandler("/api/pick/", ServeAPIPick, false)
	addHandler("/api/pickrfid/", ServeAPIPickRfid, false)
	addHandler("/api/join/", ServeAPIJoin, false)
	addHandler("/api/skip/", ServeAPISkip, false)
	addHandler("/api/prefs/", ServeAPIPrefs, true)
	addHandler("/api/setpref/", ServeAPISetPref, false)
	addHandler("/api/undopick/", ServeAPIUndoPick, false)
	addHandler("/api/userinfo/", ServeAPIUserInfo, true)
	addHandler("/api/getcardpack/", ServeAPIGetCardPack, true)
	addHandler("/api/samplepack/", ServeAPISamplePack, true)
	addHandler("/api/set/", ServeAPISet, true)

	addHandler("/api/dev/forceEnd/", ServeAPIForceEnd, false)

	mux.Handle("/", http.HandlerFunc(HandleIndex))

	return mux
}

func HandleIndex(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "client-dist/index.html")
}

// ServeAPIArchive serves the /api/archive endpoint.
func ServeAPIArchive(_ http.ResponseWriter, r *http.Request, userID int64, ob *objectbox.ObjectBox) error {
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
func ServeAPIDraft(w http.ResponseWriter, r *http.Request, userID int64, ob *objectbox.ObjectBox) error {
	re := regexp.MustCompile(`/api/draft/(\d+)`)
	parseResult := re.FindStringSubmatch(r.URL.Path)
	if parseResult == nil {
		return fmt.Errorf("bad api url")
	}
	draftID, err := strconv.ParseInt(parseResult[1], 10, 64)
	if err != nil {
		return fmt.Errorf("bad api url: %w", err)
	}

	draftJSON, err := GetFilteredJSON(ob, draftID, userID)
	if err != nil {
		return fmt.Errorf("error getting json: %w", err)
	}

	_, err = fmt.Fprint(w, draftJSON)
	return err
}

// ServeAPIDraftList serves the /api/draftlist endpoint.
func ServeAPIDraftList(w http.ResponseWriter, _ *http.Request, userId int64, ob *objectbox.ObjectBox) error {
	drafts, err := GetDraftList(userId, ob)
	if err != nil {
		return err
	}
	return json.NewEncoder(w).Encode(drafts)
}

// ServeAPIDraftPacks serves the /api/draftpacks endpoint.
func ServeAPIDraftPacks(w http.ResponseWriter, r *http.Request, userId int64, ob *objectbox.ObjectBox) error {
	if userId != 1 {
		return fmt.Errorf("not allowed")
	}
	re := regexp.MustCompile(`/api/draftpacks/(\d+)`)
	parseResult := re.FindStringSubmatch(r.URL.Path)
	if parseResult == nil {
		return fmt.Errorf("bad api url")
	}
	draftID, err := strconv.ParseInt(parseResult[1], 10, 64)
	if err != nil {
		return fmt.Errorf("bad api url: %w", err)
	}

	draft, err := schema.BoxForDraft(ob).Get(uint64(draftID))
	if err != nil {
		return fmt.Errorf("error getting draft: %w", err)
	}

	var clientPacks [][]R38CardData
	for _, seat := range draft.Seats {
		for _, pack := range seat.OriginalPacks {
			var clientPack []R38CardData
			for _, card := range pack.OriginalCards {
				var cardData R38CardData
				err = json.Unmarshal([]byte(card.Data), &cardData)
				if err != nil {
					return fmt.Errorf("error unmarshalling: %w", err)
				}
				clientPack = append(clientPack, cardData)
			}
			clientPacks = append(clientPacks, clientPack)
		}
	}
	for _, pack := range draft.UnassignedPacks {
		var clientPack []R38CardData
		for _, card := range pack.OriginalCards {
			var cardData R38CardData
			err = json.Unmarshal([]byte(card.Data), &cardData)
			if err != nil {
				return fmt.Errorf("error unmarshalling: %w", err)
			}
			clientPack = append(clientPack, cardData)
		}
		clientPacks = append(clientPacks, clientPack)
	}

	err = json.NewEncoder(w).Encode(clientPacks)
	if err != nil {
		return fmt.Errorf("error marshalling: %w", err)
	}

	return nil
}

// ServeAPIPrefs serves the /api/prefs endpoint.
func ServeAPIPrefs(w http.ResponseWriter, _ *http.Request, userId int64, _ *objectbox.ObjectBox) error {
	prefs, err := GetUserPrefs(userId)
	if err != nil {
		return err
	}
	return json.NewEncoder(w).Encode(prefs)
}

// ServeAPISetPref serves the /api/setpref endpoint.
func ServeAPISetPref(w http.ResponseWriter, r *http.Request, userId int64, ob *objectbox.ObjectBox) error {
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

	if pref.MtgoName != "" {
		if ob != nil {
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

	return ServeAPIPrefs(w, r, userId, ob)
}

// ServeAPIPick serves the /api/pick endpoint.
func ServeAPIPick(w http.ResponseWriter, r *http.Request, userID int64, ob *objectbox.ObjectBox) error {
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

	err = doHandlePostedPick(w, pick, userID, false, ob)
	if err != nil {
		return err
	}
	return nil
}

// ServeAPIPickRfid serves the /api/pickrfid endpoint.
func ServeAPIPickRfid(w http.ResponseWriter, r *http.Request, userId int64, ob *objectbox.ObjectBox) error {
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

	var pick = PostedPick{
		DraftId:   rfidPick.DraftId,
		CardIds:   cardIds,
		XsrfToken: rfidPick.XsrfToken,
	}

	err = doHandlePostedPick(w, pick, userId, true, ob)
	if err != nil {
		return err
	}
	return nil
}

func doHandlePostedPick(w http.ResponseWriter, pick PostedPick, userId int64, zoneDrafting bool, ob *objectbox.ObjectBox) error {
	var err error
	if len(pick.CardIds) == 1 {
		err = doSinglePick(ob, userId, pick.DraftId, pick.CardIds[0], zoneDrafting)
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
	draftJSON, err = GetFilteredJSON(ob, pick.DraftId, userId)
	if err != nil {
		return fmt.Errorf("error getting json: %w", err)
	}

	_, err = fmt.Fprint(w, draftJSON)
	return err
}

// ServeAPIUndoPick serves the /api/undopick endpoint.
func ServeAPIUndoPick(_ http.ResponseWriter, r *http.Request, userID int64, ob *objectbox.ObjectBox) error {
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
	return err
}

// ServeAPIJoin serves the /api/join endpoint.
func ServeAPIJoin(w http.ResponseWriter, r *http.Request, userID int64, ob *objectbox.ObjectBox) error {
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
		err = doJoin(ob, userID, draftID)
	} else if toJoin.Position > 7 {
		return fmt.Errorf("invalid position %d", toJoin.Position)
	} else {
		err = doJoinSeatPosition(ob, userID, draftID, toJoin.Position)
	}
	if err != nil {
		return fmt.Errorf("error joining draft %d: %w", draftID, err)
	}

	draftJSON, err := GetFilteredJSON(ob, draftID, userID)
	if err != nil {
		return fmt.Errorf("error getting json: %w", err)
	}

	_, err = fmt.Fprint(w, draftJSON)
	return err
}

// doJoin does the actual joining.
func doJoin(ob *objectbox.ObjectBox, userId int64, draftId int64) error {
	draft, err := schema.BoxForDraft(ob).Get(uint64(draftId))
	if err != nil {
		return err
	}

	var reservedSeat *schema.Seat
	var openSeats []*schema.Seat
	for _, seat := range draft.Seats {
		if seat.User != nil && seat.User.Id == uint64(userId) {
			return fmt.Errorf("user %d already joined %d", userId, draftId)
		}
		if seat.ReservedUser != nil && seat.ReservedUser.Id == uint64(userId) {
			reservedSeat = seat
		} else if seat.User == nil {
			openSeats = append(openSeats, seat)
		}
	}

	if reservedSeat != nil {
		err = doJoinSeat(ob, userId, draft, reservedSeat)
	} else if len(openSeats) > 0 {
		err = doJoinSeat(ob, userId, draft, openSeats[mathrand.IntN(len(openSeats))])
	} else {
		return fmt.Errorf("no non-reserved seats available for user %d in draft %d", userId, draftId)
	}

	return err
}

// doJoinSeatPosition joins at a specific seat Position.
func doJoinSeatPosition(ob *objectbox.ObjectBox, userId int64, draftId int64, position int64) error {
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

	err = doJoinSeat(ob, userId, draft, seat)
	return err
}

func doJoinSeat(ob *objectbox.ObjectBox, userId int64, draft *schema.Draft, seat *schema.Seat) error {
	user, err := schema.BoxForUser(ob).Get(uint64(userId))
	if err != nil {
		return err
	}

	seat.User = user
	_, err = schema.BoxForSeat(ob).Put(seat)
	if err != nil {
		return err
	}

	if dg != nil && draft.SpectatorChannelId != "" && user.DiscordId != "" {
		err = dg.ChannelPermissionSet(draft.SpectatorChannelId, user.DiscordId, 1, 0, discordgo.PermissionViewChannel)
		if err != nil {
			log.Printf("error locking spectator channel for user %s: %s", user.DiscordId, err.Error())
		}
	} else {
		ignoredDiscordCalls = append(ignoredDiscordCalls, DiscordCall{
			Type:      "lockChannel",
			ChannelId: draft.SpectatorChannelId,
			Message:   user.DiscordId,
		})
	}

	return err
}

// ServeAPISkip serves the /api/skip endpoint.
func ServeAPISkip(w http.ResponseWriter, r *http.Request, userId int64, ob *objectbox.ObjectBox) error {
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

	err = doSkip(ob, userId, draftId)
	if err != nil {
		return fmt.Errorf("error skipping draft %d: %w", draftId, err)
	}

	draftJSON, err := GetFilteredJSON(ob, draftId, userId)
	if err != nil {
		return fmt.Errorf("error getting json: %w", err)
	}

	_, err = fmt.Fprint(w, draftJSON)
	return err
}

// doSkip does the actual skipping.
func doSkip(ob *objectbox.ObjectBox, userId int64, draftId int64) error {
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

	newUser, err := makedraft.AssignSeats(ob, draftId, 1)
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
func ServeAPIForceEnd(_ http.ResponseWriter, r *http.Request, userID int64, ob *objectbox.ObjectBox) error {
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
		return NotifyEndOfDraft(ob, draftID)
	} else {
		return http.ErrBodyNotAllowed
	}
}

// ServeAPIUserInfo serves the /api/userinfo endpoint.
func ServeAPIUserInfo(w http.ResponseWriter, _ *http.Request, userID int64, ob *objectbox.ObjectBox) error {
	var userInfoJSON []byte
	var err error
	userInfoJSON, err = getUserJSON(userID, ob)
	if err != nil {
		return err
	}

	_, err = w.Write(userInfoJSON)
	return err
}

// ServeAPIGetCardPack serves the /api/getcardpack endpoint.
func ServeAPIGetCardPack(w http.ResponseWriter, r *http.Request, _ int64, ob *objectbox.ObjectBox) error {
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		return fmt.Errorf("error reading post body: %w", err)
	}
	var getCardPack PostedGetCardPack
	err = json.Unmarshal(bodyBytes, &getCardPack)
	if err != nil {
		return fmt.Errorf("error parsing post body: %w", err)
	}

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

// ServeAPISamplePack serves the /api/samplepack endpoint.
func ServeAPISamplePack(w http.ResponseWriter, r *http.Request, _ int64, _ *objectbox.ObjectBox) error {
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		return fmt.Errorf("error reading post body: %w", err)
	}
	var setRequest PostedSamplePack
	err = json.Unmarshal(bodyBytes, &setRequest)
	if err != nil {
		return fmt.Errorf("error parsing post body: %w", err)
	}

	setPath := "sets/" + setRequest.Set + ".json"
	settings, err := makedraft.ParseSettings([]string{"", "--set", setPath, "--seed", strconv.Itoa(setRequest.Seed)})
	if err != nil {
		return err
	}
	err = makedraft.AddDraftConfigSettings(&settings)
	if err != nil {
		return err
	}

	packs, err := makedraft.GeneratePacks(settings)
	if err != nil {
		return fmt.Errorf("error generating pack: %w", err)
	}

	pack := packs[0]

	var clientPack []R38CardData
	for _, card := range pack {
		var cardData R38CardData
		err = json.Unmarshal([]byte(card.Data), &cardData)
		if err != nil {
			return fmt.Errorf("error unmarshalling: %w", err)
		}
		clientPack = append(clientPack, cardData)
	}

	err = json.NewEncoder(w).Encode(clientPack)
	if err != nil {
		return fmt.Errorf("error marshalling: %w", err)
	}

	return nil
}

// ServeAPISet serves the /api/set endpoint.
func ServeAPISet(w http.ResponseWriter, r *http.Request, _ int64, _ *objectbox.ObjectBox) error {
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

func getUserJSON(userId int64, ob *objectbox.ObjectBox) ([]byte, error) {
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

// doSinglePick performs a normal pick based on a user id and a card id.
func doSinglePick(ob *objectbox.ObjectBox, userId int64, draftId int64, cardId int64, zoneDrafting bool) error {
	packID, announcements, round, seat, err := doPick(ob, userId, draftId, cardId, zoneDrafting)
	if err != nil {
		return err
	}
	err = doEvent(ob, draftId, announcements, cardId, nil, packID, seat, round)
	return err
}

// doPick actually performs a pick in the database.
// It returns the packID, announcements, round, and an error.
// Of those return values, packID and announcements are only really relevant for Cogwork Librarian,
// which is not currently fully implemented, but we leave them here anyway for when we want to do that.
func doPick(ob *objectbox.ObjectBox, userId int64, draftId int64, cardId int64, zoneDrafting bool) (int64, []string, int64, *schema.Seat, error) {
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
		hasPacksLeft := slices.ContainsFunc(seat.Packs, func(pack *schema.Pack) bool {
			return pack.Round == int(round)
		})

		if !hasPacksLeft {
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
					err = NotifyEndOfDraft(ob, draftId)
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

func NotifyEndOfDraft(ob *objectbox.ObjectBox, draftID int64) error {
	draft, err := schema.BoxForDraft(ob).Get(uint64(draftID))
	if err != nil {
		return err
	}

	err = PostFirstRoundPairings(ob, draft)
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

	return nil
}

func PostFirstRoundPairings(ob *objectbox.ObjectBox, draft *schema.Draft) error {
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
	err := PostPairings(ob, draft, round, pairings)
	return err
}

func PostPairings(ob *objectbox.ObjectBox, draft *schema.Draft, round int, pairings string) error {
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

func GetAdminDiscordId(ob *objectbox.ObjectBox) (string, error) {
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
	} else {
		ignoredDiscordCalls = append(ignoredDiscordCalls, DiscordCall{
			Type:      "unlock",
			ChannelId: channelId,
		})
	}
	return nil
}

// DiscordNotify posts a message to discord.
func DiscordNotify(channelId string, message string) error {
	if dg != nil {
		_, err := dg.ChannelMessageSend(channelId, message)
		return err
	} else {
		ignoredDiscordCalls = append(ignoredDiscordCalls, DiscordCall{
			Type:      "notify",
			ChannelId: channelId,
			Message:   message,
		})
	}
	return nil
}

// DiscordNotifyEmbed posts a message to discord.
func DiscordNotifyEmbed(channelId string, message *discordgo.MessageEmbed) (*discordgo.Message, error) {
	if dg != nil {
		msg, err := dg.ChannelMessageSendEmbed(channelId, message)
		return msg, err
	} else {
		ignoredDiscordCalls = append(ignoredDiscordCalls, DiscordCall{
			Type:      "notifyEmbed",
			ChannelId: channelId,
			Message:   fmt.Sprintf("%+v", message),
		})
	}
	return nil, nil
}

// GetJSONObject returns a better DraftJSON object. May be filtered.
func GetJSONObject(ob *objectbox.ObjectBox, draftId int64) (DraftJSON, error) {
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

	draftJson.Events = []DraftEvent{}
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
		eventJson.DraftModified = int64(event.Modified)
		draftJson.Events = append(draftJson.Events, eventJson)
	}

	return draftJson, err
}

// GetFilteredJSON returns a filtered json object of replay data.
func GetFilteredJSON(ob *objectbox.ObjectBox, draftId int64, userId int64) (string, error) {
	draftInfo, err := GetDraftListEntry(userId, ob, draftId)
	if err != nil {
		return "", fmt.Errorf("error getting draft list entry: %w", err)
	}

	var draft DraftJSON
	draft, err = GetJSONObject(ob, draftId)
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

	response := ""
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

		response = buff.String()
		if !strings.HasPrefix(response, "{") {
			return "", fmt.Errorf("error from filter.js: %s", response)
		}
	}

	return response, nil
}

// doEvent records an event (pick) into the database.
func doEvent(ob *objectbox.ObjectBox, draftId int64, announcements []string, cardId1 int64, cardId2 *int64, packId int64, seat *schema.Seat, round int64) error {
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
	if cardId2 != nil {
		card2, err := schema.BoxForCard(ob).Get(uint64(*cardId2))
		if err != nil {
			return err
		}
		draft.Events = append(draft.Events, &schema.Event{
			Position:     seat.Position,
			Announcement: strings.Join(announcements, "\n"),
			Card1:        card1,
			Card2:        card2,
			Pack:         pack,
			Modified:     len(draft.Events),
			Round:        int(round),
		})
	} else {
		draft.Events = append(draft.Events, &schema.Event{
			Position:     seat.Position,
			Announcement: strings.Join(announcements, "\n"),
			Card1:        card1,
			Card2:        nil,
			Pack:         pack,
			Modified:     len(draft.Events),
			Round:        int(round),
		})
	}

	_, err = draftBox.Put(draft)
	return err
}

func GetDraftList(userId int64, ob *objectbox.ObjectBox) (DraftList, error) {
	draftList := DraftList{
		Drafts: []DraftListEntry{},
	}
	drafts, err := schema.BoxForDraft(ob).Query(objectbox.Any(schema.Draft_.Archived.Equals(false),
		schema.Draft_.Archived.IsNil())).Find()
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

func GetDraftListEntry(userId int64, ob *objectbox.ObjectBox, draftId int64) (DraftListEntry, error) {
	var ret DraftListEntry

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

func GetUserPrefs(_ int64) (UserFormatPrefs, error) {
	var prefs UserFormatPrefs

	return prefs, nil
}

func DiscordReady(s *discordgo.Session, _ *discordgo.Ready) {
	err := s.UpdateCustomStatus("Tier 5 Wolf Combo")
	if err != nil {
		log.Printf("Error readying discord bot: %s", err.Error())
	}
}

func DiscordMsgCreate(ob *objectbox.ObjectBox) func(s *discordgo.Session, msg *discordgo.MessageCreate) {
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
					} else {
						err = ob.RunInWriteTx(func() error {
							return makedraft.MakeDraft(settings, ob)
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
				DiscordSendRoleReactionMessage(s, ob, msg.ChannelID,
					ForestBear, ForestBearId, DraftAlertsRole,
					"Draft alerts", "if you would like notifications for games being played")
			}
		}
	}
}

func DiscordSendRoleReactionMessage(s *discordgo.Session, ob *objectbox.ObjectBox, channelID string, emoji string, emojiId string, roleId string, title string, description string) {
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
		roleMsg := schema.RoleMsg{
			MsgId:  sent.ID,
			Emoji:  emojiId,
			RoleId: roleId,
		}
		_, err = schema.BoxForRoleMsg(ob).Put(&roleMsg)
		if err != nil {
			log.Printf("Error responding to discord bot !alerts: %s", err.Error())
		}
		err = s.MessageReactionAdd(sent.ChannelID, sent.ID, emoji)
		if err != nil {
			log.Printf("Error responding to discord bot !alerts: %s", err.Error())
		}
	}
}

func DiscordReactionAdd(ob *objectbox.ObjectBox) func(s *discordgo.Session, msg *discordgo.MessageReactionAdd) {
	return func(s *discordgo.Session, msg *discordgo.MessageReactionAdd) {
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
				CheckNextRoundPairings(ob, pairingMsgs[0].Draft, pairingMsgs[0].Round)
			}
			return nil
		})
		if err != nil {
			log.Printf("Error handling discord bot reaction add: %s", err.Error())
		}
	}
}

func DiscordReactionRemove(ob *objectbox.ObjectBox) func(s *discordgo.Session, msg *discordgo.MessageReactionRemove) {
	return func(s *discordgo.Session, msg *discordgo.MessageReactionRemove) {
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

func CheckNextRoundPairings(ob *objectbox.ObjectBox, draft *schema.Draft, round int) {
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
			adminDiscordID, err := GetAdminDiscordId(ob)
			channelId := os.Getenv("DRAFT_ANNOUNCEMENTS_CHANNEL_ID")
			message := fmt.Sprintf("Congratulations to %s, winner of *%s*!\n\n"+
				"All players, please ping <@%s> directly when you're ready to return cards.",
				player, draft.Name, adminDiscordID)
			if dg != nil {
				_, err = dg.ChannelMessageSend(channelId, message)
				if err != nil {
					log.Printf("%s", err.Error())
					return
				}
			} else {
				ignoredDiscordCalls = append(ignoredDiscordCalls, DiscordCall{
					Type:      "postWinner",
					ChannelId: channelId,
					Message:   message,
				})
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
			err = PostPairings(ob, draft, round+1, pairings)
			if err != nil {
				log.Printf("%s", err.Error())
				return
			}
		}
	}
}

func ArchiveSpectatorChannels(ob *objectbox.ObjectBox) error {
	if dg != nil {
		log.Printf("archiving spectator channels")
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
	} else {
		ignoredDiscordCalls = append(ignoredDiscordCalls, DiscordCall{
			Type: "archiveChannels",
		})
	}

	log.Printf("done archiving spectator channels")
	return nil
}
