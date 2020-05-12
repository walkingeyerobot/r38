<template>
  <div
      class="_draft-seat"
      :class="{ selected: isSelected, }"
      @click="onHeaderClick"
      >
    <div
        :class="{ selected: isSelected, }"
        class="header"
        >
      <div class="player-label">
        {{ seat.player.name }}
      </div>
    </div>

    <div class="card-cnt">
      <CardPack
          v-for="pack in seat.queuedPacks.packs"
          :key="pack.id"
          :pack="pack"
          class="opened-pack"
          />

      <div class="spacer"></div>

      <CardPack
          v-for="pack in seat.unopenedPacks.packs"
          :key="pack.id"
          :pack="pack"
          class="unopened-pack"
          />
    </div>

  </div>
</template>

<script lang="ts">
import Vue from 'vue';
import { DraftSeat } from '../../draft/DraftState';
import { navTo } from '../../router/url_manipulation';

import CardPack from './CardPack.vue';

import { replayStore as store } from '../../state/ReplayModule';


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
      const selection = store.selection;
      return selection != null
          && selection.type == 'seat'
          && selection.id == this.seat.position
    }
  },

  methods: {
    onHeaderClick() {
      navTo(store, this.$route, this.$router, {
        selection: {
          type: 'seat',
          id: this.seat.position,
        },
      });
    },
  },

});
</script>

<style scoped>
._draft-seat {
  height: 70px;
  padding: 10px 10px 0 10px;
}

._draft-seat.selected {
  background: #EAEAEA;
}

.header {
  margin-bottom: 6px;
}

.card-cnt {
  display: flex;
  flex-direction: row;
}

.spacer {
  flex: 1
}

.opened-pack {
  margin-right: 3px;
}

.unopened-pack {
  margin-left: 3px;
  filter: saturate(20%);
}

</style>
