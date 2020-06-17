<template>
<div class="_playback-controls">
  <button @click="onStartClick" class="playback-btn">Start</button>
  <button @click="onPrevClick" class="prev-btn playback-btn">Prev</button>
  <button @click="onNextClick" class="next-btn playback-btn">Next</button>
  <button @click="onEndClick" class="playback-btn">End</button>
</div>
</template>

<script lang="ts">
import Vue from 'vue';
import { replayStore } from '../../state/ReplayStore';
import { draftStore } from '../../state/DraftStore';

import { navTo } from '../../router/url_manipulation';

export default Vue.extend({
  methods: {
    onNextClick() {
      replayStore.goNext();
      navTo(draftStore, replayStore, this.$route, this.$router, {});
    },

    onPrevClick() {
      replayStore.goBack();
      navTo(draftStore, replayStore, this.$route, this.$router, {});
    },

    onStartClick() {
      navTo(draftStore, replayStore, this.$route, this.$router, {
        eventIndex: 0,
      });
    },

    onEndClick() {
      navTo(draftStore, replayStore, this.$route, this.$router, {
        eventIndex: replayStore.events.length,
      });
    },
  },
});
</script>

<style scoped>

._playback-controls {
  display: flex;
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

  flex: 1;
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
</style>
