(function() {
  let S = 0;
  const packDiv = document.querySelector('#pack');
  const picksDiv = document.querySelector('#picks');
  let R = 1;
  function getDraftObject() {
    let Draft = JSON.parse(window.DraftString);

    Draft.events = Draft.events || [];

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

    Draft.groups = [];
    var eventsIndex = Draft.events.length;
    var groupIndex = 0;
    while (--eventsIndex >= 0) {
      var event = Draft.events[eventsIndex];
      var inserted = false;
      if (i === groupIndex && Draft.groups[i].every((v) => v != null)) {
        groupIndex++;
      }
      for (var i = groupIndex; i <= Draft.groups.length; i++) {
        if (i === Draft.groups.length) {
          Draft.groups.push([null,null,null,null,null,null,null,null]);
        }
        if (!Draft.groups[i][event.player]) {
          if (Draft.groups[i].every((v) => v == null || v.round === event.round)) {
            Draft.groups[i][event.player] = event;
            inserted = true;
            break;
          }
        }
      }
      if (!inserted) {
        console.log('bad');
        debugger;
      }
    }

    Draft.pastGroups = [];

    for (var i = 0; i < Draft.seats.length; i++) {
      document.querySelector('#seat>option[value="' + i + '"]').textContent = Draft.seats[i].name;
    }
    
    return Draft;
  }
  let Draft = getDraftObject();
  window.Draft = Draft;
  function doNext() {
    if (Draft.groups.length === 0) {
      return;
    }

    var group = Draft.groups.shift();
    Draft.pastGroups.unshift(group);

    for (var i = 0; i < group.length; i++) {
      if (group[i] == null) {
        continue;
      }
      var nextEvent = group[i];

      var pickedCardIndex = Draft.seats[nextEvent.player].rounds[R].packs[0].cards.findIndex(v => v && v.name === nextEvent.card1);
      var removedCard = Draft.seats[nextEvent.player].rounds[R].packs[0].cards[pickedCardIndex];
      Draft.seats[nextEvent.player].rounds[R].packs[0].cards[pickedCardIndex] = null;
      Draft.seats[nextEvent.player].rounds[0].packs[0].cards.push(removedCard);

      if (nextEvent.card2) {
        var pickedCardIndex = Draft.seats[nextEvent.player].rounds[R].packs[0].cards.findIndex(v => v && v.name === nextEvent.card2);
        var removedCard = Draft.seats[nextEvent.player].rounds[R].packs[0].cards[pickedCardIndex];
        Draft.seats[nextEvent.player].rounds[R].packs[0].cards[pickedCardIndex] = null;
        Draft.seats[nextEvent.player].rounds[0].packs[0].cards.push(removedCard);

        var librarianIndex = Draft.seats[nextEvent.player].rounds[0].packs[0].cards.findIndex(v => v && v.name === 'Cogwork Librarian');
        var removedCard = Draft.seats[nextEvent.player].rounds[0].packs[0].cards[librarianIndex];
        Draft.seats[nextEvent.player].rounds[0].packs[0].cards[librarianIndex] = null;
        Draft.seats[nextEvent.player].rounds[R].packs[0].cards.push(removedCard);

        if (nextEvent.player === S) {
          picksDiv.removeChild(picksDiv.querySelector('#cogworklibrarian'));
        }
      }

      var nextSeat;
      if (R % 2 === 0) {
        nextSeat = nextEvent.player - 1;
        if (nextSeat === -1) {
          nextSeat = 7;
        }
      } else {
        nextSeat = nextEvent.player + 1;
        if (nextSeat === 8) {
          nextSeat = 0;
        }
      }

      var packToPass = Draft.seats[nextEvent.player].rounds[R].packs.shift()
      Draft.seats[nextSeat].rounds[R].packs.push(packToPass);

      if (nextEvent.card1 === 'Lore Seeker' || nextEvent.card2 === 'Lore Seeker') {
        Draft.seats[nextEvent.player].rounds[R].packs.unshift({
          cards: Draft.extraPack
        });
        delete Draft.extraPack;
      }
    }

    if (!Draft.groups.length) {
      console.log('all done!');
    } else if (Draft.groups[0].every((v) => v == null || v.round === R + 1)) {
      console.log('moving to next round');
      R++;
    }

    displayCurrentState();
  }
  function doPrevious() {
    if (Draft.pastGroups.length === 0) {
      return;
    }

    var group = Draft.pastGroups.shift();
    Draft.groups.unshift(group);

    if (Draft.pastGroups[0].every((v) => v == null || v.round === R - 1)) {
      console.log('moving to previous round');
      R--;
    }

    for (var i = 0; i < group.length; i++) {
      if (group[i] == null) {
        continue;
      }

      var pastEvent = group[i];

      if (pastEvent.card1 === 'Lore Seeker' || pastEvent.card2 === 'Lore Seeker') {
        Draft.extraPack = Draft.seats[pastEvent.player].rounds[R].packs.shift().cards;
      }

      var previousSeat;
      if (R % 2 === 0) {
        previousSeat = pastEvent.player + 1;
        if (previousSeat === 8) {
          previousSeat = 0;
        }
      } else {
        previousSeat = pastEvent.player - 1;
        if (previousSeat === -1) {
          previousSeat = 7;
        }
      }

      var packToPass = Draft.seats[previousSeat].rounds[R].packs.pop();
      Draft.seats[pastEvent.player].rounds[R].packs.unshift(packToPass);
    }
  }
  function displayCurrentState() {
    var picks = Draft.seats[S].rounds[0].packs[0].cards;
    for (var i = 0; i < picks.length; i++) {
      if (!picks[i]) {
        continue;
      }
      if (!picksDiv.querySelector('#' + normalizeCardName(picks[i].name))) {
        addCardImage(picksDiv, picks[i]);
      }
    }

    while (packDiv.firstChild) {
      packDiv.removeChild(packDiv.firstChild);
    }

    var pack = Draft.seats[S].rounds[R].packs[0].cards;
    for (var i = 0; i < pack.length; i++) {
      addCardImage(packDiv, pack[i]);
    }

    if (Draft.groups.length === 0) {
      var txtDiv = document.createElement('div');
      txtDiv.textContent = 'Draft Over!';
      packDiv.append(txtDiv);
    } else {
      var nextEvent = Draft.groups[0][S];
      if (nextEvent) {
        document.querySelector('#' + normalizeCardName(nextEvent.card1)).classList.add('selected');
        if (nextEvent.card2) {
          document.querySelector('#' + normalizeCardName(nextEvent.card2)).classList.add('selected');
          document.querySelector('#cogworklibrarian').classList.add('selected');
        }
      } else {
        var txtDiv = document.createElement('div');
        txtDiv.textContent = 'Waiting on other players...';
        packDiv.append(txtDiv);
      }
    }
  }
  function addCardImage(div, card) {
    if (!card) {
      return;
    }
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
  function switchSeat(newseat) {
    if (S !== newseat) {
      S = newseat;
      while (picksDiv.firstChild) {
        picksDiv.removeChild(picksDiv.firstChild);
      }
      displayCurrentState();
    } else {
      console.log('already on that seat');
    }
  }
  function switchSeatEvent(e) {
    switchSeat(parseInt(e.target.value, 10));
  }
  window.next = doNext;
  window.prev = doPrevious;
  window.seat = switchSeat;
  displayCurrentState();

  document.querySelector('#next').addEventListener('click', doNext);
  document.querySelector('#seat').addEventListener('input', switchSeatEvent);
}())
