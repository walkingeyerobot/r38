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
import { FAKE_DATA_03 } from '../fake_data/FAKE_DATA_03';

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
    const srcData = this.getServerPayload();
    const draft = parseDraft(srcData);

    this.$tstore.commit('initDraft', {
      id: parseInt(this.$route.params['draftId']),
      draft,
    });

    if (this.$tstore.state.draft.isComplete) {
      this.$tstore.commit('setTimeMode', 'synchronized');
      this.$tstore.commit('goTo', this.$tstore.state.events.length);
    }

    applyReplayUrlState(this.$tstore, this.$route.params);
  },

  methods: {
    getServerPayload() {
      if (window.DraftString != undefined) {
        console.log('Found server payload, loading!');
        return JSON.parse(window.DraftString);
      } else {
        console.log(`Couldn't find server payload, falling back to default...`);
        return FAKE_DATA_03;
      }
    },
  },

  watch: {
    $route(to, from) {
      applyReplayUrlState(this.$tstore, this.$route.params);
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
