<template>
  <div class="_tagwriter">
    <img
      :src="index >= cards.length ? '' : cards[index].data.image_uris?.[0]"
      :alt="index >= cards.length ? '' : cards[index].data.scryfall.name"
      @click="nextCard"
    />
  </div>
</template>

<script setup lang="ts">
import { onMounted, onUnmounted, ref } from "vue";
import { fetchEndpoint } from "@/fetch/fetchEndpoint.ts";
import { ROUTE_SET } from "@/rest/api/set/set.ts";
import type { SourceSet } from "@/parse/SourceData.ts";
import { RfidHandler } from "@/rfid/RfidHandler.ts";

const props = defineProps({
  set: {
    type: String,
    required: true,
  },
});

const cards = ref<SourceSet>([]);
const index = ref(0);
const rfidHandler = new RfidHandler(() => {});

function nextCard() {
  index.value++;
  setCard(cards.value[index.value].id);
}

function setCard(card: string | null) {
  rfidHandler.writeTag(card);
}

onMounted(async () => {
  await rfidHandler.start(false);
  cards.value = await fetchEndpoint(ROUTE_SET, { set: props.set });
  setCard(cards.value[0].id);
});

onUnmounted(() => {
  setCard(null);
  rfidHandler.stop();
});
</script>

<style scoped>
._tagwriter img {
  padding: 20px;
  box-sizing: border-box;
  width: 100%;
}
</style>
