<template>
  <div
      class="_replay"
      @mousedown.capture="onCaptureMouseDown"
      @mousedown="onBubbleMouseDown"
      >
    <ControlsRow />
    <div class="main">
      <PlayerSelector class="table" />
      <CardGrid class="grid" />
    </div>
  </div>
</template>

<script lang="ts">
import Vue from 'vue';
import { parseDraft } from '../parse/parseDraft';
import { SourceData } from '../parse/SourceData';
import { getServerPayload } from '../parse/getServerPayload';

import ControlsRow from './replay/ControlsRow.vue';
import PlayerSelector from './replay/PlayerSelector.vue';
import CardGrid from './replay/CardGrid.vue';
import { SelectedView } from '../state/selection';
import { applyReplayUrlState } from '../router/url_manipulation';
import { globalClickTracker } from './infra/globalClickTracker';

import { replayStore as store } from '../state/ReplayModule';


export default Vue.extend({
  name: 'Replay',

  components: {
    ControlsRow,
    PlayerSelector,
    CardGrid,
  },

  created() {
    const srcData = getServerPayload();
    const draft = parseDraft(srcData);

    store.initDraft(draft);

    document.title = `Replay of ${store.draftName}`;

    if (store.parseError == null) {
      store.setTimeMode('synchronized');
      store.goTo(store.events.length);
    }
    applyReplayUrlState(store, this.$route);
  },

  watch: {
    $route(to, from) {
      applyReplayUrlState(store, this.$route);
    },
  },

  methods: {
    onCaptureMouseDown() {
      globalClickTracker.onCaptureGlobalMouseDown();
    },

    onBubbleMouseDown(e: MouseEvent) {
      globalClickTracker.onBubbleGlobalMouseDown(e);
    },
  },
});
</script>

<style scoped>
._replay {
  height: 100%;
  display: flex;
  flex-direction: column;
}

.main {
  display: flex;
  flex: 1;
  flex-direction: row;
  overflow: hidden;
}

.table {
  width: 300px;
  flex: 0 0 auto;
}

.grid {
  flex: 1;
}
</style>
