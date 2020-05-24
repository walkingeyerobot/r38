export interface SourceData {
  name: string;
  seats: SourceSeat[];
  events: SourceEvent[];
  extraPack: SourceCard[] | null;
}

export interface SourceSeat {
  name?: string,
  // The first one is the player's picks, following by packs 1-3 in their seat
  rounds: [SourceRound, SourceRound, SourceRound, SourceRound];
}

export interface SourceRound {
  round: number;
  packs: [SourcePack];
}

export interface SourcePack {
  cards: SourceCard[];
}

export interface SourceCard {
  name: string;
  tags: string;
  number: string;
  cmc: number;
  color: string;
  mtgo?: string;
  type?: string;
  edition: string;
}

export interface SourceEvent {
  player: number;
  announcements: string[];
  card1: string;
  card2: string;    // empty string if not a pick
  cards?: string[];
  playerModified: number;
  draftModified: number;
  round: number;

  librarian?: boolean;
}
