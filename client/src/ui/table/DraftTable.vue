<template>
  <div class="_draft-table">
    <div class="controls">
      <button @click="onStartClick">Start</button>
      <button @click="onEndClick">End</button>
      <button @click="onPrevClick">Prev</button>
      <button @click="onNextClick">Next</button>
      <input
          type="checkbox"
          id="synchronize-checkbox"
          v-model="synchronizeTimeline">
      <label for="synchronize-checkbox">Synchronize timeline</label>
    </div>
    <div class="seat-cnt">
      <draft-seat-component
          v-for="seat in draft.seats"
          :key="seat.position"
          :seat="seat"
          />
    </div>
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

    synchronizeTimeline: {
      get() {
        return this.$tstore.state.timeMode == 'synchronized';
      },

      set(value) {
        this.$tstore.commit('setTimeMode', value ? 'synchronized' : 'original');
      }
    }
  },

  methods: {
    onNextClick() {
      this.$tstore.commit('goNext');
    },

    onPrevClick() {
      this.$tstore.commit('goBack');
    },

    onStartClick() {
      this.$tstore.commit('goTo', 0);
    },

    onEndClick() {
      this.$tstore.commit('goTo', this.$tstore.state.events.length);
    },
  },

});
</script>

<style scoped>
._draft-table {
  user-select: none;
  cursor: default;
}

.seat-cnt {
  display: flex;
  flex-direction: row;
  padding: 30px;
}
</style>
