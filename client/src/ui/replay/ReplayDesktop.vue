<!--

Root component for desktop layout

-->

<template>
  <div class="_replay-desktop">
    <ControlsRow />
    <div class="main">
      <PlayerSelector class="table" />
      <DraftPicker v-if="showDraftPicker" class="picker" :showDeckBuilder="true" />
      <DraftView v-else class="grid" />
      <div class="card-detail-cnt" v-if="focusedCard">
        <SimpleCardImage :card="focusedCard" class="card-detail" />
      </div>
    </div>
  </div>
</template>

<script lang="ts">
import { defineComponent } from "vue";
import ControlsRow from "./ControlsRow.vue";
import PlayerSelector from "./player_selector/PlayerSelector.vue";
import DraftPicker from "./DraftPicker.vue";
import DraftView from "./DraftView.vue";
import SimpleCardImage from "@/ui/picker/SimpleCardImage.vue";

import type { DraftCard } from "@/draft/DraftState";
import { replayFocusedCard } from "@/ui/replay/replayFocusedCard";

export default defineComponent({
  components: {
    DraftPicker,
    DraftView,
    ControlsRow,
    PlayerSelector,
    SimpleCardImage,
  },

  props: {
    showDraftPicker: {
      type: Boolean,
      required: true,
    },
  },

  computed: {
    focusedCard(): DraftCard | null {
      return replayFocusedCard.value;
    },
  },
});
</script>

<style scoped>
._replay-desktop {
  height: 100%;
  display: flex;
  flex-direction: column;
}

.main {
  display: flex;
  flex: 1;
  flex-direction: row;
  overflow: hidden;
  position: relative;
}

.table {
  width: 300px;
  flex: 0 0 auto;
}

.grid,
.picker {
  flex: 1;
}

.card-detail-cnt {
  position: absolute;
  left: 0;
  top: 0;
  width: 100%;
  height: 100%;

  display: flex;
  flex-direction: column;
  align-items: flex-end;

  pointer-events: none;
}

.card-detail {
  height: 400px;
  margin-top: 50px;
  margin-right: 50px;
}
</style>
