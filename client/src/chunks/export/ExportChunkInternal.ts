import JSZip from "jszip";
import jsPDF from "jspdf";

import { type Deck, BASICS } from "@/state/DeckBuilderModule";
import type { MtgCard } from "@/draft/DraftState";
import type { ExportChunk } from "./ExportChunk";

const EXPORT_CHUNK_INTERNAL: ExportChunk = {
  deckToXml(deck: Deck) {
    let exportStr = XML_HEADER;
    const mainMap = new Map<number, DeckEntry>();
    const sideMap = new Map<number, DeckEntry>();
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
      exportStr +=
        `<Cards CatID=\"${mtgo}\" Quantity=\"${card.quantity}\"` +
        ` Sideboard=\"false\" Name=\"${card.name}\" />\n`;
    }
    for (const [mtgo, card] of sideMap) {
      exportStr +=
        `<Cards CatID=\"${mtgo}\" Quantity=\"${card.quantity}\"` +
        ` Sideboard=\"true\" Name=\"${card.name}\" />\n`;
    }
    exportStr += "</Deck>";
    return `data:text/xml;charset=utf-8,${encodeURIComponent(exportStr)}`;
  },

  deckToBinderXml(deck: Deck): string {
    return `data:text/xml;charset=utf-8,${encodeURIComponent(deckToBinderXmlContents(deck))}`;
  },

  async decksToBinderZip(decks: Deck[], names: string[], mtgoNames: string[]): Promise<string> {
    const zip = new JSZip();
    decks.map(deckToBinderXmlContents).forEach((deckXml, i) => {
      const totalCards = decks[i].maindeck.flat().length + decks[i].sideboard.flat().length;
      const playerName = (mtgoNames[i] || names[i]).replaceAll(RegExp("[^\\w\\s]+", "g"), "-");
      zip.file(`${playerName} (${totalCards} cards).dek`, deckXml);
    });
    return zip
      .generateAsync({ type: "base64" })
      .then((base64) => `data:application/zip;base64,${base64}`);
  },

  deckToPdf(deck: Deck) {
    const pdf = new jsPDF("p", "in", "letter");
    let cardsOnLine = 0;
    let linesOnPage = 0;
    const cards = deck.maindeck.flat().concat(deck.sideboard.flat());
    Promise.all(
      cards.map(async (card) => {
        const canvas = document.createElement("canvas");
        const ctx = canvas.getContext("2d");
        const img = document.querySelector(
          `img[src="${card.definition.image_uris[0]}?_"]`,
        )! as HTMLImageElement;
        canvas.width = img.naturalWidth;
        canvas.height = img.naturalHeight;
        ctx!.drawImage(img, 0, 0);
        const blob = await new Promise((resolve) => canvas.toBlob(resolve, "image/png"));
        return await (blob as Blob).arrayBuffer();
      }),
    ).then((images) => {
      cards.forEach((_card, i) => {
        pdf.addImage(
          new Uint8Array(images[i]),
          "JPEG",
          0.25 + cardsOnLine * 2.4,
          0.25 + linesOnPage * 3.35,
          2.4,
          3.348,
        );
        cardsOnLine++;
        if (cardsOnLine === 3) {
          cardsOnLine = 0;
          linesOnPage++;
          if (linesOnPage == 3) {
            linesOnPage = 0;
            if (i < cards.length - 1) {
              pdf.addPage();
            }
          }
        }
      });
      pdf.save("r38export.pdf");
    });
  },
};
export default EXPORT_CHUNK_INTERNAL;

function deckToBinderXmlContents(deck: Deck) {
  let exportStr = XML_HEADER;
  const map = new Map<number, DeckEntry>();
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
    exportStr +=
      `<Cards CatID=\"${mtgo}\" Quantity=\"${card.quantity}\"` +
      ` Sideboard=\"false\" Name=\"${card.name}\" />\n`;
  }
  exportStr += "</Deck>";
  return exportStr;
}

function incrementQuantity(map: Map<number, DeckEntry>, card: MtgCard) {
  let entry = map.get(card.mtgo);
  if (entry == undefined) {
    entry = { name: card.name, quantity: 0 };
    map.set(card.mtgo, entry);
  }
  entry.quantity++;
}

interface DeckEntry {
  name: string;
  quantity: number;
}

const XML_HEADER = `<?xml version="1.0" encoding="utf-8"?>
<Deck xmlns:xsd="http://www.w3.org/2001/XMLSchema" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">
<NetDeckID>0</NetDeckID>
<PreconstructedDeckID>0</PreconstructedDeckID>
`;
