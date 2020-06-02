const fs = require('fs');
const net = require('net');
const SOCK_ADDR = './r38.sock';
const STOP_SEQ = new Uint8Array([13, 10, 13, 10]); //'\r\n\r\n'

const server = net.createServer((c) => {
  var allData;
  console.error('client connected');
  c.on('end', () => {
    console.error('client disconnected');
  });
  c.on('data', (data) => {
    if (!allData) {
      allData = data;
    } else {
      allData = Buffer.concat([allData, data], allData.length + data.length);
    }
    var matches = true;
    for (var i = data.length - 4; i < data.length; i++) {
      var x = i - data.length + 4;
      if (data.readUInt8(i) !== STOP_SEQ[i - data.length + 4]) {
        matches = false;
        break;
      }
    }
    if (matches) {
      doParse2(c, allData.toString());
    }
  });
});

server.on('error', (err) => {
  console.error('got error when starting server');
  if (err.code === 'EADDRINUSE') {
    console.error('address in use');
    var clientSocket = new net.Socket();
    clientSocket.on('error', (err) => {
      if (err.code === 'ECONNREFUSED') {
        console.error('removing old socket');
        fs.unlinkSync(SOCK_ADDR);
        server.listen(SOCK_ADDR);
      } else if (err.code === 'EACCES') {
        console.error('cannot remove old socket, EACCES. probably ran this as the wrong user?');
        process.exit();
      } else {
        throw err;
      }
    });
    clientSocket.connect({path: SOCK_ADDR}, () => {
      console.error('server already running, exiting...');
      process.exit();
    });
  } else {
    throw err;
  }
});

server.listen(SOCK_ADDR, () => {
  console.error('server bound');
});

function stopServer() {
  console.error('closing...');
  if (server) {
    server.unref();
    server.close();
  }
}

process.on('exit', stopServer);
process.on('SIGINT', stopServer);
process.on('SIGTERM', stopServer);

function doParse2(client, objstr) {
  var obj = JSON.parse(objstr);
  var state = JSON.parse(objstr).draft.seats;
  var myPosition = obj.draft.seats.findIndex((elem) => elem.playerId === obj.user);
  var oddPasser = myPosition === 0 ? 7 : myPosition - 1;
  var evenPasser = myPosition === 7 ? 0 : myPosition + 1;
  var passer = [undefined, oddPasser, evenPasser, oddPasser];
  var newEvents = [];
  var librarian;

  // create a map of which pack ever card lives in.
  // this isn't strictly necessary as we can limit our card searches to the pack
  // that the player has available, but it's good to have to verify all events
  // are valid.
  var cardToPackAndIndex = {};
  for (var i = 0; i < state.length; i++) {
    state[i].round = 1;
    var packs = state[i].packs
    for (var j = 0; j < packs.length; j++) {
      var pack = packs[j];
      pack.startSeat = i;
      for (var k = 0; k < pack.length; k++) {
        cardToPackAndIndex[pack[k].r38_data.id] = { pack: pack, index: k };
        if (pack[k].name === 'Cogwork Librarian') {
          if (librarian) {
            throw Error('Cannot have multiple Cogwork Librarians in a draft without rewriting this logic.');
          }
          librarian = pack[k];
        }
      }
      packs[j] = [packs[j]];
    }
  }

  // a map that indicates if a pack has been seen by the player or not
  var packSeen = [
    [false, false, false],
    [false, false, false],
    [false, false, false],
    [false, false, false],
    [false, false, false],
    [false, false, false],
    [false, false, false],
    [false, false, false],
  ];
  // the player is always allowed to see their pack 1
  packSeen[myPosition][0] = true;

  // a map from a pack's starting seat + round to what cards have been picked
  // since the watched player last saw that pack
  var shadowCards = {};
  // a map from a pack's starting seat + round to the timestamp of the event
  // that last added a card to this list. this is important because we want
  // shadow pick events to have stable draftModified values.
  var shadowModified = {};
  
  for (var i = 0; i < obj.draft.events.length; i++) {
    var event = obj.draft.events[i];
    var pi = cardToPackAndIndex[event.cards[0]];
    var shadowKey = pi.pack.startSeat + '|' + event.round;

    if (event.position === myPosition) {
      // the player we're watching made a pick. if previous cards
      // have been picked from this pack, record those picks first
      if (shadowCards[shadowKey]) {
        newEvents.push({
          announcements: [],
          cards: shadowCards[shadowKey],
          draftModified: shadowModified[shadowKey] + 0.5,
          librarian: false,
          position: -1,
          round: event.round,
          type: 'ShadowPick',
        });
        delete shadowCards[shadowKey];
        delete shadowModified[shadowKey];
      }

      // event.type should be set in go. I'll do it later.
      event.type = 'Pick';
      newEvents.push(event);
    } else {
      // another player made a pick. just note that a card was picked and when
      // so the pack can be properly passed around by the UI.
      newEvents.push({
        announcements: [],
        draftModified: event.draftModified,
        librarian: event.librarian,
        playerModified: event.playerModified,
        position: event.position,
        round: event.round,
        type: 'SecretPick'
      });

      // add the card that was picked to the list of picked cards from that pack.
      shadowCards[shadowKey] = event.cards.concat(shadowCards[shadowKey] || []);
      shadowCards[shadowKey].sort(); // sort by card id so pick order can't be deduced.
      shadowModified[shadowKey] = event.draftModified;
    }

    // now do the event. the purpose of the rest of this for loop body is to mark packs as seen by the player.
    if (event.librarian) {
      if (!librarian) {
        throw Error('tried to place librarian but could not find it');
      }
      // replace the first card picked with cogwork librarian and remove the second card picked
      var pi2 = cardToPackAndIndex[event.cards[1]];
      pi.pack[pi.index] = librarian
      pi2.pack[pi2.index] = null;
    } else {
      // remove the picked card
      pi.pack[pi.index] = null;
    }

    // figure out where the pack is going
    var nextPos = event.position;
    if (event.round % 2 === 1) {
      nextPos++;
      if (nextPos === 8) {
        nextPos = 0;
      }
    } else {
      nextPos--;
      if (nextPos === -1) {
        nextPos = 7;
      }
    }

    // sanity check the round
    var stateRound = state[event.position].round;
    if (stateRound !== event.round) {
      throw Error('problem with rounds');
    }

    // sanity check the pack being picked from agrees with our current state
    if (state[event.position].packs[event.round - 1][0] !== pi.pack) {
      throw Error('problem with pack location');
    }

    // do the actual passing
    var passedPack = state[event.position].packs[event.round - 1].shift();
    state[nextPos].packs[event.round - 1].push(passedPack);

    var startSeat = pi.pack.startSeat;
    if (event.position === myPosition) {
      // if the player we're watching just picked a card, they have seen the pack
      packSeen[startSeat][event.round - 1] = true;
    } else if (!packSeen[startSeat][event.round - 1]) {
      // if another player has picked a card from this pack, and the player
      // we're watching has never seen this pack, mark the card as forever hidden
      var oldPack = obj.draft.seats[startSeat].packs[event.round - 1];
      var oldCard = oldPack[pi.index];
      oldPack[pi.index] = {
        name: 'Forever Unknown Card',
        r38_data: {id: oldCard.r38_data.id, hidden: true}
      };
    }

    // if the whole pack is empty, increment the round for that player
    if (passedPack.every((x) => !x)) {
      state[event.position].round++;
    }
  }

  // now that we've translated all the events and hidden all the forever unknown cards,
  // we need to see if there is a pack the watched player is able to pick from.
  // if there is, mark that pack as seen and add that pack's most recent shadow pick
  // to the event list
  var myRound = state[myPosition].round;
  var availablePack = state[myPosition].packs[myRound - 1][0];
  if (availablePack) {
    var ss = availablePack.startSeat;
    packSeen[ss][myRound - 1] = true;
    var shadowKey = ss + '|' + myRound;
    if (shadowCards[shadowKey]) {
      newEvents.push({
        announcements: [],
        cards: shadowCards[shadowKey],
        draftModified: shadowModified[shadowKey] + 0.5,
        librarian: false,
        position: -1,
        round: event.round,
        type: 'ShadowPick',
      });
      delete shadowCards[shadowKey];
      delete shadowModified[shadowKey];
    }
  }

  // mark all cards not yet seen as unknown cards
  for (var i = 0; i < packSeen.length; i++) {
    for (var j = 0; j < packSeen[i].length; j++) {
      if (!packSeen[i][j]) {
        obj.draft.seats[i].packs[j] = obj.draft.seats[i].packs[j].map((card) => {
          if (card.r38_data.hidden) {
            return card;
          }
          return {
            name: 'Currently Unknown Card',
            r38_data: {id: card.r38_data.id, hidden: true}
          };
        });
      }
    }
  }

  newEvents.sort((a, b) => {
    if (a.draftModified < b.draftModified) {
      return -1;
    } else if (a.draftModified > b.draftModified) {
      return 1;
    }
    throw Error('duplicate draftModified values');
  });

  obj.draft.events = newEvents;
  // console.log(JSON.stringify(obj.draft));
  client.end(JSON.stringify(obj.draft));
  client.unref();
}
