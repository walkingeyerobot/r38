(function() {
  const S = 4;
  const packDiv = document.querySelector('#pack');
  const picksDiv = document.querySelector('#picks');
  let R = 1;
  function getDraftObject() {
    let Draft = JSON.parse(window.DraftString);

    Draft.events.sort((a, b) => {
      if (a.round < b.round) {
        return 1;
      }
      if (a.round > b.round) {
        return -1;
      }
      if (a.playerModified < b.playerModified) {
        return 1;
      }
      if (a.playerModified > b.playerModified) {
        return -1;
      }
      if (a.player < b.player) {
        return 1;
      }
      if (a.player > b.player) {
        return -1;
      }
      return 0;
    });

    Draft.pastEvents = [];
    return Draft;
  }
  let Draft = getDraftObject();
  window.Draft2 = Draft;
  function findPackInfo(cardName) {
    for (var i = 0; i < 8; i++) {
      var seat = Draft.seats[i];
      for (var j = 0; j < seat.rounds.length; j++) {
        var round = seat.rounds[j];
        for (var k = 0; k < round.packs.length; k++) {
          var pack = round.packs[k];
          for (var l = 0; l < pack.cards.length; l++) {
            var card = pack.cards[l];
            if (card.name === cardName) {
              return {
                seatIndex: i,
                roundIndex: j,
                packIndex: k,
                cardIndex: l,
                seat: seat,
                round: round,
                pack: pack,
                card: card
              }
            }
          }
        }
      }
    }
    console.log('bad');
  }
  function doNext() {
    var lastEvent = Draft.events[Draft.events.length-1];
    var modified = lastEvent.playerModified;
    var round = lastEvent.round;
    do {
      var nextEvent = Draft.events.pop();
      Draft.pastEvents.push(nextEvent);
      var packInfo = findPackInfo(nextEvent.card1);
      packInfo.seat.rounds[0].packs[0].cards.push(packInfo.card);
      packInfo.pack.cards.splice(packInfo.cardIndex, 1);
      if (nextEvent.card2) {
        packInfo = findPackInfo(nextEvent.card2);
        packInfo.seat.rounds[0].packs[0].cards.push(packInfo.card);
        packInfo.pack.cards.splice(packInfo.cardIndex, 1);

        var librarianInfo = findPackInfo('Cogwork Librarian');
        packInfo.pack.cards.push(librarianInfo.pack.cards.splice(librarianInfo.cardIndex, 1)[0]);
      }
      var nextSeat;
      if (R % 2 === 0) {
        nextSeat = packInfo.seatIndex - 1;
        if (nextSeat === -1) {
          nextSeat = 7;
        }
      } else {
        nextSeat = packInfo.seatIndex + 1;
        if (nextSeat === 8) {
          nextSeat = 0;
        }
      }
      Draft.seats[nextSeat].rounds[R].packs.push(packInfo.pack);
      packInfo.round.packs.splice(packInfo.packIndex, 1);

      if (nextEvent.card1 === 'Lore Seeker') {
        packInfo.round.packs.unshift({
          cards: Draft.extraPack
        });
      }

      var nextRound = true;
      for (var i = 0; i < 8; i++) {
        var packs = Draft.seats[i].rounds[R].packs;
        for (var j = 0; j < packs.length; j++) {
          if (packs[j].cards.length > 0) {
            nextRound = false;
            break;
          }
        }
        if (!nextRound) {
          break;
        }
      }

      if (nextRound) {
        console.log('moving to next round');
        R++;
      }
    } while (Draft.events.length > 0 && Draft.events[Draft.events.length-1].playerModified === modified && Draft.events[Draft.events.length-1].round === round);
    displayCurrentState();
  }
  function doPrevious() {
    var lastEvent = Draft.pastEvents.pop();
    Draft.events.push(lastEvent);
    displayCurrentState();
  }
  function displayCurrentState() {
    while (picksDiv.firstChild) {
      picksDiv.removeChild(picksDiv.firstChild);
    }
    var picks = Draft.seats[S].rounds[0].packs[0].cards;
    for (var i = 0; i < picks.length; i++) {
      addCardImage(picksDiv, picks[i]);
    }

    while (packDiv.firstChild) {
      packDiv.removeChild(packDiv.firstChild);
    }

    var pack = Draft.seats[S].rounds[R].packs[0].cards;
    for (var i = 0; i < pack.length; i++) {
      addCardImage(packDiv, pack[i]);
    }

    for (var i = Draft.events.length - 1; i >= 0; i--) {
      if (Draft.events[i].player !== S) {
        continue;
      }
      document.querySelector('#' + normalizeCardName(Draft.events[i].card1)).classList.add('selected');
      if (Draft.events[i].card2) {
        document.querySelector('#' + normalizeCardName(Draft.events[i].card2)).classList.add('selected');
      }
      break;
    }
  }
  function addCardImage(div, card) {
    var ret = document.createElement('div');
    var img = document.createElement('img');
    var name = document.createElement('div');
    ret.append(img, name);
    img.src = 'http://api.scryfall.com/cards/' + card.edition + '/' + card.number + '?format=image&version=normal';
    img.height = '300';
    name.textContent = card.name;
    ret.id = normalizeCardName(card.name);
    ret.classList.add('card');
    if (card.tags) {
      var tags = document.createElement('div');
      tags.textContent = card.tags;
      ret.append(tags);
    }

    div.append(ret);
  }
  function normalizeCardName(n) {
    return n.replace(/[\W]/g, '').toLowerCase();
  }
  window.next = doNext;
  window.prev = doPrevious;
  displayCurrentState();
}())
