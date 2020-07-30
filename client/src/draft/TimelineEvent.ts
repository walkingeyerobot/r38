export interface TimelineEvent {
  id: number;
  type: TimelineEventType,
  associatedSeat: number;
  round: number;
  roundEpoch: number;
  pick: number;
  actions: TimelineAction[];
}

export type TimelineEventType =
    | 'pick'
    | 'hidden-pick'
    | 'shadow-pick'
    | 'advance-round'
    ;

export type TimelineAction =
    | ActionMoveCard
    | ActionMovePack
    | ActionMarkTransfer
    | ActionIncrementSeatRound
    | ActionAssignPackRound
    | ActionAnnounce
    ;

export interface ActionMoveCard {
  type: 'move-card';
  subtype: 'pick-card' | 'return-card' | 'shadow-pick';
  cardName: string;
  card: number;
  from: number;
  to: number;
}

/**
 * Decrements from's `count` by one and increments to's `count` by one.
 *
 * Indicates that a card was transferred from one place to another. Due to the
 * shadow drafter, the card may not actually move until later. However, we
 * update the container's `count` values to reflect how many cards they contain
 * (even though we may not know exactly what those cards are).
 */
export interface ActionMarkTransfer {
  type: 'mark-transfer';
  from: number;
  to: number;
}

export interface ActionMovePack {
  type: 'move-pack';
  subtype: 'open' | 'pass' | 'discard';
  pack: number;
  from: number;
  to: number;
  epoch: 'increment' | number;
}

export interface ActionAssignPackRound {
  type: 'assign-pack-round';
  pack: number;
  from: number;
  to: number;
}

export interface ActionIncrementSeatRound {
  type: 'increment-seat-round';
  seat: number;
}

export interface ActionAnnounce {
  type: 'announce';
  message: string;
}
