import { DraftCard } from '../draft/DraftState';
import { vuexModule } from './vuex/vuexModule2';
import { rootStore } from './store';


const DEFAULT_NUM_COLUMNS = 7;
const MODULE_NAME = 'deckbuilder';

export const BASICS = ["27647", "27280", "27649", "27725", "27727"];

/**
 * Vuex module for storing state related to the deck builder.
 */
export const deckBuilderStore = vuexModule(rootStore, MODULE_NAME, {

  selectedSeat: 0,
  decks: [],
  selection: [],

} as DeckBuilderState, {

  mutations: {
    initDecks(
        state: DeckBuilderState,
        init: DeckInitializer[],
    ) {
      state.decks = [];
      for (let initializer of init) {
        const draftName = initializer.draftName;
        const seatPosition = initializer.player.seatPosition;
        const stored = localStorage.getItem(getLocalstorageKey(draftName, seatPosition));
        let deck: Deck;
        if (stored) {
          deck = JSON.parse(stored);
        } else {
          deck = {
            draftName: draftName,
            player: {
              seatPosition: seatPosition,
              name: initializer.player.name,
            },
            sideboard: (<DraftCard[][]>Array(DEFAULT_NUM_COLUMNS)).fill([]).map(() => []),
            maindeck: (<DraftCard[][]>Array(DEFAULT_NUM_COLUMNS)).fill([]).map(() => []),
          };
        }
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

    setSelectedSeat(state: DeckBuilderState, selectedSeat: number) {
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
                if (i === numColumns - 1) {
                  return card.definition.cmc >= i;
                } else {
                  return card.definition.cmc === i;
                }
              });
            }
            return newSection;
          });
    },

    sortByColor(state: DeckBuilderState, payload: { seat: number, maindeck: boolean }) {
      sort(state, payload.seat, payload.maindeck,
          (cards, numColumns) => {
            const newSection: CardColumn[] =
                (<DraftCard[][]>Array(numColumns)).fill([]).map(() => []);
            for (const card of cards) {
              if (card.definition.color.length === 1) {
                newSection[["W", "U", "B", "R", "G"].indexOf(card.definition.color)].push(card);
              } else if (card.definition.color.length === 0) {
                newSection[Math.min(5, numColumns)].push(card);
              } else {
                newSection[Math.min(6, numColumns)].push(card);
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
  for (const deck of state.decks) {
    localStorage.setItem(
        getLocalstorageKey(deck.draftName, deck.player.seatPosition),
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

function getLocalstorageKey(draftName: string, seatPosition: number) {
  return `draft|${draftName}|${seatPosition}`;
}

export type DeckBuilderStore = typeof deckBuilderStore;

interface DeckBuilderState {
  selectedSeat: number,
  decks: Deck[],
  selection: CardLocation[],
}

export interface Deck {
  draftName: string;
  player: {
    seatPosition: number;
    name: string;
  },
  maindeck: CardColumn[],
  sideboard: CardColumn[],
}

export type CardColumn = DraftCard[];

export interface DeckInitializer {
  draftName: string;
  player: {
    seatPosition: number;
    name: string;
  };
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
