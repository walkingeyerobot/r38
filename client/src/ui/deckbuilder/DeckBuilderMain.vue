<template>
  <div
      class="_deck-builder-main"
      :class="{
          horizontal,
          vertical: !horizontal,
      }"
      >
    <div class="sideboard">
      <DeckBuilderSectionControls
          :maindeck="false"
          :horizontal="horizontal"
          />
      <DeckBuilderSection
          :columns="sideboard"
          :deckIndex="store.selectedSeat"
          :maindeck="false"
          :horizontal="horizontal"
          />
    </div>
    <div class="maindeck">
      <DeckBuilderSectionControls
          :maindeck="true"
          :horizontal="horizontal"
          />
      <DeckBuilderSection
          :columns="maindeck"
          :deckIndex="store.selectedSeat"
          :maindeck="true"
          :horizontal="horizontal"
          />
    </div>
    <div
        class="exportButtons">
      <a
          :href="exportedDecksZip"
          download="r38export.zip"
          class="exportButton"
          :hidden="!admin || horizontal"
          >
        Export all
      </a>
      <a
          :href="exportedDeck"
          download="r38export.dek"
          class="exportButton"
          :hidden="!deck || horizontal"
          >
        Export deck
      </a>
      <a
          :href="exportedBinder"
          download="r38export.dek"
          class="exportButton"
          :hidden="!deck || horizontal"
          >
        Export binder
      </a>
    </div>
  </div>
</template>

<script lang="ts">
import Vue from 'vue';
import DeckBuilderSection from './DeckBuilderSection.vue';
import DeckBuilderSectionControls from './DeckBuilderSectionControls.vue';
import {
  CardColumn,
  Deck,
  deckBuilderStore,
  deckBuilderStore as store,
  DeckBuilderStore
} from '../../state/DeckBuilderModule';
import { rootStore } from '../../state/store';
import { tuple } from '../../util/tuple';
import { draftStore } from '../../state/DraftStore';
import { authStore } from "../../state/AuthStore";
import { decksToBinderZip, deckToBinderXml, deckToXml } from "../../draft/deckExport";

export default Vue.extend({
  components: {
    DeckBuilderSection,
    DeckBuilderSectionControls,
  },

  props: {
    horizontal: {type: Boolean},
  },

  data() {
    return {
      unwatchDraftStore: null as null | (() => void),
    }
  },

  created() {
    this.unwatchDraftStore = rootStore.watch(
        (state) => tuple(draftStore.currentState),
        (newProps, oldProps) => this.onDraftStoreChanged(),
        {immediate: true},
    );
  },

  destroyed() {
    if (this.unwatchDraftStore) {
      this.unwatchDraftStore();
    }
  },

  computed: {
    admin(): boolean {
      return authStore.user?.id === 1;
    },

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
      return this.deck ? deckToXml(this.deck) : '';
    },

    exportedBinder(): string {
      return this.deck ? deckToBinderXml(this.deck) : '';
    }
  },

  asyncComputed: {
    async exportedDecksZip(): Promise<string> {
      return await decksToBinderZip(store.decks, store.names);
    }
  },

  methods: {
    onDraftStoreChanged() {
      deckBuilderStore.sync(draftStore.currentState);
    },
  },
});

</script>

<style scoped>
._deck-builder-main {
  display: flex;
  overflow-x: scroll;
  min-height: 300px;
}

._deck-builder-main::-webkit-scrollbar {
  display: none;
}

.vertical {
  flex-direction: column;
}

.horizontal {
  flex-direction: row-reverse;
  border-top: 1px solid #666;
}

.maindeck {
  flex: 5 0 0;
  border-bottom: 1px solid #EAEAEA;
}

.horizontal .maindeck {
  border-bottom: none;
  border-right: 1px solid #666;
}

.sideboard {
  flex: 2 0 0;
}

.horizontal .maindeck, .horizontal .sideboard {
  overflow-x: hidden;
}

.exportButtons {
  position: absolute;
  top: 20px;
  right: 20px;
}

.exportButton {
  padding: 10px;
  margin-left: 20px;
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
