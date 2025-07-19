package main

import (
	"context"
	"database/sql"
	"flag"
	"github.com/objectbox/objectbox-go/objectbox"
	"github.com/walkingeyerobot/r38/makedraft"
	"github.com/walkingeyerobot/r38/schema"
	"log"
	"maps"
	"os"
	"slices"
)

func main() {
	flagSet := flag.NewFlagSet(os.Args[0], flag.ContinueOnError)

	settings := makedraft.Settings{}
	settings.Database = flagSet.String(
		"database", "draft.db",
		"The sqlite3 database to read from.")
	settings.DatabaseDir = flagSet.String(
		"database_dir", "objectbox",
		"The objectbox database directory to write to. WARNING: if this exists, it is deleted first")

	err := flagSet.Parse(os.Args[1:])
	if err != nil {
		log.Printf(err.Error())
		os.Exit(1)
	}

	database, err := sql.Open("sqlite3", *settings.Database)
	if err != nil {
		log.Printf("error opening database %s: %s", *settings.Database, err.Error())
		os.Exit(1)
	}
	err = database.Ping()
	if err != nil {
		log.Printf("error pinging database: %s", err.Error())
		os.Exit(1)
	}

	tx, err := database.BeginTx(context.Background(), &sql.TxOptions{ReadOnly: true})
	if err != nil {
		log.Printf("can't create a context: %s", err.Error())
		os.Exit(1)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	err = os.RemoveAll(*settings.DatabaseDir + "/data.mdb")
	if err != nil {
		log.Printf("can't remove existing objectbox dir: %s", err.Error())
		os.Exit(1)
	}
	ob, err := objectbox.NewBuilder().Model(schema.ObjectBoxModel()).
		Directory(*settings.DatabaseDir).Build()
	defer ob.Close()

	err = ob.RunInWriteTx(func() error {
		userBox := schema.BoxForUser(ob)
		users := make(map[int]*schema.User)
		rows, err := tx.Query("select id, discord_id, discord_name, picture, mtgo_name from users order by id")
		if err != nil {
			return err
		}
		for rows.Next() {
			user := schema.User{
				Skips: []*schema.Skip{},
			}
			var mtgoName sql.NullString
			err = rows.Scan(
				&user.Id,
				&user.DiscordId,
				&user.DiscordName,
				&user.Picture,
				&mtgoName,
			)
			if user.Picture == "http://draftcu.be/static/favicon.png" {
				user.Picture = "https://cdn.discordapp.com/embed/avatars/0.png"
			}
			if mtgoName.Valid {
				user.MtgoName = mtgoName.String
			}
			if err != nil {
				return err
			}
			users[int(user.Id)] = &user
		}

		draftBox := schema.BoxForDraft(ob)
		drafts := make(map[int]*schema.Draft)
		rows, err = tx.Query("select id, name, format, spectatorchannelid, inperson from drafts order by id")
		if err != nil {
			return err
		}
		for rows.Next() {
			draft := schema.Draft{
				Seats:           make([]*schema.Seat, 0),
				UnassignedPacks: make([]*schema.Pack, 0),
			}
			err = rows.Scan(
				&draft.Id,
				&draft.Name,
				&draft.Format,
				&draft.SpectatorChannelId,
				&draft.InPerson,
			)
			if err != nil {
				return err
			}
			drafts[int(draft.Id)] = &draft
		}

		seatBox := schema.BoxForSeat(ob)
		seats := make(map[int]*schema.Seat)
		rows, err = tx.Query("select id, position, user, draft, round, reserveduser, scansound, errorsound from seats where position <> 8 order by id")
		if err != nil {
			return err
		}
		for rows.Next() {
			seat := schema.Seat{
				Packs:         make([]*schema.Pack, 0),
				OriginalPacks: make([]*schema.Pack, 0),
				PickedCards:   make([]*schema.Card, 0),
			}
			var userId sql.NullInt64
			var reservedUserId sql.NullInt64
			var draftId uint64
			err = rows.Scan(
				&seat.Id,
				&seat.Position,
				&userId,
				&draftId,
				&seat.Round,
				&reservedUserId,
				&seat.ScanSound,
				&seat.ErrorSound,
			)
			if err != nil {
				return err
			}
			if userId.Valid {
				seat.User = users[int(userId.Int64)]
				drafts[int(draftId)].Seats = append(drafts[int(draftId)].Seats, &seat)
			}
			if reservedUserId.Valid {
				seat.ReservedUser = users[int(reservedUserId.Int64)]
			}
			seats[int(seat.Id)] = &seat
		}

		packBox := schema.BoxForPack(ob)
		pickedPackIdToSeatId := make(map[int]int)
		packs := make(map[int]*schema.Pack)
		rows, err = tx.Query("select id, seat, round, original_seat from packs order by id")
		if err != nil {
			return err
		}
		for rows.Next() {
			pack := schema.Pack{
				Cards:         []*schema.Card{},
				OriginalCards: []*schema.Card{},
			}
			var seatId int
			var originalSeatId uint64
			err = rows.Scan(
				&pack.Id,
				&seatId,
				&pack.Round,
				&originalSeatId,
			)
			if err != nil {
				return err
			}
			if pack.Round != 0 {
				seats[seatId].Packs = append(seats[seatId].Packs, &pack)
				seats[int(originalSeatId)].OriginalPacks = append(seats[int(originalSeatId)].OriginalPacks, &pack)
				packs[int(pack.Id)] = &pack
			} else {
				pickedPackIdToSeatId[int(pack.Id)] = seatId
			}
		}

		cardBox := schema.BoxForCard(ob)
		cards := make(map[int]*schema.Card)
		rows, err = tx.Query("select id, pack, original_pack, data, cardid from cards order by id")
		if err != nil {
			return err
		}
		for rows.Next() {
			card := schema.Card{}
			var packId int
			var originalPackId int
			err = rows.Scan(
				&card.Id,
				&packId,
				&originalPackId,
				&card.Data,
				&card.CardId,
			)
			if err != nil {
				return err
			}
			_, isRealPack := packs[packId]
			if isRealPack {
				packs[packId].Cards = append(packs[packId].Cards, &card)
			} else {
				seat := seats[pickedPackIdToSeatId[packId]]
				seat.PickedCards = append(seat.PickedCards, &card)
			}
			packs[originalPackId].OriginalCards = append(packs[originalPackId].OriginalCards, &card)
			cards[int(card.Id)] = &card
		}

		eventBox := schema.BoxForEvent(ob)
		events := make(map[int]*schema.Event)
		rows, err = tx.Query("select id, draft, card1, modified, round, position, pack from events order by id")
		if err != nil {
			return err
		}
		for rows.Next() {
			event := schema.Event{}
			var draftId int
			var cardId int
			var packId int
			err = rows.Scan(
				&event.Id,
				&draftId,
				&cardId,
				&event.Modified,
				&event.Round,
				&event.Position,
				&packId,
			)
			if err != nil {
				return err
			}
			event.Pack = packs[packId]
			event.Card1 = cards[cardId]
			drafts[draftId].Events = append(drafts[draftId].Events, &event)
			events[int(event.Id)] = &event
		}

		_, err = eventBox.PutMany(slices.Collect(maps.Values(events)))
		if err != nil {
			return err
		}
		_, err = cardBox.PutMany(slices.Collect(maps.Values(cards)))
		if err != nil {
			return err
		}
		_, err = packBox.PutMany(slices.Collect(maps.Values(packs)))
		if err != nil {
			return err
		}
		_, err = seatBox.PutMany(slices.Collect(maps.Values(seats)))
		if err != nil {
			return err
		}
		_, err = userBox.PutMany(slices.Collect(maps.Values(users)))
		if err != nil {
			return err
		}
		_, err = draftBox.PutMany(slices.Collect(maps.Values(drafts)))
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		log.Printf(err.Error())
		os.Exit(1)
	}

	draftsCount, _ := schema.BoxForDraft(ob).Count()
	usersCount, _ := schema.BoxForUser(ob).Count()
	seatsCount, _ := schema.BoxForSeat(ob).Count()
	packsCount, _ := schema.BoxForPack(ob).Count()
	cardsCount, _ := schema.BoxForCard(ob).Count()
	eventsCount, _ := schema.BoxForEvent(ob).Count()

	log.Printf("migrated %d users, %d drafts, %d seats, %d packs, %d cards, %d events", usersCount, draftsCount, seatsCount, packsCount, cardsCount, eventsCount)
	os.Exit(0)
}
