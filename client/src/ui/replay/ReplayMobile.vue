<!--

An interface for mobile use

-->

<template>
  <div class="_replay-mobile">
    <div class="title-bar">
      {{ draftStore.draftName }}
    </div>

    <template>
      <DraftPicker
          v-if="showDraftPicker"
          class="main-content picker"
          :showDeckBuilder="false"
          />
      <CardGrid v-else class="main-content grid" />
    </template>

    <div class="footer">

      <img class="icon" :src="selectedUserIcon">

      <div class="footer-center">
        <button
            v-long-press="400"
            @click="onPrevClick"
            @long-press-start="onPrevLongPress"
            @contextmenu.prevent
            class="left-button"
            >
          Prev
        </button>
        <TimelineButton
            class="timeline-button"
            popover-alignment="center above" />
        <button
            v-long-press="400"
            @click="onNextClick"
            @long-press-start="onNextLongPress"
            @contextmenu.prevent
            class="right-button"
            >
          Next
        </button>
      </div>

      <div class="footer-right"></div>
    </div>
  </div>
</template>

<script lang="ts">
import Vue from 'vue';
import LongPress from 'vue-directive-long-press';

import CardGrid from './CardGrid.vue';
import DraftPicker from './DraftPicker.vue';
import TimelineButton from './controls_row/TimelineButton.vue';

import { draftStore, DraftStore } from '../../state/DraftStore';
import { replayStore } from '../../state/ReplayStore';

import { pushDraftUrlRelative, pushDraftUrlFromState } from '../../router/url_manipulation';
import { FALLBACK_USER_PICTURE } from '../../parse/fallbacks';


export default Vue.extend({
  components: {
    DraftPicker,
    CardGrid,
    TimelineButton,
  },

  directives: {
    'long-press': LongPress
  },

  props: {
    showDraftPicker: {
      type: Boolean,
      required: true,
    },
  },

  computed: {

    draftStore(): DraftStore {
      return draftStore;
    },

    selectedUserIcon(): string {
      if (replayStore.selection == null) {
        // TODO Fill this in with something that makes more sense
        return FALLBACK_USER_PICTURE;
      } else if (replayStore.selection.type == 'pack') {
        // TODO Fill this in with something that makes more sense
        return FALLBACK_USER_PICTURE;
      } else {
        return replayStore.draft.seats[replayStore.selection.id].player.iconUrl;
      }
    },
  },

  methods: {
    onPrevClick() {
      replayStore.goBack();
      pushDraftUrlFromState(this, draftStore, replayStore);
    },

    onPrevLongPress() {
      pushDraftUrlRelative(this, {
        eventIndex: 0,
      });
    },

    onNextClick() {
      replayStore.goNext();
      pushDraftUrlFromState(this, draftStore, replayStore);
    },

    onNextLongPress() {
      pushDraftUrlRelative(this, {
        eventIndex: replayStore.events.length,
      });
    },

  }
})
</script>

<style scoped>
._replay-mobile {
  height: 100%;
  display: flex;
  flex-direction: column;
}

.title-bar {
  text-align: center;
  background: #fff;
  color: #000;
  border-bottom: 1px solid #eaeaea;
  padding: 10px 0;
}

.main-content {
  flex: 1;
}

.footer {
  display: flex;
  align-items: center;
  background: #ffffff;
  border-top: 1px solid #eaeaea;
  padding: 7px 0;
}

.icon {
  width: 35px;
  height: 35px;
  box-sizing: border-box;
  border: 1px solid #c7c7c7;
  border-radius: 35px;
  margin: 0 7px;
}

.footer-center {
  flex: 1;
  display: flex;
  justify-content: center;
  align-items: center;
}

.left-button, .right-button {
  padding: 5px 10px;
  min-width: 52px;
  text-align: center;
  user-select: none;
  cursor: default;
  font-size: 14px;
  font-family: inherit;
  border: 1px solid #c7c7c7;
  border-radius: 5px;
  color: inherit;
}

.timeline-button {
  margin-left: 6px;
  margin-right: 6px;
}

.footer-right {
  width: 45px;
}

</style>
