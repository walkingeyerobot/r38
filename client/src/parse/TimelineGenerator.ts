import { SourceEvent } from './SourceData';
import { commitTimelineEvent } from '../draft/mutate';
import { DraftState, CardContainer, CardPack, DraftSeat, PackContainer } from '../draft/DraftState';
import { TimelineEvent } from '../draft/TimelineEvent';
import { ParseError } from './ParseError';
import { MutationError } from '../draft/MutationError';
import { deepCopy } from '../util/deepCopy';

export class TimelineGenerator {
  private _state!: DraftState;

  private _outEvents = [] as TimelineEvent[];

  private _nextTimelineId = 0;
  private _playerTrackers = [] as PlayerTracker[];

  generate(
      state: DraftState,
      events: SourceEvent[],
  ): GeneratedTimeline {
    this.initialize(deepCopy(state), events);

    let parseError: Error | null = null;
    for (let i = 0; i < events.length; i++) {
      const srcEvent = events[i];
      try {
        this.parseEvent(srcEvent);
      } catch (e) {
        if (e instanceof ParseError || e instanceof MutationError) {
          console.error('Error while parsing event', srcEvent, e);
          parseError = e;
          break;
        } else {
          throw e;
        }
      }
    }

    return {
      events: this._outEvents.concat(),
      isComplete: this.isDraftComplete(),
      parseError,
    };
  }

  private initialize(state: DraftState, events: SourceEvent[]) {
    this._state = state;
    this._outEvents = [];
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

  private parseEvent(srcEvent: SourceEvent) {
    const playerData = this._playerTrackers[srcEvent.player];
    if (srcEvent.round != playerData.currentRound) {
      throw new ParseError(`Unexpected round: ${srcEvent.round}`);
    }
    const seat = getSeat(srcEvent, this._state);
    if (seat.queuedPacks.packs.length == 0) {
      throw new ParseError(
          `Seat ${seat.position} doesn't have any open packs!`);
    }
    const activePack = seat.queuedPacks.packs[0];

    const outEvent = this.createEvent(playerData);

    for (let cardId of srcEvent.cards) {
      const card = findCard(cardId, activePack);
      outEvent.actions.push({
        type: 'move-card' as const,
        subtype: 'pick-card',
        cardName: card.definition.name,
        card: card.id,
        from: activePack.id,
        to: seat.player.picks.id,
      });
    }
    playerData.nextPick++;

    // Check to see if the player used a power that allowed them to draft a
    // second card
    this.handleClockworkLibrarian(srcEvent, outEvent, seat, activePack);

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

    // Sigh -____-
    this.handleLoreSeeker(srcEvent, outEvent, seat);

    // Commit this event
    this.commitEvent(outEvent);

    // Finally, check to see if we should open a new pack
    // (and advance the round)
    // TODO: This check could be fooled by a lore seeker pack
    if (activePack.cards.length == 0 && seat.unopenedPacks.packs.length > 0) {
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

  private handleClockworkLibrarian(
    srcEvent: SourceEvent,
    outEvent: TimelineEvent,
    seat: DraftSeat,
    activePack: CardPack,
  ) {
    if (srcEvent.cards.length > 1) {
      const librarianCard =
          findCardByName('Cogwork Librarian', seat.player.picks);
      outEvent.actions.push({
        type: 'move-card' as const,
        subtype: 'return-card',
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
      if (this._state.unusedPacks.packs.length == 0) {
        throw new ParseError(`No more packs to add`);
      }
      const newPack = this._state.unusedPacks.packs[0];

      outEvent.actions.push({
        type: 'move-pack' as const,
        pack: newPack.id,
        from: this._state.unusedPacks.id,
        to: seat.queuedPacks.id,
        queuePosition: 'front',
      });
    }
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

      event.actions = [{
        type: 'move-pack' as const,
        pack: pack.id,
        from: seat.unopenedPacks.id,
        to: seat.queuedPacks.id,
        queuePosition: 'front',
      }];

      this.commitEvent(event);
    }
  }

  private isDraftComplete() {
    for (let seat of this._state.seats) {
      if (seat.unopenedPacks.packs.length > 0
          || seat.queuedPacks.packs.length > 0) {
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
    // console.log('EVENT', event);
    // for (const action of event.actions){
    //   console.log('  ', action);
    // }

    commitTimelineEvent(event, this._state);

    this._outEvents.push(event);
  }
}

function getSeat(srcEvent: SourceEvent, state: DraftState) {
  const seat = state.seats[srcEvent.player];
  if (seat == null) {
    throw new ParseError(
        `Unknown player "${srcEvent.player}" in event `
            + JSON.stringify(srcEvent));
  }
  return seat;
}

function findCard(cardId: number, container: CardContainer) {
  for (const card of container.cards) {
    if (card.id == cardId) {
      return card;
    }
  }
  throw new ParseError(
      `Card "${cardId}" not found in container ${container.id}.`);
}

function findCardByName(cardName: string, container: CardContainer) {
  for (const card of container.cards) {
    if (card.definition.name == cardName) {
      return card;
    }
  }
  throw new ParseError(
      `Card "${cardName}" not found in container ${container.id}.`);
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
