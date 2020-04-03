<template>
  <div class="_home">
    <DraftTable class="table" />
    <CardGrid class="grid" />
  </div>
</template>

<script lang="ts">
import Vue from 'vue';
import { parseInitialState } from '../parse/parseInitialState';
import { TimelineGenerator } from '../parse/TimelineGenerator';
import { SourceData } from '../parse/SourceData';
import { FAKE_DATA_03 } from '../fake_data/FAKE_DATA_03';

import DraftTable from './table/DraftTable.vue';
import CardGrid from './table/CardGrid.vue';

const timeline = new TimelineGenerator();

export default Vue.extend({
  name: 'Home',

  components: {
    DraftTable,
    CardGrid,
  },

  created() {
    const srcData = this.getServerPayload();
    const draft = parseInitialState(srcData);
    const events = timeline.generate(draft, srcData.events);

    this.$tstore.commit('initDraft', { draft, events} );
  },

  methods: {
    getServerPayload() {
      if (window.DraftString != undefined) {
        return JSON.parse(window.DraftString);
      } else {
        return FAKE_DATA_03;
      }
    },
  },

});
</script>

<style scoped>
._home {
  height: 100%;
  display: flex;
  flex-direction: column;
}

.table {
  height: 300px;
  flex: 0 0;
}

.grid {
  flex: 1;
  overflow-y: scroll;
}
</style>
