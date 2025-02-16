<!--

An interface for mobile use

-->

<template>
  <div class="_replay-mobile">
    <div class="title-bar">
      {{ draftStore.draftName }}
    </div>

    <DraftPicker
        v-if="showDraftPicker"
        class="main-content picker"
        :showDeckBuilder="false"
        :inPersonDraft="inPersonDraft"
        />
    <CardGrid v-else class="main-content grid" />

    <div class="footer">

      <img class="icon" :src="selectedUserIcon">

      <div class="footer-center">
        <button
            v-long-press="400"
            ref="prevButton"
            @click="onPrevClick"
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
            ref="nextButton"
            @click="onNextClick"
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
import { defineComponent, ref } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { onLongPress } from '@vueuse/core';

import CardGrid from './CardGrid.vue';
import DraftPicker from './DraftPicker.vue';
import TimelineButton from './controls_row/TimelineButton.vue';

import { draftStore, DraftStore } from '../../state/DraftStore';
import { replayStore } from '../../state/ReplayStore';

import { pushDraftUrlRelative, pushDraftUrlFromState } from '../../router/url_manipulation';
import { FALLBACK_USER_PICTURE } from '../../parse/fallbacks';


export default defineComponent({
  components: {
    DraftPicker,
    CardGrid,
    TimelineButton,
  },

  props: {
    showDraftPicker: {
      type: Boolean,
      required: true,
    },
    inPersonDraft: {
      type: Boolean,
      required: true,
    },
  },

  setup() {
    const prevButton = ref<HTMLElement | null>(null);
    const nextButton = ref<HTMLElement | null>(null);

    const route = useRoute();
    const router = useRouter();

    const routeProvider = {
      $route: route,
      $router: router,
    };

    onLongPress(
      prevButton,
      onPrevLongPress,
      { modifiers: { prevent: true } },
    );

    onLongPress(
      nextButton,
      onNextLongPress,
      { modifiers: { prevent: true } },
    );

    function onPrevLongPress() {
      pushDraftUrlRelative(routeProvider, {
        eventIndex: 0,
      });
    }

    function onNextLongPress() {
      pushDraftUrlRelative(routeProvider, {
        eventIndex: replayStore.events.length,
      });
    }
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

    onNextClick() {
      replayStore.goNext();
      pushDraftUrlFromState(this, draftStore, replayStore);
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
