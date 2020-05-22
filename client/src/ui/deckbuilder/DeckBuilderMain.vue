<template>
  <div class="_deck-builder-main">
    <DeckBuilderSectionControls
        :maindeck="false"
        />
    <DeckBuilderSection
        class="sideboard"
        :columns="sideboard"
        :deckIndex="store.selectedSeat"
        :maindeck="false"
        />
    <DeckBuilderSectionControls
        :maindeck="true"
        />
    <DeckBuilderSection
        class="maindeck"
        :columns="maindeck"
        :deckIndex="store.selectedSeat"
        :maindeck="true"
        />
    <a
        :href="exportedDeck"
        download="r38export.dek"
        class="exportButton"
        ref="exportButton"
        :hidden="!deck"
        >
      Export to MTGO
    </a>
  </div>
</template>

<script lang="ts">
import Vue from 'vue';
import DeckBuilderSection from './DeckBuilderSection.vue';
import DeckBuilderSectionControls from "./DeckBuilderSectionControls.vue";
import { CardColumn, Deck, deckBuilderStore as store, DeckBuilderStore } from '../../state/DeckBuilderModule';

export default Vue.extend({
  components: {
    DeckBuilderSection,
    DeckBuilderSectionControls,
  },

  computed: {
    store(): DeckBuilderStore {
      return store;
    },

    deck(): Deck | undefined {
      return store.decks[store.selectedSeat];
    },

    sideboard(): CardColumn[] {
      return this.deck ? this.deck.sideboard : [];
    },

    maindeck(): CardColumn[] {
      return this.deck ? this.deck.maindeck : [];
    },

    exportedDeck(): string {
      if (this.deck) {
        let exportStr = "<?xml version=\"1.0\" encoding=\"utf-8\"?>\n<Deck xmlns:xsd=\"http://www.w3.org/2001/XMLSchema\" xmlns:xsi=\"http://www.w3.org/2001/XMLSchema-instance\">\n<NetDeckID>0</NetDeckID>\n<PreconstructedDeckID>0</PreconstructedDeckID>\n";
        for (const card of this.deck.maindeck.flat()) {
          exportStr += `<Cards CatID=\"${card.definition.mtgo}\" Quantity=\"1\" Sideboard=\"false\" Name=\"${card.definition.name}\" />\n`
        }
        for (const card of this.deck.sideboard.flat()) {
          exportStr += `<Cards CatID=\"${card.definition.mtgo}\" Quantity=\"1\" Sideboard=\"true\" Name=\"${card.definition.name}\" />\n`
        }
        exportStr += "</Deck>";
        return `data:text/xml;charset=utf-8,${encodeURIComponent(exportStr)}`;
      } else {
        return "";
      }
    }
  },

  methods: {},
});
</script>

<style scoped>
._deck-builder-main {
  display: flex;
  flex-direction: column;
  overflow-x: scroll;
}

.maindeck {
  flex: 5 0 0;
  border-bottom: 1px solid #EAEAEA;
}

.sideboard {
  flex: 2 0 0;
}

.exportButton {
  position: absolute;
  top: 20px;
  right: 20px;
  padding: 10px;
  text-decoration: none;
  color: inherit;
  background: white;
  border: 1px solid #bbb;
  border-radius: 5px;
  cursor: default;
}

.exportButton:hover {
  background: #ddd;
}
</style>
