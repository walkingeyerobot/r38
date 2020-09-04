<template>
  <div class="_deck-builder-export-menu">
    <template v-if="libLoaded && deck != null">
      <a
          :href="exportedDecksZip"
          download="r38export.zip"
          class="exportButton"
          v-if="admin"
          >
        Export all
      </a>
      <a
          :href="exportedDeck"
          download="r38export.dek"
          class="exportButton"
          >
        Export deck
      </a>
      <a
          :href="exportedBinder"
          download="r38export.dek"
          class="exportButton"
          >
        Export binder
      </a>
      <a
          @click="exportToPdf"
          download="r38export.pdf"
          class="exportButton"
          >
        Print deck
      </a>
    </template>
    <div v-else class="loading-message">
      Loading...
    </div>
  </div>
</template>

<script lang="ts">
import Vue from 'vue';
import { authStore } from '../../state/AuthStore';
import { Deck, deckBuilderStore as store } from '../../state/DeckBuilderModule';
import { ExportChunk, exportLoader } from '../../chunks/export/ExportChunk';

export default Vue.extend({
  props: {
    deckIndex: {type: Number},
  },

  created() {
    this.loadExportLib();
  },

  data() {
    return {
      libLoaded: false,
    };
  },

  computed: {
    admin(): boolean {
      return authStore.user?.id === 1;
    },

    deck(): Deck | undefined {
      return store.selectedSeat !== null ? store.decks[store.selectedSeat] : undefined;
    },

    exportedDeck(): string {
      return this.deck ? exportLoader.chunk.deckToXml(this.deck) : '';
    },

    exportedBinder(): string {
      return this.deck ? exportLoader.chunk.deckToBinderXml(this.deck) : '';
    },
  },

  asyncComputed: {
    async exportedDecksZip(): Promise<string> {
      if (this.libLoaded) {
        return await exportLoader.chunk.decksToBinderZip(
            store.decks, store.names);
      } else {
        return '';
      }
    }
  },

  methods: {
    loadExportLib() {
      exportLoader.load().then(() => {
        this.libLoaded = true;
      })
    },

    exportToPdf() {
      if (this.deck) {
        exportLoader.chunk.deckToPdf(this.deck);
      }
    },
  },
});
</script>

<style scoped>
._deck-builder-export-menu {
  background-color: #FFF;
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

.loading-message {
  padding: 10px;
  height: 40px;
}

</style>