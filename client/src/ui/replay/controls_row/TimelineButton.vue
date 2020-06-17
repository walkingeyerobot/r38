<!--

A button that displays the current pack and pick number

Clicking on the button opens a browser for jumping to a particular place in
the timeline.

-->

<template>
  <div
      class="_timeline-button"
      @mousedown.capture="onRootMouseDown"
      >
    <button
        class="button"
        @click="onButtonClick"
        >
      <div class="location-p1">{{ labels[0] }}</div>
      <div v-if="labels[1]" class="location-p2">
        {{ labels[1] }}
      </div>
    </button>
    <TimelineSelector
        v-if="timelineOpen"
        class="popover"
        />
  </div>
</template>

<script lang="ts">
import Vue from 'vue';
import TimelineSelector from './TimelineSelector.vue';
import { globalClickTracker, UnhandledClickListener } from '../../infra/globalClickTracker';
import { TimelineEvent } from '../../../draft/TimelineEvent';

import { replayStore as store, ReplayStore, replayStore } from '../../../state/ReplayStore';
import { CardPack, DraftSeat, DraftState } from '../../../draft/DraftState';
import { draftStore } from '../../../state/DraftStore';
import { find } from '../../../util/collection';
import { isPickEvent } from '../../../state/util/isPickEvent';
import { getPack, getSeat } from '../../../state/util/getters';
import { checkNotNil } from '../../../util/checkNotNil';
import { eventToString } from '../../../state/util/eventToString';


export default Vue.extend({
  components: {
    TimelineSelector,
  },

  data() {
    return {
      timelineOpen: false,
      globalMouseDownListener: null as UnhandledClickListener | null,
    };
  },

  computed: {
    labels(): [string, string | null] {
      // If at the end of the draft, say that
      // If a pack is selected, say what that pack is
      // If a player is selected and has a pack, say what that pack is
      // If a player is selected and doesn't have a pack, say what their most
      //    recent pack and pick was
      // If nothing is selected, just show the event number I guess

      if (draftStore.isComplete
          && replayStore.eventPos == replayStore.events.length) {
        return ['End of draft', null];

      } else if (replayStore.selection?.type == 'pack') {
        const pack = getPack(replayStore.draft, replayStore.selection.id);

        // TODO: Don't hard-code pack size
        return [`Pack ${pack.round}`, `Pick ${15 - pack.cards.length + 1}`];

      } else if (replayStore.selection?.type == 'seat') {
        const seat = getSeat(replayStore.draft, replayStore.selection.id);
        const queuedPack = seat.queuedPacks.packs[0];
        if (queuedPack != undefined) {
          const pick =
              getPickCountForPlayer(
                  replayStore,
                  seat.position,
                  queuedPack.round);
          return [`Pack ${queuedPack.round}`, `Pick ${pick + 1}`];
        } else {
          const event = getMostRecentPickEvent(replayStore, seat.position);
          if (event != null) {
            return [`Pack ${event.round}`, `Pick ${event.pick + 1}`];
          } else {
            // Player hasn't picked anything yet, so assume p1p1
            return [`Pack 1`, `Pick 1`];
          }
        }

      } else {
        return [`Event ${replayStore.eventPos}`, null];

      }
    },
  },

  created() {
    this.globalMouseDownListener = () => this.onGlobalMouseDown();
    globalClickTracker
        .registerUnhandledClickListener(this.globalMouseDownListener);
  },

  destroyed() {
    if (this.globalMouseDownListener != null) {
      globalClickTracker
          .unregisterUnhandledClickListener(this.globalMouseDownListener);
    }
  },

  methods: {
    onButtonClick() {
      this.timelineOpen = !this.timelineOpen;
    },

    onRootMouseDown() {
      globalClickTracker.onCaptureLocalMouseDown();
    },

    onGlobalMouseDown() {
      this.timelineOpen = false;
    },
  },

});

function getPickCountForPlayer(
    store: ReplayStore,
    seat: number,
    round: number,
) {
  let count = 0;
  for (let i = store.eventPos - 1; i >= 0; i--) {
    const event = store.events[i];
    if (event.associatedSeat == seat && isPickEvent(event)) {
      if (event.round != round) {
        break;
      }
      count++;
    }
  }
  return count;
}

function getMostRecentPickEvent(store: ReplayStore, seatId: number) {
  for (let i = store.eventPos - 1; i >= 0; i--) {
    const event = store.events[i];
    if (event.associatedSeat == seatId && isPickEvent(event)) {
      return event;
    }
  }
  return null;
}
</script>

<style scoped>

._timeline-button {
  font-size: 14px;
  flex: 0 0 auto;
  position: relative;
}

.button {
  padding: 5px 10px;
  min-width: 125px;
  text-align: left;
  user-select: none;
  cursor: default;
  display: flex;

  /* Override default button styling */
  font-size: 100%;
  font-family: inherit;
  border: 1px solid #c7c7c7;
  border-radius: 5px;
  color: inherit;
}

.button:focus {
  border-color: #D0D0D0;
  outline: none;
}

.location-p1, .location-p2 {
  flex: 1 0 0;
  white-space: nowrap;
}

.location-p2 {
  margin-left: 13px;
}

.popover {
  position: absolute;
  top: calc(100% + 5px);
  left: 0;
  width: 300px;
  height: calc(100vh - 70px);
  background-color: #FFF;
  border-radius: 5px;
  box-shadow: 0px 1px 4px rgba(0, 0, 0, 0.3);
  border: 1px solid #ccc;
}

</style>
