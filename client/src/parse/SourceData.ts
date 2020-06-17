import { ScryfallColor, ScryfallRarity, ScryfallLayout } from '../draft/scryfall';

export interface SourceData {
  draftId: number;
  draftName: string;
  seats: SourceSeat[];
  events: SourceEvent[];
}

export interface SourceSeat {
  playerId: number;
  playerName: string;
  playerImage: string;
  packs: SourcePack[];
}

export type SourcePack = SourceCard[];

export interface SourceCard {
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
  r38_data: {
    id: number;
    foil: boolean;
    mtgo_id: number;
    image_uris?: string[];
    rating?: number;
  };
  card_faces?: Array<{
    name: string;
    colors?: ScryfallColor[];
    mana_cost: string;
    type_line: string;
  }>;
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
