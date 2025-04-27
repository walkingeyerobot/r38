<template>
  <div class="_simple-card-image">
    <img
      class="card-img"
      :src="rewriteImgUri(card.definition.image_uris[imageIndex ?? 0])"
      :title="card.definition.name"
    />
  </div>
</template>

<script setup lang="ts">
import type { DraftCard } from "@/draft/DraftState";

defineProps<{
  card: DraftCard;
  imageIndex?: number;
}>();

function rewriteImgUri(uri: string) {
  return uri.replace("https://img.scryfall.com/cards/", "https://cards.scryfall.io/");
}
</script>

<style scoped>
._simple-card-image {
  display: flex;
  align-items: center;
  justify-content: center;
  box-sizing: border-box;

  /*
   * This is extremely load-bearing for reasons I can't really explain, see
   * below
   */
  min-width: 0;
}

.card-img {
  border-radius: 4.5% / 3.2%;
  box-shadow: rgba(0, 0, 0, 0.5) 0px 2px 6px;

  /*
   * These, plus the min-width in the above rule, are the method by which we
   * ensure that clients can set any form of width, height, or lack thereof
   * on the root element and things will Just Work properly.
   *
   * (a) If only one dimension is set, or if the component is placed within a
   *     flexible container like a flexbox, the component will resize to fit
   *     given all appropriate constraints, preserving the aspect ratio of the
   *     image.
   * (b) If both dimensions are specified or constrained, the component will
   *     distort to fill the specified space but the image itself will retain
   *     its original aspect ratio, remaining centered (letterboxed or pillar
   *     boxed) within the component.
   *
   * Why does this work? Oh geez I dunno man.
   */
  min-width: 0;
  max-height: 100%;

  aspect-ratio: 2.5 / 3.5 auto;

  background-color: #000;
}
</style>
