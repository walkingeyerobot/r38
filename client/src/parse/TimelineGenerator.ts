import { SourceEvent } from './SourceData';
import { commitTimelineEvent } from '../draft/mutate';
import { cloneDraftState } from '../draft/cloneDraftState';
import { DraftState, CardContainer, CardPack, DraftSeat } from '../draft/DraftState';
import { TimelineEvent, PACK_LOCATION_UNUSED, PACK_LOCATION_DEAD } from '../draft/TimelineEvent';

export class TimelineGenerator {
  private _state!: DraftState;
  private _srcEvents!: SourceEvent[];

  private _outEvents = [] as TimelineEvent[];

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

  isComplete() {
    return this._state.isComplete;
  }

  private initialize(state: DraftState, events: SourceEvent[]) {
    this._state = state;
    this._srcEvents = events;
    this._outEvents = [];
    this._eventIndex = 0;
    this._nextTimelineId = 0;
    this._playerTrackers = [];

    // Fill out our player trackers
    for (const seat of this._state.seats) {
      this._playerTrackers.push({
        seatId: seat.position,
        currentRound: 1,
        nextEpoch: 0,
        nextPick: 0,
      });
    }

    this.openFirstPacks();
  }

  next(): boolean {
    if (this._eventIndex >= this._srcEvents.length) {
      this._state.isComplete = this.isDraftComplete();
      return false;
    }
    const srcEvent = this._srcEvents[this._eventIndex];
    this._eventIndex++;

    try {
      this.parseEvent(srcEvent);
    } catch (e) {
      console.log(
          `Error parsing event ${this._eventIndex - 1}`,
          this._srcEvents[this._eventIndex - 1]);
      throw e;
    }
    return true;
  }

  private parseEvent(srcEvent: SourceEvent) {
    const playerData = this._playerTrackers[srcEvent.player];
    if (srcEvent.round != playerData.currentRound) {
      throw new Error(`Unexpected round: ${srcEvent.round}`);
    }
    const seat = getSeat(srcEvent, this._state);
    if (seat.queuedPacks.length == 0) {
      throw new Error(`Seat ${seat.position} doesn't have any open packs!`);
    }
    const activePack = seat.queuedPacks[0];

    const outEvent = this.createEvent(playerData);

    // Draft the selected card
    const card = findCard(srcEvent.card1, activePack);
    outEvent.actions.push({
      type: 'move-card' as const,
      cardName: card.definition.name,
      card: card.id,
      from: activePack.id,
      to: seat.player.picks.id,
    });
    playerData.nextPick++;

    // Check to see if the player used a power that allowed them to draft a
    // second card
    this.handleSecondPick(srcEvent, outEvent, seat, activePack);

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

    // Sigh -____-
    this.handleLoreSeeker(srcEvent, outEvent, seat);

    // Commit this event
    this.commitEvent(outEvent);

    // Finally, check to see if we should open a new pack
    // (and advance the round)
    if (activePack.cards.length == 0 && seat.unopenedPacks.length > 0) {
      playerData.currentRound++;
      playerData.nextEpoch = 0;
      playerData.nextPick = 0;

      const nextPack = seat.unopenedPacks[0];
      const openPackEvent = this.createEvent(playerData);
      openPackEvent.actions.push({
        type: 'move-pack' as const,
        pack: nextPack.id,
        from: { seat: seat.position, queue: 'unopened' },
        to: { seat: seat.position, queue: 'opened' },
        insertAction: 'unshift',
      })
      this.commitEvent(openPackEvent);
    }
  }

  private handleSecondPick(
    srcEvent: SourceEvent,
    outEvent: TimelineEvent,
    seat: DraftSeat,
    activePack: CardPack,
  ) {
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
  }

  private handleLoreSeeker(
    srcEvent: SourceEvent,
    outEvent: TimelineEvent,
    seat: DraftSeat,
  ) {
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
  }

  private openFirstPacks() {
    for (const seat of this._state.seats) {
      if (seat.unopenedPacks.length == 0) {
        console.warn(`Seat ${seat.position} doesn't have any packs!`);
        continue;
      }

      const playerData = this._playerTrackers[seat.position];
      const event = this.createEvent(playerData);

      event.actions = [{
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
      }];

      this.commitEvent(event);
    }
  }

  private isDraftComplete() {
    for (let seat of this._state.seats) {
      if (seat.unopenedPacks.length > 0 || seat.queuedPacks.length > 0) {
        return false;
      }
    }
    return true;
  }

  private createEvent(playerData: PlayerTracker): TimelineEvent {
    return {
      id: this._nextTimelineId++,
      associatedSeat: playerData.seatId,
      round: playerData.currentRound,
      roundEpoch: playerData.nextEpoch++,
      pick: playerData.nextPick,
      actions: [],
    };
  }

  private commitEvent(event: TimelineEvent) {
    this._outEvents.push(event);

    // console.log('EVENT', event);
    // for (const action of event.actions){
    //   console.log('  ', action);
    // }

    commitTimelineEvent(event, this._state);
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
  seatId: number;
  currentRound: number;
  nextEpoch: number;
  nextPick: number;
}
