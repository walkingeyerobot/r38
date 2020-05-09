// Much of this is based on the Typescript Vue Starter page,
// https://github.com/Microsoft/TypeScript-Vue-Starter
// Sadly, some of the page is out of date.

// See https://webpack.js.org/configuration/ for docs on this config format.

const path = require('path');
const webpack = require('webpack');
const { CleanWebpackPlugin } = require('clean-webpack-plugin');
const VueLoaderPlugin = require('vue-loader/lib/plugin')


const PROJECT_ROOT = path.resolve(__dirname, '../../');
const CLIENT_ROOT = path.resolve(__dirname, '../');
const CLIENT_SRC_ROOT = path.join(CLIENT_ROOT, 'src');

module.exports = mode => {
  return {
    // Our compilation targets. Each one will include all of its dependencies
    // Right now, we just have one, named "app"
    entry: {
      app: path.resolve(CLIENT_SRC_ROOT, 'main.ts'),
    },

    output: {
      // The all-important `path` field is specified in .dev and .prod

      // Our compiled entry point -- the thing we include as a script tag in
      // our HTML. Right now we have just one entry point app, so this will be
      // app.bundle.js
      filename: '[name].bundle.js',

      // Our app is broken into chunks and loaded lazily when we need them. This
      // specifies their filenames. The chunks themselves are specified in
      // src/router/index.ts
      chunkFilename: '[name].bundle.js',

      // URL route that the webserver will serve our output from
      publicPath: '/static/dist/',
    },

    module: {
      rules: [

        // Compilation for Vue single file components (*.vue)
        {
          test: /\.vue$/,
          loader: 'vue-loader',
        },

        // Compilation for Typescript files
        {
          test: /\.tsx?$/,
          loader: 'ts-loader',
          exclude: /node_modules/,
          options: {
            configFile: path.resolve(PROJECT_ROOT, 'tsconfig.json'),
            appendTsSuffixTo: [/\.vue$/],
          },
        },

        // CSS processing (for both .vue files and normal .css files)
        {
          test: /\.css$/,
          use: [
            'vue-style-loader',
            // Converts url() and import@ references to dependencies and changes
            // them to refer to the final output filenames
            'css-loader'
          ]
        },

        // Images
        // TODO: Check if we want to include the hash here
        {
          test: /\.(png|jpg|gif|svg)$/,
          loader: 'file-loader',
          options: {
            name: '[name].[ext]?[hash]'
          }
        },
      ]
    },

    plugins: [
      // Cleans up any obsolete build artifacts (e.g. images that have since been
      // deleted).
      new CleanWebpackPlugin(),

      // Required for loading .vue files
      new VueLoaderPlugin(),

      new webpack.DefinePlugin({
        DEVELOPMENT: JSON.stringify(mode == 'development'),
        'process.env.NODE_ENV': JSON.stringify(mode),
      })
    ],

    resolve: {
      // Files with these extensions can be imported without specifying the
      // extension (e.g. './foo' vs. './foo.ts');
      extensions: [ '.tsx', '.ts', '.js', '.json' ],

      alias: {
        // Import alias. Causes `import vue from 'vue'` to point to precompiled
        // vue distributable.
        // The $ indicates an exact match is required.
        'vue$': 'vue/dist/vue.runtime.esm.js',
      },
    },
  };
}
