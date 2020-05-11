<template>
  <div class="_deck-builder-section-controls">
    <p
        class="cardsCount"
        :class="{tooFewCards}"
        >
      {{ numCards }} cards
    </p>
    <label class="sortLabel">Sort:</label>
    <button class="sortButton" @click="sortCmc">CMC</button>
    <button class="sortButton" @click="sortColor">Color</button>
  </div>
</template>

<script lang="ts">
import Vue from "vue";
import { deckBuilderStore as store } from '../../state/DeckBuilderModule';

export default Vue.extend({
  name: 'DeckBuilderSectionControls',

  props: {
    maindeck: {
      type: Boolean
    },
  },

  computed: {
    numCards(): number {
      return (this.maindeck
          ? store.decks[store.selectedSeat].maindeck
          : store.decks[store.selectedSeat].sideboard)
          .flat().length;
    },
    tooFewCards(): boolean {
      return this.maindeck && this.numCards < 40;
    }
  },

  methods: {
    sortCmc() {
      store.sortByCmc({seat: store.selectedSeat, maindeck: this.maindeck});
    },
    sortColor() {
      store.sortByColor({seat: store.selectedSeat, maindeck: this.maindeck});
    },
  },
});
</script>

<style scoped>
._deck-builder-section-controls {
  display: flex;
  align-items: center;
  padding: 20px 20px 0;
}

.cardsCount {
  width: 5em;
}

.tooFewCards {
  color: #aa2222
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