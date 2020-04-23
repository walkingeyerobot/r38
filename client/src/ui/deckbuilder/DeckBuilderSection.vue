<template>
  <div class="section">
    <DeckBuilderColumn
        v-for="(column, index) in columns"
        :key="index"
        :column="column"
        :deckIndex="deckIndex"
        :maindeck="maindeck"
        :columnIndex="index"
        />
  </div>
</template>

<script lang="ts">
  import Vue from 'vue'
  import DeckBuilderColumn from "./DeckBuilderColumn.vue";
  import { CardColumn } from '../../state/DeckBuilderModule';

  export default Vue.extend({
    name: 'DeckBuilderSection',

    components: {
      DeckBuilderColumn
    },

    props: {
      columns: {
        type: Array as () => CardColumn[]
      },
      deckIndex: {
        type: Number
      },
      maindeck: {
        type: Boolean
      }
    },

  });
</script>

<style scoped>
  .section {
    display: flex;
    flex-direction: row;
    height: 50%;
    overflow-y: scroll;
  }
</style>
