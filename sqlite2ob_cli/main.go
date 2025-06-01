package main

import (
	"context"
	"database/sql"
	"flag"
	"github.com/objectbox/objectbox-go/objectbox"
	"github.com/walkingeyerobot/r38/makedraft"
	"github.com/walkingeyerobot/r38/schema"
	"log"
	"os"
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

	err = os.RemoveAll(*settings.DatabaseDir)
	if err != nil {
		log.Printf("can't remove existing objectbox dir: %s", err.Error())
		os.Exit(1)
	}
	ob, err := objectbox.NewBuilder().Model(schema.ObjectBoxModel()).
		Directory(*settings.DatabaseDir).Build()
	defer ob.Close()

	err = ob.RunInWriteTx(func() error {

		userBox := schema.BoxForUser(ob)
		var users []*schema.User
		rows, err := tx.Query("select discord_id, discord_name, picture, mtgo_name from users order by id")
		if err != nil {
			return err
		}
		for rows.Next() {
			user := schema.User{
				Skips: []*schema.Skip{},
			}
			var mtgoName sql.NullString
			err = rows.Scan(
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
			users = append(users, &user)
		}

		draftBox := schema.BoxForDraft(ob)
		var drafts []*schema.Draft
		rows, err = tx.Query("select name, format, spectatorchannelid, inperson from drafts order by id")
		if err != nil {
			return err
		}
		for rows.Next() {
			draft := schema.Draft{}
			err = rows.Scan(
				&draft.Name,
				&draft.Format,
				&draft.SpectatorChannelId,
				&draft.InPerson,
			)
			if err != nil {
				return err
			}
			drafts = append(drafts, &draft)
		}

		seatBox := schema.BoxForSeat(ob)
		var seats []*schema.Seat
		rows, err = tx.Query("select position, user, draft, round, reserveduser, scansound, errorsound from seats order by id")
		if err != nil {
			return err
		}
		for rows.Next() {
			seat := schema.Seat{}
			var userId sql.NullInt64
			var reservedUserId sql.NullInt64
			var draftId uint64
			err = rows.Scan(
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
				seat.User = users[userId.Int64-1]
				drafts[draftId-1].Seats = append(drafts[draftId-1].Seats, &seat)
			}
			if reservedUserId.Valid {
				seat.ReservedUser = users[reservedUserId.Int64-1]
			}
			seats = append(seats, &seat)
		}

		packBox := schema.BoxForPack(ob)
		sqlPackIdToObPackId := map[uint64]uint64{}
		pickedPackIdToSeatId := map[uint64]uint64{}
		var packs []*schema.Pack
		rows, err = tx.Query("select id, seat, round, original_seat from packs order by id")
		if err != nil {
			return err
		}
		for rows.Next() {
			pack := schema.Pack{
				Cards:         []*schema.Card{},
				OriginalCards: []*schema.Card{},
			}
			var packId uint64
			var seatId uint64
			var originalSeatId uint64
			err = rows.Scan(
				&packId,
				&seatId,
				&pack.Round,
				&originalSeatId,
			)
			if err != nil {
				return err
			}
			if pack.Round != 0 {
				seats[seatId-1].Packs = append(seats[seatId-1].Packs, &pack)
				seats[originalSeatId-1].OriginalPacks = append(seats[originalSeatId-1].OriginalPacks, &pack)
				packs = append(packs, &pack)
				sqlPackIdToObPackId[packId] = uint64(len(packs))
			} else {
				pickedPackIdToSeatId[packId] = seatId
			}
		}

		cardBox := schema.BoxForCard(ob)
		var cards []*schema.Card
		rows, err = tx.Query("select pack, original_pack, data, cardid from cards order by id")
		if err != nil {
			return err
		}
		for rows.Next() {
			card := schema.Card{}
			var packId uint64
			var originalPackId uint64
			err = rows.Scan(
				&packId,
				&originalPackId,
				&card.Data,
				&card.CardId,
			)
			if err != nil {
				return err
			}
			obPackId, isRealPack := sqlPackIdToObPackId[packId]
			if isRealPack {
				packs[obPackId-1].Cards = append(packs[obPackId-1].Cards, &card)
			} else {
				seat := seats[pickedPackIdToSeatId[packId]]
				seat.PickedCards = append(seat.PickedCards, &card)
			}
			obOriginalPackId := sqlPackIdToObPackId[originalPackId]
			packs[obOriginalPackId-1].OriginalCards = append(packs[obOriginalPackId-1].OriginalCards, &card)
			cards = append(cards, &card)
		}

		eventBox := schema.BoxForEvent(ob)
		var events []*schema.Event
		rows, err = tx.Query("select draft, card1, modified, round, position, pack from events order by id")
		if err != nil {
			return err
		}
		for rows.Next() {
			event := schema.Event{}
			var draftId uint64
			var cardId uint64
			var packId uint64
			err = rows.Scan(
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
			event.Pack = packs[sqlPackIdToObPackId[packId]-1]
			event.Card1 = cards[cardId-1]
			drafts[draftId-1].Events = append(drafts[draftId-1].Events, &event)
			events = append(events, &event)
		}

		_, err = eventBox.PutMany(events)
		if err != nil {
			return err
		}
		_, err = cardBox.PutMany(cards)
		if err != nil {
			return err
		}
		_, err = packBox.PutMany(packs)
		if err != nil {
			return err
		}
		_, err = seatBox.PutMany(seats)
		if err != nil {
			return err
		}
		_, err = userBox.PutMany(users)
		if err != nil {
			return err
		}
		_, err = draftBox.PutMany(drafts)
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

	log.Printf("migrated %d drafts and %d users", draftsCount, usersCount)
	os.Exit(0)
}
