<template>
  <div class="_controls-row">
    <div class="start">
      <TimelineButton class="timeline-btn" />
      <button
          v-if="draftStore.isFilteredDraft"
          @click="onPicksClick"
          class="picks-btn playback-btn"
          >
        {{ numPicks }} {{ numPicks == 1 ? 'pick' : 'picks' }} available
      </button>
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
      <SearchBox />
      <img v-if="authStore.user" class="user-img" :src="authStore.user.picture">
    </div>
  </div>
</template>

<script lang="ts">
import Vue from 'vue';
import TimelineButton from './controls_row/TimelineButton.vue';
import SearchBox from './controls_row/SearchBox.vue';

import { authStore, AuthStore } from '../../state/AuthStore';
import { draftStore, DraftStore } from '../../state/DraftStore';
import { replayStore, ReplayStore } from '../../state/ReplayStore';

import { navTo } from '../../router/url_manipulation';
import { TimelineEvent } from '../../draft/TimelineEvent';
import { globalClickTracker, UnhandledClickListener } from '../infra/globalClickTracker';
import { isAuthedUserSelected } from './isAuthedUserSelected';
import { getUserPosition } from '../../state/util/userIsSeated';

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
    },

    numPicks(): number {
      const seatId =
          getUserPosition(authStore.user?.id, draftStore.currentState);
      if (seatId == -1) {
        return 0;
      } else {
        return draftStore.currentState.seats[seatId].queuedPacks.packs.length;
      }
    },
  },

  methods: {
    onPicksClick() {
      const seatId =
          getUserPosition(authStore.user?.id, draftStore.currentState);
      navTo(draftStore, replayStore, this.$route, this.$router, {
        eventIndex: replayStore.events.length,
        selection: {
          type: 'seat',
          id: seatId
        },
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

.picks-btn {
  width: auto;
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

.playback-btn {
  margin: 0 4px;
  height: 28px;
  min-width: 100px;
  padding: 0 10px;

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
</style>
