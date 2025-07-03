var fs = require('fs');
var https = require('https');
var filename = process.argv[2];
var finalCards = [];
var fileType;
filename = '../cube_new.csv'
var foilIndex = 9;
var correctLineLen = null;
debugger;
fs.readFile(filename, 'utf8', (err, data) => {
  if (err) {
    console.error(err);
    return
  }
  fileType = filename.split('.')[0];
  fileType = 'cube';
  console.error('fileType: ' + fileType);
  if (fileType === 'cube') {
    var lines = data.split('\n');
    var rawCards = lines.map((e, i, a) => {
      var rawline = e.split(',');
      var line = [];
      for (var ii = 0; ii < rawline.length; ii++) {
        if (rawline[ii].endsWith('"') && !rawline[ii].startsWith('"')) {
          line[line.length - 1] += ',' + rawline[ii];
        } else {
          line.push(rawline[ii]);
        }
      }
      if (correctLineLen == null) {
        correctLineLen = line.length;
      } else {
        if (line.length !== correctLineLen) {
          console.error(line);
          console.error(line.length);
          throw Error("AHHHHHHHHHHHHHHHHHHHHHHHHHHHHHHHHHHHHHHHHHHH");
        }
      }
      console.error(line);
      var foilStatus = false;
      if (line[foilIndex] === 'Foil' || line[foilIndex] === 'Foil\r') {
        foilStatus = true;
      } else if (line[foilIndex] !== 'Non-foil' && line[foilIndex] !== 'Non-foil\r' && line[foilIndex] !== '') {
        throw Error('cannot determine foil status of: "' + line + '" (found "' + line[foilIndex] + '")');
      }
      return {
        set: line[4].replaceAll('"',''),
        collector_number: line[5].replaceAll('"',''),
        foil: foilStatus
      };
    });
    GetIndividualCards(rawCards);
  } else {
    var lines = data.split('\n');
    var rawCards = lines.map((e) => {
      var line = e.split(',');
      return {
        set: fileType,
        collector_number: line[0],
        foil: "FOIL_STATUS",
        rating: parseFloat(line[1])
      };
    });
    GetEntireSet('https://api.scryfall.com/cards/search?order=set&q=e%3A' + fileType + '&unique=prints', ProcessStandardCards.bind(null, rawCards));
  }
});

function GetEntireSet(url, cb) {
  https.get(url, (resp) => {
    var data = '';

    resp.on('data', (chunk) => {
      data += chunk;
    });

    resp.on('end', () => {
      var obj = JSON.parse(data);
      if (obj.has_more) {
        GetEntireSet(obj.next_page, (obj2) => {
          cb(obj.data.concat(obj2));
        });
      } else {
        cb(obj.data);
      }
    });
  }).on('error', (err) => {
    throw Error(err);
  });
}

function GetIndividualCards(rawCards) {
  var idx = 0;
  function GetSingleCard(rawCard) {
    console.error(rawCard);
    var url = 'https://api.scryfall.com/cards/' + rawCard.set + '/' + rawCard.collector_number;
    var opts = {
      headers: {
          'User-Agent': 'cube stuff (mitch@thefoley.net)',
          'Accept': 'text/json',
      }
  };
    console.error('requesting ' + idx + '/' + rawCards.length + ' ' + url);
    https.get(url, opts, (resp) => {
      var data = '';

      resp.on('data', (chunk) => {
        data += chunk;
      });

      resp.on('end', () => {
        var obj = JSON.parse(data);
        ProcessIndividualCard(rawCard, obj);
        if (idx < rawCards.length) {
          var qqqq = rawCards[idx++];
          var gsc = GetSingleCard.bind(undefined, qqqq);
          setTimeout(gsc, 500);
          //GetSingleCard(rawCards[idx++]);
        } else {
          ProcessAllCards();
        }
      });
    }).on('error', (err) => {
      throw Error(err);
    });
  }
  GetSingleCard(rawCards[idx++]);
}

function ProcessStandardCards(rawCards, scryfallCards) {
  var x = rawCards.forEach((rawCard) => {
    var scryfallCard = scryfallCards.find((elem) => {
      return elem.collector_number === rawCard.collector_number;
    });
    ProcessIndividualCard(rawCard, scryfallCard);
  });
  ProcessAllCards();
}

function ProcessIndividualCard(rawCard, scryfallCard) {
  if (!scryfallCard) {
    throw Error('couldn\'t find card "' + JSON.stringify(rawCard) + '"');
  }

  var r38Card = {
    foil: rawCard.foil,
    scryfall: {
      //id: scryfallCard.id, // this gets deleted later
      //oracle_id: scryfallCard.oracle_id, // this gets deleted later
      id: scryfallCard.oracle_id, // this gets deleted later

      cmc: scryfallCard.cmc,
      color_identity: scryfallCard.color_identity,
      layout: scryfallCard.layout,
      name: scryfallCard.name,
      type_line: scryfallCard.type_line,

      collector_number: scryfallCard.collector_number,
      rarity: scryfallCard.rarity,
      set: scryfallCard.set,
    },
  };

  if (scryfallCard.image_uris) {
    r38Card.image_uris = [scryfallCard.image_uris.normal];
  } else if (scryfallCard.card_faces && scryfallCard.card_faces.length === 2) {
    r38Card.image_uris = [
      scryfallCard.card_faces[0].image_uris.normal,
      scryfallCard.card_faces[1].image_uris.normal
    ];
  } else {
    throw Error('no face? no image? what?\n' + JSON.stringify(scryfallCard));
  }

  if (rawCard.rating != null) {
    r38Card.rating = rawCard.rating;
  }

  r38Card.mtgo_id = 0;
  /*
  if (fileType === 'cube') {
    if (scryfallCard.mtgo_id && scryfallCard.mtgo_foil_id) {
      r38Card.mtgo_id = rawCard.foil ? scryfallCard.mtgo_foil_id : scryfallCard.mtgo_id;
    } else if (scryfallCard.mtgo_id) {
      r38Card.mtgo_id = scryfallCard.mtgo_id;
      if (rawCard.foil) {
        r38Card.mtgo_id++;
      }
    } else {
      throw Error('card is weird:\n' + JSON.stringify(scryfallCard));
    }
  } else {
    r38Card.mtgo_id = scryfallCard.mtgo_id;
  }

  if (!r38Card.mtgo_id) {
    throw Error('no mtgo id set!');
  }
  */

  if (scryfallCard.card_faces) {
    r38Card.scryfall.card_faces = [];
    for (var i = 0; i < scryfallCard.card_faces.length; i++) {
      var face = scryfallCard.card_faces[i];
      var myFace = {
        mana_cost: face.mana_cost,
        name: face.name,
        type_line: face.type_line
      };
      if (face.colors != null) {
        myFace.colors = face.colors;
      }
      r38Card.scryfall.card_faces.push(myFace);
    }
  }

  if (scryfallCard.colors) {
    r38Card.scryfall.colors = scryfallCard.colors;
  }

  if (scryfallCard.mana_cost) {
    r38Card.scryfall.mana_cost = scryfallCard.mana_cost;
  }

  finalCards.push(r38Card);
}

function ProcessAllCards() {
  var finalObject;
  if (fileType === 'cube') {
    finalObject = {
      hoppers: [
        { type: 'CubeHopper' },
        { type: 'Pointer', refs: [0] },
        { type: 'Pointer', refs: [0] },
        { type: 'Pointer', refs: [0] },
        { type: 'Pointer', refs: [0] },
        { type: 'Pointer', refs: [0] },
        { type: 'Pointer', refs: [0] },
        { type: 'Pointer', refs: [0] },
        { type: 'Pointer', refs: [0] },
        { type: 'Pointer', refs: [0] },
        { type: 'Pointer', refs: [0] },
        { type: 'Pointer', refs: [0] },
        { type: 'Pointer', refs: [0] },
        { type: 'Pointer', refs: [0] },
        { type: 'Pointer', refs: [0] },
      ],
      flags: [],
    };
  } else {
    finalObject = {
      /*hoppers: [
        { type: 'RareHopper' },
        { type: 'UncommonHopper' },
        { type: 'Pointer', refs: [1] },
        { type: 'Pointer', refs: [1] },
        { type: 'CommonHopper' },
        { type: 'Pointer', refs: [4] },
        { type: 'Pointer', refs: [4] },
        { type: 'CommonHopper' },
        { type: 'Pointer', refs: [7] },
        { type: 'Pointer', refs: [7] },
        { type: 'CommonHopper' },
        { type: 'Pointer', refs: [10] },
        { type: 'DfcHopper' },
        { type: 'FoilHopper', refs: [4, 7, 10] },
        { type: 'BasicLandHopper' },
      ],
      flags: [
        "-dfc-mode=true",
        "-pack-common-color-stdev-max=1.5",
        "-pack-common-rating-min=1.5",
        "-pack-common-rating-max=3",
        "-draft-common-color-stdev-max=4",
      ],*/
      hoppers: [
        { type: 'RareHopper' },
        { type: 'UncommonHopper' },
        { type: 'Pointer', refs: [1] },
        { type: 'Pointer', refs: [1] },
        { type: 'CommonHopper' },
        { type: 'Pointer', refs: [4] },
        { type: 'Pointer', refs: [4] },
        { type: 'CommonHopper' },
        { type: 'Pointer', refs: [7] },
        { type: 'Pointer', refs: [7] },
        { type: 'CommonHopper' },
        { type: 'Pointer', refs: [10] },
        { type: 'Pointer', refs: [10] },
        { type: 'FoilHopper', refs: [4, 7, 10] },
        { type: 'BasicLandHopper' },
      ],
      flags: [
        "-pack-common-color-identity-stdev-max=1.55",
        "-pack-common-rating-min=1.8",
        "-pack-common-rating-max=3",
        "-draft-common-color-stdev-max=3",
        "-abort-missing-common-color-identity=true",
        "-abort-duplicate-three-color-identity-uncommons=true",
      ],
    };
  }
  finalObject.cards = finalCards.map((card) => {
    var id = card.scryfall.id;
    delete card.scryfall.id;
    var oracle_id = card.scryfall.oracle_id;
    delete card.scryfall.oracle_id;
    return {
      color: card.scryfall.colors ? card.scryfall.colors.join('') : card.scryfall.card_faces[0].colors.join(''),
      color_identity: card.scryfall.color_identity.join(''),
      dfc: card.scryfall.layout === 'transform',
      id: id,
      oracle_id: oracle_id,
      rarity: card.scryfall.type_line.includes('Basic Land') ? 'basic' : card.scryfall.rarity,
      rating: card.rating,
      data: JSON.stringify(card)
    }
  });
  console.log(JSON.stringify(finalObject, null, '  '));
}
