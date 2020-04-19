<template>
  <div>
    <DeckBuilderSection
        :columns="sideboard"
        :deckIndex="deckIndex"
        :maindeck="false"/>
    <DeckBuilderSection
        :columns="maindeck"
        :deckIndex="deckIndex"
        :maindeck="true"/>
  </div>
</template>

<script lang="ts">
  import Vue from 'vue';
  import {SelectedView} from "../../state/selection.js";
  import {DraftSeat} from "../../draft/DraftState.js";
  import {CardColumn, Deck} from "../../state/store.js";
  import DeckBuilderSection from "./DeckBuilderSection.vue";

  export default Vue.extend({
    name: 'DeckBuilder',

    components: {
      DeckBuilderSection,
    },

    computed: {
      selection(): SelectedView | null {
        return this.$tstore.state.selection;
      },

      selectedSeat(): DraftSeat | null {
        return this.selection == null || this.selection.type == 'pack' ? null :
            this.$tstore.state.draft.seats[this.selection.id];
      },

      deckIndex(): number {
        return this.selectedSeat === null ? 0 : this.selectedSeat.position;
      },

      deck(): Deck | null {
        if (this.selectedSeat === null) {
          return null;
        } else {
          if (!this.$tstore.state.decks[this.selectedSeat.position]) {
            const cards = this.selectedSeat.player.picks.cards.map(card => card.definition);
            this.$set(this.$tstore.state.decks, this.selectedSeat.position, {
              sideboard: [cards, [], [], [], [], [], []],
              maindeck: [[], [], [], [], [], [], []],
            });
          }
          return this.$tstore.state.decks[this.selectedSeat.position];
        }
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

</style>