package migrations

import (
	"github.com/BurntSushi/migration"
)

var Migrations = []migration.Migrator{
	initialSchema,
	discordBotSupport,
	addMtgoName,
	addResultsTimestamp,
	addCardId,
}

func initialSchema(tx migration.LimitedTx) error {
	_, err := tx.Exec(`CREATE TABLE IF NOT EXISTS users ( id integer primary key autoincrement, discord_id text unique, discord_name text, picture text );
		CREATE TABLE IF NOT EXISTS "users_old"( id integer primary key autoincrement, google_id text unique, email text, picture text, slack string, discord string, webhook string);
		CREATE TABLE IF NOT EXISTS seats( id integer primary key autoincrement, position number, user number, draft number, round number default 1);
		CREATE TABLE IF NOT EXISTS packs( id integer primary key autoincrement, seat number, modified number, round number , original_seat number);
		CREATE TABLE IF NOT EXISTS cards( id integer primary key autoincrement, pack number, edition text, number text, tags text, name text, faceup number default false, original_pack number, cmc number, type text, color text, modified number default 0, mtgo string);
		CREATE TABLE IF NOT EXISTS drafts( id integer primary key autoincrement, name text);
		CREATE TABLE IF NOT EXISTS revealed( id integer primary key autoincrement, draft number, message text);
		CREATE TABLE IF NOT EXISTS events( id integer primary key autoincrement, draft number, user number, announcement text, card1 number, card2 number, modified number, round number);
		CREATE VIEW IF NOT EXISTS v_packs as select packs.*, count(cards.id) as count from packs left join cards on packs.id=cards.pack group by packs.id
		/* v_packs(id,seat,modified,round,original_seat,count) */;`)
	return err
}

func discordBotSupport(tx migration.LimitedTx) error {
	_, err := tx.Exec(`CREATE TABLE rolemsgs ( id integer primary key autoincrement, msgid text, emoji text, roleid text);
		CREATE TABLE pairingmsgs ( id integer primary key autoincrement, msgid text, draft number, round number);
		CREATE TABLE results ( id integer primary key autoincrement, draft number, round number, user number, win number);
		CREATE TABLE skips ( id integer primary key autoincrement, user number, draft number);
		CREATE TABLE userformats ( id integer primary key autoincrement, user number, format string, epoch number, elig number);
		ALTER TABLE drafts ADD COLUMN spectatorchannelid string;
		ALTER TABLE seats ADD COLUMN reserveduser number;`)
	return err
}

func addMtgoName(tx migration.LimitedTx) error {
	_, err := tx.Exec(`ALTER TABLE users ADD COLUMN mtgo_name string;`)
	return err
}

func addResultsTimestamp(tx migration.LimitedTx) error {
	_, err := tx.Exec(`ALTER TABLE results ADD COLUMN timestamp text;`)
	return err
}

func addCardId(tx migration.LimitedTx) error {
	_, err := tx.Exec(`ALTER TABLE cards ADD COLUMN cardid string;
		CREATE INDEX cardid_idx ON cards (cardid);`)
	return err
}
