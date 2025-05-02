FROM golang:1.24 AS gobuilder

WORKDIR /src

# then copy the rest of the source code
COPY . .
RUN go mod download
RUN go build -v .
WORKDIR /src/makedraft_cli
RUN go build -v .


FROM node:23-slim AS nodebuilder
WORKDIR /src/client

COPY client/package.json client/package-lock.json* ./
RUN npm ci && npm cache clean --force

COPY client/. .
RUN npm run build



FROM node:23-slim AS filter_deploy
WORKDIR /srv/r38
COPY filter.js .
#RUN ln -s socket/r38.sock r38.sock
WORKDIR /srv/r38/socket

ENTRYPOINT ["node", "../filter.js"]



FROM debian:stable-slim AS go_deploy

RUN apt-get update && apt-get install -y ca-certificates  sqlite3 curl && rm -rf /var/lib/apt/lists/*

WORKDIR /srv/r38

COPY --from=nodebuilder /src/client-dist /srv/r38/client-dist 
COPY --from=gobuilder /src/r38 /srv/r38/r38
COPY --from=gobuilder /src/makedraft_cli/makedraft_cli /srv/r38/makedraft_cli

RUN ln -s db/draft.db draft.db
RUN ln -s socket/r38.sock r38.sock
EXPOSE 12264

CMD ["/srv/r38/r38"]
