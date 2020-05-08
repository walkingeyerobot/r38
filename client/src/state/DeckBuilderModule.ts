import { DraftCard } from '../draft/DraftState';
import { vuexModule } from './vuex/vuexModule2';
import { rootStore } from './store';


const NUM_COLUMNS = 7;


/**
 * Vuex module for storing state related to the deck builder.
 */
export const deckBuilderStore = vuexModule(rootStore, 'deckbuilder', {

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
        const sideboard: CardColumn[] =
            (<DraftCard[][]>Array(NUM_COLUMNS)).fill([]).map(() => []);
        for (const card of initializer.pool) {
          sideboard[Math.min(card.definition.cmc, sideboard.length - 1)]
              .push(card);
        }
        state.decks.push({
          player: {
            seatPosition: initializer.player.seatPosition,
            name: initializer.player.name,
          },
          sideboard,
          maindeck: (<DraftCard[][]>Array(NUM_COLUMNS)).fill([]).map(() => []),
        });
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
      let targetSection: CardColumn[];
      if (move.target.maindeck) {
        targetSection = state.decks[move.deckIndex].maindeck;
      } else {
        targetSection = state.decks[move.deckIndex].sideboard;
      }
      targetSection[move.target.columnIndex]
          .splice(move.target.cardIndex, 0, ...cards);
      state.selection = [];
    },

    selectCards(state: DeckBuilderState, selection: CardLocation[]) {
      state.selection = selection;
    },
  },

  getters: {},

  actions: {},
});

export type DeckBuilderStore = typeof deckBuilderStore;

interface DeckBuilderState {
  selectedSeat: number,
  decks: Deck[],
  selection: CardLocation[],
}

export interface Deck {
  player: {
    seatPosition: number;
    name: string;
  },
  maindeck: CardColumn[],
  sideboard: CardColumn[],
}

export type CardColumn = DraftCard[];

export interface DeckInitializer {
  player: {
    seatPosition: number;
    name: string;
  },
  pool: DraftCard[],
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
