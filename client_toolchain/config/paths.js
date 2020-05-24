const path = require('path');
const fs = require('fs');

const PROJECT_ROOT = findProjectRoot(__dirname);
const CLIENT_ROOT = path.resolve(PROJECT_ROOT, 'client');
const CLIENT_SRC_ROOT = path.resolve(PROJECT_ROOT, 'client/src');

module.exports = {
  PROJECT_ROOT, CLIENT_ROOT, CLIENT_SRC_ROOT
};


function findProjectRoot(currentPath) {
  if (currentPath == '/') {
    throw new Error(`Cannot find project root`);
  }
  if (fs.existsSync(path.join(currentPath, 'r38'))) {
    return currentPath;
  } else {
    return findProjectRoot(path.resolve(currentPath, '../'));
  }
}
