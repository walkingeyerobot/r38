<template>
  <div class="_deckbuilder-player-selector">
    <div
        v-for="(name, index) in state.names"
        :key="index"
        class="player"
        :class="{
          selected: index === state.selectedSeat
        }"
        @click="onPlayerClick(index)"
        >
      {{ name }}
    </div>
  </div>
</template>

<script lang="ts">
import { defineComponent } from 'vue';
import { deckBuilderStore as store, DeckBuilderStore } from '../../state/DeckBuilderModule';

export default defineComponent({
  computed: {
    state(): DeckBuilderStore {
      return store;
    },
  },

  methods: {
    onPlayerClick(index: number) {
      store.setSelectedSeat(index);
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
