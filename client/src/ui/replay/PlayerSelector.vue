<template>
  <div class="_player-selector">
    <draft-seat-component
        v-for="seat in draft.seats"
        :key="seat.position"
        :seat="seat"
        />
  </div>
</template>

<script lang="ts">
import Vue from "vue";
import DraftSeatComponent from "./DraftSeat.vue"
import { DraftState } from "../../draft/DraftState";

import { replayStore as store } from '../../state/ReplayModule';


export default Vue.extend({
  components: {
    DraftSeatComponent,
  },

  computed: {
    draft(): DraftState {
      console.log('Reading draft');
      return store.draft;
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

  border-right: 1px solid #EAEAEA;
}
</style>
