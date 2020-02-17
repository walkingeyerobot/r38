(function() {
  for (var i = 0; i < Draft.events.length; i++) {
    Draft.events[i].announcements = Draft.events[i].announcements.filter( v => v != '' );
  }

  Draft.events.sort((a, b) => {
    if (a.playerModified < b.playerModified) {
      return -1;
    }
    if (a.playerModified > b.playerModified) {
      return 1;
    }
    if (a.player < b.player) {
      return -1;
    }
    if (a.player > b.player) {
      return 1;
    }
    return 0;
  });
}())
