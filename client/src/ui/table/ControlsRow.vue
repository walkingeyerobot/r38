<template>
  <div class="_controls-row">
    <div class="start"></div>
    <div class="center">
      <button @click="onStartClick" class="playback-btn">« Start</button>
      <button @click="onPrevClick" class="playback-btn">‹ Prev</button>
      <button @click="onNextClick" class="playback-btn">Next ›</button>
      <button @click="onEndClick" class="playback-btn">End »</button>
    </div>
    <div class="end">
      <input
          type="checkbox"
          id="synchronize-checkbox"
          v-model="synchronizeTimeline">
      <label
          for="synchronize-checkbox"
          class="synchronize-label"
          >
        Synchronize timeline
      </label>
    </div>
  </div>
</template>

<script lang="ts">
import Vue from 'vue'
export default Vue.extend({
  computed: {
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
._controls-row {
  display: flex;
  flex-direction: row;

  padding: 10px;
  border-bottom: 1px solid #EAEAEA;
}

.start {
  flex: 1 0 0;
}

.center {
  flex: 1 0 0;
  text-align: center;
}

.end {
  flex: 1 0 0;
  text-align: end;
  display: flex;
  align-items: center;
  justify-content: flex-end;
}

.playback-btn {
  margin: 0 5px;
  width: 70px;
  height: 30px;
  border-radius: 3px;
}

.synchronize-label {
  margin-left: 4px;
}
</style>
