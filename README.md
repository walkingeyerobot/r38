# r38

R-38 is an insulation strength. For managing drafts.

# Installation

[check out the repo, mess around with golang stuff]

## Install golang dependencies:

```bash
go get -v github.com/walkingeyerobot/r38/...
```

## Install ObjectBox

```bash
bash <(curl -s https://raw.githubusercontent.com/objectbox/objectbox-go/main/install.sh)
```

## Run the server without OAuth

You can now run the server without OAuth. You will always be considered logged in as userId 1. To be logged in as a different user, add ?as=x to the end of the url you want to view, where x is the id of the user you want to view the page as.

### Using sqlite

```bash
source r38-secret*.env; go run . -auth=false
```

### Using objectbox

```bash
source r38-secret*.env; go run . -auth=false -objectbox=true
```

## Configure OAuth for a local environment:

### Google OAuth

- [Using OAuth 2.0 to Access Google APIs](https://developers.google.com/identity/protocols/oauth2)
- Set up the OAuth consent screen
- Origin URI should be `http://${SITE}:${PORT:-12264}`, wherever you'll run R38
- Authorized redirect URI should be `http://${SITE}:${PORT:-12264}/auth/google/callback`

### Discord OAuth

- [Using OAuth 2.0 to Access Discord APIs](https://discordapp.com/developers/docs/topics/oauth2)
- Set up the OAuth consent screen
- Origin URI should be `http://${SITE}:${PORT:-12264}`, wherever you'll run R38
  - Discord maybe doesn't care about origin URI?
- Authorized redirect URI should be `http://${SITE}:${PORT:-12264}/auth/discord/callback`
- Should only need `email` and `identify` scopes

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

### Using sqlite

MTGO draft:

```bash
go run makedraft_cli/*.go --name="name of draft"
```

Paper draft:

```bash
go run makedraft_cli/*.go --inPerson --name="name of draft"
```

Paper draft with automatic seat and/or pack assignment, for testing:

```bash
go run makedraft_cli/*.go --inPerson --name="name of draft" [--assignSeats] [--assignPacks]
```

### Using objectbox

MTGO draft:

```bash
go run makedraft_cli/*.go --name="name of draft" --database_dir=objectbox
```

Paper draft:

```bash
go run makedraft_cli/*.go --inPerson --name="name of draft" --database_dir=objectbox
```

Paper draft with automatic seat and/or pack assignment, for testing:

```bash
go run makedraft_cli/*.go --inPerson --name="name of draft" --database_dir=objectbox [--assignSeats] [--assignPacks]
```

## Initialize fake users

The following will prepopulate the server with 11 fake users that you can
impersonate with the `?as=` query param.

```bash
sqlite3 draft.db < development/InitUsers.sql
```

## Start the server

### Using sqlite

```bash
source r38-secret*.env; go run .
```

### Using objectbox

```bash
source r38-secret*.env; go run . --objectbox=true
```
