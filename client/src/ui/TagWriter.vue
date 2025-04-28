<template>
  <div class="_tagwriter">
    <img
      :src="index >= cards.length ? '' : cards[index].data.image_uris?.[0]"
      :alt="index >= cards.length ? '' : cards[index].data.scryfall.name"
      @click="nextCard"
    />
  </div>
</template>

<script lang="ts">
import { defineComponent } from "vue";
import { fetchEndpoint } from "@/fetch/fetchEndpoint.ts";
import { ROUTE_SET } from "@/rest/api/set/set.ts";
import type { SourceSet } from "@/parse/SourceData.ts";

export default defineComponent({
  name: "TagWriter",

  props: {
    set: {
      type: String,
      required: true,
    },
  },

  data() {
    return {
      cards: [] as SourceSet,
      index: 0,
    };
  },

  methods: {
    nextCard() {
      this.index++;
      this.setCard(this.cards[this.index].id);
    },

    setCard(card: string | null) {
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      if ((window as any).kmpJsBridge) {
        // eslint-disable-next-line @typescript-eslint/no-explicit-any
        (window as any).kmpJsBridge.callNative(
          "setCard",
          card ? `{card: "${card}"}` : "{card: null}",
        );
      }
    },
  },

  async mounted() {
    this.cards = await fetchEndpoint(ROUTE_SET, { set: this.set });
    this.setCard(this.cards[0].id);
  },

  beforeUnmount() {
    this.setCard(null);
  },
});
</script>

<style scoped>
._tagwriter img {
  padding: 20px;
  box-sizing: border-box;
  width: 100%;
}
</style>
