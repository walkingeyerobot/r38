<template>
  <div
    class="_deck-builder-screen"
    @mousedown.capture="onCaptureMouseDown"
    @mousedown="onBubbleMouseDown"
  >
    <div class="main" v-if="status == 'loaded'">
      <DeckBuilderPlayerSelector class="player-selector" />
      <DeckBuilderMain class="deckbuilder" :horizontal="false" />
    </div>
  </div>
</template>

<script lang="ts">
import { defineComponent } from "vue";

import DeckBuilderMain from "./deckbuilder/DeckBuilderMain.vue";
import DeckBuilderPlayerSelector from "./deckbuilder/DeckBuilderPlayerSelector.vue";

import { authStore } from "@/state/AuthStore";
import { draftStore } from "@/state/DraftStore";
import { deckBuilderStore as deckStore } from "@/state/DeckBuilderModule";

import { fetchEndpoint } from "@/fetch/fetchEndpoint";
import { routeDraft } from "@/rest/api/draft/draft";
import type { FetchStatus } from "./infra/FetchStatus";
import { globalClickTracker } from "./infra/globalClickTracker";

export default defineComponent({
  components: {
    DeckBuilderMain,
    DeckBuilderPlayerSelector,
  },

  data() {
    return {
      status: "missing" as FetchStatus,
    };
  },

  created() {
    const draftId = parseInt(this.$route.params["draftId"] as string);
    this.fetchDraft(draftId);
  },

  methods: {
    async fetchDraft(draftId: number) {
      const payload = await fetchEndpoint(routeDraft, {
        id: draftId.toString(),
        as: authStore.user?.id,
      });
      this.status = "loaded";

      // TODO: Handle fetch error

      draftStore.loadDraft(payload);
      document.title = `${draftStore.draftName}`;

      const state = draftStore.currentState;
      deckStore.sync(state);
    },

    onCaptureMouseDown() {
      globalClickTracker.onCaptureGlobalMouseDown();
    },

    onBubbleMouseDown(e: MouseEvent) {
      globalClickTracker.onBubbleGlobalMouseDown(e);
    },
  },
});
</script>

<style scoped>
._deck-builder-screen {
  height: 100%;
  display: flex;
  flex-direction: column;
}

.main {
  display: flex;
  flex: 1;
  flex-direction: row;
  overflow: hidden;
}

.player-selector {
  width: 200px;
  flex: 0 0 auto;
}

.deckbuilder {
  flex: 1;
}
</style>
