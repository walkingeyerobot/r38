import { ScryfallColor, ScryfallRarity, ScryfallLayout } from './scryfall';

export interface DraftState {
  seats: DraftSeat[];
  shadowPool: CardContainer;
  unusedPacks: PackContainer;
  deadPacks: PackContainer;
  packs: Map<number, CardContainer>;
  locations: Map<number, PackContainer>;
}

export interface DraftSeat {
  position: number;
  player: DraftPlayer;
  picks: CardContainer;
  queuedPacks: PackContainer;
  originalPacks: number[];
  round: number;
  colorCounts: {
    w: number;
    u: number;
    b: number;
    r: number;
    g: number;
  };
}

export interface PackContainer {
  id: number;
  packs: CardPack[];
  label: string;
}

export type CardContainer = CardPack | PlayerPicks;

interface BaseContainer {
  id: number;
  cards: number[];
  count: number;
}

export interface CardPack extends BaseContainer {
  type: 'pack';
  labelId: number;
  originalSeat: number;
  round: number;
  epoch: number;
}

export interface PlayerPicks extends BaseContainer {
  type: 'seat' | 'shadow-realm';
  owningSeat: number;
}

export interface DraftPlayer {
  id: number;
  name: string;
  mtgoName: string;
  iconUrl: string;
  seatPosition: number;
}

export interface DraftCard {
  id: number;
  definition: MtgCard;
  /** The index position of this card in its original pack */
  sourcePackIndex: number;
  hidden: boolean;
  pickedIn: CardPick[];
}

export interface CardPick {
  /** The seat from whose packs the card was picked. */
  fromSeat: number,

  /**
   * The seat that did the picking. If the shadow player is picking, this will
   * be -1, otherwise it will match fromSeat.
   */
  bySeat: number,

  round: number,
  pick: number,
  eventId: number,
}

export interface MtgCard {
  // scryfall-compatible (ish)
  name: string;
  set: string;
  collector_number: string;
  mana_cost: string;
  cmc: number;
  colors: ScryfallColor[];
  color_identity: ScryfallColor[];
  rarity: ScryfallRarity;
  type_line: string;
  layout: ScryfallLayout;

  image_uris: string[];

  card_faces: Array<{
    name: string;
    colors: ScryfallColor[];
    mana_cost: string;
    type_line: string;
  }>;

  // custom stuff

  // MTGO CatID
  mtgo: number;

  foil: boolean;

  // Post-processed name for quick string comparison
  searchName: string;
}

export const PACK_LOCATION_UNUSED = -1;
export const PACK_LOCATION_DEAD = -2;

export const CONTAINER_SHADOW = -1;
