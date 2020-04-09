const path = require('path');
const merge = require('webpack-merge');

const common = require('./webpack.common');

const PROJECT_ROOT = path.resolve(__dirname, '../../');
const CLIENT_ROOT = path.resolve(__dirname, '../');
const CLIENT_SRC_ROOT = path.join(CLIENT_ROOT, 'src');

const OUT_PATH = path.join(CLIENT_ROOT, 'srv/static/dist');

module.exports = merge.smart(common, {

  mode: 'development',

  // Which approach to use while serving source maps
  // There are a dizzying array of options that trade accuracy for speed, etc.
  // See https://webpack.js.org/configuration/devtool/
  devtool: 'cheap-module-eval-source-map',

  output: {
    // Directory to write compiled JS and any static assets to
    // In development, we never actually write assets, but we use a different
    // location so we don't accidentally mess with checked-in assets.
    path: OUT_PATH,
  },

  devServer: {
    // Where to serve assets from
    // Currently we serve a "fake" index.html that we've just made to look very
    // similar to the real one that the go server has. At some point we should
    // merge them.
    contentBase: [
      path.join(CLIENT_ROOT, "srv")
    ],

    // The URL where the compiled assets should be served from
    publicPath: common.output.publicPath,

    // Use hot module replacement, allowing Vue components to be hot-replaced
    // without needing to refresh the page
    hot: true,

    // Causes the dev server to server index.html instead of 404s
    // Important if we want to handle URLs other than /
    historyApiFallback: true,
  },

  stats: 'minimal',
});
