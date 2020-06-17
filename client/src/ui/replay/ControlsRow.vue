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
      <div class="draft-name">
        {{ draftStore.draftName }}
        <span
            v-if="draftStore.parseError != null"
            class="parse-error-warning"
            >
          [parse error]
        </span>
      </div>
    </div>
    <div class="end">
      <SearchBox v-if="!draftStore.isActiveDraft" />
      <img v-if="authStore.user" class="user-img" :src="authStore.user.picture">
    </div>
  </div>
</template>

<script lang="ts">
import Vue from 'vue';
import TimelineButton from './controls_row/TimelineButton.vue';
import SearchBox from './controls_row/SearchBox.vue';

import { navTo } from '../../router/url_manipulation';
import { TimelineEvent } from '../../draft/TimelineEvent';
import { globalClickTracker, UnhandledClickListener } from '../infra/globalClickTracker';

import { authStore, AuthStore } from '../../state/AuthStore';
import { draftStore, DraftStore } from '../../state/DraftStore';
import { replayStore, ReplayStore } from '../../state/ReplayStore';

export default Vue.extend({
  components: {
    TimelineButton,
    SearchBox,
  },

  computed: {
    replayStore(): ReplayStore {
      return replayStore;
    },

    authStore(): AuthStore {
      return authStore;
    },

    draftStore(): DraftStore {
      return draftStore;
    }
  },

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

.parse-error-warning {
  color: #F00;
}

.user-img {
  width: 28px;
  margin-left: 10px;
  border-radius: 20px;
}
</style>
