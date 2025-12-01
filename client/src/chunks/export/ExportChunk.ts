import type { Deck } from "@/state/DeckBuilderModule";
import { ChunkLoader } from "../ChunkLoader";

export interface ExportChunk {
  deckToTxt(deck: Deck): string;
  deckToXml(deck: Deck): string;
  deckToBinderXml(deck: Deck): string;
  decksToBinderZip(decks: Deck[], names: string[], mtgoNames: string[]): Promise<string>;
  deckToPdf(deck: Deck, scale: number): void;
}

export const exportLoader = new ChunkLoader(() =>
  import(/* webpackChunkName: "export" */ "./ExportChunkInternal").then((chunk) => chunk.default),
);
