package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"./makedraft"

	"github.com/bwmarrin/discordgo"
	"github.com/google/shlex"
	"github.com/gorilla/sessions"
	_ "github.com/mattn/go-sqlite3"
)

type r38handler func(w http.ResponseWriter, r *http.Request, userId int64, tx *sql.Tx) error

const FOREST_BEAR_ID = "700900270153924608"
const FOREST_BEAR = ":forestbear:" + FOREST_BEAR_ID
const DRAFT_ALERTS_ROLE = "692079611680653442"
const BOSS = "176164707026206720"
const PINK = 0xE50389

var secretKeyNoOneWillEverGuess = []byte(os.Getenv("SESSION_SECRET"))
var store = sessions.NewCookieStore(secretKeyNoOneWillEverGuess)
var sock string
var dg *discordgo.Session

func main() {
	useAuthPtr := flag.Bool("auth", true, "bool")
	flag.Parse()

	database, err := sql.Open("sqlite3", "draft.db")
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
		Handler: NewHandler(database, *useAuthPtr),
	}

	dg, err = discordgo.New("Bot " + os.Getenv("DISCORD_BOT_TOKEN"))
	if err != nil {
		log.Printf("%s", err.Error())
	} else {
		defer func() {
			log.Printf("Closing discord bot")
			err = dg.Close()
			if err != nil {
				log.Printf("%s", err.Error())
			}
		}()
		dg.AddHandler(DiscordReady)
		dg.AddHandler(DiscordMsgCreate(database))
		dg.AddHandler(DiscordReactionAdd(database))
		dg.AddHandler(DiscordReactionRemove(database))
		err = dg.Open()
		if err != nil {
			log.Printf("%s", err.Error())
		}
	}

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

// NewHandler creates all server routes for serving the html.
func NewHandler(database *sql.DB, useAuth bool) http.Handler {
	mux := http.NewServeMux()

	addHandler := func(route string, serveFunc r38handler, readonly bool) {
		isAuthRoute := strings.HasPrefix(route, "/auth/")
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var userID int64
			if useAuth {
				if isAuthRoute {
					userID = 0
				} else {
					session, err := store.Get(r, "session-name")
					if err != nil {
						userID = 0
					} else {
						userIDStr := session.Values["userid"]
						if userIDStr == nil {
							userID = 0
						} else {
							userIDInt, err := strconv.Atoi(userIDStr.(string))
							if err != nil {
								userID = 0
							} else {
								userID = int64(userIDInt)
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

			tx, err := database.BeginTx(ctx, &sql.TxOptions{ReadOnly: readonly})
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			err = serveFunc(w, r, userID, tx)
			if err != nil {
				tx.Rollback()
				if strings.HasPrefix(route, "/api/") {
					w.WriteHeader(http.StatusInternalServerError)
					json.NewEncoder(w).Encode(JSONError{Error: err.Error()})
				} else {
					http.Error(w, err.Error(), http.StatusInternalServerError)
				}
			} else {
				tx.Commit()
			}
		})
		mux.Handle(route, handler)
	}

	fs := http.FileServer(http.Dir("static"))
	mux.Handle("/static/", http.StripPrefix("/static/", fs))

	if useAuth {
		log.Printf("setting up auth routes...")
		addHandler("/auth/discord/login", oauthDiscordLogin, true) // don't actually need db at all
		addHandler("/auth/discord/callback", oauthDiscordCallback, false)
	}

	addHandler("/api/draft/", ServeAPIDraft, true)
	addHandler("/api/draftlist/", ServeAPIDraftList, true)
	addHandler("/api/pick/", ServeAPIPick, false)
	addHandler("/api/join/", ServeAPIJoin, false)

	addHandler("/api/dev/forceEnd/", ServeAPIForceEnd, false)

	addHandler("/", ServeVueApp, true)

	return mux
}

func HandleLogin(w http.ResponseWriter, r *http.Request, userID int64, tx *sql.Tx) error {
	t := template.Must(template.ParseFiles("login.tmpl"))
	t.Execute(w, nil)
	return nil
}

// ServeAPIDraft serves the /api/draft endpoint.
func ServeAPIDraft(w http.ResponseWriter, r *http.Request, userID int64, tx *sql.Tx) error {
	re := regexp.MustCompile(`/api/draft/(\d+)`)
	parseResult := re.FindStringSubmatch(r.URL.Path)
	if parseResult == nil {
		return fmt.Errorf("bad api url")
	}
	draftID, err := strconv.ParseInt(parseResult[1], 10, 64)
	if err != nil {
		return fmt.Errorf("bad api url: %s", err.Error())
	}

	draftJSON, err := GetFilteredJSON(tx, draftID, userID)
	if err != nil {
		return fmt.Errorf("error getting json: %s", err.Error())
	}

	fmt.Fprint(w, draftJSON)
	return nil
}

// ServeAPIDraftList serves the /api/draftlist endpoint.
func ServeAPIDraftList(w http.ResponseWriter, r *http.Request, userID int64, tx *sql.Tx) error {
	drafts, err := GetDraftList(userID, tx)
	if err != nil {
		return err
	}
	json.NewEncoder(w).Encode(drafts)
	return nil
}

// ServeAPIPick serves the /api/pick endpoint.
func ServeAPIPick(w http.ResponseWriter, r *http.Request, userID int64, tx *sql.Tx) error {
	if r.Method != "POST" {
		// we have to return an error manually here because we want to return
		// a different http status code.
		tx.Rollback()
		http.Error(w, "invalid request method", http.StatusMethodNotAllowed)
		return nil
	}

	bodyBytes, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return fmt.Errorf("error reading post body: %s", err.Error())
	}
	var pick PostedPick
	err = json.Unmarshal(bodyBytes, &pick)
	if err != nil {
		return fmt.Errorf("error parsing post body: %s", err.Error())
	}
	var draftID int64
	if len(pick.CardIds) == 1 {
		draftID, err = doSinglePick(tx, userID, pick.CardIds[0])
		if err != nil {
			// We can't send the actual error back to the client without leaking information about
			// where the card they tried to pick actually is.
			log.Printf("error making pick: %s", err.Error())
			return fmt.Errorf("error making pick")
		}
	} else if len(pick.CardIds) == 2 {
		return fmt.Errorf("cogwork librarian power not implemented yet")
	} else {
		return fmt.Errorf("invalid number of picked cards: %d", len(pick.CardIds))
	}

	draftJSON, err := GetFilteredJSON(tx, draftID, userID)
	if err != nil {
		return fmt.Errorf("error getting json: %s", err.Error())
	}

	fmt.Fprint(w, draftJSON)
	return nil
}

// ServeAPIJoin serves the /api/join endpoint.
func ServeAPIJoin(w http.ResponseWriter, r *http.Request, userID int64, tx *sql.Tx) error {
	if r.Method != "POST" {
		// we have to return an error manually here because we want to return
		// a different http status code.
		tx.Rollback()
		http.Error(w, "invalid request method", http.StatusMethodNotAllowed)
		return nil
	}

	bodyBytes, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return fmt.Errorf("error reading post body: %s", err.Error())
	}
	var toJoin PostedJoin
	err = json.Unmarshal(bodyBytes, &toJoin)
	if err != nil {
		return fmt.Errorf("error parsing post body: %s", err.Error())
	}

	draftID := toJoin.ID

	err = doJoin(tx, userID, draftID)
	if err != nil {
		return fmt.Errorf("error joining draft %d: %s", draftID, err.Error())
	}

	draftJSON, err := GetFilteredJSON(tx, draftID, userID)
	if err != nil {
		return fmt.Errorf("error getting json: %s", err.Error())
	}

	fmt.Fprint(w, draftJSON)
	return nil
}

// doJoin does the actual joining.
func doJoin(tx *sql.Tx, userID int64, draftID int64) error {
	query := `select
                    count(1)
                  from seats
                  where draft = ?
                    and user = ?`
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
                   and user is null
                 order by random()
                 limit 1`
	row = tx.QueryRow(query, draftID)
	var emptySeatID int64
	err = row.Scan(&emptySeatID)
	if err != nil {
		return err
	}

	query = `update seats set user = ? where id = ?`
	_, err = tx.Exec(query, userID, emptySeatID)
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
				err = dg.ChannelPermissionSet(channelID, discordID, "1", 0, discordgo.PermissionViewChannel)
				if err != nil {
					log.Printf("error locking spectator channel for user %s: %s", discordID, err.Error())
				}
			}
		}
	}

	return nil
}

// ServeAPIJoin serves the /api/dev/forceEnd testing endpoint.
func ServeAPIForceEnd(_ http.ResponseWriter, r *http.Request, userID int64, tx *sql.Tx) error {
	if userID == 1 {
		bodyBytes, err := ioutil.ReadAll(r.Body)
		if err != nil {
			return fmt.Errorf("error reading post body: %s", err.Error())
		}
		var toJoin PostedJoin
		err = json.Unmarshal(bodyBytes, &toJoin)
		if err != nil {
			return fmt.Errorf("error parsing post body: %s", err.Error())
		}

		draftID := toJoin.ID
		return NotifyEndOfDraft(tx, draftID)
	} else {
		return http.ErrBodyNotAllowed
	}
}

// ServeVueApp serves to vue.
func ServeVueApp(w http.ResponseWriter, r *http.Request, userID int64, tx *sql.Tx) error {
	var userInfo UserInfo

	if userID != 0 {
		query := `select
                            id,
                            discord_name,
                            picture
                          from users
                          where id = ?`
		row := tx.QueryRow(query, userID)
		err := row.Scan(&userInfo.ID, &userInfo.Name, &userInfo.Picture)
		if err != nil {
			return err
		}
	}

	userInfoJSON, err := json.Marshal(userInfo)
	if err != nil {
		return err
	}

	data := VuePageData{UserJSON: string(userInfoJSON)}

	t := template.Must(template.ParseFiles("vue.tmpl"))

	t.Execute(w, data)
	return nil
}

// doSinglePick performs a normal pick based on a user id and a card id. It returns the draft id and an error.
func doSinglePick(tx *sql.Tx, userID int64, cardID int64) (int64, error) {
	draftID, _, announcements, round, err := doPick(tx, userID, cardID, true)
	if err != nil {
		return draftID, err
	}
	err = doEvent(tx, draftID, userID, announcements, cardID, sql.NullInt64{}, round)
	if err != nil {
		return draftID, err
	}
	return draftID, nil
}

// doPick actually performs a pick in the database.
// It returns the draftID, packID, announcements, round, and an error.
// Of those return values, packID and announcements are only really relevant for Cogwork Librarian,
// which is not currently fully implemented, but we leave them here anyway for when we want to do that.
func doPick(tx *sql.Tx, userID int64, cardID int64, pass bool) (int64, int64, []string, int64, error) {
	announcements := []string{}

	// First we need information about the card. Determine which pack the card is in,
	// where that pack is at the table, who sits at that position, which draft that
	// pack is a part of, and which round that card is in.
	query := `select
                   packs.id,
                   seats.position,
                   seats.draft,
                   seats.user,
                   seats.round
                 from cards
                 join packs on cards.pack = packs.id
                 join seats on packs.seat = seats.id
                 where cards.id = ?`

	row := tx.QueryRow(query, cardID)
	var myPackID int64
	var position int64
	var draftID int64
	var userID2 int64
	var round int64
	err := row.Scan(&myPackID, &position, &draftID, &userID2, &round)

	if err != nil {
		return draftID, myPackID, announcements, round, err
	} else if userID != userID2 {
		return draftID, myPackID, announcements, round, fmt.Errorf("card does not belong to the user.")
	} else if round == 0 {
		return draftID, myPackID, announcements, round, fmt.Errorf("card has already been picked.")
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

	row = tx.QueryRow(query, userID, draftID)
	var myPackID2 int64
	err = row.Scan(&myPackID2)

	if err != nil {
		return draftID, myPackID, announcements, round, err
	} else if myPackID != myPackID2 {
		return draftID, myPackID, announcements, round, fmt.Errorf("card is not in the next available pack.")
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

	row = tx.QueryRow(query, userID, draftID)
	var myPicksID int64
	var myCount int64
	err = row.Scan(&myPicksID, &myCount)

	if err != nil {
		return draftID, myPackID, announcements, round, err
	}

	// Are we passing the pack after we've picked the card?
	if pass {
		// Get the seat position that the pack will be passed to.
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

		// Now get the seat id that the pack will be passed to.
		query = `select
                           seats.id,
                           users.discord_id
                         from seats
                         left join users on seats.user = users.id
                         where seats.draft = ?
                           and seats.position = ?`

		row = tx.QueryRow(query, draftID, newPosition)
		var newPositionID int64
		var newPositionDiscordID sql.NullString
		err = row.Scan(&newPositionID, &newPositionDiscordID)
		if err != nil {
			return draftID, myPackID, announcements, round, err
		}

		// Put the picked card into the player's picks.
		query = `update cards set pack = ? where id = ?`

		_, err = tx.Exec(query, myPicksID, cardID)
		if err != nil {
			return draftID, myPackID, announcements, round, err
		}

		// Move the pack to the next seat.
		query = `update packs set seat = ? where id = ?`
		_, err = tx.Exec(query, newPositionID, myPackID)
		if err != nil {
			return draftID, myPackID, announcements, round, err
		}

		// Get the number of remaining packs in the seat.
		query = `select
                           count(1)
                         from v_packs
                         join seats on v_packs.seat = seats.id
                         where seats.user = ?
                           and v_packs.round = ?
                           and v_packs.count > 0
                           and seats.draft = ?`
		row = tx.QueryRow(query, userID, round, draftID)
		var packsLeftInSeat int64
		err = row.Scan(&packsLeftInSeat)
		if err != nil {
			return draftID, myPackID, announcements, round, err
		}

		if packsLeftInSeat == 0 {
			// If there are 0 packs left in the seat, check to see if the player we passed the pack to
			// is in the same round as us. If the rounds match, NotifyByDraftAndPosition.
			query = `select
                                   count(1)
                                 from seats a
                                 join seats b on a.draft = b.draft
                                 where a.user = ?
                                   and b.position = ?
                                   and a.draft = ?
                                   and a.round = b.round`
			row = tx.QueryRow(query, userID, newPosition, draftID)
			var roundsMatch int64
			err = row.Scan(&roundsMatch)
			if err != nil {
				log.Printf("cannot determine if rounds match for notify")
			} else if roundsMatch == 1 && newPositionDiscordID.Valid {
				log.Printf("attempting to notify position %d draft %d", newPosition, draftID)
				err = NotifyByDraftAndDiscordID(draftID, newPositionDiscordID.String)
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

			_, err = tx.Exec(query, (myCount+1)/15+1, userID, draftID)
			if err != nil {
				return draftID, myPackID, announcements, round, err
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
                                         group by round
                                         order by round desc
                                         limit 1`

				row = tx.QueryRow(query, draftID)
				var nextRoundPlayers int64
				err = row.Scan(&nextRoundPlayers)
				if err != nil {
					log.Printf("error counting players and rounds")
				} else if nextRoundPlayers == 8 && myCount+1 == 45 {
					// The draft is over. Notify the admin.
					err = NotifyEndOfDraft(tx, draftID)
					if err != nil {
						log.Printf("error notifying end of draft: %s", err.Error())
					}
				} else if nextRoundPlayers > 1 {
					// Now we know that we are not the only player in this round.
					// Get the position of all players that currently have a pick.
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

					rows, err := tx.Query(query, draftID)
					if err != nil {
						log.Printf("error determining if there's a blocking player")
					} else {
						defer rows.Close()

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
							err = NotifyByDraftAndDiscordID(draftID, blockingDiscordID.String)
							if err != nil {
								log.Printf("error with blocking notify")
							}
						}
					}
				}
			}
		}
	} else {
		// we're in some sort of non-working cogwork librarian situation
		// just take the card from the pack.
		query = `update cards set pack = ? where id = ?`

		_, err = tx.Exec(query, myPicksID, cardID)
		if err != nil {
			return draftID, myPackID, announcements, round, err
		}
	}

	log.Printf("player %d in draft %d took card %d", userID, draftID, cardID)

	return draftID, myPackID, announcements, round, nil
}

// NotifyByDraftAndDiscordID sends a discord alert to a user.
func NotifyByDraftAndDiscordID(draftID int64, discordID string) error {
	return DiscordNotify(os.Getenv("PICK_ALERTS_CHANNEL_ID"),
		fmt.Sprintf(`<@%s> you have new picks <http://draft.thefoley.net/draft/%d>`, discordID, draftID))
}

func NotifyEndOfDraft(tx *sql.Tx, draftID int64) error {
	draftName, err := GetDraftName(tx, draftID)
	if err != nil {
		return err
	}

	err = PostFirstRoundPairings(tx, draftID, draftName)
	if err != nil {
		return err
	}

	err = NotifyAdminOfDraftCompletion(tx, draftID)
	if err != nil {
		return err
	}

	err = UnlockSpectatorChannel(tx, draftID)

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

func PostPairings(tx *sql.Tx, draftID int64, draftName string, round int, pairings string) error {
	msg, err := DiscordNotifyEmbed(
		os.Getenv("DRAFT_ANNOUNCEMENTS_CHANNEL_ID"),
		&discordgo.MessageEmbed{
			Title:       fmt.Sprintf("%s, Round %d", draftName, round),
			Description: pairings,
			Color:       PINK,
			Footer: &discordgo.MessageEmbedFooter{
				Text: "React with 🏆 if you win and 💀 if you lose.",
			},
		})
	if err != nil {
		log.Print(err.Error())
		return err
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

func NotifyAdminOfDraftCompletion(tx *sql.Tx, draftID int64) error {
	adminDiscordID, err := GetAdminDiscordId(tx)
	if err != nil {
		return err
	}
	return DiscordNotify(os.Getenv("PICK_ALERTS_CHANNEL_ID"),
		fmt.Sprintf(`<@%s> draft %d is finished!`, adminDiscordID, draftID))
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

func UnlockSpectatorChannel(tx *sql.Tx, draftID int64) error {
	if dg != nil {
		result := tx.QueryRow("select spectatorchannelid from drafts where id=?", draftID)
		var channelID sql.NullString
		err := result.Scan(&channelID)
		if !channelID.Valid {
			// OK to not find channel
			return nil
		}
		channel, err := dg.Channel(channelID.String)
		if err != nil {
			return err
		}
		for _, perm := range channel.PermissionOverwrites {
			err = dg.ChannelPermissionDelete(channelID.String, perm.ID)
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

// DiscordNotify posts a message to discord.
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
                    seats.position,
                    packs.round,
                    users.discord_name,
                    cards.id,
                    users.id,
                    cards.data,
                    users.picture
                  from seats
                  left join users on users.id = seats.user
                  join drafts on drafts.id = seats.draft
                  join packs on packs.original_seat = seats.id
                  join cards on cards.original_pack = packs.id
                  where drafts.id = ?`

	rows, err := tx.Query(query, draftID)
	if err != nil {
		return draft, err
	}
	defer rows.Close()
	var indices [8][3]int64
	for rows.Next() {
		var position int64
		var packRound int64
		var cardID int64
		var nullableDiscordID sql.NullString
		var draftUserID sql.NullInt64
		var cardData string
		var nullablePicture sql.NullString
		err = rows.Scan(&draft.DraftID, &draft.DraftName, &position, &packRound, &nullableDiscordID, &cardID, &draftUserID, &cardData, &nullablePicture)
		if err != nil {
			return draft, err
		}

		dataObj := make(map[string]interface{})
		err = json.Unmarshal([]byte(cardData), &dataObj)
		if err != nil {
			log.Printf("making nil card data because of error %s", err.Error())
			dataObj = nil
		}
		dataObj["id"] = cardID

		packRound--

		nextIndex := indices[position][packRound]

		draft.Seats[position].Packs[packRound][nextIndex] = dataObj
		draft.Seats[position].PlayerName = nullableDiscordID.String
		draft.Seats[position].PlayerID = draftUserID.Int64
		draft.Seats[position].PlayerImage = nullablePicture.String

		indices[position][packRound]++
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
	if err != nil && err != sql.ErrNoRows {
		return draft, err
	}
	defer rows.Close()
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

// GetFilteredJSON returns a filtered json object of replay data.
func GetFilteredJSON(tx *sql.Tx, draftID int64, userID int64) (string, error) {
	draftInfo, err := GetDraftListEntry(userID, tx, draftID)
	if err != nil {
		return "", err
	}

	draft, err := GetJSONObject(tx, draftID)
	if err != nil {
		return "", err
	}

	var returnFullReplay bool
	if draftInfo.Finished {
		// If the draft is over, everyone can see the full replay.
		returnFullReplay = true
	} else if draftInfo.Joined {
		// If we're a member of the draft and it's NOT over,
		// we need to see if we're done with the draft. If we are,
		// we can see the full replay. Otherwise, we need to
		// filter.
		query := `select
                            round
                          from seats
                          where draft = ?
                            and user = ?`
		row := tx.QueryRow(query, draftID, userID)
		var myRound sql.NullInt64
		err = row.Scan(&myRound)
		if err != nil {
			return "", err
		}
		if myRound.Valid && myRound.Int64 >= 4 {
			returnFullReplay = true
		}
	} else if userID != 0 && draftInfo.AvailableSeats == 0 {
		// If we're logged in AND the draft is full,
		// we can see the full replay.
		returnFullReplay = true
	}

	if returnFullReplay {
		ret, err := json.Marshal(draft)
		if err != nil {
			return "", err
		}
		return string(ret), nil
	}

	conn, err := net.Dial("unix", sock)
	if err != nil {
		return "", err
	}
	defer conn.Close()

	ret, err := json.Marshal(Perspective{User: userID, Draft: draft})
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

// doEvent records an event (pick) into the database.
func doEvent(tx *sql.Tx, draftID int64, userID int64, announcements []string, cardID1 int64, cardID2 sql.NullInt64, round int64) error {
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
		query = `insert into events (round, draft, position, announcement, card1, card2, modified) VALUES (?, ?, ?, ?, ?, ?, ?)`
		_, err = tx.Exec(query, round, draftID, position, strings.Join(announcements, "\n"), cardID1, cardID2.Int64, count)
	} else {
		query = `insert into events (round, draft, position, announcement, card1, card2, modified) VALUES (?, ?, ?, ?, ?, null, ?)`
		_, err = tx.Exec(query, round, draftID, position, strings.Join(announcements, "\n"), cardID1, count)
	}

	return err
}

func GetDraftList(userID int64, tx *sql.Tx) (DraftList, error) {
	var drafts DraftList

	query := `select
                    drafts.id,
                    drafts.name,
                    sum(seats.user is null and seats.position is not null) as empty_seats,
                    coalesce(sum(seats.user = ?), 0) as joined,
                    min(seats.round) > 3 as finished
                  from drafts
                  left join seats on drafts.id = seats.draft
                  group by drafts.id
                  order by drafts.id asc`

	rows, err := tx.Query(query, userID)
	if err != nil {
		return drafts, fmt.Errorf("can't get draft list: %s", err.Error())
	}
	defer rows.Close()
	for rows.Next() {
		var d DraftListEntry
		err = rows.Scan(&d.ID, &d.Name, &d.AvailableSeats, &d.Joined, &d.Finished)
		if err != nil {
			return drafts, fmt.Errorf("can't get draft list: %s", err.Error())
		}

		// It looks like Status is able to be derrived by the client, but in the future
		// we'll want to restrict who can join a given draft more.
		if d.Joined {
			d.Status = "member"
		} else if d.Finished {
			d.Status = "spectator"
		} else if userID == 0 {
			d.Status = "closed"
		} else if d.AvailableSeats == 0 {
			d.Status = "spectator"
		} else {
			d.Status = "joinable"
		}

		drafts.Drafts = append(drafts.Drafts, d)
	}
	return drafts, nil
}

func GetDraftListEntry(userID int64, tx *sql.Tx, draftID int64) (DraftListEntry, error) {
	var ret DraftListEntry

	drafts, err := GetDraftList(userID, tx)
	if err != nil {
		return ret, err
	}

	draftCount := len(drafts.Drafts)
	i := sort.Search(draftCount, func(i int) bool { return draftID <= drafts.Drafts[i].ID })
	if i < draftCount && i >= 0 {
		return drafts.Drafts[i], nil
	}

	return ret, fmt.Errorf("could not find draft id %d", draftID)
}

func DiscordReady(s *discordgo.Session, event *discordgo.Ready) {
	err := s.UpdateStatus(0, "Tier 5 Wolf Combo")
	if err != nil {
		log.Printf("%s", err.Error())
	}
}

func DiscordMsgCreate(database *sql.DB) func(s *discordgo.Session, msg *discordgo.MessageCreate) {
	return func(s *discordgo.Session, msg *discordgo.MessageCreate) {
		if msg.Author.ID == BOSS {
			if msg.GuildID == "" {
				args, err := shlex.Split(msg.Content)
				if err != nil {
					_, _ = dg.ChannelMessageSend(msg.ChannelID, err.Error())
					return
				}
				if strings.HasPrefix(msg.Content, "makedraft") {
					tx, err := database.Begin()
					if err != nil {
						log.Printf("%s", err.Error())
					} else {
						flagSet := flag.NewFlagSet(args[0], flag.ContinueOnError)

						settings := makedraft.Settings{}
						settings.Set = flagSet.String(
							"set", "sets/cube.json",
							"A .json file containing relevant set data.")
						settings.Database = flagSet.String(
							"database", "draft.db",
							"The sqlite3 database to insert to.")
						settings.Seed = flagSet.Int(
							"seed", 0,
							"The random seed to use to generate the draft. If 0, time.Now().UnixNano() will be used.")
						settings.Verbose = flagSet.Bool(
							"v", false,
							"If true, will enable verbose output.")
						settings.Simulate = flagSet.Bool(
							"simulate", false,
							"If true, won't commit to the database.")
						settings.Name = flagSet.String(
							"name", "untitled draft",
							"The name of the draft.")

						flagSet.Parse(args[1:])

						err = makedraft.MakeDraft(settings, tx)

						var resp string
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
						_, err = dg.ChannelMessageSend(msg.ChannelID, resp)
						if err != nil {
							log.Printf("%s", err)
						}
					}
				}
			} else if msg.Content == "!alerts" {
				DiscordSendRoleReactionMessage(s, database, msg.ChannelID,
					FOREST_BEAR, FOREST_BEAR_ID, DRAFT_ALERTS_ROLE,
					"Draft alerts", "if you would like notifications for games being played")
			}
		}
	}
}

func DiscordSendRoleReactionMessage(s *discordgo.Session, database *sql.DB, channelID string, emoji string, emojiId string, roleID string, title string, description string) {
	sent, err := DiscordNotifyEmbed(
		channelID,
		&discordgo.MessageEmbed{
			Title: title,
			Description: "\nReact with <" + emoji + "> " + description + ".\n\n" +
				"If you would like to remove the role, simply remove your reaction.\n",
			Color: PINK,
		})
	if err != nil {
		log.Printf("%s", err.Error())
	} else if sent != nil {
		_, err = database.Exec(
			`insert into rolemsgs (msgid, emoji, roleid) values (?, ?, ?)`,
			sent.ID, emojiId, roleID)
		if err != nil {
			log.Printf("%s", err.Error())
		}
		err = s.MessageReactionAdd(sent.ChannelID, sent.ID, emoji)
		if err != nil {
			log.Printf("%s", err.Error())
		}
	}
}

func DiscordReactionAdd(database *sql.DB) func(s *discordgo.Session, msg *discordgo.MessageReactionAdd) {
	return func(s *discordgo.Session, msg *discordgo.MessageReactionAdd) {
		tx, err := database.Begin()
		if err != nil {
			log.Printf("%s", err.Error())
			return
		}
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
		} else if err == sql.ErrNoRows {
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
						_, err = tx.Exec(`insert into results (draft, round, user, win) values (?, ?, ?, 1)`,
							draftID, round, user)
					} else if msg.Emoji.Name == "💀" {
						_, err = tx.Exec(`insert into results (draft, round, user, win) values (?, ?, ?, 0)`,
							draftID, round, user)
					}
					if err != nil {
						log.Printf("%s", err.Error())
					}
					CheckNextRoundPairings(tx, draftID, round)
				} else {
					log.Printf("%s", err.Error())
				}
			} else if err != sql.ErrNoRows {
				log.Printf("%s", err.Error())
			}
		} else {
			log.Printf("%s", err.Error())
		}
		err = tx.Commit()
		if err != nil {
			log.Printf("%s", err.Error())
		}
	}
}

func DiscordReactionRemove(database *sql.DB) func(s *discordgo.Session, msg *discordgo.MessageReactionRemove) {
	return func(s *discordgo.Session, msg *discordgo.MessageReactionRemove) {
		tx, err := database.Begin()
		if err != nil {
			log.Printf("%s", err.Error())
			return
		}
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
		} else if err == sql.ErrNoRows {
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
			} else if err != sql.ErrNoRows {
				log.Printf("%s", err.Error())
			}
		} else {
			log.Printf("%s", err.Error())
		}
		err = tx.Commit()
		if err != nil {
			log.Printf("%s", err.Error())
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
