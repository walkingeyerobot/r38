<template>
  <template v-if="prompt == 'request-permission'">
    <div>
      <button class="boop-btn" @click="onRequestNfcPermission">Enable booping</button>
    </div>
  </template>
  <div class="_shuffler">
    <Transition mode="out-in">
      <span :key="pack">{{ pack }}</span>
    </Transition>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from "vue";
import { fetchEndpoint } from "@/fetch/fetchEndpoint.ts";
import { ROUTE_GET_CARD_PACK } from "@/rest/api/getcardpack/getcardpack.ts";
import { useSound } from "@vueuse/sound";
import beep from "../sfx/beep.mp3";
import { type BoopPrompt, RfidHandler } from "@/rfid/RfidHandler.ts";

const props = defineProps({
  draftId: {
    type: String,
    required: true,
  },
});

const pack = ref("SCAN NOW");
const beepRef = ref(useSound(beep));
const rfidHandler = new RfidHandler(onRfidScan);

const prompt = computed<BoopPrompt>(() => {
  return rfidHandler.getPrompt();
});

function onRequestNfcPermission() {
  rfidHandler.scanForTag();
}

async function onRfidScan(cardRfid: string) {
  pack.value = "";
  const response = await fetchEndpoint(ROUTE_GET_CARD_PACK, {
    draftId: Number(props.draftId),
    cardRfid,
  });
  pack.value = response.pack === 0 ? "DISCARD" : `PACK ${response.pack}`;
  beepRef.value.play();
}

onMounted(() => {
  rfidHandler.start();
});

onUnmounted(() => {
  rfidHandler.stop();
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
