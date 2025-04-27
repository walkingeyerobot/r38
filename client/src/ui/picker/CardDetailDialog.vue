<template>
  <DismissableDialog @close="$emit('close')">
    <template #header>
      {{ card.definition.name }}
    </template>
    <div class="card-cnt">
      <SimpleCardImage
        v-for="(imageUri, index) in card.definition.image_uris"
        :key="imageUri"
        :card="card"
        :image-index="index"
        class="card"
      />
      <!-- <SimpleCardImage :card="card" class="card" /> -->
    </div>
    <button class="pick-btn" @click="$emit('pick', card)">Pick this card</button>
  </DismissableDialog>
</template>

<script setup lang="ts">
import SimpleCardImage from "./SimpleCardImage.vue";
import DismissableDialog from "@/ui/picker/DismissableDialog.vue";

import type { DraftCard } from "@/draft/DraftState";

defineProps<{
  card: DraftCard;
}>();

defineEmits<{
  (e: "close"): void;
  (e: "pick", card: DraftCard): void;
}>();
</script>

<style scoped>
.card-cnt {
  display: flex;
  flex-direction: row;
  align-items: center;
  justify-content: center;

  flex-wrap: wrap;
}

.card {
  margin: 0 10px;
  max-width: 350px;
}
.card + .card {
  margin-top: 30px;
}

.pick-btn {
  padding: 20px 30px;
  margin: 50px 0 40px 0;
  width: max-content;
  align-self: center;
  font-size: 16px;
}
</style>
