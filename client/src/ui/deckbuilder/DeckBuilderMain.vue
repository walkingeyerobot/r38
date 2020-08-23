<template>
  <div
      class="_deck-builder-main"
      :class="{
          horizontal,
          vertical: !horizontal,
      }"
      >
    <div
        class="sideboard"
        v-if="deck"
        >
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
    <div
        class="maindeck"
        v-if="deck"
        >
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
        class="exportButtons"
        @mousedown.capture="onRootMouseDown"
        >
      <a
          class="exportButton"
          @click="exportButtonClick"
          v-if="deck"
          >
        Export
      </a>
      <DeckBuilderExportMenu
          :deckIndex="store.selectedSeat"
          v-if="exportMenuOpen"
          class="exportMenu"
          />
    </div>
  </div>
</template>

<script lang="ts">
import Vue from 'vue';
import DeckBuilderExportMenu from './DeckBuilderExportMenu.vue';
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
import { globalClickTracker, UnhandledClickListener } from '../infra/globalClickTracker';

export default Vue.extend({
  components: {
    DeckBuilderExportMenu,
    DeckBuilderSection,
    DeckBuilderSectionControls,
  },

  props: {
    horizontal: {type: Boolean},
  },

  data() {
    return {
      unwatchDraftStore: null as null | (() => void),
      exportMenuOpen: false,
      globalMouseDownListener: null as UnhandledClickListener | null,
    }
  },

  created() {
    this.unwatchDraftStore = rootStore.watch(
        (state) => tuple(draftStore.currentState),
        (newProps, oldProps) => this.onDraftStoreChanged(),
        {immediate: true},
    );
    this.globalMouseDownListener = () => this.onGlobalMouseDown();
    globalClickTracker
        .registerUnhandledClickListener(this.globalMouseDownListener);
  },

  destroyed() {
    if (this.unwatchDraftStore) {
      this.unwatchDraftStore();
    }
    if (this.globalMouseDownListener != null) {
      globalClickTracker
          .unregisterUnhandledClickListener(this.globalMouseDownListener);
    }
  },

  computed: {

    store(): DeckBuilderStore {
      return store;
    },

    deck(): Deck | undefined {
      return store.selectedSeat !== null ? store.decks[store.selectedSeat] : undefined;
    },

    sideboard(): CardColumn[] {
      return this.deck ? this.deck.sideboard : [];
    },

    maindeck(): CardColumn[] {
      return this.deck ? this.deck.maindeck : [];
    },
  },

  methods: {
    onDraftStoreChanged() {
      deckBuilderStore.sync(draftStore.currentState);
    },

    exportButtonClick() {
      this.exportMenuOpen = !this.exportMenuOpen;
    },

    onRootMouseDown() {
      globalClickTracker.onCaptureLocalMouseDown();
    },

    onGlobalMouseDown() {
      this.exportMenuOpen = false;
    },
  },
});

</script>

<style scoped>
._deck-builder-main {
  display: flex;
  overflow-x: scroll;
  min-height: 300px;
  position: relative;
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

.maindeck, .sideboard {
  min-height: 0;
  display: flex;
  flex-direction: column;
}

.maindeck {
  flex: 5;
}

.horizontal .maindeck {
  border-bottom: none;
  border-right: 1px solid #666;
}

.sideboard {
  flex: 2;
  border-bottom: 1px solid #EAEAEA;
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

.exportMenu {
  position: absolute;
  top: calc(100% + 15px);
  right: 0;
}
</style>
