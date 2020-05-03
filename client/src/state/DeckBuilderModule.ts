import { DraftCard } from '../draft/DraftState';
import { VuexModule } from './vuex/VuexModule';

const NUM_COLUMNS = 7;

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
        const sideboard: CardColumn[] = (<DraftCard[][]>Array(NUM_COLUMNS)).fill([]).map(() => []);
        for (const card of initializer.pool) {
          sideboard[Math.min(card.definition.cmc, sideboard.length - 1)].push(card);
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
      let card: DraftCard;
      let source: CardColumn[];
      if (move.source.maindeck) {
        source = state.decks[move.deckIndex].maindeck;
      } else {
        source = state.decks[move.deckIndex].sideboard;
      }
      if (move.source.maindeck !== move.target.maindeck
          || move.source.columnIndex !== move.target.columnIndex) {
        [card] = source[move.source.columnIndex]
            .splice(move.source.cardIndex, 1);
        let target: CardColumn[];
        if (move.target.maindeck) {
          target = state.decks[move.deckIndex].maindeck;
        } else {
          target = state.decks[move.deckIndex].sideboard;
        }
        target[move.target.columnIndex]
            .splice(move.target.cardIndex, 0, card);
      } else if (move.source.cardIndex !== move.target.cardIndex
          && move.source.cardIndex !== move.target.cardIndex + 1) {
        [card] = source[move.source.columnIndex]
            .splice(move.source.cardIndex, 1);
        const targetCardIndex =
            (move.target.cardIndex < move.source.cardIndex)
                ? move.target.cardIndex
                : move.target.cardIndex - 1;
        source[move.target.columnIndex]
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

export interface CardLocation {
  columnIndex: number,
  cardIndex: number,
  maindeck: boolean,
}

export interface CardMove {
  deckIndex: number,
  source: CardLocation,
  target: CardLocation,
}
