<template>
  <div class="_controls-row">
    <div class="start">
      <div class="draft-name">{{ state.draftName }}</div>
      <TimelineButton />
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
import TimelineButton from './controls_row/TimelineButton.vue';
import SearchBox from './controls_row/SearchBox.vue';

import { navTo } from '../../router/url_manipulation';
import { CoreState } from '../../state/store';
import { getNextPickEventForSelectedPlayer, getNextPickEvent } from '../../state/util/getNextPickEventForSelectedPlayer';
import { TimelineEvent } from '../../draft/TimelineEvent';
import { globalClickTracker, UnhandledClickListener } from '../infra/globalClickTracker';

export default Vue.extend({
  components: {
    TimelineButton,
    SearchBox,
  },

  computed: {
    state(): CoreState {
      return this.$tstore.state;
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
</style>
