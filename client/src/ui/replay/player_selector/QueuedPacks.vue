<template>
  <div class="_queued-packs">
    <CardPack v-for="pack in activePacks" :key="pack.id" :pack="pack" class="pack active-pack" />

    <CardPack v-for="pack in futurePacks" :key="pack.id" :pack="pack" class="pack future-pack" />
  </div>
</template>

<script lang="ts">
import { defineComponent } from "vue";
import CardPack from "./CardPack.vue";

import type { DraftSeat, CardPack as CardPackModel } from "@/draft/DraftState";

export default defineComponent({
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
    activePacks(): CardPackModel[] {
      return this.seat.queuedPacks.packs.filter((pack) => pack.round == this.seat.round);
    },

    futurePacks(): CardPackModel[] {
      return this.seat.queuedPacks.packs.filter((pack) => pack.round > this.seat.round);
    },
  },
});
</script>

<style scoped>
._queued-packs {
  display: flex;
  flex-direction: row;
}

.pack + .pack {
  margin-left: 3px;
}

.future-pack {
  filter: saturate(20%);
}
</style>
