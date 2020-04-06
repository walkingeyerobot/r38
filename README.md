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
wget -O vintagecube.csv 'https://cubecobra.com/cube/download/csv/5e3cfa78fab99c24464f76ee?primary=Color%20Category&secondary=Types-Multicolor&tertiary=CMC2'
```

## Configure the sqlite3 database (draft.db)

```sqlite3
sqlite> .schema
CREATE TABLE sqlite_sequence(name,seq);
CREATE TABLE users( id integer primary key autoincrement, google_id text unique, email text, picture text, slack string, discord string, webhook string);
CREATE TABLE seats( id integer primary key autoincrement, position number, user number, draft number, round number default 1);
CREATE TABLE packs( id integer primary key autoincrement, seat number, modified number, round number , original_seat number);
CREATE TABLE cards( id integer primary key autoincrement, pack number, edition text, number text, tags text, name text, faceup number default false, original_pack number, modified number default 0, mtgo string);
CREATE TABLE drafts( id integer primary key autoincrement, name text);
CREATE TABLE revealed( id integer primary key autoincrement, draft number, message text);
CREATE TABLE events( id integer primary key autoincrement, draft number, user number, announcement text, card1 number, card2 number, modified number, round number);
CREATE VIEW v_packs as select packs.*, count(cards.id) as count from packs left join cards on packs.id=cards.pack group by packs.id
/* v_packs(id,seat,modified,round,original_seat,count) */;
```

## Run the server without OAuth

You can now run the server without OAuth. You will always be considered logged in as userId 1. To be logged in as a differnt user, add ?as=x to the end of the url you want to view, where x is the id of the user you want to view the page as.

```bash
source ~/r38-secret.env; go run main.go -auth=false
```

## Configure OAuth for a local environment:

* [Using OAuth 2.0 to Access Google APIs](https://developers.google.com/identity/protocols/oauth2)
* Set up the OAuth consent screen
* Origin URI should be `http://${SITE}:${PORT:-12264}`, wherever you'll run R38
* Authorized redirect URI should be `http://${SITE}:${PORT:-12264}/auth/google/callback`

## Configure local environment variables

```bash
SESSION_SECRET=$(sort --random-sort </usr/share/dict/words | \
  grep -E '^[a-z]+$' | head -n 3 | xargs | \
  sed 's/.*/\L&/; s/[a-z]*/\u&/g; s/\ //g') && \
echo "export SESSION_SECRET='${SESSION_SECRET}'" > ~/r38-secret.env
```

### Add generated OAuth values to local environment variables

```bash
cat <<EOF >> ~/r38-secret.env
export GOOGLE_CLIENT_ID='${ClientID}'
export GOOGLE_CLIENT_SECRET='${ClientSecret}'
export GOOGLE_REDIRECT_URL='http://${SITE}:${PORT:-12264}/auth/google/callback'
EOF
```

## Configure a draft

```bash
go run makedraft.go
```

## Start the server

```bash
source ~/r38-secret.env; go run main.go
```
