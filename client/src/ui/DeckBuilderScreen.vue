<template>
  <div class="_deck-builder-screen">
    <div class="main">
      <DeckBuilderPlayerSelector class="player-selector" />
      <DeckBuilder class="deckbuilder" />
    </div>
  </div>
</template>

<script lang="ts">
import Vue from 'vue';

import DeckBuilder from './deckbuilder/DeckBuilder.vue';
import DeckBuilderPlayerSelector from './deckbuilder/DeckBuilderPlayerSelector.vue';
import { parseDraft } from "../parse/parseDraft";
import { MtgCard, DraftCard } from '../draft/DraftState';
import { DeckInitializer } from '../state/DeckBuilderModule';
import { commitTimelineEvent } from '../draft/mutate';
import { getServerPayload } from '../parse/getServerPayload';

export default Vue.extend({
  name: 'DeckBuilderScreen',

  components: {
    DeckBuilder,
    DeckBuilderPlayerSelector,
  },

  created() {
    const draftState = this.generateCurrentDraftState();

    const init = [] as DeckInitializer[];
    for (let seat of draftState.seats) {
      init.push({
        player: {
          seatPosition: seat.player.seatPosition,
          name: seat.player.name,
        },
        pool: seat.player.picks.cards.concat(),
      });
    }
    this.$tstore.commit('deckbuilder/initDecks', init);
  },

  methods: {
    generateCurrentDraftState() {
      const srcData = getServerPayload();
      const draft = parseDraft(srcData);
      for (let event of draft.events) {
        commitTimelineEvent(event, draft.state);
      }
      return draft.state;
    },
  }
})
</script>

<style scoped>
._deck-builder-screen {
  height: 100%;
  display: flex;
  flex-direction: column;
}

.main {
  display: flex;
  flex: 1;
  flex-direction: row;
  overflow: hidden;
}

.player-selector {
  width: 200px;
  flex: 0 0 auto;
}

.deckbuilder {
  flex: 1;
}
</style>
