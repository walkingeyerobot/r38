<template>
  <div class="_deck-builder-section-controls">
    <p
        class="cardsCount"
        :class="{tooFewCards}"
        :hidden="horizontal"
        >
      {{ maindeck ? "Main" : "Sideboard" }} - {{ numCards }} cards
    </p>
    <label class="sortLabel" :class="{horizontal}">Sort:</label>
    <button class="sortButton" :class="{horizontal}" @click="sortCmc">CMC</button>
    <button class="sortButton" :class="{horizontal}" @click="sortColor">Color</button>
    <button class="landButton" @click="addPlains" :hidden="horizontal">
      <img
          alt="Plains"
          src="../shared/mana/W_small.png"
          >
    </button>
    <button class="landButton" @click="addIsland" :hidden="horizontal">
      <img
          alt="Island"
          src="../shared/mana/U_small.png"
          >
    </button>
    <button class="landButton" @click="addSwamp" :hidden="horizontal">
      <img
          class="mana-symbol"
          alt="Swamp"
          src="../shared/mana/B_small.png"
          >
    </button>
    <button class="landButton" @click="addMountain" :hidden="horizontal">
      <img
          alt="Mountain"
          src="../shared/mana/R_small.png"
          >
    </button>
    <button class="landButton" @click="addForest" :hidden="horizontal">
      <img
          alt="Forest"
          src="../shared/mana/G_small.png"
          >
    </button>
  </div>
</template>

<script lang="ts">
import Vue from 'vue';
import { CardColumn, deckBuilderStore as store } from '../../state/DeckBuilderModule';
import { MtgCard } from '../../draft/DraftState';

export default Vue.extend({
  name: 'DeckBuilderSectionControls',

  props: {
    maindeck: {
      type: Boolean
    },
    horizontal: {
      type: Boolean
    },
  },

  computed: {
    section(): CardColumn[] {
      const selectedSeat = store.selectedSeat;
      const decks = store.decks;
      if (selectedSeat !== null && decks[selectedSeat]) {
        return this.maindeck
            ? decks[selectedSeat].maindeck
            : decks[selectedSeat].sideboard;
      } else {
        return [];
      }
    },
    numCards(): number {
      return this.section.flat().length;
    },
    tooFewCards(): boolean {
      return this.maindeck && this.numCards < 40;
    }
  },

  methods: {
    sortCmc() {
      if (store.selectedSeat !== null) {
        store.sortByCmc({seat: store.selectedSeat, maindeck: this.maindeck});
      }
    },
    sortColor() {
      if (store.selectedSeat !== null) {
        store.sortByColor({seat: store.selectedSeat, maindeck: this.maindeck});
      }
    },

    addPlains() {
      this.addLand(PLAINS);
    },

    addIsland() {
      this.addLand(ISLAND);
    },

    addSwamp() {
      this.addLand(SWAMP);
    },

    addMountain() {
      this.addLand(MOUNTAIN);
    },

    addForest() {
      this.addLand(FOREST);
    },

    addLand(definition: MtgCard) {
      this.section[0].push({
        id: performance.now(),
        sourcePackIndex: 0,
        pickedIn: [],
        hidden: false,
        definition: definition,
      });
    },

  },
});

const PLAINS = buildBasic(
  'Plains',
  'm21',
  '262',
  81203,
  'https://img.scryfall.com/cards/normal/front/8/a/8a299a1e-1ce9-4668-a5f5-c587081acf6b.jpg?1592761983',
);

const ISLAND = buildBasic(
  'Island',
  'm21',
  '263',
  81205,
  'https://img.scryfall.com/cards/normal/front/f/c/fc9a66a1-367c-4035-a22e-00fab55be5a0.jpg?1592761988',
);

const SWAMP = buildBasic(
  'Swamp',
  'm21',
  '266',
  81211,
  'https://img.scryfall.com/cards/normal/front/3/0/30b3d647-3546-4ade-b395-f2370750a7a6.jpg?1592762002',
);

const MOUNTAIN = buildBasic(
  'Mountain',
  'm21',
  '271',
  81221,
  'https://img.scryfall.com/cards/normal/front/e/d/ed6fd37e-a5b3-48c6-b881-cedadfd94833.jpg?1592762029',
);

const FOREST = buildBasic(
  'Forest',
  'm21',
  '274',
  81227,
  'https://img.scryfall.com/cards/normal/front/d/4/d4558304-7c17-4aa0-bd50-cdd95547f1a7.jpg?1592762045',
);

function buildBasic(
  name: string,
  set: string,
  collectorNumber: string,
  mtgoId: number,
  imageUri: string,
): MtgCard {
  return Object.freeze({
    name,
    set,
    collector_number: collectorNumber,
    mana_cost: [],
    cmc: 0,
    colors: [],
    color_identity: [],
    mtgo: mtgoId,
    rarity: 'common',
    type_line: `Basic Land - ${name}`,
    layout: 'normal',
    card_faces: [],
    foil: false,
    image_uris: [imageUri],
    searchName: name.toLocaleLowerCase(),
  });
}

</script>

<style scoped>
._deck-builder-section-controls {
  display: flex;
  align-items: center;
  padding: 20px 20px 0;
}

.cardsCount {
  width: 14em;
}

.tooFewCards {
  color: #aa2222
}

.sortLabel {
  margin-right: 1em;
  font-size: 80%;
}

.sortLabel.horizontal {
  color: white;
}

.sortButton {
  background: transparent;
  border: 1px solid #bbb;
  border-radius: 1.5em;
  padding: 2px 1em;
  margin-right: 1em;
}

.sortButton.horizontal {
  background: white;
}

.sortButton:hover {
  background: #ddd;
}

.landButton {
  border: none;
  border-radius: 12px;
  padding: 4px;
  line-height: 0;
}

.landButton:hover {
  background: #ddd;
}

.landButton img {
  width: 16px;
  height: 16px;
}

</style>