
export interface DraftState {
  seats: DraftSeat[];
  unusedPacks: CardPack[];
  packs: Map<number, CardContainer>;
  isComplete: boolean;
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
