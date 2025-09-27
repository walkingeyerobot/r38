<!-- eslint-disable vue/no-deprecated-v-on-native-modifier -->
<template>
  <div class="_draft-picker">
    <div v-if="availablePack" class="main-content picks">
      <CardView
        v-for="(cardId, i) in availablePack.cards"
        :key="cardId"
        :card="draftStore.getCard(cardId)"
        class="card"
        :class="getCardCssClass(cardId)"
        :style="{ 'animation-delay': animationDelays[i] * -15000 + 'ms' }"
        @click.native="onPackCardClicked(cardId)"
      />
    </div>
    <div v-else class="main-content no-picks">You don't have any picks (yet)</div>
    <DeckBuilderMain v-if="showDeckBuilder" class="pool" :horizontal="true" />
  </div>
</template>

<script lang="ts">
import { defineComponent } from "vue";
import CardView from "./CardView.vue";
import DeckBuilderMain from "../deckbuilder/DeckBuilderMain.vue";

import { authStore } from "@/state/AuthStore";
import { draftStore, type DraftStore } from "@/state/DraftStore";
import { replayStore } from "@/state/ReplayStore";
import type { CardPack, DraftSeat } from "@/draft/DraftState";
import { fetchEndpoint } from "@/fetch/fetchEndpoint";
import { ROUTE_PICK } from "@/rest/api/pick/pick";
import { delay } from "@/util/delay";

export default defineComponent({
  components: {
    CardView,
    DeckBuilderMain,
  },

  props: {
    showDeckBuilder: {
      type: Boolean,
      required: true,
    },
  },

  data() {
    const animationDelays = [];
    for (let i = 0; i < 15; i++) {
      animationDelays.push(Math.random());
    }

    return {
      animationDelays,
      submittingPick: false,
      pickedCardId: null as number | null,
    };
  },

  computed: {
    draftStore(): DraftStore {
      return draftStore;
    },

    availablePack(): CardPack | null {
      const pack = this.currentSeat.queuedPacks.packs[0];
      if (pack != null && pack.round == this.currentSeat.round) {
        return pack;
      } else {
        return null;
      }
    },

    currentSeat(): DraftSeat {
      if (authStore.user == null) {
        throw new Error(`Must have a logged-in user`);
      }
      for (const seat of replayStore.draft.seats) {
        if (seat.player?.id == authStore.user.id) {
          return seat;
        }
      }
      throw new Error(`No active user found with ID ${authStore.user.id}`);
    },

    currentPool(): number[] {
      return this.currentSeat.picks.cards;
    },
  },

  methods: {
    async onPackCardClicked(cardId: number) {
      if (this.submittingPick) {
        return;
      }
      this.submittingPick = true;
      this.pickedCardId = cardId;

      const start = Date.now();
      // TODO: Error handling
      const response = await fetchEndpoint(ROUTE_PICK, {
        draftId: draftStore.draftId,
        cards: [cardId],
        xsrfToken: draftStore.pickXsrf,
        as: authStore.user?.id,
      });
      const elapsed = Date.now() - start;
      await delay(500 - elapsed);

      draftStore.loadDraft(response);

      this.submittingPick = false;
      this.pickedCardId = null;
    },

    getCardCssClass(cardId: number) {
      const isPicked = cardId == this.pickedCardId;
      return {
        "picked-fade": isPicked,
        "not-picked-fade": this.submittingPick && !isPicked,
      };
    },
  },
});
</script>

<style scoped>
._draft-picker {
  display: flex;
  flex-direction: column;
  background: #3c3c3c;
  overflow-y: scroll;
}

.main-content {
  flex: 1 0 400px;
}

.picks {
  display: flex;
  flex-direction: row;
  flex-wrap: wrap;
  align-content: flex-start;
  padding: 15px;
  box-sizing: border-box;
  max-width: 1200px;
}

.scan {
  display: flex;
  flex-direction: column;
  justify-content: center;
  align-items: center;
  text-transform: uppercase;
  color: white;
  font-size: 300%;
}

.scan > button {
  margin: 30px;
  padding: 5px 15px;

  font-size: 50%;
  font-family: inherit;
  border: 1px solid #c54818;
  border-radius: 5px;
  background: white;
  color: #c54818;
}

.no-picks {
  display: flex;
  align-items: center;
  justify-content: center;
  color: #ccc;
}

.card {
  margin: 0 15px 15px 0;
  opacity: 1;
}

.picked-fade {
  opacity: 0;
  transition: opacity 300ms cubic-bezier(0.5, 1, 0.89, 1);
  transition-delay: 200ms;
}

.not-picked-fade {
  opacity: 0;
  transition: opacity 200ms cubic-bezier(0.33, 1, 0.68, 1);
}

.pool {
  margin-top: 60px;
}
</style>
