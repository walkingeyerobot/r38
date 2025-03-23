<template>
  <div class="_shuffler">
    <Transition mode="out-in">
      <span :key="pack">{{ pack }}</span>
    </Transition>
  </div>
</template>

<script lang="ts">
import { defineComponent } from "vue";
import { fetchEndpoint } from "@/fetch/fetchEndpoint.ts";
import { routeGetCardPack } from "@/rest/api/getcardpack/getcardpack.ts";
import { useSound } from "@vueuse/sound";
import beep from "../sfx/beep.mp3";

export default defineComponent({
  name: "CubeShuffler",

  props: {
    draftId: {
      type: String,
      required: true,
    },
  },

  data() {
    return {
      pack: "SCAN NOW",
      scanListener: this.onRfidScan as EventListener,
      beep: useSound(beep),
    };
  },

  methods: {
    async onRfidScan(event: CustomEvent) {
      this.pack = "";
      const response = await fetchEndpoint(routeGetCardPack, {
        draftId: Number(this.draftId),
        cardRfid: event.detail as string,
      });
      this.pack = response.pack === 0 ? "DISCARD" : `PACK ${response.pack}`;
      this.beep.play();
    },
  },

  mounted() {
    document.body.addEventListener("rfidScan", this.scanListener);
  },

  beforeUnmount() {
    document.body.removeEventListener("rfidScan", this.scanListener);
  },
});
</script>

<style scoped>
._shuffler {
  height: 100%;
  display: flex;
  justify-content: center;
  align-items: center;
  font-size: 4em;
}

.v-enter-active {
  transition: color 2s ease-out;
}

.v-leave-active {
  transition: opacity 0.1s;
}

.v-enter-from {
  color: #01723f;
}

.v-enter-to {
  color: #000000;
}

.v-leave-to {
  opacity: 0;
}
</style>
