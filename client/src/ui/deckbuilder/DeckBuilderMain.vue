<template>
  <div
    v-if="store.selectedSeat != null"
    class="_deck-builder-main"
    :class="{
      horizontal,
      vertical: !horizontal,
    }"
  >
    <div class="sideboard" v-if="deck">
      <DeckBuilderSectionControls :maindeck="false" :horizontal="horizontal" />
      <DeckBuilderSection
        ref="sideboard"
        :columns="sideboard"
        :deckIndex="store.selectedSeat"
        :maindeck="false"
        :horizontal="horizontal"
      />
    </div>
    <div class="maindeck" v-if="deck">
      <DeckBuilderSectionControls :maindeck="true" :horizontal="horizontal" />
      <DeckBuilderSection
        ref="maindeck"
        :columns="maindeck"
        :deckIndex="store.selectedSeat"
        :maindeck="true"
        :horizontal="horizontal"
      />
    </div>
    <div class="exportButtons" @mousedown.capture="onRootMouseDown">
      <a class="exportButton" @click="exportButtonClick" v-if="deck"> Export </a>
      <DeckBuilderExportMenu
        :deckIndex="store.selectedSeat"
        v-if="exportMenuOpen"
        class="exportMenu"
      />
    </div>
    <img
      v-if="zoomedCard"
      class="card-zoom"
      :src="zoomedCard.image_uris[0]"
      :alt="zoomedCard.name"
      :style="zoomedCardPos"
    />
  </div>
</template>

<script lang="ts">
import { defineComponent, ref, type StyleValue } from "vue";
import DeckBuilderExportMenu from "./DeckBuilderExportMenu.vue";
import DeckBuilderSection from "./DeckBuilderSection.vue";
import DeckBuilderSectionControls from "./DeckBuilderSectionControls.vue";

import {
  type DeckBuilderStore,
  type Deck,
  type CardColumn,
  deckBuilderStore as store,
  deckBuilderStore,
} from "@/state/DeckBuilderModule";
import { rootStore } from "@/state/store";
import { tuple } from "@/util/tuple";
import { draftStore } from "@/state/DraftStore";
import { globalClickTracker, type UnhandledClickListener } from "../infra/globalClickTracker";
import type { MtgCard } from "@/draft/DraftState";

export default defineComponent({
  name: "DeckBuilderMain",

  components: {
    DeckBuilderExportMenu,
    DeckBuilderSection,
    DeckBuilderSectionControls,
  },

  setup() {
    const maindeckElem = ref<InstanceType<typeof DeckBuilderSection> | null>();
    const sideboardElem = ref<InstanceType<typeof DeckBuilderSection> | null>();

    return { maindeckElem, sideboardElem };
  },

  props: {
    horizontal: { type: Boolean },
  },

  data: () => ({
    unwatchDraftStore: null as null | (() => void),
    exportMenuOpen: false,
    globalMouseDownListener: null as UnhandledClickListener | null,
  }),

  created() {
    this.unwatchDraftStore = rootStore.watch(
      (_state) => tuple(draftStore.currentState),
      (_newProps, _oldProps) => this.onDraftStoreChanged(),
      { immediate: true },
    );
    this.globalMouseDownListener = () => this.onGlobalMouseDown();
    globalClickTracker.registerUnhandledClickListener(this.globalMouseDownListener);
  },

  unmounted() {
    if (this.unwatchDraftStore) {
      this.unwatchDraftStore();
    }
    if (this.globalMouseDownListener != null) {
      globalClickTracker.unregisterUnhandledClickListener(this.globalMouseDownListener);
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

    zoomedCard(): MtgCard | null {
      return this.deck && store.zoomed
        ? (store.zoomed.maindeck ? this.deck.maindeck : this.deck.sideboard)[
            store.zoomed.columnIndex
          ][store.zoomed.cardIndex].definition
        : null;
    },

    zoomedCardPos(): Partial<StyleValue> {
      if (store.zoomed) {
        const section = store.zoomed.maindeck ? this.maindeckElem : this.sideboardElem;
        const left = (store.zoomed.columnIndex + 1) * section?.columnElems[0].$el.clientWidth;
        const top = store.zoomed.cardIndex * (this.horizontal ? 15 : 30);
        return {
          left: left + "px",
          top: top + "px",
        };
      } else {
        return {};
      }
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
      store.zoomCard(null);
    },
  },
});
</script>

<style scoped>
._deck-builder-main {
  display: flex;
  position: relative;
  align-items: stretch;
}

.vertical {
  flex-direction: column;
  overflow-y: scroll;
}

.horizontal {
  flex-direction: row-reverse;
  border-top: 1px solid #666;
}

.maindeck,
.sideboard {
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
  border-bottom: 1px solid #eaeaea;
}

.horizontal .maindeck,
.horizontal .sideboard {
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

.card-zoom {
  position: absolute;
  width: 200px;
  height: 279px;
  border-radius: 10px;
}
</style>
