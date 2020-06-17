import { rootStore } from './store';
import { authStore } from './AuthStore';
import { draftStore, DraftStore } from './DraftStore';

import { SelectedView } from './selection';
import { vuexModule } from './vuex/vuexModule';
import { TimelineEvent } from "../draft/TimelineEvent";
import { DraftState } from '../draft/DraftState';
import { buildEmptyDraftState } from '../draft/buildEmptyDraftState';
import { commitTimelineEvent, rollbackTimelineEvent } from '../draft/mutate';
import { CardStore } from '../draft/CardStore';
import { getUserPosition } from '../draft/util/userIsSeated';
import { deepCopy } from '../util/deepCopy';
import { indexOf } from '../util/collection';
import { isPickEvent } from './util/isPickEvent';
import { getNextPickEvent, getPreviousPickEvent } from './util/eventSearch';


/**
 * Vuex module for storing state related to the replay viewer.
 */
export const replayStore = vuexModule(rootStore, 'replay', {

  selectedDraft: 0,
  rawSelection: null,
  events: [],
  draft: buildEmptyDraftState(),
  eventPos: 0,
  timeMode: 'original',

} as ReplayState, {

  mutations: {
    setSelection(state: ReplayState, selection: SelectedView) {
      state.rawSelection = selection;
    },

    sync(state: ReplayState) {
      console.log('Syncing replay state...');

      const event = state.events[state.eventPos];

      state.events = draftStore.events.concat();
      sortEvents(state.events, state.timeMode);

      const newIndex = event
          ? indexOf(state.events, { id: event.id })
          : state.events.length;

      freshJumpTo(draftStore, state, newIndex);
    },

    goTo(state: ReplayState, index: number) {
      freshJumpTo(draftStore, state, index);
    },

    goNext(state: ReplayState) {
      switch (state.timeMode) {
        case 'original':
          if (state.eventPos < state.events.length) {
            const event = state.events[state.eventPos];
            commitTimelineEvent(draftStore, event, state.draft);
            state.eventPos++;
          }
          break;

        case 'synchronized':
          const seatId = replayStore.selection?.type == 'seat'
              ? replayStore.selection.id : null;
          const targetEvent = getNextPickEvent(replayStore, seatId);

          commitWhile(
              state,
              event => targetEvent == null || !occursAfter(event, targetEvent));

          const nextNextEvent = getNextPickEvent(replayStore, seatId);
          if (nextNextEvent == null) {
            // No more events for this player, just fast-forward to the end
            commitWhile(state, event => true);
          }
          break;

        default:
          throw new Error(`Unrecognized timeMode: ${state.timeMode}`);
      }
    },

    goBack(state: ReplayState) {
      switch (state.timeMode) {
        case 'original':
          if (state.eventPos > 0) {
            state.eventPos--;
            const event = state.events[state.eventPos];
            rollbackTimelineEvent(draftStore, event, state.draft);
          }
          break;

        case 'synchronized':
          const seatId = replayStore.selection?.type == 'seat'
              ? replayStore.selection.id
              : null;
          const targetEvent = getPreviousPickEvent(replayStore, seatId);

          rollbackWhile(
              state,
              event =>
                  targetEvent == null || !occursBefore(event, targetEvent));
          break;

        default:
          throw new Error(`Unrecognized timeMode: ${state.timeMode}`);
      }
    },

    setTimeMode(state: ReplayState, mode: TimeMode) {
      if (state.timeMode == mode) {
        return;
      }
      state.timeMode = mode;

      if (mode == 'synchronized') {

        const [currentRound, currentEpoch] = getCurrentEpoch(state);

        state.events.sort(sortEventsLockstep);
        state.draft = deepCopy(draftStore.initialState);
        state.eventPos = 0;

        let i = 0;
        for (; i < state.events.length; i++) {
          const event = state.events[i];
          if (event.round > currentRound
              || (event.round == currentRound
                  && event.roundEpoch > currentEpoch)) {
            break;
          }
          commitTimelineEvent(draftStore, event, state.draft);
        }
        state.eventPos = i;

      } else {
        // If seat selected, find the next event for that seat; use it
        // If no seat selected, just use the next event
        let targetEvent: TimelineEvent | null = null;
        if (state.rawSelection == null || state.rawSelection.type == 'pack') {
          targetEvent = state.events[state.eventPos];
        } else {
          for (let i = state.eventPos; i < state.events.length; i++) {
            const event = state.events[i];
            if (event.associatedSeat == state.rawSelection.id) {
              targetEvent = event;
              break;
            }
          }
        }

        state.events.sort((a, b) => a.id - b.id);

        state.draft = deepCopy(draftStore.initialState);
        state.eventPos = 0;

        if (targetEvent != null) {
          let i = 0;
          for (; i < state.events.length; i++) {
            const event = state.events[i];
            if (event.id == targetEvent.id) {
              break;
            }
            commitTimelineEvent(draftStore, event, state.draft);
          }
          state.eventPos = i;
        }
      }
    },

  },

  getters: {
    selection(): SelectedView | null {
      // Build in some protection so that it's impossible for us to show active
      // drafters anyone else's cards.
      if (draftStore.isActiveDraft) {
        return {
          type: 'seat',
          id: getUserPosition(authStore.user!.id, draftStore.currentState),
        };
      } else {
        return replayStore.rawSelection;
      }
    },
  },

  actions: { },
});

function sortEvents(events: TimelineEvent[], timeMode: TimeMode) {
  const sortFunc = timeMode == 'synchronized'
          ? sortEventsLockstep
          : (a: TimelineEvent, b: TimelineEvent) => a.id - b.id;
  events.sort(sortFunc);
}

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

function getCurrentEpoch(state: ReplayState): [number, number] {
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

function freshJumpTo(
    draftStore: DraftStore,
    state: ReplayState,
    index: number,
) {
  const start = performance.now();
  const draft = deepCopy(draftStore.initialState);
  let i = 0;
  for (; i < state.events.length && i < index; i++) {
    commitTimelineEvent(draftStore, state.events[i], draft);
  }
  state.draft = draft;
  state.eventPos = i;
  console.log(`Applied ${i} events in ${performance.now() - start}ms`);

  consumeTransitionEvents(draftStore, state);
}

function rollbackWhile(
    state: ReplayState,
    predicate: (event: TimelineEvent, index: number) => boolean,
) {
  let i = state.eventPos - 1;
  for (; i >= 0; i--) {
    const event = state.events[i];
    if (!predicate(event, i)) {
      break;
    }
    rollbackTimelineEvent(draftStore, event, state.draft);
  }
  const prevPos = state.eventPos;
  state.eventPos = i + 1;

  consumeTransitionEvents(draftStore, state);
}

function commitWhile(
    state: ReplayState,
    predicate: (event: TimelineEvent, index: number) => boolean,
) {
  let i = state.eventPos;
  for (; i < state.events.length; i++) {
    const event = state.events[i];
    // console.log('Event', i, 'round:', event.round, 'epoch:', event.roundEpoch);
    if (!predicate(event, i)) {
      break;
    }
    commitTimelineEvent(draftStore, event, state.draft);
  }
  const prevPos = state.eventPos;
  state.eventPos = i;

  consumeTransitionEvents(draftStore, state);

  // console.log('eventPos:', prevPos, '->', state.eventPos);
}

function occursAfter(a: TimelineEvent, b: TimelineEvent) {
  return cmpEpoch(a, b) > 0;
}

function occursBefore(a: TimelineEvent, b: TimelineEvent) {
  return cmpEpoch(a, b) < 0;
}

function cmpEpoch(a: TimelineEvent, b: TimelineEvent) {
  if (a.round != b.round) {
    return a.round - b.round;
  } else {
    return a.roundEpoch - b.roundEpoch;
  }
}

function consumeTransitionEvents(
    cardStore: CardStore,
    state: ReplayState
) {
  for (let i = state.eventPos; i < state.events.length; i++) {
    const event = state.events[i];
    if (!isPickEvent(event)) {
      commitTimelineEvent(cardStore, event, state.draft);
      state.eventPos++;
    } else {
      break;
    }
  }
}

export type ReplayStore = typeof replayStore;

interface ReplayState {
  rawSelection: SelectedView | null,
  draft: DraftState,
  events: TimelineEvent[],
  eventPos: number,
  timeMode: TimeMode,
}

export type TimeMode = 'original' | 'synchronized';
