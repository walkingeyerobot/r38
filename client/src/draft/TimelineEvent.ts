export interface TimelineEvent {
  id: number;
  associatedSeat: number;
  round: number;
  roundEpoch: number;
  pick: number;
  actions: TimelineAction[];
}

export type TimelineAction =
    | ActionMoveCard
    | ActionMovePack
    | ActionAssignRound
    | ActionAnnounce
    ;

export interface ActionMoveCard {
  type: 'move-card';
  subtype: 'pick-card' | 'return-card',
  cardName: string;
  card: number;
  from: number;
  to: number;
}

export interface ActionMovePack {
  type: 'move-pack';
  pack: number;
  from: number;
  to: number;
  queuePosition: 'front' | 'end';
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
