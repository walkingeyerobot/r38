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
    | 'open-pack'
    ;

export type TimelineAction =
    | ActionMoveCard
    | ActionMovePack
    | ActionAssignRound
    | ActionAnnounce
    ;

export interface ActionMoveCard {
  type: 'move-card';
  subtype: 'pick-card' | 'return-card' | 'shadow-pick',
  cardName: string;
  card: number;
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

export interface ActionAssignRound {
  type: 'assign-round';
  pack: number;
  from: number;
  to: number;
}

export interface ActionAnnounce {
  type: 'announce';
  message: string;
}
