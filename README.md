# r38

R-38 is an insulation strength. For managing drafts.

# Installation

[check out the repo, mess around with golang stuff]

## Install golang dependencies:

```bash
go get -v github.com/walkingeyerobot/r38/...
```

## Pull a usable Cube list (vintagecube.csv):


```bash
wget -O cube.csv 'https://cubecobra.com/cube/download/csv/5e3cfa78fab99c24464f76ee?primary=Color%20Category&secondary=Types-Multicolor&tertiary=CMC2'
```

NOTE: As of 2020-04-24, the CSV output needs a minor hack to formatting: add `,Blank` to the CSV header line if the last field is "MTGO ID".


## Configure the sqlite3 database (draft.db)

```sqlite3
sqlite> .schema
CREATE TABLE sqlite_sequence(name,seq);
CREATE TABLE users ( id integer primary key autoincrement, discord_id text unique, discord_name text, picture text );
CREATE TABLE IF NOT EXISTS "users_old"( id integer primary key autoincrement, google_id text unique, email text, picture text, slack string, discord string, webhook string);
CREATE TABLE seats( id integer primary key autoincrement, position number, user number, reserveduser number, draft number, round number default 1);
CREATE TABLE packs( id integer primary key autoincrement, seat number, modified number, round number , original_seat number);
CREATE TABLE cards( id integer primary key autoincrement, pack number, edition text, number text, tags text, name text, faceup number default false, original_pack number, cmc number, type text, color text, modified number default 0, mtgo string);
CREATE TABLE drafts( id integer primary key autoincrement, name text, spectatorchannelid string);
CREATE TABLE revealed( id integer primary key autoincrement, draft number, message text);
CREATE TABLE events( id integer primary key autoincrement, draft number, user number, announcement text, card1 number, card2 number, modified number, round number);
CREATE TABLE rolemsgs ( id integer primary key autoincrement, msgid text, emoji text, roleid text);
CREATE TABLE pairingmsgs ( id integer primary key autoincrement, msgid text, draft number, round number);
CREATE TABLE results ( id integer primary key autoincrement, draft number, round number, user number, win number);
CREATE TABLE skips ( id integer primary key autoincrement, user number, draft number);
CREATE TABLE userformats ( id integer primary key autoincrement, user number, format string, epoch number, elig number);
CREATE VIEW v_packs as select packs.*, count(cards.id) as count from packs left join cards on packs.id=cards.pack group by packs.id
/* v_packs(id,seat,modified,round,original_seat,count) */;
```

## Run the server without OAuth

You can now run the server without OAuth. You will always be considered logged in as userId 1. To be logged in as a different user, add ?as=x to the end of the url you want to view, where x is the id of the user you want to view the page as.

```bash
source ~/r38-secret*.env; go run main.go -auth=false
```

## Configure OAuth for a local environment:

### Google OAuth

* [Using OAuth 2.0 to Access Google APIs](https://developers.google.com/identity/protocols/oauth2)
* Set up the OAuth consent screen
* Origin URI should be `http://${SITE}:${PORT:-12264}`, wherever you'll run R38
* Authorized redirect URI should be `http://${SITE}:${PORT:-12264}/auth/google/callback`

### Discord OAuth

* [Using OAuth 2.0 to Access Discord APIs](https://discordapp.com/developers/docs/topics/oauth2)
* Set up the OAuth consent screen
* Origin URI should be `http://${SITE}:${PORT:-12264}`, wherever you'll run R38
    * Discord maybe doesn't care about origin URI?
* Authorized redirect URI should be `http://${SITE}:${PORT:-12264}/auth/discord/callback`
* Should only need `email` and `identify` scopes

## Configure local environment variables

Generate a session secret and copy it to either or both secret files:
```bash
SESSION_SECRET=$(sort --random-sort </usr/share/dict/words | \
  grep -E '^[a-z]+$' | head -n 3 | xargs | \
  sed 's/.*/\L&/; s/[a-z]*/\u&/g; s/\ //g')

for s in r38-secret-{goog,discord}.env; do
  echo "export SESSION_SECRET='${SESSION_SECRET}'" > ~/${s}
done
```

### Add generated OAuth values to local environment variables

Google oauth values:
```bash
cat <<EOF >> ~/r38-secret-goog.env
export GOOGLE_CLIENT_ID='${ClientID}'
export GOOGLE_CLIENT_SECRET='${ClientSecret}'
export GOOGLE_REDIRECT_URL='http://${SITE}:${PORT:-12264}/auth/google/callback'
EOF
```

Discord oauth values:
```bash
cat <<EOF >> ~/r38-secret-discord.env
export DISCORD_CLIENT_ID='${ClientID}'
export DISCORD_CLIENT_SECRET='${ClientSecret}'
export DISCORD_REDIRECT_URL='http://${SITE}:${PORT:-12264}/auth/discord/callback'
EOF
```

## Configure a draft

```bash
go run makedraft_cli/main.go
```

## Start the server

```bash
source ~/r38-secret*.env; go run main.go
```
