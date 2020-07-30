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
          v-for="pack in activePacks"
          :key="pack.id"
          :pack="pack"
          class="active-pack"
          />

      <div class="spacer"></div>

      <CardPack
          v-for="pack in futurePacks"
          :key="pack.id"
          :pack="pack"
          class="future-pack"
          />
    </div>

  </div>
</template>

<script lang="ts">
import Vue from 'vue';
import { DraftSeat, CardPack as CardPackModel } from '../../draft/DraftState';
import { pushDraftUrlRelative } from '../../router/url_manipulation';

import CardPack from './CardPack.vue';

import { replayStore } from '../../state/ReplayStore';
import { draftStore } from '../../state/DraftStore';

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
      const selection = replayStore.selection;
      return selection != null
          && selection.type == 'seat'
          && selection.id == this.seat.position
    },

    activePacks(): CardPackModel[] {
      return this.seat.queuedPacks.packs.filter(
          pack => pack.round == this.seat.round);
    },

    futurePacks(): CardPackModel[] {
      return this.seat.queuedPacks.packs.filter(
          pack => pack.round > this.seat.round);
    },
  },

  methods: {
    onHeaderClick() {
      pushDraftUrlRelative(this, {
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

.active-pack {
  margin-right: 3px;
}

.future-pack {
  margin-left: 3px;
  filter: saturate(20%);
}

</style>
