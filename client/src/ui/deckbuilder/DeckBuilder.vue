<template>
  <div>
    <DeckBuilderSection
        class="maindeck"
        :columns="sideboard"
        :deckIndex="state.selectedSeat"
        :maindeck="false"
        />
    <DeckBuilderSection
        :columns="maindeck"
        :deckIndex="state.selectedSeat"
        :maindeck="true"
        />
  </div>
</template>

<script lang="ts">
  import Vue from 'vue';
  import { SelectedView } from "../../state/selection.js";
  import { DraftSeat } from "../../draft/DraftState.js";
  import DeckBuilderSection from "./DeckBuilderSection.vue";
  import { DeckBuilderState, CardColumn, Deck } from '../../state/DeckBuilderModule.js';

  export default Vue.extend({
    name: 'DeckBuilder',

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
.maindeck {
  border-bottom: 1px solid #EAEAEA;
}
</style>
