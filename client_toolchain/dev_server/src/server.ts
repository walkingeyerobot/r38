import express from 'express';
import webpack from 'webpack';
import webpackDevMiddleware from 'webpack-dev-middleware';
import webpackHotMiddleware from 'webpack-hot-middleware';
import bodyParser from 'body-parser';
import { serveIndex } from './route/serveIndex';

const webpackConfig = require('../../config/webpack.dev');

function main() {
  let app = express();

  configureExpress(app);
  configureRoutes(app);

  app.listen(8080, 'localhost', () => {
    console.log('Listening on http://localhost:8080');
  });
}

function configureExpress(app: express.Express) {
  app.use(bodyParser.json());

  const compiler = webpack(webpackConfig);
  app.use(webpackDevMiddleware(compiler, {
    publicPath: webpackConfig.output.publicPath,
    stats: "minimal",
  }));
  app.use(webpackHotMiddleware(compiler));
}

function configureRoutes(app: express.Express) {
  for (let htmlPath of HTML_PATHS) {
    app.get(htmlPath, serveIndex);
  }
}

const HTML_PATHS = [
  '/',
  '/login',

  '/draft/:id',

  '/replay/*',
  '/deckbuilder/*',
]

main();
