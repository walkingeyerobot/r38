<template>
  <div class="_controls-row">
    <div class="start">
      <div class="draft-name">{{ state.draftName }}</div>
      <div
          class="location-cnt"
          @mousedown.capture="onTimelineMouseDown"
          >
        <button
            class="location-btn"
            @click="onLocationClick"
            >
          <div class="location-p1">{{ firstLocationLabel }}</div>
          <div v-if="secondLocationLabel" class="location-p2">
            {{ secondLocationLabel }}
          </div>
        </button>
        <TimelineSelector
            v-if="timelineOpen"
            class="timeline-popover"
            />
      </div>
    </div>
    <div class="center">
      <button @click="onStartClick" class="playback-btn">« Start</button>
      <button @click="onPrevClick" class="playback-btn">‹ Prev</button>
      <button @click="onNextClick" class="playback-btn">Next ›</button>
      <button @click="onEndClick" class="playback-btn">End »</button>
    </div>
    <div class="end">
      <SearchBox />
    </div>
  </div>
</template>

<script lang="ts">
import Vue from 'vue';
import TimelineSelector from './TimelineSelector.vue';
import SearchBox from './controls_row/SearchBox.vue';

import { navTo } from '../../router/url_manipulation';
import { CoreState } from '../../state/store';
import { getNextPickEventForSelectedPlayer, getNextPickEvent } from '../../state/util/getNextPickEventForSelectedPlayer';
import { TimelineEvent } from '../../draft/TimelineEvent';
import { globalClickTracker, UnhandledClickListener } from '../infra/globalClickTracker';

export default Vue.extend({
  components: {
    TimelineSelector,
    SearchBox,
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
    onNextClick() {
      this.$tstore.commit('goNext');
      navTo(this.$tstore, this.$route, this.$router, {});
    },

    onPrevClick() {
      this.$tstore.commit('goBack');
      navTo(this.$tstore, this.$route, this.$router, {});
    },

    onStartClick() {
      navTo(this.$tstore, this.$route, this.$router, {
        eventIndex: 0,
      });
    },

    onEndClick() {
      navTo(this.$tstore, this.$route, this.$router, {
        eventIndex: this.state.events.length,
      });
    },

    onLocationClick() {
      this.timelineOpen = !this.timelineOpen;
    },

    onTimelineMouseDown() {
      globalClickTracker.onCaptureLocalMouseDown();
    },

    onGlobalMouseDown() {
      this.timelineOpen = false;
    },
  },
});
</script>


<style scoped>
._controls-row {
  display: flex;
  flex-direction: row;

  padding: 10px;
  border-bottom: 1px solid #EAEAEA;
  z-index: 1;
}

.start {
  flex: 1 0 0;
  display: flex;
  align-items: center;
}

.center {
  flex: 1 0 0;
  text-align: center;
}

.end {
  flex: 1 0 0;
  display: flex;
  align-items: center;
  justify-content: flex-end;
}

.playback-btn {
  margin: 0 5px;
  width: 70px;
  height: 30px;
  border-radius: 3px;
}

.synchronize-label {
  margin-left: 4px;
}

.draft-name {
  color: #828282;
}

.location-cnt {
  margin-left: 15px;
  font-size: 14px;
  flex: 0 0 auto;
  position: relative;
}

.location-btn {
  padding: 6px 10px;
  min-width: 125px;
  text-align: left;
  user-select: none;
  cursor: default;
  display: flex;

  /* Override default button styling */
  font-size: 100%;
  font-family: inherit;
  border: 1px solid #EAEAEA;
  border-radius: 5px;
}

.location-btn:focus {
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

.timeline-popover {
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
