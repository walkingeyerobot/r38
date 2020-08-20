import express from 'express';
import webpack from 'webpack';
import webpackDevMiddleware from 'webpack-dev-middleware';
import webpackHotMiddleware from 'webpack-hot-middleware';
import { createProxyMiddleware } from 'http-proxy-middleware';

const webpackConfig = require('../../config/webpack.dev');

function main() {
  let app = express();

  configureDevServing(app);
  configureProxy(app);

  app.listen(8080, () => {
    console.log('Listening on http://localhost:8080');
    console.log();
  });
}

function configureDevServing(app: express.Express) {
  const compiler = webpack(webpackConfig);
  app.use(webpackDevMiddleware(compiler, {
    publicPath: webpackConfig.output.publicPath,
    stats: "minimal",
  }));
  app.use(webpackHotMiddleware(compiler));
}

function configureProxy(app: express.Express) {
  app.use('/', createProxyMiddleware({
    target: 'http://localhost:12270',
    changeOrigin: true,
  }));
}

main();
