FROM golang:1.24 AS gobuilder

ENV GOCACHE=/tmp/go-cache
RUN --mount=type=cache,target=/tmp/go-cache \
    --mount=type=cache,target=/go/pkg/mod \
    mkdir -p /tmp/go-cache /go/pkg/mod

WORKDIR /src

# then copy the rest of the source code
# TODO there must be a better way to write this
COPY . .
RUN --mount=type=cache,target=/tmp/go-cache \
    --mount=type=cache,target=/go/pkg/mod \
 go mod download
RUN wget https://raw.githubusercontent.com/objectbox/objectbox-go/main/install.sh && \
     chmod +x install.sh && \
     ./install.sh && \     
     rm install.sh
RUN --mount=type=cache,target=/tmp/go-cache \
    --mount=type=cache,target=/go/pkg/mod \
 go build -v .
WORKDIR /src/makedraft_cli
RUN --mount=type=cache,target=/tmp/go-cache \
    --mount=type=cache,target=/go/pkg/mod \
 go build -v .


FROM node:23-slim AS nodebuilder
WORKDIR /src/client

COPY client/package.json client/package-lock.json* ./
RUN npm ci && npm cache clean --force

COPY client/. .
RUN npm run build



FROM node:23-slim AS filter_deploy
WORKDIR /srv/r38
COPY filter.js .
WORKDIR /srv/r38/socket

ENTRYPOINT ["node", "../filter.js"]



FROM debian:stable-slim AS go_deploy

RUN apt-get update && apt-get install -y ca-certificates curl sqlite3 && rm -rf /var/lib/apt/lists/*

WORKDIR /srv/r38

COPY --from=nodebuilder /src/client-dist /srv/r38/client-dist 
COPY --from=gobuilder /src/r38 /srv/r38/r38
COPY --from=gobuilder /src/makedraft_cli/makedraft_cli /srv/r38/makedraft_cli
COPY --from=gobuilder /src/sets /srv/r38/sets
COPY --from=gobuilder /src/objectboxlib/lib/libobjectbox.so /usr/local/lib/


# race condition with objectbox admin web app
RUN mkdir -p /srv/r38/db && \
    touch /srv/r38/db/data.mdb && \
    touch /srv/r38/db/lock.mdb

# the go app expects these in its working directory but they need to live on mounted volumes
RUN ln -s db/draft.db draft.db && \
    ln -s socket/r38.sock r38.sock && \
    ldconfig

EXPOSE 12264

CMD ["sh", "-c", "chown -R 1000:1000 /srv/r38/db && /srv/r38/r38"]
