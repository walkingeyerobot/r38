<template>
  <div
      class="_draft-seat"
      :class="{ selected: isSelected, }"
      @click="onHeaderClick"
      @mouseleave="onQueuedPacksUnhover"
      >
    <img class="icon" :src="seat.player.iconUrl" >

    <div class="player-name">
      {{ seat.player.name }}
    </div>

    <div
        v-if="packCount > 0"
        class="pack-count"
        @mouseenter="onPackCountHover"
        >
      <img
          class="pack-icon"
          :class="{ 'no-packs': activePackCount == 0 }"
          src="./card_back.png"
          >
      <div v-if="activePackCount > 0" class="pack-count-label">
        {{ activePackCount }}
      </div>

      <QueuedPacks
          v-if="showPacks"
          :seat="seat"
          class="queued-packs"
          />
    </div>

  </div>
</template>

<script lang="ts">
import Vue from 'vue';
import QueuedPacks from './QueuedPacks.vue';

import { replayStore } from '../../../state/ReplayStore';

import { DraftSeat, CardPack as CardPackModel } from '../../../draft/DraftState';
import { pushDraftUrlRelative } from '../../../router/url_manipulation';


export default Vue.extend({
  components: {
    QueuedPacks,
  },

  props: {
    seat: {
      type: Object as () => DraftSeat,
      required: true,
    },
  },

  data() {
    return {
      showPacks: false,
    };
  },

  computed: {
    isSelected(): boolean {
      const selection = replayStore.selection;
      return selection != null
          && selection.type == 'seat'
          && selection.id == this.seat.position
    },

    packCount(): number {
      return this.seat.queuedPacks.packs.length;
    },

    activePackCount(): number {
      return this.seat.queuedPacks.packs.filter(
          pack => pack.round == this.seat.round).length;
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

    onPackCountHover() {
      this.showPacks = true;
    },

    onQueuedPacksUnhover() {
      this.showPacks = false;
    }
  },

});
</script>

<style scoped>
._draft-seat {
  display: flex;
  flex-direction: row;
  align-items: center;
}

._draft-seat.selected {
  background: #EAEAEA;
}

.icon {
  width: 40px;
  height: 40px;
  border-radius: 40px;
  margin: 10px;
  margin-right: 11px;
}

.player-name {
  flex: 1;
}

.pack-count {
  display: flex;
  position: relative;
  width: 46px;
  height: 100%;
  box-sizing: border-box;
  padding: 4px 6px;
  flex-direction: row;
  align-items: center;
  font-size: 14px;
}

.pack-count-label {
  margin-left: 5px;
  color: #555;
}

.pack-icon {
  height: 24px;
}

.pack-icon.no-packs {
  filter: saturate(0%) brightness(1.8);
}

.queued-packs {
  position: absolute;
  right: 7px;
  top: 7px;

  padding: 4px 4px;
  background: white;
  border: 1px solid #cacaca;
  border-radius: 4px;
  box-shadow: 0px 2px 4px rgba(0, 0, 0, 0.15);
}

</style>
