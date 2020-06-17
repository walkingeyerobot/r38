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
        :hidden="!deck"
        >
      Export deck
    </a>
    <a
        :href="exportedBinder"
        download="r38export.dek"
        class="exportButton"
        :hidden="!deck"
        >
      Export binder
    </a>
  </div>
</template>

<script lang="ts">
import Vue from 'vue';
import DeckBuilderSection from './DeckBuilderSection.vue';
import DeckBuilderSectionControls from "./DeckBuilderSectionControls.vue";
import { BASICS, CardColumn, Deck, deckBuilderStore as store, DeckBuilderStore } from '../../state/DeckBuilderModule';
import { MtgCard } from '../../draft/DraftState';

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
        let exportStr = XML_HEADER;
        let mainMap = new Map<number, DeckEntry>();
        let sideMap = new Map<number, DeckEntry>();
        for (const card of this.deck.maindeck.flat()) {
          if (!card.definition.mtgo) {
            continue;
          }
          incrementQuantity(mainMap, card.definition);
        }
        for (const card of this.deck.sideboard.flat()) {
          if (!card.definition.mtgo) {
            continue;
          }
          incrementQuantity(sideMap, card.definition);
        }
        for (const [mtgo, card] of mainMap) {
          exportStr += `<Cards CatID=\"${mtgo}\" Quantity=\"${card.quantity}\"`
              + ` Sideboard=\"false\" Name=\"${card.name}\" />\n`;
        }
        for (const [mtgo, card] of sideMap) {
          exportStr += `<Cards CatID=\"${mtgo}\" Quantity=\"${card.quantity}\"`
              + ` Sideboard=\"true\" Name=\"${card.name}\" />\n`;
        }
        exportStr += "</Deck>";
        return `data:text/xml;charset=utf-8,${encodeURIComponent(exportStr)}`;
      } else {
        return '';
      }
    },

    exportedBinder(): string {
      if (this.deck) {
        let exportStr = XML_HEADER;
        let map = new Map<number, DeckEntry>();
        for (const card of this.deck.maindeck.flat()) {
          if (!card.definition.mtgo || BASICS.includes(card.definition.mtgo)) {
            continue;
          }
          incrementQuantity(map, card.definition);
        }
        for (const card of this.deck.sideboard.flat()) {
          if (!card.definition.mtgo) {
            continue;
          }
          incrementQuantity(map, card.definition);
        }
        for (const [mtgo, card] of map) {
          exportStr += `<Cards CatID=\"${mtgo}\" Quantity=\"${card.quantity}\"`
              + ` Sideboard=\"false\" Name=\"${card.name}\" />\n`;
        }
        exportStr += "</Deck>";
        return `data:text/xml;charset=utf-8,${encodeURIComponent(exportStr)}`;
      } else {
        return '';
      }
    }
  },

  methods: {},
});

function incrementQuantity(map: Map<number, DeckEntry>, card: MtgCard) {
  let entry = map.get(card.mtgo);
  if (entry == undefined) {
    entry = {name: card.name, quantity: 0};
    map.set(card.mtgo, entry);
  }
  entry.quantity++;
}

interface DeckEntry {
  name: string;
  quantity: number;
}

const XML_HEADER =
    `<?xml version="1.0" encoding="utf-8"?>
<Deck xmlns:xsd="http://www.w3.org/2001/XMLSchema" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">
<NetDeckID>0</NetDeckID>
<PreconstructedDeckID>0</PreconstructedDeckID>
`;

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

.exportButton + .exportButton {
  right: calc(7em + 20px);
}

.exportButton:hover {
  background: #ddd;
}
</style>
