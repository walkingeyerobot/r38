<template>
  <header class="header">
    <a class="button" :href="`/samplepack/${set}`"> New pack </a>
  </header>
  <div class="pack">
    <SimpleCardImage
      v-for="card in cards"
      :key="card.id"
      :card="card"
      :image-index="0"
      class="card"
    />
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from "vue";
import { fetchEndpoint } from "@/fetch/fetchEndpoint.ts";
import SimpleCardImage from "@/ui/picker/SimpleCardImage.vue";
import { ROUTE_SAMPLE_PACK } from "@/rest/api/samplepack/samplepack.ts";
import { parseCard } from "@/parse/parseInitialState.ts";
import type { DraftCard } from "@/draft/DraftState.ts";

const props = defineProps({
  set: {
    type: String,
    required: true,
  },
  seed: {
    type: Number,
    required: true,
  },
});

const cards = ref<DraftCard[]>([]);

onMounted(async () => {
  cards.value = (await fetchEndpoint(ROUTE_SAMPLE_PACK, { set: props.set, seed: props.seed })).map(
    (card) => parseCard(card, 0),
  );
});
</script>

<style scoped>
.pack {
  display: flex;
  flex-wrap: wrap;
  margin: 10px;
  gap: 10px;
}

.card {
  max-width: 200px;
}

.header {
  display: flex;
  justify-content: end;
  padding: 20px;
}

.button {
  flex-basis: content;
  display: flex;
  height: 34px;
  flex-direction: row;
  align-items: center;
  margin-right: 30px;

  font-size: 14px;
  font-weight: bold;
  color: white;
  background: #7187dd;
  border-radius: 4px;
  text-decoration: none;
  padding: 0 20px;
}
</style>
