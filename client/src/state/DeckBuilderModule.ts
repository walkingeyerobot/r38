import { DraftCard } from '../draft/DraftState';
import { VuexModule } from './vuex/VuexModule';

/**
 * Vuex module for storing state related to the deck builder.
 */
export const DeckBuilderModule = VuexModule({
  namespaced: true,

  state: {
    selectedSeat: 0,
    decks: [],
  } as DeckBuilderState,

  mutations: {
    initDecks(
      state: DeckBuilderState,
      init: DeckInitializer[],
    ) {
      state.decks = [];
      for (let initializer of init) {
        state.decks.push({
          player: {
            seatPosition: initializer.player.seatPosition,
            name: initializer.player.name,
          },
          sideboard: [initializer.pool, [], [], [], [], [], []],
          maindeck: [[], [], [], [], [], [], []],
        });
      }
    },

    setSelectedSeat(state: DeckBuilderState, selectedSeat: number) {
      state.selectedSeat = selectedSeat;
    },

    moveCard(state: DeckBuilderState, move: CardMove) {
      let card: DraftCard;
      let source: CardColumn[];
      if (move.sourceMaindeck) {
        source = state.decks[move.deckIndex].maindeck;
      } else {
        source = state.decks[move.deckIndex].sideboard;
      }
      if (move.sourceMaindeck !== move.targetMaindeck
          || move.sourceColumnIndex !== move.targetColumnIndex) {
        [card] = source[move.sourceColumnIndex]
            .splice(move.sourceCardIndex, 1);
        let target: CardColumn[];
        if (move.targetMaindeck) {
          target = state.decks[move.deckIndex].maindeck;
        } else {
          target = state.decks[move.deckIndex].sideboard;
        }
        target[move.targetColumnIndex]
            .splice(move.targetCardIndex, 0, card);
      } else if (move.sourceCardIndex !== move.targetCardIndex
          && move.sourceCardIndex !== move.targetCardIndex + 1) {
        [card] = source[move.sourceColumnIndex]
            .splice(move.sourceCardIndex, 1);
        const targetCardIndex =
            (move.targetCardIndex < move.sourceCardIndex)
                ? move.targetCardIndex
                : move.targetCardIndex - 1;
        source[move.targetColumnIndex]
            .splice(targetCardIndex, 0, card);
      }
    },

  },
});

export interface DeckBuilderState {
  selectedSeat: number,
  decks: Deck[],
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

export interface CardMove {
  deckIndex: number,
  sourceColumnIndex: number,
  sourceCardIndex: number,
  sourceMaindeck: boolean,
  targetColumnIndex: number,
  targetCardIndex: number,
  targetMaindeck: boolean,
}
