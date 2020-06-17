const path = require('path');
const webpack = require('webpack');
const merge = require('webpack-merge');

const common = require('./webpack.common')('production');

const { PROJECT_ROOT, CLIENT_ROOT, CLIENT_SRC_ROOT } = require('./paths')

module.exports = merge.smart(common, {
  mode: 'production',

  // Emit a source map, even for production. Recommended by webpack, but means
  // we have to serve the source map as well
  devtool: 'source-map',

  output: {
    // Directory to write compiled JS and any static assets to.
    // For production, we write these to the place where the go server serves
    // from.
    path: path.resolve(PROJECT_ROOT, 'static/dist'),
  },
});
