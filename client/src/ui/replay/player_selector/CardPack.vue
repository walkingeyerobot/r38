<template>
  <div class="_card-pack" :class="{ selected: isSelected }" @click.stop="onClick">
    {{ pack.id }}
  </div>
</template>

<script lang="ts">
import { defineComponent } from "vue";

import type { CardPack } from "@/draft/DraftState";
import { replayStore } from "@/state/ReplayStore";
import { pushDraftUrlRelative } from "@/router/url_manipulation";

export default defineComponent({
  props: {
    pack: {
      type: Object as () => CardPack,
      required: true,
    },
  },

  computed: {
    isSelected(): boolean {
      const selection = replayStore.selection;
      return selection != null && selection.type == "pack" && selection.id == this.pack.id;
    },
  },

  methods: {
    onClick() {
      pushDraftUrlRelative(this, {
        selection: {
          type: "pack",
          id: this.pack.id,
        },
      });
    },
  },
});
</script>

<style scoped>
._card-pack {
  width: 25px;
  height: 35px;
  background-color: #a24e30;
  border-radius: 2px;
  display: flex;
  align-items: center;
  justify-content: center;

  font-size: 14px;
  color: white;

  background-image: url("./card_back.png");
  background-size: cover;
}

.selected {
  box-shadow: 0px 0px 5px 2px #3206de;
}
</style>
