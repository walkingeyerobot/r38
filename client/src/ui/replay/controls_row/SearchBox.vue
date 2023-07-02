<!--

General search box for the draft

Typing into the search box will cause a popover to appear that contains
results.

-->

<template>
  <div
      class="_search-box"
      @mousedown.capture="onRootMouseDown"
      >
    <input
        v-model.trim="searchStr"
        class="input"
        :class="{
          expanded: inputFocused || resultsOpen,
        }"
        @focus="onInputFocus"
        @blur="onInputBlur"
        @input="onInputChange"
        placeholder="Search"
        >
    <div
        v-if="resultsOpen"
        class="results"
        >
      <div
          v-for="(result, index) in searchResults"
          :key="index"
          class="card-result"
          >
        <div
            class="card-nameslate"
            @click="onCardNameClick(result)"
            >
          <div class="card-name">
            {{ result.card.name }}
          </div>
          <ManaSymbol
              v-for="(code, i) in result.card.mana_cost"
              :key="i"
              :code="code"
              class="mana-symbol"
              />
        </div>
        <div
            class="card-picked"
            >
          <template v-if="result.picked">
            Picked by
            {{ result.picked.playerName }}
            in
            <span
                class="pick-time"
                @click="onPickTimeClick(result.picked)">
              pack
              {{ result.picked.round }}
              pick
              {{ result.picked.pick + 1 }}
            </span>
          </template>
          <template v-else>
            Not picked yet
          </template>
        </div>
      </div>

      <div
          v-if="searchResults.length == 0"
          class="no-results-message"
          >
        No results
      </div>
    </div>
  </div>
</template>

<script lang="ts">
import { defineComponent } from 'vue';
import ManaSymbol from '../../shared/mana/ManaSymbol.vue';

import { replayStore } from '../../../state/ReplayStore';
import { draftStore } from '../../../state/DraftStore';

import { CardContainer, MtgCard } from '../../../draft/DraftState';
import { pushDraftUrlRelative } from '../../../router/url_manipulation';
import { SelectedView } from '../../../state/selection';
import { indexOf } from '../../../util/collection';
import { TimelineEvent, TimelineAction } from '../../../draft/TimelineEvent';
import { globalClickTracker, UnhandledClickListener } from '../../infra/globalClickTracker';
import { CardLocation } from '../../../state/DeckBuilderModule';

export default defineComponent({
  components: {
    ManaSymbol,
  },

  data() {
    return {
      searchStr: '',
      resultsOpen: false,
      inputFocused: false,
      globalMouseDownListener: null as UnhandledClickListener | null,
    };
  },

  computed: {
    searchResults(): CardSearchResult[] {
      if (!this.resultsOpen) {
        return [];
      } else {
        if (this.searchStr.length <= 1) {
          return [];
        } else {
          return this.performSearch(this.searchStr);
        }
      }
    },
  },

  created() {
    this.globalMouseDownListener = () => this.onGlobalMouseDown();
    globalClickTracker
        .registerUnhandledClickListener(this.globalMouseDownListener);
  },

  destroyed() {
    if (this.globalMouseDownListener != null) {
      globalClickTracker
          .unregisterUnhandledClickListener(this.globalMouseDownListener);
    }
  },

  methods: {
    onInputFocus() {
      if (this.searchStr != '') {
        this.resultsOpen = true;
      }
      this.inputFocused = true;
    },

    onInputBlur() {
      this.inputFocused = false;
    },

    onInputChange() {
      this.resultsOpen = this.searchStr != '';
    },

    onRootMouseDown() {
      globalClickTracker.onCaptureLocalMouseDown();
    },

    onCardNameClick(result: CardSearchResult) {
      if (result.packType != 'shadow-realm') {
        pushDraftUrlRelative(this, {
          selection: {
            id: result.packId,
            type: result.packType,
          },
        });
      }
    },

    onPickTimeClick(pick: CardSearchResult['picked']) {
      if (pick != null) {
        const index = indexOf(replayStore.events, { id: pick.eventId });
        if (index != -1) {
          pushDraftUrlRelative(this, {
            eventIndex: index,
            selection: {
              id: pick.fromSeat,
              type: 'seat',
            },
          });
        }
      }
    },

    onGlobalMouseDown() {
      this.resultsOpen = false;
    },

    performSearch(query: string): CardSearchResult[] {
      const startTime = Date.now();

      const finalQuery = query.toLocaleLowerCase().normalize();
      const results = [] as CardSearchResult[];
      for (let pack of replayStore.draft.packs.values()) {
        for (let cardId of pack.cards) {
          const card = draftStore.getCard(cardId);

          if (card.definition.searchName.indexOf(query) != -1) {
            const packId = pack.type == 'seat' ? pack.owningSeat : pack.id;

            const result: CardSearchResult = {
              type: 'card',
              packType: pack.type,
              packId: packId,
              card: card.definition,
              picked: null,
            };

            const pick = card.pickedIn[card.pickedIn.length - 1];
            if (pick != null) {
              const playerName = pick.bySeat == -1
                  ? 'someone'
                  : replayStore.draft.seats[pick.bySeat].player.name;
              result.picked = {
                playerName,
                fromSeat: pick.fromSeat,
                eventId: pick.eventId,
                round: pick.round,
                pick: pick.pick,
              };
            }

            results.push(Object.freeze(result));
          }
        }
      }

      results.sort((a, b) => a.card.name.localeCompare(b.card.name));

      console.log('Search took', (Date.now() - startTime), 'ms');

      return results;
    },
  },
});

type SearchResult = CardSearchResult;

interface CardSearchResult {
  type: 'card',
  packId: number,
  packType: CardContainer['type'],
  card: MtgCard,
  picked: {
    playerName: string,
    fromSeat: number,
    eventId: number,
    round: number,
    pick: number,
  } | null,
}
</script>

<style scoped>
._search-box {
  position: relative;
}

.input {
  width: 200px;
  box-sizing: border-box;

  padding: 5px 10px;
  border: 1px solid #c7c7c7;
  border-radius: 100px;

  font-family: inherit;
  font-size: 14px;
  color: #2c3e50;

  transition: width 300ms cubic-bezier(0.33, 1, 0.68, 1);
}

.input:focus {
  outline: none;
  border-color: #aaa;
}

.input.expanded {
  width: 300px;
}

.results {
  position: absolute;
  top: calc(100% + 5px);
  right: 0;
  box-sizing: border-box;
  width: 300px;
  min-height: 125px;
  max-height: calc(100vh - 70px);
  padding: 10px 9px;
  overflow-y: scroll;

  background-color: white;
  font-size: 14px;
  color: #000;

  border: 1px solid #ccc;
  border-radius: 5px;
  box-shadow: 0px 1px 4px rgba(0, 0, 0, 0.3);

  cursor: default;
}

.card-result {
  margin-bottom: 10px;
}

.card-nameslate {
  padding: 4px 6px;
  border-radius: 6px;
  background-color: #f7f7f7;
  border: 1px solid #a5a5a5;

  cursor: pointer;
  user-select: none;

  display: flex;
  flex-direction: row;
  align-items: baseline;
}

.card-nameslate:hover {
  border-color: #9a9a9a;
  box-shadow: 0 0 2px rgba(0, 0, 0, 0.2);
}

.card-name {
  flex: 1;
}

.mana-symbol {
  margin-left: 2px;
}

.card-picked {
  padding: 0 7px;
  font-size: 12px;
  margin-top: 2px;
}

.pick-time {
  cursor: pointer;
}

.pick-time:hover {
  text-decoration: underline;
}

.pick-time:active {
  color: #000;
}

.no-results-message {
  margin-top: 10px;
  color: #666;
  text-align: center;
}
</style>
