<template>
  <div class="_deck-builder-screen">
    <div class="main" v-if="status == 'loaded'">
      <DeckBuilderPlayerSelector class="player-selector" />
      <DeckBuilderMain class="deckbuilder" :horizontal="false" />
    </div>
  </div>
</template>

<script lang="ts">
import Vue from 'vue';

import DeckBuilderMain from './deckbuilder/DeckBuilderMain.vue';
import DeckBuilderPlayerSelector from './deckbuilder/DeckBuilderPlayerSelector.vue';

import { authStore } from '../state/AuthStore';
import { draftStore } from '../state/DraftStore';
import { deckBuilderStore as deckStore, DeckInitializer } from '../state/DeckBuilderModule';

import { fetchEndpoint } from '../fetch/fetchEndpoint';
import { routeDraft } from '../rest/api/draft/draft';
import { FetchStatus } from './infra/FetchStatus';


export default Vue.extend({
  components: {
    DeckBuilderMain,
    DeckBuilderPlayerSelector,
  },

  data() {
    return {
      status: 'missing' as FetchStatus,
    };
  },

  created() {
    const draftId = parseInt(this.$route.params['draftId']);
    this.fetchDraft(draftId);
  },

  methods: {
    async fetchDraft(draftId: number) {
      const payload =
          await fetchEndpoint(routeDraft, {
            id: draftId,
            as: authStore.user?.id,
          });
      this.status = 'loaded';

      // TODO: Handle fetch error

      draftStore.loadDraft(payload);
      document.title = `${draftStore.draftName}`;

      const state = draftStore.currentState;
      const init = [] as DeckInitializer[];
      deckStore.initNames(state.seats.map(seat => seat.player.name));
      for (let seat of state.seats) {
        init.push({
          draftName: draftStore.draftName,
          pool: seat.player.picks.cards
              .map(cardId => draftStore.getCard(cardId)),
        });
      }
      deckStore.initDecks(init);
    },
  },

});
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
