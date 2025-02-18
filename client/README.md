# Client

## Setup

_Note: all commands should be executed from the `/client` directory._

Install the "Current" version of [Node.js](https://nodejs.org/).

Then:

```
$ npm ci
```

## Deploying

To build the production version of the client, run:

```
$ npm run build
```

The build artifacts will be placed in `<git root>/static/dist`.

## Development

Start the development server:

```
$ npm run dev
```

Then visit [http://localhost:5173] (or whatever it tells you) in your browser
to see your local version.

Note that you must also be running the Go server in another shell. The dev
server will proxy all REST calls to the backend.

## Recommended editor setup

I strongly recommend using [VS Code](https://code.visualstudio.com/).

In VSCode, go `File > New Window` and then `File > Open` and select the root of
the (entire) project.

Install the extensions that it prompts you to install, the most important of
them being the one named "Vue - Official".
