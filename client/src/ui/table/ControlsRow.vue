<template>
  <div class="_controls-row">
    <div class="start">
      <div class="draft-name">{{ state.draftName }}</div>
    </div>
    <div class="center">
      <button @click="onStartClick" class="playback-btn">« Start</button>
      <button @click="onPrevClick" class="playback-btn">‹ Prev</button>
      <button @click="onNextClick" class="playback-btn">Next ›</button>
      <button @click="onEndClick" class="playback-btn">End »</button>
    </div>
    <div class="end">
      <input
          type="checkbox"
          id="synchronize-checkbox"
          v-model="synchronizeTimeline">
      <label
          for="synchronize-checkbox"
          class="synchronize-label"
          >
        Synchronize timeline
      </label>
      <button class="location-btn">
        <div class="location-p1">{{ firstLocationLabel }}</div>
        <div v-if="secondLocationLabel" class="location-p2">
          {{ secondLocationLabel }}
        </div>
      </button>
    </div>
  </div>
</template>

<script lang="ts">
import Vue from 'vue'
import { navTo } from '../../router/url_manipulation';
import { CoreState } from '../../state/store';
import { getNextPickEventForSelectedPlayer } from '../../state/util/getNextPickEventForSelectedPlayer';
import { TimelineEvent } from '../../draft/TimelineEvent';

export default Vue.extend({
  computed: {
    synchronizeTimeline: {
      get() {
        return this.$tstore.state.timeMode == 'synchronized';
      },

      set(value) {
        this.$tstore.commit('setTimeMode', value ? 'synchronized' : 'original');
        navTo(this.$tstore, this.$route, this.$router, {});
      }
    },

    state(): CoreState {
      return this.$tstore.state;
    },

    nextPickEvent(): TimelineEvent | null {
      return getNextPickEventForSelectedPlayer(this.state);
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
  },
});
</script>


<style scoped>
._controls-row {
  display: flex;
  flex-direction: row;

  padding: 10px;
  border-bottom: 1px solid #EAEAEA;
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
  text-align: end;
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

.location-btn {
  padding: 6px 15px;
  min-width: 150px;
  text-align: left;
  margin-left: 15px;
  user-select: none;
  cursor: default;
  display: flex;
  flex: 0 0 auto;

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

</style>
