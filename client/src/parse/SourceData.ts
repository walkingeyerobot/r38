import { ScryfallColor, ScryfallRarity, ScryfallLayout } from '../draft/scryfall';

export interface SourceData {
  draftId: number;
  draftName: string;
  seats: SourceSeat[];
  events: SourceEvent[];
  pickXsrf: string;
  inPerson: boolean;

  // If this source data is from the perspective of a specific player, then
  // that player's ID
  playerId?: number;
}

export interface SourceSeat {
  playerId: number;
  playerName: string;
  mtgoName: string;
  playerImage: string;
  packs: SourcePack[];
}

export type SourcePack = SourceCard[];

export type SourceCard = KnownCard | HiddenCard;

export interface KnownCard {
  id: number;
  hidden?: false;

  mtgo_id: number;
  foil: boolean;
  rating?: number;
  image_uris?: string[];

  scryfall: {
    name: string;
    set: string;
    collector_number: string;
    mana_cost?: string;
    cmc: number;
    colors?: ScryfallColor[];
    color_identity: ScryfallColor[];
    rarity: ScryfallRarity;
    type_line: string;
    layout: ScryfallLayout;
    card_faces?: Array<{
      name: string;
      colors?: ScryfallColor[];
      mana_cost: string;
      type_line: string;
    }>;
  };
}

export interface HiddenCard {
  id: number;
  hidden: true;

  scryfall: {
    name: string;
  };
}

// Notes:
// - `mana_cost` is missing for lands
// - `colors` is missing for cards with more than one face
// - `card_faces.mana_cost` is NOT missing for faces
// - `r38_data.image_uris` is only very rarely present

export type SourceEvent = NormalPickEvent | SecretPickEvent | ShadowPickEvent;

export interface NormalPickEvent extends BaseEvent {
  type: 'Pick';
  cards: number[];
  playerModified: number;
}

export interface SecretPickEvent extends BaseEvent {
  type: 'SecretPick';
  draftModified: number;
  playerModified: number;
}

export interface ShadowPickEvent extends BaseEvent {
  type: 'ShadowPick';
  cards: number[];
}

export interface BaseEvent {
  position: number;
  round: number;
  announcements: string[];
  draftModified: number;
  librarian: boolean;
}
