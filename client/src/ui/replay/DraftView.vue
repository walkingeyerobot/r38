<template>
  <div class="_draft-view">
    <div>
      <button
        class="tab"
        @click="selectTab(0)"
        :class="{ selected: selectedTab === 0 }"
        v-if="selectedSeat !== null"
      >
        History
      </button>
      <button
        class="tab"
        @click="selectTab(1)"
        :class="{ selected: selectedTab === 1 }"
        v-if="selectedSeat !== null"
      >
        Deck
      </button>
    </div>
    <CardGrid v-if="selectedTab === 0 || selectedSeat === null" class="tab-content" />
    <DeckBuilderMain
      v-if="selectedTab === 1 && selectedSeat !== null"
      :horizontal="false"
      class="tab-content"
    />
  </div>
</template>

<script lang="ts">
import { defineComponent } from "vue";
import DeckBuilderMain from "../deckbuilder/DeckBuilderMain.vue";
import CardGrid from "./CardGrid.vue";

import { replayStore } from "@/state/ReplayStore";

export default defineComponent({
  components: {
    CardGrid,
    DeckBuilderMain,
  },

  data: () => ({
    selectedTab: 0,
  }),

  computed: {
    selectedSeat(): number | null {
      return replayStore.selection?.type === "seat" ? replayStore.selection.id : null;
    },
  },

  methods: {
    selectTab(tab: number) {
      this.selectedTab = tab;
    },
  },
});
</script>

<style scoped>
._draft-view {
  display: flex;
  flex-direction: column;
  min-width: 0;
}

.tab {
  border: 1px solid #ccc;
  background: white;
  padding: 10px;
  outline: none;
  cursor: pointer;
  z-index: 2;
  position: relative;
  border-top-left-radius: 5px;
  border-top-right-radius: 5px;
  box-shadow: inset 0 -4px 3px rgba(128, 128, 128, 0.1);
}

.tab.selected {
  border-bottom-color: white;
  box-shadow: none;
}

.tab-content {
  border-top: 1px solid #ccc;
  margin-top: -1px;
}
</style>
