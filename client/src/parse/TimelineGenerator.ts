import { SourceEvent, SecretPickEvent, NormalPickEvent, ShadowPickEvent } from './SourceData';
import { commitTimelineEvent } from '../draft/mutate';
import { DraftState, CardPack, DraftSeat, DraftCard } from '../draft/DraftState';
import { TimelineEvent, TimelineEventType, ActionMovePack } from '../draft/TimelineEvent';
import { ParseError } from './ParseError';
import { checkExhaustive } from '../util/checkExhaustive';
import { checkNotNil } from '../util/checkNotNil';

export class TimelineGenerator {
  private _state: DraftState;
  private _cards: Map<number, DraftCard>;
  private _outEvents: TimelineEvent[];
  private _activePlayerId: number | null;

  private _nextTimelineId = 0;
  private _playerTrackers = new Map<number, PlayerTracker>();

  constructor(
    state: DraftState,
    cards: Map<number, DraftCard>,
    events: TimelineEvent[],
    activePlayerId: number | null,
  ) {
    this._state = state;
    this._outEvents = events;
    this._cards = cards;
    this._activePlayerId = activePlayerId;

    // Fill out our player trackers
    for (const seat of this._state.seats) {
      this._playerTrackers.set(seat.position, {
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
    switch (srcEvent.type) {
      case 'Pick':
        this.parsePickEvent(srcEvent);
        break;
      case 'SecretPick':
        this.parsePickEvent(srcEvent);
        break;
      case 'ShadowPick':
        this.parseShadowPickEvent(srcEvent);
        break;
      default:
        checkExhaustive(srcEvent);
    }
  }

  pickCard(seatId: number, cardId: number) {
    const seat = this._state.seats[seatId];
    const playerData = this.getPlayerData(seatId);
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

  isDraftComplete() {
    for (let seat of this._state.seats) {
      if (seat.unopenedPacks.packs.length > 0
          || seat.queuedPacks.packs.length > 0) {
        return false;
      }
    }
    return true;
  }

  private openFirstPacks() {
    for (const seat of this._state.seats) {
      if (seat.unopenedPacks.packs.length == 0) {
        console.warn(`Seat ${seat.position} doesn't have any packs!`);
        continue;
      }
      const pack = seat.unopenedPacks.packs[0];

      const playerData = this.getPlayerData(seat.position);
      const event = this.createEvent('open-pack', playerData);

      event.actions.push({
        type: 'move-pack',
        subtype: 'open',
        pack: pack.id,
        from: seat.unopenedPacks.id,
        to: seat.queuedPacks.id,
        epoch: 'increment',
      });

      this.commitEvent(event);
    }
  }

  private parsePickEvent(srcEvent: NormalPickEvent | SecretPickEvent) {
    const seat = checkNotNil(this._state.seats[srcEvent.position]);
    const playerData = this.getPlayerData(srcEvent.position);

    this.maybeAdvancePlayerToNextRound(seat, playerData, srcEvent);

    if (seat.queuedPacks.packs.length == 0) {
      throw new ParseError(
          `Seat ${seat.position} doesn't have a pack to pick from`);
    }
    const activePack: CardPack = seat.queuedPacks.packs[0];

    let event: TimelineEvent;
    let netPickCount = 0;
    switch (srcEvent.type) {
      case 'Pick':
        event = this.createEvent('pick', playerData);
        for (let cardId of srcEvent.cards) {
          const card = this.getCard(cardId);
          event.actions.push({
            type: 'move-card',
            subtype: 'pick-card',
            card: cardId,
            cardName: card.definition.name,
            from: activePack.id,
            to: seat.player.picks.id
          });

          // TODO: Do we really need to embed all this info? Can we just get it
          // out of the event object as-needed?
          card.pickedIn.push({
            eventId: event.id,
            pick: event.pick,
            round: event.round,
            fromSeat: event.associatedSeat,
            bySeat: event.associatedSeat,
          });

          netPickCount++;
        }
        break;

      case 'SecretPick':
        event = this.createEvent('hidden-pick', playerData);
        break;

      default:
        throw checkExhaustive(srcEvent);
    }
    playerData.nextPick++;

    // Pass the pack to the next player
    event.actions.push(
        this.buildPassAction(seat, playerData, activePack, netPickCount));

    this.commitEvent(event);
  }

  private buildPassAction(
    seat: DraftSeat,
    playerData: PlayerTracker,
    pack: CardPack,
    pickCount: number,
  ): ActionMovePack {
    if (pack.cards.length <= pickCount) {
      return {
        type: 'move-pack',
        subtype: 'discard',
        pack: pack.id,
        from: seat.queuedPacks.id,
        to: this._state.deadPacks.id,
        epoch: 'increment',
      };
    } else {
      const seatId =
          getSeatToPassTo(
              seat.position,
              this._state.seats.length,
              playerData.currentRound);
      return {
        type: 'move-pack',
        subtype: 'pass',
        pack: pack.id,
        from: seat.queuedPacks.id,
        to: this._state.seats[seatId].queuedPacks.id,
        epoch: 'increment',
      };
    }
  }

  private parseShadowPickEvent(srcEvent: ShadowPickEvent) {
    const shadowedEvent = this.getMostRecentNonShadowEvent();
    const outEvent = this.createShadowEvent(shadowedEvent);

    // How many cards are picked from each pack
    const packPickCounts = new Map<CardPack, number>();

    for (let cardId of srcEvent.cards) {
      const card = this.getCard(cardId);
      const { pack, seat } = this.findPackContainingCard(cardId);

      const pickCount = (packPickCounts.get(pack) || 0) + 1;
      packPickCounts.set(pack, pickCount);

      outEvent.actions.push({
        type: 'move-card',
        subtype: 'shadow-pick',
        cardName: cardDisplayName(card),
        card: cardId,
        from: pack.id,
        to: this._state.shadowPool.id,
      });

      card.pickedIn.push({
        eventId: shadowedEvent.id,
        pick: shadowedEvent.pick,
        round: shadowedEvent.round,
        fromSeat: shadowedEvent.associatedSeat,
        bySeat: -1,
      });

      // If the pack no longer has cards in it, banish it to the SHADOW REALM
      if (pack.cards.length == pickCount) {
        outEvent.actions.push({
          type: 'move-pack',
          subtype: 'discard',
          pack: pack.id,
          from: seat.queuedPacks.id,
          to: this._state.deadPacks.id,
          epoch: 'increment',
        });
      }
    }

    this.commitEvent(outEvent);
  }

  private maybeAdvancePlayerToNextRound(
    seat: DraftSeat,
    playerData: PlayerTracker,
    srcEvent: NormalPickEvent | SecretPickEvent,
  ) {
    if (srcEvent.round == playerData.currentRound) {
      return;
    }
    if (srcEvent.round != playerData.currentRound + 1) {
      throw new ParseError(
          `Seat ${srcEvent.position} can't just from round`
          + ` ${playerData.currentRound} to ${srcEvent.round}`);
    }

    playerData.currentRound++;
    playerData.nextEpoch = 0;
    playerData.nextPick = 0;

    const nextPack = seat.unopenedPacks.packs[0];
    const openPackEvent = this.createEvent('open-pack', playerData);
    openPackEvent.actions.push({
      type: 'move-pack',
      subtype: 'open',
      pack: nextPack.id,
      from: seat.unopenedPacks.id,
      to: seat.queuedPacks.id,
      epoch: 'increment',
    });
    this.commitEvent(openPackEvent);
  }

  private createEvent(
      type: TimelineEventType,
      playerData: PlayerTracker,
  ): TimelineEvent {
    return Object.freeze({
      id: this._nextTimelineId++,
      type: type,
      associatedSeat: playerData.seatId,
      round: playerData.currentRound,
      roundEpoch: playerData.nextEpoch++,
      pick: playerData.nextPick,
      actions: [],
    });
  }

  private createShadowEvent(shadowedEvent: TimelineEvent): TimelineEvent {
    // One way to think about things is that the shadow player always acts
    // "for" another player (usually, the active player). In order to ensure
    // that the shadow player's actions properly reorder in synchronized mode,
    // we use the round/roundEpoch information of the most recent non-shadow
    // event.

    return Object.freeze({
      id: this._nextTimelineId++,
      type: 'shadow-pick',
      associatedSeat: -1,
      round: shadowedEvent.round,
      roundEpoch: shadowedEvent.roundEpoch,
      pick: shadowedEvent.pick,
      actions: [],
    });
  }

  private getMostRecentNonShadowEvent() {
    for (let i = this._outEvents.length - 1; i >= 0; i--) {
      const event = this._outEvents[i];
      if (event.associatedSeat != -1) {
        return event;
      }
    }
    throw new ParseError(
        `No event to shadow; event list is ${this._outEvents.length} long`);
  }

  private findPackContainingCard(cardId: number) {
    // TODO: This is not super efficient
    for (let seat of this._state.seats) {
      for (let pack of seat.queuedPacks.packs) {
        if (pack.cards.includes(cardId)) {
          return { seat, pack };
        }
      }
    }
    throw new ParseError(
        `Cannot find a pack to pick card ${cardId}:`
            + `"${cardDisplayName(this.getCard(cardId))}" from`);
  }

  private commitEvent(event: TimelineEvent) {
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

  private getPlayerData(position: number) {
    return checkNotNil(this._playerTrackers.get(position));
  }
}

function getSeatToPassTo(
    currentSeatPosition: number,
    numSeats: number,
    round: number,
) {
  let nextSeatId = round % 2 == 1
        ? currentSeatPosition + 1
        : currentSeatPosition - 1;

  // Equivalent to (nextSeatId % numSeats), but also works for negative numbers
  nextSeatId = ((nextSeatId % numSeats) + numSeats) % numSeats;

  return nextSeatId;
}

function cardDisplayName(card: DraftCard) {
  return card.hidden ? `Hidden Card ${card.id}` : card.definition.name;
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
