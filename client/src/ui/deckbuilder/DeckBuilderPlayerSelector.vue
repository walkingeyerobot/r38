<template>
  <div class="_deckbuilder-player-selector">
    <div
        v-for="(deck, index) in state.decks"
        :key="deck.player.seatPosition"
        class="player"
        :class="{
          selected: index == state.selectedSeat
        }"
        @click="onPlayerClick(index)"
        >
      {{ deck.player.name }}
    </div>
  </div>
</template>

<script lang="ts">
import Vue from 'vue';

export default Vue.extend({
  computed: {
    state() {
      return this.$tstore.state.deckbuilder;
    },
  },

  methods: {
    onPlayerClick(index: number) {
      this.$tstore.commit('deckbuilder/setSelectedSeat', index);
    },
  },
});
</script>

<style scoped>
._deckbuilder-player-selector {
  border-right: 1px solid #EAEAEA;
}

.player {
  padding: 10px 10px;
  cursor: default;
  user-select: none;
}

.player.selected {
  background: #EAEAEA;
}
</style>
