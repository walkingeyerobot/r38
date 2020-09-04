import { DraftCard, DraftState } from '../draft/DraftState';
import { vuexModule } from './vuex/vuexModule';
import { rootStore } from './store';
import { draftStore } from './DraftStore';


type DataVersion = 1 | 2;
const DATA_VERSION = 2;
const DEFAULT_NUM_COLUMNS = 8;
const MODULE_NAME = 'deckbuilder';

export const BASICS = [27647, 27280, 27649, 27725, 27727];

/**
 * Vuex module for storing state related to the deck builder.
 */
export const deckBuilderStore = vuexModule(rootStore, MODULE_NAME, {

  selectedSeat: 0,
  names: [],
  decks: [],
  selection: [],

} as DeckBuilderState, {

  mutations: {
    sync(state: DeckBuilderState, draftState: DraftState) {
      const init = [] as DeckInitializer[];
      state.names = draftState.seats.map(seat => seat.player.name);
      for (let seat of draftState.seats) {
        init.push({
          draftName: draftStore.draftName,
          pool: seat.picks.cards
              .map(cardId => draftStore.getCard(cardId)),
        });
      }
      state.decks = [];
      for (let [index, initializer] of init.entries()) {
        const draftName = initializer.draftName;
        let deck = getStoredDeck(draftName, index);
        const cardsInDeck = new Set<number>();
        deck.sideboard.flat(2).forEach(card => {
          cardsInDeck.add(card.id);
        });
        deck.maindeck.flat(2).forEach(card => {
          cardsInDeck.add(card.id);
        });
        for (const card of initializer.pool) {
          if (!cardsInDeck.has(card.id)) {
            deck.maindeck[Math.min(card.definition.cmc, deck.sideboard.length - 1)]
                .push(card);
          }
        }
        state.decks.push(deck);
      }
    },

    setSelectedSeat(state: DeckBuilderState, selectedSeat: number | null) {
      state.selectedSeat = selectedSeat;
    },

    moveCard(state: DeckBuilderState, move: CardMove) {
      if (move.source.length === 0) {
        return;
      }

      let sourceSection: CardColumn[];
      if (move.source[0].maindeck) {
        sourceSection = state.decks[move.deckIndex].maindeck;
      } else {
        sourceSection = state.decks[move.deckIndex].sideboard;
      }
      let targetSection: CardColumn[];
      if (move.target.maindeck) {
        targetSection = state.decks[move.deckIndex].maindeck;
      } else {
        targetSection = state.decks[move.deckIndex].sideboard;
      }

      while (move.target.columnIndex >= targetSection.length) {
        targetSection.push([]);
      }

      if (move.target.cardIndex < 0) {
        move.target.cardIndex = targetSection[move.target.columnIndex].length;
      }

      const cards: DraftCard[] = move.source.map(location =>
          sourceSection[location.columnIndex][location.cardIndex]);
      if (move.source.some(location =>
          location.maindeck === move.target.maindeck
          && location.columnIndex === move.target.columnIndex)) {
        move.target.cardIndex -= move.source.filter(location =>
            location.maindeck === move.target.maindeck
            && location.columnIndex === move.target.columnIndex
            && location.cardIndex < move.target.cardIndex
        ).length;
      }
      move.source.forEach((location, index) => {
        sourceSection[location.columnIndex].splice(
            sourceSection[location.columnIndex].indexOf(cards[index]), 1);
      });
      targetSection[move.target.columnIndex]
          .splice(move.target.cardIndex, 0, ...cards);
      state.selection = [];
    },

    selectCards(state: DeckBuilderState, selection: CardLocation[]) {
      state.selection = selection;
    },

    sortByCmc(state: DeckBuilderState, payload: { seat: number, maindeck: boolean }) {
      sort(state, payload.seat, payload.maindeck,
          (cards, numColumns) => {
            const newSection: CardColumn[] =
                (<DraftCard[][]>Array(numColumns)).fill([]).map(() => []);
            for (let i = 0; i < numColumns; i++) {
              newSection[i] = cards.filter(card => {
                const isDfc = card.definition.card_faces.length > 0;
                const type = isDfc
                    ? card.definition.card_faces[0].type_line
                    : card.definition.type_line;
                if (type.indexOf("Land") != -1) {
                  return i === 0;
                } else {
                  if (i === numColumns - 1) {
                    return card.definition.cmc >= i - 1;
                  } else {
                    return card.definition.cmc === i - 1;
                  }
                }
              });
            }
            return newSection;
          });
    },

    sortByColor(state: DeckBuilderState, payload: { seat: number, maindeck: boolean }) {
      sort(state, payload.seat, payload.maindeck,
          (cards, numColumns) => {
            const newSection: CardColumn[] = fillArray(numColumns, () => []);

            for (const card of cards) {
              const isDfc = card.definition.card_faces.length > 0;
              const colors = isDfc
                  ? card.definition.card_faces[0].colors
                  : card.definition.colors;
              const type = isDfc
                  ? card.definition.card_faces[0].type_line
                  : card.definition.type_line;
              if (colors.length === 1) {
                const index = getColorIndex(colors[0]);
                newSection[index].push(card);
              } else if (card.definition.colors.length === 0) {
                if (type.indexOf("Land") != -1) {
                  newSection[Math.min(0, numColumns)].push(card);
                } else {
                  newSection[Math.min(6, numColumns)].push(card);
                }
              } else {
                if (newSection.length < 8) {
                  newSection.push([]);
                }
                newSection[Math.min(7, numColumns)].push(card);
              }
            }
            for (let i = 0; i < newSection.length - 1; i++) {
              let movedCols = 0;
              while (newSection[i].length === 0 && movedCols < newSection.length - i) {
                newSection.push(newSection.splice(i, 1)[0]);
                movedCols++;
              }
            }
            return newSection;
          });
    },
  },

  getters: {},

  actions: {},
});

deckBuilderStore.subscribe((mutation, state) => {
  for (const [index, deck] of state.decks.entries()) {
    localStorage.setItem(
        getLocalstorageKey(deck.draftName, index, DATA_VERSION),
        JSON.stringify(deck));
  }
});

function sort(state: DeckBuilderState,
              seat: number,
              maindeck: boolean,
              sort: (cards: DraftCard[], numColumns: number) => CardColumn[]) {
  const section = maindeck ? state.decks[seat].maindeck : state.decks[seat].sideboard;
  const cards = section.flat();

  const newSection = sort(cards, section.length);
  if (maindeck) {
    state.decks[seat].maindeck = newSection;
  } else {
    state.decks[seat].sideboard = newSection;
  }
}

function getStoredDeck(draftName: string, seatPosition: number): Deck {
  let deck: Deck | null = null;
  let version;
  for (version = DATA_VERSION; version > 0; version--) {
    const stored = localStorage.getItem(getLocalstorageKey(draftName, seatPosition, version));
    if (stored) {
      deck = JSON.parse(stored);
      break;
    }
  }
  if (deck) {
    for (; version < DATA_VERSION && deck != null; version++) {
      deck = migrateDeckVersion(deck, <DataVersion>(version + 1));
    }
  }
  if (!deck) {
    deck = {
      draftName: draftName,
      sideboard: (<DraftCard[][]>Array(DEFAULT_NUM_COLUMNS)).fill([]).map(() => []),
      maindeck: (<DraftCard[][]>Array(DEFAULT_NUM_COLUMNS)).fill([]).map(() => []),
    };
  }

  return deck;
}

function migrateDeckVersion(deck: Deck, newVersion: DataVersion): Deck | null {
  function unreachable(x: never) {
    return new Error(x);
  }

  switch (newVersion) {
    case 1:
      // no migration to version 1
      return deck;
    case 2:
      // no migration to version 2 - version 1 didn't have image URIs
      return null;
    default:
      throw unreachable(newVersion);
  }
}

function getLocalstorageKey(draftName: string, seatPosition: number, dataVersion: number) {
  return `draft|${draftName}|${seatPosition}|${dataVersion}`;
}

/**
 * Creates an array of [length] the contents of which are filled from repeated
 * calls to [entryGenerator].
 */
function fillArray<T>(
    length: number,
    entryGenerator: (index: number) => T,
): T[] {
  const arr = [] as T[];
  for (let i = 0; i < length; i++) {
    arr.push(entryGenerator(i));
  }
  return arr;
}

const COLOR_ORDER = ['', 'W', 'U', 'B', 'G', 'R'];

function getColorIndex(color: string) {
  const index = COLOR_ORDER.indexOf(color);
  if (index == -1) {
    throw new Error(`Unknown color ${color}`);
  }
  return index;
}

export type DeckBuilderStore = typeof deckBuilderStore;

interface DeckBuilderState {
  selectedSeat: number | null,
  names: string[],
  decks: Deck[],
  selection: CardLocation[],
}

export interface Deck {
  draftName: string;
  maindeck: CardColumn[],
  sideboard: CardColumn[],
}

export type CardColumn = DraftCard[];

export interface DeckInitializer {
  draftName: string;
  pool: DraftCard[];
}

export interface CardLocation {
  columnIndex: number,
  cardIndex: number,
  maindeck: boolean,
}

export interface CardMove {
  deckIndex: number,
  source: CardLocation[],
  target: CardLocation,
}
