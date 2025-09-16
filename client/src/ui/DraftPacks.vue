<template>
  <div class="pack" v-for="(pack, index) in packs" :key="index">
    <SimpleCardImage
      v-for="card in pack"
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
import { parseCard } from "@/parse/parseInitialState.ts";
import type { DraftCard } from "@/draft/DraftState.ts";
import { ROUTE_DRAFT_PACKS } from "@/rest/api/draftpacks/draftpacks.ts";

const props = defineProps({
  id: {
    type: String,
    required: true,
  },
});

const packs = ref<DraftCard[][]>([]);

onMounted(async () => {
  packs.value = (await fetchEndpoint(ROUTE_DRAFT_PACKS, { id: props.id })).map((pack) =>
    pack.map((card) => parseCard(card, 0)),
  );
});
</script>

<style scoped>
.pack {
  display: flex;
  flex-wrap: wrap;
  margin: 30px 20px;
  padding: 5px;
  border: 1px solid #999;
  background: #ccc;
  gap: 10px;
}

.card {
  max-width: 200px;
}
</style>
