<template>
  <div class="_controls-row">
    <div class="start">
      <TimelineButton class="timeline-btn" />
      <button @click="onStartClick" class="playback-btn">Start</button>
      <button @click="onPrevClick" class="prev-btn playback-btn">Prev</button>
      <button @click="onNextClick" class="next-btn playback-btn">Next</button>
      <button @click="onEndClick" class="playback-btn">End</button>
    </div>
    <div class="center">
      <div class="draft-name">{{ store.draftName }}</div>
    </div>
    <div class="end">
      <SearchBox />
    </div>
  </div>
</template>

<script lang="ts">
import Vue from 'vue';
import TimelineButton from './controls_row/TimelineButton.vue';
import SearchBox from './controls_row/SearchBox.vue';

import { navTo } from '../../router/url_manipulation';
import { getNextPickEventForSelectedPlayer, getNextPickEvent } from '../../state/util/getNextPickEventForSelectedPlayer';
import { TimelineEvent } from '../../draft/TimelineEvent';
import { globalClickTracker, UnhandledClickListener } from '../infra/globalClickTracker';

import { replayStore as store, ReplayModule } from '../../state/ReplayModule';


export default Vue.extend({
  components: {
    TimelineButton,
    SearchBox,
  },

  computed: {
    store(): ReplayModule {
      return store;
    },
  },

  methods: {
    onNextClick() {
      store.goNext();
      navTo(store, this.$route, this.$router, {});
    },

    onPrevClick() {
      store.goBack();
      navTo(store, this.$route, this.$router, {});
    },

    onStartClick() {
      navTo(store, this.$route, this.$router, {
        eventIndex: 0,
      });
    },

    onEndClick() {
      navTo(store, this.$route, this.$router, {
        eventIndex: store.events.length,
      });
    },
  },
});
</script>

<style scoped>
._controls-row {
  display: flex;
  flex-direction: row;

  font-size: 14px;

  padding: 13px 12px;
  border-bottom: 1px solid #EAEAEA;
  z-index: 1;

  box-shadow: 0 1px 2px rgba(0, 0, 0, 0.05);
}

.start {
  flex: 1 0 0;
  display: flex;
  align-items: center;
}

.center {
  flex: 1 0 0;
  display: flex;
  align-items: center;
  justify-content: center;
}

.end {
  flex: 1 0 0;
  display: flex;
  align-items: center;
  justify-content: flex-end;
}

.timeline-btn {
  margin-right: 4px;
}

.playback-btn {
  margin: 0 4px;
  width: 55px;
  height: 28px;

  font-family: inherit;
  font-size: 14px;

  border: 1px solid #dcdcdc;
  border-radius: 5px;

  color: #444;
  -webkit-appearance: none;
}

.playback-btn:focus {
  outline: none;
  border-color: #aaa;
}

.playback-btn:active {
  border-color: #777;
}

.prev-btn, .next-btn {
  width: 70px;
}

.synchronize-label {
  margin-left: 4px;
}

.draft-name {
  font-size: 16px;
}
</style>
