import { vuexModule } from "./vuex/vuexModule";
import { rootStore } from "./store";
import { buildEmptyDraftState } from "@/draft/buildEmptyDraftState";
import type { DraftCard, DraftState } from "@/draft/DraftState";
import { deepCopy } from "@/util/deepCopy";
import type { TimelineEvent } from "@/draft/TimelineEvent";
import type { SourceData, SourceEvent } from "@/parse/SourceData";
import { MutationError } from "@/draft/MutationError";
import { parseInitialState } from "@/parse/parseInitialState";
import { ParseError } from "@/parse/ParseError";
import { TimelineGenerator } from "@/parse/TimelineGenerator";

import { authStore } from "./AuthStore";
import { checkNotNil } from "@/util/checkNotNil";
import { userIsSeated } from "@/state/util/userIsSeated";

let timelineGenerator: TimelineGenerator;

export const draftStore = vuexModule(
  rootStore,
  "draft",
  {
    draftId: 0,
    draftName: "Unknown draft",
    pickXsrf: "",
    initialState: buildEmptyDraftState(),
    currentState: buildEmptyDraftState(),
    cards: new Map<number, DraftCard>(),
    events: [],
    isComplete: false,
    inPerson: false,
    parseError: null,
  } as State,
  {
    mutations: {
      loadDraft(state: State, payload: SourceData) {
        // For debugging purposes
        window.draftData = payload;

        const parsed = parseInitialState(payload);

        state.draftId = payload.draftId;
        state.draftName = payload.draftName;
        state.pickXsrf = payload.pickXsrf;
        state.initialState = parsed.state;
        const currentState = deepCopy(parsed.state);
        const events = [] as TimelineEvent[];

        console.log("Player ID is", payload.playerId);

        timelineGenerator = new TimelineGenerator(
          currentState,
          parsed.cards,
          events,
          payload.playerId || null,
        );

        const start = performance.now();
        for (let i = 0; i < payload.events.length; i++) {
          const srcEvent = payload.events[i];
          try {
            timelineGenerator.parseEvent(srcEvent);
          } catch (e) {
            if (e instanceof ParseError || e instanceof MutationError) {
              console.error("Error while parsing event", srcEvent, e);
              state.parseError = e;
              break;
            } else {
              throw e;
            }
          }
        }
        console.log(`Parsing draft data took ${performance.now() - start}ms`);
        state.currentState = currentState;
        state.cards = parsed.cards;
        state.events = events;
        state.isComplete = timelineGenerator.isDraftComplete();
        state.inPerson = parsed.state.inPerson;
      },

      pushEvent(state: State, srcEvent: SourceEvent) {
        timelineGenerator.parseEvent(srcEvent);
        state.isComplete = timelineGenerator.isDraftComplete();
      },
    },

    getters: {
      isFilteredDraft(): boolean {
        return (
          !draftStore.isComplete &&
          authStore.user != null &&
          userIsSeated(authStore.user.id, draftStore.currentState)
        );
      },

      isInPersonDraft(): boolean {
        return draftStore.currentState.inPerson;
      },

      getCard(): (id: number) => DraftCard {
        return (id: number) => checkNotNil(draftStore.cards.get(id));
      },

      hasSeatsAvailable(): boolean {
        for (const seat of draftStore.currentState.seats) {
          if (!seat.player.isPresent) {
            return true;
          }
        }
        return false;
      },
    },

    actions: {},
  },
);

export type DraftStore = typeof draftStore;

interface State {
  draftId: number;
  draftName: string;
  pickXsrf: string;
  initialState: DraftState;
  currentState: DraftState;
  cards: Map<number, DraftCard>;
  events: TimelineEvent[];
  isComplete: boolean;
  inPerson: boolean;

  /** Non-null if there was an error while parsing the event stream */
  parseError: Error | null;
}
