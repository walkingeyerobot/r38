<!--

Displays a single card

-->

<template>
  <div class="_card">
    <div class="under-layer" :class="underlayerClass">
      <div class="shadow"></div>
      <div class="selection-outline" :class="outlineClass"></div>
    </div>
    <img class="card-img" :alt="renderInfo.displayName" :src="renderInfo.imageUrl" />
  </div>
</template>

<script lang="ts">
import { defineComponent } from "vue";
import type { DraftCard } from "@/draft/DraftState";
import { checkNotNil } from "@/util/checkNotNil";

export default defineComponent({
  props: {
    card: {
      type: Object as () => DraftCard,
      required: true,
    },

    selectionStyle: {
      type: String as () => "none" | "picked" | "returned",
    },
  },

  computed: {
    renderInfo(): CardRenderInfo {
      return {
        displayName: this.card.hidden ? "Unknown card" : this.card.definition.name,
        imageUrl: getImageSrc(this.card),
      };
    },

    underlayerClass() {
      switch (this.selectionStyle) {
        case "picked":
        case "returned":
          return "selected";
        default:
          return undefined;
      }
    },

    outlineClass() {
      switch (this.selectionStyle) {
        case "picked":
          return "action-picked";
        case "returned":
          return "action-returned";
        default:
          return undefined;
      }
    },
  },

  methods: {},
});

interface CardRenderInfo {
  displayName: string;
  imageUrl: string;
}

function getImageSrc(card: DraftCard): string {
  if (card.hidden) {
    // HACK HACK HACK
    return (
      `https://media.magic.wizards.com/image_legacy_migration/magic/` +
      `images/mtgcom/fcpics/making/mr224_back.jpg`
    );
  } else {
    return checkNotNil(card.definition.image_uris[0]);
  }
}
</script>

<style scoped>
._card {
  cursor: pointer;
  display: flex;
  position: relative;
}

.under-layer {
  position: absolute;
  left: 0;
  top: 0;
  right: 0;
  bottom: 0;
  z-index: -1;
}

.under-layer.selected {
  left: -4px;
  top: -4px;
  right: -4px;
  bottom: -4px;
}

/*
 * We want the shadow to fade in nicely on hover, but animating box-shadow is
 * computationally expensive. Instead, we create an extra element with a static
 * shadow and animate that element's opacity when the user hovers.
 */
.shadow {
  position: absolute;
  left: 0;
  top: 0;
  right: 0;
  bottom: 0;
  box-shadow: 0 1px 4px 1.2px rgba(0, 0, 0, 0.7);
  opacity: 0;
  transition: opacity 110ms cubic-bezier(0.33, 1, 0.68, 1);
  border-radius: 9.5px;
}

.selected > .shadow {
  border-radius: 13px;
}

._card:hover .shadow {
  opacity: 1;
}

.selection-outline {
  position: absolute;
  left: 0;
  top: 0;
  right: 0;
  bottom: 0;
  border-radius: 13px;
  display: none;
}

.action-picked {
  display: block;
  background-color: #00f;
}

.action-returned {
  display: block;
  background-color: #f00;
}

/* native is 745 × 1040 */
.card-img {
  width: 200px;
  height: 279px;
  background: #aaa;
  border-radius: 10px;
  overflow: hidden;
}
</style>
