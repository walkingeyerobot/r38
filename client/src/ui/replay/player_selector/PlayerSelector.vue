<template>
<div class="_player-selector">
  <PlaybackControls class="playback-controls" />

  <draft-seat-component
      v-for="seat in draft.seats"
      :key="seat.position"
      :seat="seat"
      />
</div>
</template>

<script lang="ts">
import { defineComponent } from 'vue';
import DraftSeatComponent from './DraftSeat.vue'
import PlaybackControls from './PlaybackControls.vue';

import { replayStore } from '../../../state/ReplayStore';
import { draftStore } from '../../../state/DraftStore';

import { DraftState } from '../../../draft/DraftState';


export default defineComponent({
  components: {
    DraftSeatComponent,
    PlaybackControls,
  },

  computed: {
    draft(): DraftState {
      return replayStore.draft;
    },
  },
});
</script>

<style scoped>
._player-selector {
  user-select: none;
  cursor: default;

  display: flex;
  flex-direction: column;
  overflow-y: auto;

  border-right: 1px solid #EAEAEA;
}

.playback-controls {
  padding: 10px 5px;
}
</style>
