import { TimelineEvent } from './TimelineEvent';

export interface DraftState {
  seats: DraftSeat[];
  unusedPacks: PackContainer;
  deadPacks: PackContainer;
  packs: Map<number, CardContainer>;
  locations: Map<number, PackContainer>;
}

export interface DraftSeat {
  position: number;
  player: DraftPlayer;
  queuedPacks: PackContainer;
  unopenedPacks: PackContainer;
  originalPacks: number[];
}

export interface PackContainer {
  id: number;
  packs: CardPack[];
  label: string;
}

export type CardContainer = CardPack | PlayerPicks;

export interface CardPack {
  type: 'pack';
  id: number;
  cards: DraftCard[];
  originalSeat: number;
  round: number;
}

export interface PlayerPicks {
  type: 'seat';
  id: number;
  cards: DraftCard[];
}

export interface DraftPlayer {
  seatPosition: number;
  name: string;
  picks: CardContainer;
}

export interface DraftCard {
  id: number;
  definition: MtgCard;
  /** The index position of this card in its original pack */
  sourcePackIndex: number;
  pickedIn: CardPick[];
}

export interface CardPick {
  seat: number,
  round: number,
  pick: number,
  eventId: number,
}

export interface MtgCard {
  // scryfall-compatible
  name: string;
  set: string;
  collector_number: string;
  cmc: number;
  color: string;
  // MTGO CatID
  mtgo: string;

  // custom stuff
  tags: string[];

  // Post-processed name for quick string comparison
  searchName: string;
}

export const PACK_LOCATION_UNUSED = -1;
export const PACK_LOCATION_DEAD = -2;
