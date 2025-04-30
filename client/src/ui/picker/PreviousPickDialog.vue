<template>
  <DismissableDialog @close="$emit('close')">
    <template #header>Previous pick</template>

    <div class="card-cnt">
      <SimpleCardImage
        v-for="card in cards"
        :key="card.id"
        :card="card"
        :image-index="0"
        class="card"
      />
      <!-- <SimpleCardImage :card="cards[0]" :image-index="0" class="card" /> -->
    </div>

    <button class="undo-btn" @click="$emit('undo', cards[0])">Undo pick</button>
  </DismissableDialog>
</template>

<script setup lang="ts">
import SimpleCardImage from "@/ui/picker/SimpleCardImage.vue";
import DismissableDialog from "@/ui/picker/DismissableDialog.vue";

import type { DraftCard } from "@/draft/DraftState";

defineProps<{
  cards: DraftCard[];
}>();

defineEmits<{
  (e: "close"): void;
  (e: "undo", card: DraftCard): void;
}>();
</script>

<style scoped>
.card-cnt {
  display: flex;
  justify-content: center;
  margin: 20px;
  margin-top: 0px;
}

.card {
  max-width: 350px;
}

.card + .card {
  margin-left: 10px;
}

.undo-btn {
  padding: 20px 30px;
  margin: 50px 0 40px 0;
  width: max-content;
  align-self: center;
  font-size: 16px;
}
</style>
