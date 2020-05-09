<template>
  <div class="_deck-builder-section-controls">
    <label class="sortLabel">Sort:</label>
    <button class="sortButton" @click="sortCmc">CMC</button>
    <button class="sortButton" @click="sortColor">Color</button>
  </div>
</template>

<script lang="ts">
import Vue from "vue";
import { deckBuilderStore as store } from '../../state/DeckBuilderModule';

export default Vue.extend({
  name: "DeckBuilderSectionControls",

  props: {
    deckIndex: {
      type: Number
    },
    maindeck: {
      type: Boolean
    },
  },

  methods: {
    sortCmc() {
      store.sortByCmc({seat: this.deckIndex, maindeck: this.maindeck});
    },
    sortColor() {
      store.sortByColor({seat: this.deckIndex, maindeck: this.maindeck});
    },
  },
});
</script>

<style scoped>
._deck-builder-section-controls {
  display: flex;
  align-items: center;
  padding: 20px;
}

.sortLabel {
  margin-right: 1em;
  font-size: 80%;
}

.sortButton {
  background: transparent;
  border: 1px solid #bbb;
  border-radius: 1.5em;
  padding: 2px 1em;
  margin-right: 1em;
}

.sortButton:hover {
  background: #ddd;
}

</style>