FROM golang:1.24 AS gobuilder

WORKDIR /src

# then copy the rest of the source code
COPY . .
RUN go mod download
RUN go build -v .



FROM node:23-slim AS nodebuilder
WORKDIR /src/client

COPY client/package.json client/package-lock.json* ./
RUN npm ci && npm cache clean --force

COPY client/. .
RUN npm run build



FROM node:23-slim AS filter_deploy
WORKDIR /srv/r38-filter
COPY filter.js .
ENTRYPOINT ["node", "filter.js"]



FROM debian:stable-slim AS go_deploy

RUN apt-get update && apt-get install -y ca-certificates && rm -rf /var/lib/apt/lists/*

WORKDIR /srv/r38

COPY --from=nodebuilder /src/client-dist /srv/r38/client-dist 
COPY --from=gobuilder /src/r38 /srv/r38/r38

EXPOSE 12264
WORKDIR /srv/r38/db
CMD ["/srv/r38/r38"]
