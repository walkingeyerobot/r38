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
      <div class="location-p1">{{ firstLocationLabel }}</div>
      <div v-if="secondLocationLabel" class="location-p2">
        {{ secondLocationLabel }}
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
import { CoreState } from '../../../state/store';
import { getNextPickEventForSelectedPlayer, getNextPickEvent } from '../../../state/util/getNextPickEventForSelectedPlayer';
import { TimelineEvent } from '../../../draft/TimelineEvent';

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
    state(): CoreState {
      return this.$tstore.state;
    },

    nextPickEvent(): TimelineEvent | null {
      if (this.state.selection?.type == 'seat') {
        return getNextPickEventForSelectedPlayer(this.state);
      } else {
        return getNextPickEvent(this.state);
      }
    },

    firstLocationLabel(): string {
      const pickEvent = this.nextPickEvent;
      if (pickEvent != null) {
        return `Pack ${pickEvent.round}`;
      } else if (this.state.eventPos >= this.state.events.length) {
        return `End of draft`;
      } else {
        return `Event ${this.state.events[this.state.eventPos].id}`;
      }
    },

    secondLocationLabel(): string | null {
      if (this.nextPickEvent != null) {
        return `Pick ${this.nextPickEvent.pick + 1}`;
      } else {
        return null;
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
