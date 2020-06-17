import { SourceEvent } from './SourceData';
import { commitTimelineEvent } from '../draft/mutate';
import { DraftState, CardPack, DraftSeat, PackContainer, DraftCard } from '../draft/DraftState';
import { TimelineEvent } from '../draft/TimelineEvent';
import { ParseError } from './ParseError';
import { getSeat } from '../state/util/getters';
import { checkUnreachable } from '../util/checkUnreachable';

export class TimelineGenerator {
  private _state: DraftState;
  private _outEvents: TimelineEvent[];
  private _cards: Map<number, DraftCard>;

  private _nextTimelineId = 0;
  private _playerTrackers = [] as PlayerTracker[];

  constructor(
    state: DraftState,
    cards: Map<number, DraftCard>,
    events: TimelineEvent[],
  ) {
    this._state = state;
    this._outEvents = events;
    this._cards = cards;

    // Fill out our player trackers
    for (const seat of this._state.seats) {
      this._playerTrackers.push({
        seatId: seat.position,
        currentRound: 1,
        nextEpoch: 0,
        nextPick: 0,
      });
    }
  }

  init() {
    this.openFirstPacks();
  }

  parseEvent(srcEvent: SourceEvent) {
    const playerData = this._playerTrackers[srcEvent.position];
    if (srcEvent.round != playerData.currentRound) {
      throw new ParseError(`Unexpected round: ${srcEvent.round}`);
    }
    const seat = getSeat(this._state, srcEvent.position);
    if (seat.queuedPacks.packs.length == 0) {
      throw new ParseError(
          `Seat ${seat.position} doesn't have any open packs!`);
    }
    const activePack = seat.queuedPacks.packs[0];

    const outEvent = this.createEvent(playerData);

    this.pickCardsForEvent(srcEvent, seat, activePack, playerData, outEvent);

    // Move the pack to the next seat
    const dstLocation =
        getLocationToPassTo(activePack, seat, playerData, this._state);

    outEvent.actions.push({
      type: 'move-pack' as const,
      pack: activePack.id,
      from: seat.queuedPacks.id,
      to: dstLocation.id,
      queuePosition: 'end',
    });

    // Commit this event
    this.commitEvent(outEvent);

    // Finally, check to see if we should open a new pack
    // (and advance the round)
    // TODO: This check could be fooled by a lore seeker pack
    if (seat.position != -1
        && activePack.cards.length == 0
        && seat.unopenedPacks.packs.length > 0) {
      playerData.currentRound++;
      playerData.nextEpoch = 0;
      playerData.nextPick = 0;

      const nextPack = seat.unopenedPacks.packs[0];
      const openPackEvent = this.createEvent(playerData);
      openPackEvent.actions.push({
        type: 'assign-round',
        pack: nextPack.id,
        from: nextPack.round,
        to: playerData.currentRound,
      });
      openPackEvent.actions.push({
        type: 'move-pack' as const,
        pack: nextPack.id,
        from: seat.unopenedPacks.id,
        to: seat.queuedPacks.id,
        queuePosition: 'front',
      });
      this.commitEvent(openPackEvent);
    }
  }

  pickCard(seatId: number, cardId: number) {
    const seat = this._state.seats[seatId];
    const playerData = this._playerTrackers[seatId];
    if (seat == undefined || playerData == undefined) {
      throw new Error(`No seat with ID ${seatId}`);
    }
    const pack = seat.queuedPacks.packs[0];
    if (pack == undefined) {
      throw new Error(`Seat ${seatId} doesn't have any packs to pick from.`);
    }

    const fakeEvent: SourceEvent = {
      type: 'Pick',
      cards: [cardId],
      position: seatId,
      round: pack.round,
      announcements: [],
      draftModified: -1,
      playerModified: -1,
      librarian: false,
    };

    this.parseEvent(fakeEvent);
  }

  private openFirstPacks() {
    for (const seat of this._state.seats) {
      if (seat.unopenedPacks.packs.length == 0) {
        console.warn(`Seat ${seat.position} doesn't have any packs!`);
        continue;
      }
      const pack = seat.unopenedPacks.packs[0];

      const playerData = this._playerTrackers[seat.position];
      const event = this.createEvent(playerData);

      event.actions.push({
        type: 'move-pack' as const,
        pack: pack.id,
        from: seat.unopenedPacks.id,
        to: seat.queuedPacks.id,
        queuePosition: 'front',
      });

      this.commitEvent(event);
    }
  }

  isDraftComplete() {
    for (let seat of this._state.seats) {
      if (seat.unopenedPacks.packs.length > 0
          || seat.queuedPacks.packs.length > 0) {
        return false;
      }
    }
    return true;
  }

  private createEvent(playerData: PlayerTracker): TimelineEvent {
    return Object.freeze({
      id: this._nextTimelineId++,
      associatedSeat: playerData.seatId,
      round: playerData.currentRound,
      roundEpoch: playerData.nextEpoch++,
      pick: playerData.nextPick,
      actions: [],
    });
  }

  private commitEvent(event: TimelineEvent) {
    // console.log('EVENT', event);
    // for (const action of event.actions){
    //   console.log('  ', action);
    // }

    commitTimelineEvent(this, event, this._state);

    this._outEvents.push(event);
  }

  getCard(id: number) {
    const card = this._cards.get(id);
    if (card == undefined) {
      throw new Error(`Unknown card: ${id}`);
    }
    return card;
  }

  pickCardsForEvent(
      srcEvent: SourceEvent,
      seat: DraftSeat,
      activePack: CardPack,
      playerData: PlayerTracker,
      outEvent: TimelineEvent,
  ) {
    switch (srcEvent.type) {
      case 'SecretPick':
        // Nothing to actually pick here, we just pass the pack along
        playerData.nextPick++;
        break;
      case 'Pick':
      case 'ShadowPick':
        for (let cardId of srcEvent.cards) {
          const card = this.getCard(cardId);
          outEvent.actions.push({
            type: 'move-card' as const,
            subtype: 'pick-card',
            cardName: card.definition.name,
            card: card.id,
            from: activePack.id,
            to: seat.player.picks.id,
          });
          card.pickedIn.push({
            eventId: outEvent.id,
            pick: playerData.nextPick,
            round: playerData.currentRound,
            seat: seat.position,
          });
        }
        playerData.nextPick++;
        break;
      default:
        checkUnreachable(srcEvent);
    }
  }
}

function getLocationToPassTo(
    pack: CardPack,
    seat: DraftSeat,
    playerData: PlayerTracker,
    state: DraftState,
): PackContainer {
  if (pack.cards.length <= 1) {
    return state.deadPacks;
  } else {
    const numSeats = state.seats.length;

    let nextSeatId = playerData.currentRound % 2 == 1
        ? seat.position + 1
        : seat.position - 1;

    // This is essentially just (nextSeatId % numSeats), but also works for
    // negative numbers
    nextSeatId = ((nextSeatId % numSeats) + numSeats) % numSeats;

    return state.seats[nextSeatId].queuedPacks;
  }
}

interface PlayerTracker {
  seatId: number;
  currentRound: number;
  nextEpoch: number;
  nextPick: number;
}

export interface GeneratedTimeline {
  events: TimelineEvent[];
  isComplete: boolean;
  parseError: Error | null;
}
