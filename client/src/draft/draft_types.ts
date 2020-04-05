
export interface DraftState {
  seats: DraftSeat[];
  unusedPacks: CardPack[];
  packs: Map<number, CardContainer>
}

export interface DraftSeat {
  position: number;
  player: DraftPlayer;
  queuedPacks: CardPack[];
  unopenedPacks: CardPack[];
  originalPacks: number[];
}

export type CardContainer = CardPack | PlayerPicks;

export interface CardPack {
  type: 'pack';
  id: number;
  cards: DraftCard[];
  originalSeat: number;
}

export interface PlayerPicks {
  type: 'player-picks';
  id: number;
  cards: DraftCard[];
}

export interface DraftPlayer {
  seatPosition: number;
  name: string;
  picks: CardContainer
}

export interface DraftCard {
  id: number;
  definition: MtgCard;
  /** The index position of this card in its original pack */
  sourcePackIndex: number,
  draftedBy: DraftedBy | null;
}

export interface MtgCard {
  // scryfall-compatible
  name: string;
  set: string;
  collector_number: string;

  // custom stuff
  tags: string[];
}

export interface DraftedBy {
  player: DraftPlayer;
  round: number;
}




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
