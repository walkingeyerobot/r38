<template>
  <div class="_home">
    <div class="main">
      <DraftTable class="table"/>
      <DeckBuilder class="deckbuilder"/>
    </div>
  </div>
</template>

<script lang="ts">
  import Vue from 'vue';

  import DraftTable from './table/DraftTable.vue';
  import DeckBuilder from './deckbuilder/DeckBuilder.vue';
  import {parseDraft} from "../parse/parseDraft";
  import {FAKE_DATA_03} from "../fake_data/FAKE_DATA_03";

  export default Vue.extend({
    name: 'DeckBuilderScreen',

    components: {
      DraftTable,
      DeckBuilder,
    },

    created() {
      const srcData = this.getServerPayload();
      const draft = parseDraft(srcData);
      this.$tstore.commit('initDraft', draft);
      this.$tstore.commit('goTo', this.$tstore.state.events.length);
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
    }
  })
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

  .deckbuilder {
    flex: 1;
  }
</style>