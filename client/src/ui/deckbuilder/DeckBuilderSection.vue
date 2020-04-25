<template>
  <div class="_deck-builder-section">
    <div class="column-cnt">
      <DeckBuilderColumn
          v-for="(column, index) in columns"
          :key="index"
          :column="column"
          :deckIndex="deckIndex"
          :maindeck="maindeck"
          :columnIndex="index"
          />
    </div>
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
._deck-builder-section {
  overflow: scroll;
}

.column-cnt {
  display: flex;
  flex-direction: row;
}
</style>
