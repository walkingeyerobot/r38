import { SourceEvent } from './SourceData';
import { commitTimelineEvent } from '../draft/mutate';
import { checkNotNil } from '../util/checkNotNil';
import { cloneDraftState } from '../draft/cloneDraftState';
import { DraftState, CardContainer, CardPack, DraftSeat } from '../draft/DraftState';
import { TimelineEvent, PACK_LOCATION_UNUSED, PACK_LOCATION_DEAD } from '../draft/TimelineEvent';

export class TimelineGenerator {
  private _state!: DraftState;
  private _srcEvents!: SourceEvent[];

  private _outEvents = [] as TimelineEvent[];
  private _committedIndex = 0;

  private _eventIndex = 0;
  private _nextTimelineId = 0;
  private _playerTrackers = [] as PlayerTracker[];

  generate(state: DraftState, events: SourceEvent[]) {
    this.initialize(cloneDraftState(state), events);
    while (this.next()) {}
    return this._outEvents.concat();
  }

  manage(state: DraftState, events: SourceEvent[]) {
    this.initialize(state, events);
  }

  private initialize(state: DraftState, events: SourceEvent[]) {
    this._state = state;
    this._srcEvents = events;
    this._outEvents = [];
    this._committedIndex = 0;
    this._eventIndex = 0;
    this._nextTimelineId = 0;
    this._playerTrackers = [];

    // Fill out our player trackers
    for (const seat of this._state.seats) {
      this._playerTrackers.push({
        currentRound: 1,
        nextEpoch: 0,
      });
    }

    this.openFirstPacks();
  }

  next(): boolean {
    if (this._committedIndex >= this._outEvents.length) {
      try {
        this.pushNext();
      } catch (e) {
        console.log(
            `Error parsing event`,
            this._srcEvents[this._eventIndex - 1]);
        throw e;
      }
    }

    if (this._committedIndex < this._outEvents.length) {
      // const event = this._outEvents[this._committedIndex];
      // console.log('EVENT', this._committedIndex, event);
      // for (const action of event.actions){
      //   console.log('  ', action);
      // }
      commitTimelineEvent(this._outEvents[this._committedIndex], this._state);
      this._committedIndex++;
      return true;
    } else {
      return false;
    }
  }

  private pushNext() {
    if (this._eventIndex >= this._srcEvents.length) {
      return;
    }
    const srcEvent = this._srcEvents[this._eventIndex];
    this._eventIndex++;

    const playerData = this._playerTrackers[srcEvent.player];

    const outEvent: TimelineEvent = {
      id: this._nextTimelineId++,
      round: srcEvent.round,
      roundEpoch: playerData.nextEpoch++,
      associatedSeat: srcEvent.player,
      pick: 45, // TODO
      actions: [],
    };

    const seat = getSeat(srcEvent, this._state);

    if (srcEvent.round != playerData.currentRound) {
      if (srcEvent.round != playerData.currentRound + 1) {
        throw new Error(`Unexpected round: ${srcEvent.round}`);
      }
      this._eventIndex--;
      playerData.currentRound++;
      playerData.nextEpoch = 1;
      outEvent.roundEpoch = 0;

      const nextPack = checkNotNil(seat.unopenedPacks[0]);

      outEvent.actions.push({
        type: 'move-pack' as const,
        pack: nextPack.id,
        from: { seat: seat.position, queue: 'unopened' },
        to: { seat: seat.position, queue: 'opened' },
        insertAction: 'unshift',
      });
      this._outEvents.push(outEvent);

      // TODO: Don't do an early return here
      return;
    }

    if (seat.queuedPacks.length == 0) {
      throw new Error(`Seat ${seat.position} doesn't have any open packs!`);
    }
    const activePack = seat.queuedPacks[0];

    const card = findCard(srcEvent.card1, activePack);
    outEvent.actions.push({
      type: 'move-card' as const,
      cardName: card.definition.name,
      card: card.id,
      from: activePack.id,
      to: seat.player.picks.id,
    });


    if (srcEvent.card2 != "") {
      // This means that someone used Cogwork Librarian's ability to draft
      // multiple cards. There doesn't appear to be a better way to know that
      // this happened
      const card2 = findCard(srcEvent.card2, activePack);
      outEvent.actions.push({
        type: 'move-card' as const,
        cardName: card2.definition.name,
        card: card2.id,
        from: activePack.id,
        to: seat.player.picks.id,
      });

      const librarianCard = findCard('Cogwork Librarian', seat.player.picks);
      outEvent.actions.push({
        type: 'move-card' as const,
        cardName: librarianCard.definition.name,
        card: librarianCard.id,
        from: seat.player.picks.id,
        to: activePack.id,
      });
    }

    // Move the pack to the next seat
    const nextSeat = getSeatToPassTo(activePack, seat, playerData, this._state);

    outEvent.actions.push({
      type: 'move-pack' as const,
      pack: activePack.id,
      from: {
        seat: seat.position,
        queue: 'opened',
      },
      to: {
        seat: nextSeat,
        queue: 'opened',
      },
      insertAction: 'enqueue',
    });

    // Finally, Lore Seeker -____-
    // If a player drafts Lore Seeker, we assume they chose to add a new pack
    // to the draft
    if (srcEvent.card1 == 'Lore Seeker' || srcEvent.card2 == 'Lore Seeker') {
      if (this._state.unusedPacks.length == 0) {
        throw new Error(`No more packs to add`);
      }

      outEvent.actions.push({
        type: 'move-pack' as const,
        pack: this._state.unusedPacks[0].id,
        from: {
          seat: PACK_LOCATION_UNUSED,
          queue: 'unopened',
        },
        to: {
          seat: seat.position,
          queue: 'opened',
        },
        insertAction: 'unshift',
      });
    }

    this._outEvents.push(outEvent);
  }

  private openFirstPacks() {
    for (const seat of this._state.seats) {
      if (seat.unopenedPacks.length == 0) {
        console.warn(`Seat ${seat.position} doesn't have any packs!`);
        continue;
      }

      const event: TimelineEvent = {
        id: this._nextTimelineId++,
        associatedSeat: seat.position,
        round: 1,
        roundEpoch: this._playerTrackers[seat.position].nextEpoch++,
        pick: 0,
        actions: [{
          type: 'move-pack' as const,
          pack: seat.unopenedPacks[0].id,
          from: {
            seat: seat.position,
            queue: 'unopened',
          },
          to: {
            seat: seat.position,
            queue: 'opened',
          },
          insertAction: 'enqueue',
        }]
      };
      this._outEvents.push(event);
    }
  }
}

function getSeat(srcEvent: SourceEvent, state: DraftState) {
  const seat = state.seats[srcEvent.player];
  if (seat == null) {
    throw new Error(
        `Unknown player "${srcEvent.player}" in event `
            + JSON.stringify(srcEvent));
  }
  return seat;
}

function findCard(cardName: string, container: CardContainer) {
  for (const card of container.cards) {
    if (card.definition.name == cardName) {
      return card;
    }
  }
  throw new Error(`Card "${cardName}" not found in container ${container.id}.`);
}

function getSeatToPassTo(
    pack: CardPack,
    seat: DraftSeat,
    playerData: PlayerTracker,
    state: DraftState,
) {
  if (pack.cards.length <= 1) {
    return PACK_LOCATION_DEAD
  } else {
    let nextSeatId = seat.position;
    if (playerData.currentRound % 2 == 1) {
      nextSeatId++;
    } else {
      nextSeatId--;
    }
    const numSeats = state.seats.length;

    // This is essentially just (nextSeatId % numSeats), but also works for
    // negative numbers
    return ((nextSeatId % numSeats) + numSeats) % numSeats;
  }
}

interface PlayerTracker {
  currentRound: number;
  nextEpoch: number;
}
