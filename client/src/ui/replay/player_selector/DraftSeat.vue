<template>
  <div class="_draft-seat" :class="{ selected: isSelected }" @click="onHeaderClick">
    <img class="icon" :src="seat.player.iconUrl" />

    <div class="name-cnt">
      <div class="player-name">
        {{ seat.player.name }}
      </div>

      <div class="mana-counts">
        <ManaSymbol
          v-for="colorWeight in colorWeights"
          :key="colorWeight.color"
          :code="colorWeight.color"
          class="mana-symbol"
        >
        </ManaSymbol>
      </div>
    </div>

    <div
      v-if="packCount > 0"
      class="pack-count"
      @mouseenter="onPackCountHover"
      @mouseleave="onQueuedPacksUnhover"
    >
      <img class="pack-icon" :class="{ 'no-packs': activePackCount == 0 }" src="./card_back.png" />
      <div v-if="activePackCount > 0" class="pack-count-label">
        {{ activePackCount }}
      </div>

      <QueuedPacks v-if="showPacks" :seat="seat" class="queued-packs" />
    </div>
  </div>
</template>

<script lang="ts">
import { defineComponent } from "vue";

import ManaSymbol from "../../shared/mana/ManaSymbol.vue";
import QueuedPacks from "./QueuedPacks.vue";

import { replayStore } from "@/state/ReplayStore";
import type { DraftSeat } from "@/draft/DraftState";
import { pushDraftUrlRelative } from "@/router/url_manipulation";
import type { ScryfallColor } from "@/draft/scryfall";

export default defineComponent({
  components: {
    ManaSymbol,
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
      return selection != null && selection.type == "seat" && selection.id == this.seat.position;
    },

    packCount(): number {
      return this.seat.queuedPacks.packs.length;
    },

    activePackCount(): number {
      return this.seat.queuedPacks.packs.filter((pack) => pack.round == this.seat.round).length;
    },

    colorWeights(): ColorWeight[] {
      const totalPicks = this.seat.picks.count;

      return [
        {
          color: "W" as const,
          defaultIndex: 0,
          weight: this.generateColorWeight(this.seat.colorCounts.w, totalPicks),
        },
        {
          color: "U" as const,
          defaultIndex: 1,
          weight: this.generateColorWeight(this.seat.colorCounts.u, totalPicks),
        },
        {
          color: "B" as const,
          defaultIndex: 2,
          weight: this.generateColorWeight(this.seat.colorCounts.b, totalPicks),
        },
        {
          color: "R" as const,
          defaultIndex: 3,
          weight: this.generateColorWeight(this.seat.colorCounts.r, totalPicks),
        },
        {
          color: "G" as const,
          defaultIndex: 4,
          weight: this.generateColorWeight(this.seat.colorCounts.g, totalPicks),
        },
      ]
        .filter((colorWeight) => colorWeight.weight > 0)
        .sort((a, b) => {
          let cmp = b.weight - a.weight;
          if (cmp == 0) {
            cmp = a.defaultIndex - b.defaultIndex;
          }
          return cmp;
        });
    },
  },

  methods: {
    onHeaderClick() {
      pushDraftUrlRelative(this, {
        selection: {
          type: "seat",
          id: this.seat.position,
        },
      });
    },

    onPackCountHover() {
      this.showPacks = true;
    },

    onQueuedPacksUnhover() {
      this.showPacks = false;
    },

    generateColorWeight(pickCount: number, totalPickCount: number) {
      if (pickCount < 2 || pickCount / totalPickCount < 0.16) {
        return 0;
      } else {
        return pickCount;
      }
    },
  },
});

interface ColorWeight {
  color: ScryfallColor;
  defaultIndex: number;
  weight: number;
}
</script>

<style scoped>
._draft-seat {
  display: flex;
  flex-direction: row;
  align-items: center;
  border: 1px solid transparent;
  margin-left: 4px;
  margin-right: 4px;
}

._draft-seat.selected {
  border: 1px solid #f5c747;
  border-radius: 9px;
}

.icon {
  width: 40px;
  height: 40px;
  border-radius: 40px;
  margin: 10px;
  margin-right: 9px;
  margin-left: 7px;
}

.name-cnt {
  flex: 1;
}

.player-name {
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  flex: 1;
}

.mana-counts {
  display: flex;
  align-items: center;
  justify-content: start;
  position: relative;
  height: 14px;
  margin-left: 0.5px;
  margin-top: 2px;
  padding-bottom: 1px;
}

.mana-symbol + .mana-symbol {
  margin-left: 3px;
}

.pack-count {
  display: flex;
  position: relative;
  width: 46px;
  height: 100%;
  box-sizing: border-box;
  padding: 4px 6px;
  margin-left: 6px;
  flex-direction: row;
  align-items: center;
  font-size: 14px;
  flex-shrink: 0;
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
