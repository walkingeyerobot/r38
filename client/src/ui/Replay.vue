<template>
  <div class="_home">
    <ControlsRow />
    <div class="main">
      <DraftTable class="table" />
      <CardGrid class="grid" />
    </div>
  </div>
</template>

<script lang="ts">
import Vue from 'vue';
import { parseDraft } from '../parse/parseDraft';
import { SourceData } from '../parse/SourceData';
import { getServerPayload } from '../parse/getServerPayload';

import ControlsRow from './table/ControlsRow.vue';
import DraftTable from './table/DraftTable.vue';
import CardGrid from './table/CardGrid.vue';
import { SelectedView } from '../state/selection';
import { applyReplayUrlState } from '../router/url_manipulation';

export default Vue.extend({
  name: 'Home',

  components: {
    ControlsRow,
    DraftTable,
    CardGrid,
  },

  created() {
    const srcData = getServerPayload();
    const draft = parseDraft(srcData);

    this.$tstore.commit('initDraft', draft);

    if (this.$tstore.state.draft.isComplete) {
      this.$tstore.commit('setTimeMode', 'synchronized');
      this.$tstore.commit('goTo', this.$tstore.state.events.length);
    }

    applyReplayUrlState(this.$tstore, this.$route);
  },

  watch: {
    $route(to, from) {
      applyReplayUrlState(this.$tstore, this.$route);
    },
  }
});
</script>

<style scoped>
._home {
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
