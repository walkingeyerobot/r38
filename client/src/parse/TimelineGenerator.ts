import { SourceEvent, SecretPickEvent, NormalPickEvent, ShadowPickEvent } from './SourceData';
import { commitTimelineEvent } from '../draft/mutate';
import { DraftState, CardPack, DraftSeat, DraftCard, MtgCard } from '../draft/DraftState';
import { TimelineEvent, TimelineEventType, ActionMovePack, ActionIncrementPickedColors } from '../draft/TimelineEvent';
import { ParseError } from './ParseError';
import { checkExhaustive } from '../util/checkExhaustive';
import { checkNotNil } from '../util/checkNotNil';
import { getActivePackForSeat } from '../state/util/getters';

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

  isDraftComplete() {
    for (let seat of this._state.seats) {
      if (seat.queuedPacks.packs.length > 0) {
        return false;
      }
    }
    return true;
  }

  private parsePickEvent(srcEvent: NormalPickEvent | SecretPickEvent) {
    const seat = checkNotNil(this._state.seats[srcEvent.position]);
    const playerData = this.getPlayerData(srcEvent.position);

    if (srcEvent.round != playerData.currentRound ) {
      throw new ParseError(
          `Seat ${srcEvent.position} is in round ${playerData.currentRound} `
          + `but event is round ${srcEvent.round}; `
          + `event=${JSON.stringify(srcEvent)}`);
    }

    const activePack = getActivePackForSeat(this._state, seat.position);
    if (activePack == null) {
      throw new ParseError(
          `Seat ${seat.position} doesn't have a pack to pick from`);
    }

    let event: TimelineEvent;
    let netPickCount = 0;
    switch (srcEvent.type) {
      case 'Pick':
        event = this.createEvent('pick', playerData);
        for (let cardId of srcEvent.cards) {
          const card = this.getCard(cardId);
          event.actions.push(
            {
              type: 'move-card',
              subtype: 'pick-card',
              card: cardId,
              cardName: card.definition.name,
              from: activePack.id,
              to: seat.picks.id
            }, {
              type: 'mark-transfer',
              from: activePack.id,
              to: seat.picks.id,
            },
            buildIncrementPickedColorsAction(
                seat.position,
                [card.definition],
                [],
            ),
          );

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
        event.actions.push({
          type: 'mark-transfer',
          from: activePack.id,
          to: seat.picks.id,
        });
        break;

      default:
        throw checkExhaustive(srcEvent);
    }
    playerData.nextPick++;

    // Pass the pack to the next player
    event.actions.push(
        this.buildPassAction(seat, playerData, activePack, netPickCount));

    this.commitEvent(event);

    this.maybeAdvancePlayerToNextRound(seat, playerData);
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
  ) {
    // TODO: This can be fooled by drafts that introduce more packs or that
    // have packs that aren't 15 cards
    if (![15, 30].includes(seat.picks.count)) {
      return;
    }

    playerData.currentRound++;
    playerData.nextEpoch = 0;
    playerData.nextPick = 0;

    const event = this.createEvent('advance-round', playerData);
    event.actions.push({
      type: 'increment-seat-round',
      seat: seat.position,
    });
    this.commitEvent(event);
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

function buildIncrementPickedColorsAction(
  seat: number,
  gainedCards: MtgCard[],
  lostCards: MtgCard[],
) {
  const action: ActionIncrementPickedColors = {
    type: 'increment-picked-colors',
    seat,
    w: 0,
    u: 0,
    b: 0,
    r: 0,
    g: 0,
  };

  for (let gainedCard of gainedCards) {
    if (gainedCard.color_identity.includes('W')) {
      action.w++;
    }
    if (gainedCard.color_identity.includes('U')) {
      action.u++;
    }
    if (gainedCard.color_identity.includes('B')) {
      action.b++;
    }
    if (gainedCard.color_identity.includes('R')) {
      action.r++;
    }
    if (gainedCard.color_identity.includes('G')) {
      action.g++;
    }
  }

  for (let gainedCard of lostCards) {
    if (gainedCard.color_identity.includes('W')) {
      action.w--;
    }
    if (gainedCard.color_identity.includes('U')) {
      action.u--;
    }
    if (gainedCard.color_identity.includes('B')) {
      action.b--;
    }
    if (gainedCard.color_identity.includes('R')) {
      action.r--;
    }
    if (gainedCard.color_identity.includes('G')) {
      action.g--;
    }
  }

  return action;
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
