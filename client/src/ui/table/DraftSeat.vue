<template>
  <div
      class="_draft-seat"
      >
    <div
        :class="{ selected: isSelected, }"
        @click="onHeaderClick"
        >
      <div class="player-label">
        {{ seat.player.name }}
      </div>
    </div>

    <div class="unopened-packs">
      <CardPack
          v-for="pack in seat.unopenedPacks"
          :key="pack.id"
          :pack="pack"
          />
    </div>

    <div class="active-packs">
      <CardPack
          v-for="pack in seat.queuedPacks"
          :key="pack.id"
          :pack="pack"
          />
    </div>
  </div>
</template>

<script lang="ts">
import Vue from 'vue';
import { DraftSeat } from '../../draft/draft_types';

import CardPack from './CardPack.vue';

export default Vue.extend({
  components: {
    CardPack,
  },

  props: {
    seat: {
      type: Object as () => DraftSeat,
      required: true,
    },
  },

  computed: {
    isSelected(): boolean {
      const selection = this.$tstore.state.selection;
      return selection != null
          && selection.type == 'seat'
          && selection.id == this.seat.position
    }
  },

  methods: {
    onHeaderClick() {
      this.$tstore.commit('setSelection', {
        type: 'seat',
        id: this.seat.position,
      });
    },
  },

});
</script>

<style scoped>
._draft-seat {
  width: 0;
  flex: 1;
}

.selected {
  font-weight: bold;
}

.unopened-packs {
  margin-top: 10px;
}

.active-packs {
  min-height: 80px;
  margin-top: 10px;
}

</style>
