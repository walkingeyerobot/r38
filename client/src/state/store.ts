import Vue from 'vue';
import Vuex from 'vuex';
import {SelectedView} from './selection';
import {DraftState, MtgCard} from '../draft/DraftState';
import {TimelineEvent} from '../draft/TimelineEvent';
import {buildEmptyDraftState} from '../draft/buildEmptyDraftState';
import {commitTimelineEvent, rollbackTimelineEvent} from '../draft/mutate';
import {cloneDraftState} from '../draft/cloneDraftState';
import {find} from '../util/collection';

Vue.use(Vuex);

export interface RootState {
  selection: SelectedView | null,
  draft: DraftState,
  events: TimelineEvent[],
  eventPos: number,
  timeMode: TimeMode,
  decks: Deck[],
}

export type TimeMode = 'original' | 'synchronized';

export type CardColumn = MtgCard[];

export interface Deck {
  maindeck: CardColumn[],
  sideboard: CardColumn[],
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

const state: RootState = {
  selection: null,
  draft: buildEmptyDraftState(),
  events: [],
  eventPos: 0,
  timeMode: 'original',
  decks: [],
};

let initialDraftState: DraftState = buildEmptyDraftState();

const store = new Vuex.Store({
  state,

  mutations: {
    setSelection(state: RootState, selection: SelectedView) {
      state.selection = selection;
    },

    initDraft(
        state: RootState,
        payload: { state: DraftState, events: TimelineEvent[] }
    ) {
      initialDraftState = cloneDraftState(payload.state);
      state.draft = cloneDraftState(payload.state);
      state.events = payload.events;
      state.eventPos = 0;
    },

    pushEvent(state: RootState, event: TimelineEvent) {
      state.events.push(event);
    },

    goTo(state: RootState, index: number) {
      state.draft = cloneDraftState(initialDraftState);
      let i = 0;
      for (; i < state.events.length && i < index; i++) {
        commitTimelineEvent(state.events[i], state.draft);
      }
      state.eventPos = i;
    },

    goNext(state: RootState) {
      switch (state.timeMode) {
        case 'original':
          if (state.eventPos < state.events.length) {
            const event = state.events[state.eventPos];
            commitTimelineEvent(event, state.draft);
            state.eventPos++;
          }
          break;
        case 'synchronized':
          const nextEvent = state.events[state.eventPos];
          if (nextEvent == null) {
            return;
          }
          for (let i = state.eventPos; i < state.events.length; i++) {
            const event = state.events[i];
            if (event.roundEpoch == nextEvent.roundEpoch
                // Always skip over roundEpoch 0, which is just opening packs
                || event.roundEpoch == 0) {
              commitTimelineEvent(event, state.draft);
              state.eventPos++;
            } else {
              break;
            }
          }
          break;
        default:
          throw new Error(`Unrecognized timeMode: ${state.timeMode}`);
      }
    },

    goBack(state: RootState) {
      switch (state.timeMode) {
        case 'original':
          if (state.eventPos > 0) {
            state.eventPos--;
            const event = state.events[state.eventPos];
            rollbackTimelineEvent(event, state.draft);
          }
          break;
        case 'synchronized':
          // We want to roll back epochs until we roll back an epoch that
          // contains a pick (some epochs just contain things like opening
          // packs). So we roll back events until we find one that picks a card,
          // then set the targetEpoch to that event's epoch.
          let targetEpoch = null as number | null;

          for (let i = state.eventPos - 1; i >= 0; i--) {
            const event = state.events[i];

            if (targetEpoch == null
                && find(event.actions, {type: 'move-card'}) != -1) {
              targetEpoch = event.roundEpoch;
            }

            if (targetEpoch == null || event.roundEpoch == targetEpoch) {
              rollbackTimelineEvent(event, state.draft);
              state.eventPos--;
            } else {
              break;
            }
          }
          break;
        default:
          throw new Error(`Unrecognized timeMode: ${state.timeMode}`);
      }
    },

    setTimeMode(state: RootState, mode: TimeMode) {
      if (state.timeMode == mode) {
        return;
      }
      state.timeMode = mode;

      if (mode == 'synchronized') {

        const [currentRound, currentEpoch] = getCurrentEpoch(state);

        state.events.sort(sortEventsLockstep);
        state.draft = cloneDraftState(initialDraftState);

        console.log('Fast-forwarding...');
        state.eventPos = 0;
        let i = 0;
        for (; i < state.events.length; i++) {
          const event = state.events[i];
          if (event.round > currentRound
              || (event.round == currentRound
                  && event.roundEpoch > currentEpoch)) {
            break;
          }
          console.log('  Applying event', event.id);
          commitTimelineEvent(event, state.draft);
        }
        state.eventPos = i;

      } else {
        // If seat selected, find the next event for that seat; use it
        // If no seat selected, just use the next event
        let targetEvent: TimelineEvent | null = null;
        if (state.selection == null || state.selection.type == 'pack') {
          targetEvent = state.events[state.eventPos];
        } else {
          for (let i = state.eventPos; i < state.events.length; i++) {
            const event = state.events[i];
            if (event.associatedSeat == state.selection.id) {
              targetEvent = event;
              break;
            }
          }
        }

        state.events.sort((a, b) => a.id - b.id);

        console.log('Fast-forwarding...');
        state.draft = cloneDraftState(initialDraftState);
        state.eventPos = 0;

        if (targetEvent != null) {
          let i = 0;
          for (; i < state.events.length; i++) {
            const event = state.events[i];
            if (event.id == targetEvent.id) {
              break;
            }
            console.log('  Applying event', event.id);
            commitTimelineEvent(event, state.draft);
          }
          state.eventPos = i;
        }
      }
    },

    moveCard(state: RootState, move: CardMove) {
      let card: MtgCard;
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

  getters: {},

  actions: {},

});

export default store;

function sortEventsLockstep(a: TimelineEvent, b: TimelineEvent) {
  let cmp = a.round - b.round;
  if (cmp != 0) {
    return cmp;
  }

  cmp = a.roundEpoch - b.roundEpoch;
  if (cmp != 0) {
    return cmp;
  }

  cmp = a.associatedSeat - b.associatedSeat;
  if (cmp != 0) {
    return cmp;
  }

  return a.id - b.id;
}

function getCurrentEpoch(state: RootState): [number, number] {
  let currentRound = 1;
  let currentEpoch = 0;
  const mostRecentEvent =
      state.events[state.eventPos - 1] as TimelineEvent | undefined;
  if (mostRecentEvent != null) {
    currentRound = mostRecentEvent.round;
    currentEpoch = mostRecentEvent.roundEpoch;
  }
  return [currentRound, currentEpoch];
}
