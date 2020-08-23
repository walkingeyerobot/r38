<template>
  <div class="_deck-builder-export-menu">
    <a
        :href="exportedDecksZip"
        download="r38export.zip"
        class="exportButton"
        v-if="admin && deck"
        >
      Export all
    </a>
    <a
        :href="exportedDeck"
        download="r38export.dek"
        class="exportButton"
        v-if="deck"
        >
      Export deck
    </a>
    <a
        :href="exportedBinder"
        download="r38export.dek"
        class="exportButton"
        v-if="deck"
        >
      Export binder
    </a>
    <a
        @click="exportToPdf"
        download="r38export.pdf"
        class="exportButton"
        v-if="deck"
        >
      Print deck
    </a>
  </div>
</template>

<script lang="ts">
import Vue from 'vue';
import { authStore } from '../../state/AuthStore';
import { Deck, deckBuilderStore as store } from '../../state/DeckBuilderModule';
import { decksToBinderZip, deckToBinderXml, deckToPdf, deckToXml } from '../../draft/deckExport';

export default Vue.extend({
  props: {
    deckIndex: {type: Number},
  },

  computed: {
    admin(): boolean {
      return authStore.user?.id === 1;
    },

    deck(): Deck | undefined {
      return store.selectedSeat !== null ? store.decks[store.selectedSeat] : undefined;
    },

    exportedDeck(): string {
      return this.deck ? deckToXml(this.deck) : '';
    },

    exportedBinder(): string {
      return this.deck ? deckToBinderXml(this.deck) : '';
    },
  },

  asyncComputed: {
    async exportedDecksZip(): Promise<string> {
      return await decksToBinderZip(store.decks, store.names);
    }
  },

  methods: {
    exportToPdf() {
      if (this.deck) {
        deckToPdf(this.deck);
      }
    }
  },
});
</script>

<style scoped>
._deck-builder-export-menu {
  background: white;
  box-shadow: #a5a5a5 3px 2px 2px;
  border: 1px solid #eee;
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

</style>