<!--

Root component for desktop layout

-->

<template>
  <div class="_replay-desktop">
    <ControlsRow />
    <div class="main">
      <PlayerSelector class="table" />
      <DraftPicker
          v-if="showDraftPicker"
          class="picker"
          :showDeckBuilder="true"
          :inPersonDraft="inPersonDraft"
          />
      <DraftView v-else class="grid" />
    </div>
  </div>
</template>

<script lang="ts">
import { defineComponent } from 'vue'
import ControlsRow from './ControlsRow.vue';
import PlayerSelector from './player_selector/PlayerSelector.vue';
import DraftPicker from './DraftPicker.vue';
import DraftView from './DraftView.vue';

export default defineComponent({
  components: {
    DraftPicker,
    DraftView,
    ControlsRow,
    PlayerSelector,
  },

  props: {
    showDraftPicker: {
      type: Boolean,
      required: true,
    },
    inPersonDraft: {
      type: Boolean,
      required: true,
    },
  },
})

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
}

.table {
  width: 300px;
  flex: 0 0 auto;
}

.grid, .picker {
  flex: 1;
}
</style>

