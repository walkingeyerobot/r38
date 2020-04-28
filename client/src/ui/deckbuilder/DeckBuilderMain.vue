<template>
  <div class="_deck-builder-main">
    <DeckBuilderSection
        class="sideboard"
        :columns="sideboard"
        :deckIndex="state.selectedSeat"
        :maindeck="false"
        />
    <DeckBuilderSection
        class="maindeck"
        :columns="maindeck"
        :deckIndex="state.selectedSeat"
        :maindeck="true"
        />
  </div>
</template>

<script lang="ts">
import Vue from 'vue';
import DeckBuilderSection from "./DeckBuilderSection.vue";
import { CardColumn, Deck, DeckBuilderState } from '../../state/DeckBuilderModule.js';

export default Vue.extend({
  components: {
    DeckBuilderSection,
  },

  computed: {
    state(): DeckBuilderState {
      return this.$tstore.state.deckbuilder;
    },

    deck(): Deck | undefined {
      return this.state.decks[this.state.selectedSeat];
    },

    sideboard(): CardColumn[] {
      return this.deck ? this.deck.sideboard : [];
    },

    maindeck(): CardColumn[] {
      return this.deck ? this.deck.maindeck : [];
    },
  },

  methods: {},
});
</script>

<style scoped>
._deck-builder-main {
  display: flex;
  flex-direction: column;
  overflow-x: scroll;
}

.maindeck {
  flex: 3 0 0;
  border-bottom: 1px solid #EAEAEA;
}

.sideboard {
  flex: 2 0 0;
}
</style>
