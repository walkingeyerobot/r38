var fs = require('fs');
var https = require('https');
var filename = process.argv[2];
var finalCards = [];
var fileType;

fs.readFile(filename, 'utf8', (err, data) => {
  if (err) {
    console.error(err);
    return
  }
  fileType = filename.split('.')[0];
  console.error('fileType: ' + fileType);
  if (fileType === 'cube') {
    var lines = data.split('\n');
    var rawCards = lines.map((e) => {
      var line = e.split(',');
      var foilStatus = false;
      if (line[2] === 'Foil') {
        foilStatus = true;
      } else if (line[2] !== 'Non-foil') {
        throw Error('cannot determine foil status of: "' + line + '"');
      }
      return {
        set: line[0],
        collector_number: line[1],
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
    var url = 'https://api.scryfall.com/cards/' + rawCard.set + '/' + rawCard.collector_number;
    console.error('requesting ' + idx + '/' + rawCards.length);
    https.get(url, (resp) => {
      var data = '';

      resp.on('data', (chunk) => {
        data += chunk;
      });

      resp.on('end', () => {
        var obj = JSON.parse(data);
        if (idx < rawCards.length) {
          ProcessIndividualCard(rawCard, obj);
          GetSingleCard(rawCards[idx++]);
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
  var card = scryfallCard;
  if (!card) {
    throw Error('couldn\'t find card "' + JSON.stringify(rawCard) + '"');
  }
  var myCard = {
    r38_data: {
      foil: rawCard.foil,
    },

    id: card.id, // this gets deleted later

    cmc: card.cmc,
    color_identity: card.color_identity,
    layout: card.layout,
    name: card.name,
    type_line: card.type_line,

    collector_number: card.collector_number,
    rarity: card.rarity,
    set: card.set,
  };

  if (card.image_uris) {
    myCard.r38_data.image_uris = [card.image_uris.normal];
  } else if (card.card_faces && card.card_faces.length === 2) {
    myCard.r38_data.image_uris = [
      card.card_faces[0].image_uris.normal,
      card.card_faces[1].image_uris.normal
    ];
  } else {
    throw Error('no face? no image? what?\n' + JSON.stringify(card));
  }

  if (rawCard.rating != null) {
    myCard.r38_data.rating = rawCard.rating;
  }

  if (fileType === 'cube') {
    if (card.mtgo_id && card.mtgo_foil_id) {
      myCard.r38_data.mtgo_id = rawCard.foil ? card.mtgo_foil_id : card.mtgo_id;
    } else if (card.mtgo_id) {
      myCard.r38_data.mtgo_id = card.mtgo_id;
      if (rawCard.foil) {
        myCard.r38_data.mtgo_id++;
      }
    } else {
      throw Error('card is weird:\n' + JSON.stringify(card));
    }
  } else {
    myCard.r38_data.mtgo_id = card.mtgo_id;
  }

  if (!myCard.r38_data.mtgo_id) {
    throw Error('no mtgo id set!');
  }

  if (card.card_faces) {
    myCard.card_faces = [];
    for (var i = 0; i < card.card_faces.length; i++) {
      var face = card.card_faces[i];
      var myFace = {
        mana_cost: face.mana_cost,
        name: face.name,
        type_line: face.type_line
      };
      if (face.colors != null) {
        myFace.colors = face.colors;
      }
      myCard.card_faces.push(myFace);
    }
  }

  if (card.colors) {
    myCard.colors = card.colors;
  }

  if (card.mana_cost) {
    myCard.mana_cost = card.mana_cost;
  }

  finalCards.push(myCard);
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
      ],
/*      hoppers: [
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
        "-pack-common-color-identity-stdev-max=0.8",
        "-pack-common-rating-min=1.8",
        "-pack-common-rating-max=3",
        "-draft-common-color-stdev-max=3",
        "-abort-missing-common-color-identity=true",
        "-abort-duplicate-three-color-identity-uncommons=true",
      ],*/
    };
  }
  finalObject.cards = finalCards.map((card) => {
    var id = card.id;
    delete card.id;
    return {
      cmc: card.cmc, // temporary
      collector_number: card.collector_number,
      color: card.colors ? card.colors.join('') : card.card_faces[0].colors.join(''),
      color_identity: card.color_identity.join(''),
      dfc: card.layout === 'transform',
      id: id,
      mtgo_id: card.r38_data.mtgo_id, // temporary
      name: card.name, // temporary
      rarity: card.type_line.includes('Basic Land') ? 'basic' : card.rarity,
      rating: card.r38_data.rating,
      set: card.set, // temporary
      type_line: card.type_line, // temporary
      data: JSON.stringify(card)
    }
  });
  console.log(JSON.stringify(finalObject, null, '  '));
}
