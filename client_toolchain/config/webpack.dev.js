const path = require('path');
const webpack = require('webpack');
const { merge } = require('webpack-merge');

const common = require('./webpack.common')('development');

const { PROJECT_ROOT, CLIENT_ROOT, CLIENT_SRC_ROOT } = require('./paths')

const OUT_PATH = path.join(CLIENT_ROOT, 'srv/static/dist');

module.exports = merge(common, {

  mode: 'development',

  // Add another entry point to make sure we include the hot module replacement
  // client (this will be in addition to main.ts, which is defined in common)
  entry: {
    app: ['webpack-hot-middleware/client?noInfo=true'],
  },

  // Which approach to use while serving source maps
  // There are a dizzying array of options that trade accuracy for speed, etc.
  // See https://webpack.js.org/configuration/devtool/
  devtool: 'eval-cheap-module-source-map',

  output: {
    // Directory to write compiled JS and any static assets to
    // In development, we never actually write assets, but we use a different
    // location so we don't accidentally mess with checked-in assets.
    path: OUT_PATH,
  },

  plugins: [
    // Allows for code replacement without page refresh
    new webpack.HotModuleReplacementPlugin(),
  ],
});
