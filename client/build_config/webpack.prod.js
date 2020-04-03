const path = require('path');
const merge = require('webpack-merge');

const common = require('./webpack.common');

const PROJECT_ROOT = path.resolve(__dirname, '../../');
const CLIENT_ROOT = path.resolve(__dirname, '../');
const CLIENT_SRC_ROOT = path.join(CLIENT_ROOT, 'src');

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
