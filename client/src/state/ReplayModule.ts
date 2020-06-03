import { TimelineEvent } from "../draft/TimelineEvent";
import { SelectedView } from './selection';
import { DraftState } from '../draft/DraftState';
import { buildEmptyDraftState } from '../draft/buildEmptyDraftState';
import { vuexModule } from './vuex/vuexModule';
import { ParsedDraft } from '../parse/parseDraft';
import { commitTimelineEvent, rollbackTimelineEvent } from '../draft/mutate';
import { isPickEvent } from './util/isPickEvent';
import { rootStore } from './store';
import { printEvent } from './util/printEvent';
import { deepCopy } from '../util/deepCopy';

// TODO: Remove the need for this
let initialDraftState: DraftState = buildEmptyDraftState();


/**
 * Vuex module for storing state related to the replay viewer.
 */
export const replayStore = vuexModule(rootStore, 'replay', {

  selection: null,
  draft: buildEmptyDraftState(),
  draftId: 0,
  draftName: 'Unknown draft',
  events: [],
  eventPos: 0,
  timeMode: 'original',
  parseError: null,

} as ReplayState, {

  mutations: {
    setSelection(state: ReplayState, selection: SelectedView) {
      state.selection = selection;
    },

    initDraft(
        state: ReplayState,
        payload: ParsedDraft,
    ) {
      initialDraftState = deepCopy(payload.state);

      state.draftName = payload.name,
      state.draft = deepCopy(payload.state);
      state.events = payload.events;
      state.eventPos = 0;
      state.selection = {
        type: 'seat',
        id: 0,
      };
      state.parseError = payload.parseError;

      console.log('Initialized draft with', state.events.length, 'events');
    },

    pushEvent(state: ReplayState, event: TimelineEvent) {
      state.events.push(event);
    },

    goTo(state: ReplayState, index: number) {
      state.draft = deepCopy(initialDraftState);
      let i = 0;
      for (; i < state.events.length && i < index; i++) {
        commitEvent(state.events[i], state.draft);
      }
      state.eventPos = i;
    },

    goNext(state: ReplayState) {
      switch (state.timeMode) {
        case 'original':
          if (state.eventPos < state.events.length) {
            const event = state.events[state.eventPos];
            commitEvent(event, state.draft);
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
              commitEvent(event, state.draft);
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

    goBack(state: ReplayState) {
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

            if (targetEpoch == null && isPickEvent(event)) {
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

    setDraftId(state: ReplayState, draftId: number) {
      state.draftId = draftId;
    },

    setTimeMode(state: ReplayState, mode: TimeMode) {
      if (state.timeMode == mode) {
        return;
      }
      state.timeMode = mode;

      if (mode == 'synchronized') {

        const [currentRound, currentEpoch] = getCurrentEpoch(state);

        state.events.sort(sortEventsLockstep);
        state.draft = deepCopy(initialDraftState);

        state.eventPos = 0;
        let i = 0;
        for (; i < state.events.length; i++) {
          const event = state.events[i];
          if (event.round > currentRound
              || (event.round == currentRound
                  && event.roundEpoch > currentEpoch)) {
            break;
          }
          commitEvent(event, state.draft);
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

        state.draft = deepCopy(initialDraftState);
        state.eventPos = 0;

        if (targetEvent != null) {
          let i = 0;
          for (; i < state.events.length; i++) {
            const event = state.events[i];
            if (event.id == targetEvent.id) {
              break;
            }
            commitEvent(event, state.draft);
          }
          state.eventPos = i;
        }
      }
    },

  },

  getters: {
  },

  actions: { },
});

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

const DEBUG = false;

function commitEvent(event: TimelineEvent, state: DraftState) {
  if (DEBUG) {
    printEvent(event, state);
  }
  commitTimelineEvent(event, state);
}



export type ReplayModule = typeof replayStore;

interface ReplayState {
  selection: SelectedView | null,
  draft: DraftState,
  draftId: number,
  draftName: string,
  events: TimelineEvent[],
  eventPos: number,
  timeMode: TimeMode,

  /** Non-null if there was an error while parsing the event stream */
  parseError: Error | null,
}

export type TimeMode = 'original' | 'synchronized';
