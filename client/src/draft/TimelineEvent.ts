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
    | ActionAnnounce
    ;

export interface ActionMoveCard {
  type: 'move-card';
  cardName: string;
  card: number;
  from: number;
  to: number;
}

export interface ActionMovePack {
  type: 'move-pack';
  pack: number;
  from: PackLocation;
  to: PackLocation;
  insertAction: 'enqueue' | 'unshift';
}

export interface ActionAnnounce {
  type: 'announce';
  message: string;
}

export interface PackLocation {
  seat: number;
  queue: 'unopened' | 'opened';
}

export const PACK_LOCATION_UNUSED = -1;
export const PACK_LOCATION_DEAD = -2;
