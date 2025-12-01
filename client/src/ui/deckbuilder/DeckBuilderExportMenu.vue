<template>
  <div class="_deck-builder-export-menu">
    <template v-if="libLoaded && deck != null">
      <a
        v-if="admin && exportedDecksZip"
        :href="exportedDecksZip"
        :download="exportedDecksFilename"
        class="exportButton"
      >
        Export all
      </a>
      <a :href="exportedDeck" download="r38export.txt" class="exportButton"> Export deck </a>
      <a :href="exportedBinder" download="r38export.dek" class="exportButton"> Export binder </a>
      <a @click="exportToPdf" download="r38export.pdf" class="exportButton"> Print deck </a>
      <div class="exportButton">
        <label for="scale">Scale:</label>
        <input class="scaleInput" name="scale" v-model="scale" type="number" step=".01" />
      </div>
    </template>
    <div v-else class="loading-message">Loading...</div>
  </div>
</template>

<script lang="ts">
import { defineComponent } from "vue";
import { authStore } from "@/state/AuthStore";
import { type Deck, deckBuilderStore as store } from "@/state/DeckBuilderModule";
import { exportLoader } from "@/chunks/export/ExportChunk";
import { draftStore } from "@/state/DraftStore";

export default defineComponent({
  props: {
    deckIndex: { type: Number },
  },

  created() {
    this.loadExportLib();
  },

  data() {
    return {
      libLoaded: false,
      exportedDecksZip: null as string | null,
      scale: 0.96,
    };
  },

  computed: {
    admin(): boolean {
      return authStore.userId === 1;
    },

    deck(): Deck | undefined {
      return store.selectedSeat !== null ? store.decks[store.selectedSeat] : undefined;
    },

    exportedDeck(): string {
      return this.deck ? exportLoader.chunk.deckToTxt(this.deck) : "";
    },

    exportedBinder(): string {
      return this.deck ? exportLoader.chunk.deckToBinderXml(this.deck) : "";
    },

    exportedDecksFilename(): string {
      return `${draftStore.draftName} decks.zip`;
    },

    zipDeps() {
      return [this.libLoaded, store.decks, store.names, store.mtgoNames] as const;
    },
  },

  watch: {
    async libLoaded(_oldVal, libLoaded) {
      if (libLoaded) {
        this.exportedDecksZip = await exportLoader.chunk.decksToBinderZip(
          store.decks,
          store.names,
          store.mtgoNames,
        );
      }
    },
  },

  methods: {
    loadExportLib() {
      exportLoader.load().then(() => {
        this.libLoaded = true;
      });
    },

    exportToPdf() {
      if (this.deck) {
        exportLoader.chunk.deckToPdf(this.deck, this.scale);
      }
    },
  },
});
</script>

<style scoped>
._deck-builder-export-menu {
  background-color: #fff;
  border-radius: 5px;
  box-shadow: 0px 1px 4px rgba(0, 0, 0, 0.3);
  border: 1px solid #ccc;
  min-width: 150px;
}

.exportButton {
  display: block;
  padding: 10px;
  color: inherit;
  text-decoration: none;
  white-space: nowrap;
  cursor: pointer;
}

.exportButton:hover {
  background: #ddd;
}

.scaleInput {
  margin-left: 4px;
}

.loading-message {
  padding: 10px;
  height: 40px;
}
</style>
