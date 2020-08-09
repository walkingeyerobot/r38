import { MtgCard } from "./DraftState";
import { BASICS, Deck } from "../state/DeckBuilderModule";
import JSZip from "jszip";

const XML_HEADER =
    `<?xml version="1.0" encoding="utf-8"?>
<Deck xmlns:xsd="http://www.w3.org/2001/XMLSchema" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">
<NetDeckID>0</NetDeckID>
<PreconstructedDeckID>0</PreconstructedDeckID>
`;

interface DeckEntry {
  name: string;
  quantity: number;
}

export function deckToXml(deck: Deck) {
  let exportStr = XML_HEADER;
  let mainMap = new Map<number, DeckEntry>();
  let sideMap = new Map<number, DeckEntry>();
  for (const card of deck.maindeck.flat()) {
    if (!card.definition.mtgo) {
      continue;
    }
    incrementQuantity(mainMap, card.definition);
  }
  for (const card of deck.sideboard.flat()) {
    if (!card.definition.mtgo) {
      continue;
    }
    incrementQuantity(sideMap, card.definition);
  }
  for (const [mtgo, card] of mainMap) {
    exportStr += `<Cards CatID=\"${mtgo}\" Quantity=\"${card.quantity}\"`
        + ` Sideboard=\"false\" Name=\"${card.name}\" />\n`;
  }
  for (const [mtgo, card] of sideMap) {
    exportStr += `<Cards CatID=\"${mtgo}\" Quantity=\"${card.quantity}\"`
        + ` Sideboard=\"true\" Name=\"${card.name}\" />\n`;
  }
  exportStr += "</Deck>";
  return `data:text/xml;charset=utf-8,${encodeURIComponent(exportStr)}`;
}

function deckToBinderXmlContents(deck: Deck) {
  let exportStr = XML_HEADER;
  let map = new Map<number, DeckEntry>();
  for (const card of deck.maindeck.flat()) {
    if (!card.definition.mtgo || BASICS.includes(card.definition.mtgo)) {
      continue;
    }
    incrementQuantity(map, card.definition);
  }
  for (const card of deck.sideboard.flat()) {
    if (!card.definition.mtgo) {
      continue;
    }
    incrementQuantity(map, card.definition);
  }
  for (const [mtgo, card] of map) {
    exportStr += `<Cards CatID=\"${mtgo}\" Quantity=\"${card.quantity}\"`
        + ` Sideboard=\"false\" Name=\"${card.name}\" />\n`;
  }
  exportStr += "</Deck>";
  return exportStr;
}

export function deckToBinderXml(deck: Deck): string {
  return `data:text/xml;charset=utf-8,${encodeURIComponent(deckToBinderXmlContents(deck))}`;
}

export async function decksToBinderZip(decks: Deck[], names: string[]): Promise<string> {
  const zip = new JSZip();
  decks.map(deckToBinderXmlContents).forEach((deckXml, i) => {
    zip.file(`${names[i]} (${decks[i].maindeck.flat().length + decks[i].sideboard.flat().length}).dek`,
        deckXml);
  });
  return zip.generateAsync({type: "base64"})
      .then(base64 => `data:application/zip;base64,${base64}`);
}

function incrementQuantity(map: Map<number, DeckEntry>, card: MtgCard) {
  let entry = map.get(card.mtgo);
  if (entry == undefined) {
    entry = {name: card.name, quantity: 0};
    map.set(card.mtgo, entry);
  }
  entry.quantity++;
}
