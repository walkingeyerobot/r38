<template>
  <div class="_draft-table">
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

export default Vue.extend({
  components: {
    DraftSeatComponent,
  },

  computed: {
    draft(): DraftState {
      return this.$tstore.state.draft;
    },
  },
});
</script>

<style scoped>
._draft-table {
  user-select: none;
  cursor: default;

  display: flex;
  flex-direction: column;

  border-right: 1px solid #EAEAEA;
}
</style>
