<!-- eslint-disable vue/no-deprecated-v-on-native-modifier -->
<template>
  <div class="_draft-picker">
    <div v-if="availablePack" class="main-content picks">
      <CardView
        v-for="cardId in availablePack.cards"
        :key="cardId"
        :card="draftStore.getCard(cardId)"
        :selection-style="pickedCards.includes(cardId) ? 'will-pick' : undefined"
        class="card"
        :class="getCardCssClass(cardId)"
        @click.native="onPackCardClicked(cardId)"
      />
    </div>
    <div v-else class="main-content no-picks">You don't have any picks (yet)</div>
    <div class="pick-cnt">
      <button v-if="pickedCards.length > 0" class="pick-btn" @click="onPickConfirmed">
        {{ pickButtonLabel }}
      </button>
    </div>
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
    return {
      submittingPick: false,
      pickedCards: [] as number[],
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

    cardsPerPick(): number {
      return this.draftStore.pickTwo ? 2 : 1;
    },

    pickButtonLabel(): string {
      if (this.pickedCards.length >= this.cardsPerPick) {
        return this.pickedCards.length == 1 ? "Pick card" : "Pick cards";
      } else {
        return `Pick ${this.cardsPerPick - this.pickedCards.length} more card`;
      }
    },
  },

  methods: {
    onPackCardClicked(cardId: number) {
      const idx = this.pickedCards.indexOf(cardId);
      if (idx == -1) {
        if (this.pickedCards.length >= this.cardsPerPick) {
          this.pickedCards.shift();
        }
        this.pickedCards.push(cardId);
      } else {
        this.pickedCards.splice(idx, 1);
      }
    },

    async onPickConfirmed() {
      if (this.submittingPick) {
        return;
      }
      this.submittingPick = true;

      const start = Date.now();
      // TODO: Error handling
      const response = await fetchEndpoint(ROUTE_PICK, {
        draftId: draftStore.draftId,
        cards: [...this.pickedCards],
        xsrfToken: draftStore.pickXsrf,
        as: authStore.user?.id,
      });
      const elapsed = Date.now() - start;
      await delay(500 - elapsed);

      draftStore.loadDraft(response);

      this.submittingPick = false;
    },

    getCardCssClass(cardId: number) {
      if (!this.submittingPick) {
        return undefined;
      }
      const isPicked = this.pickedCards.includes(cardId);
      return isPicked ? "picked-fade" : "not-picked-fade";
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
  padding-bottom: 0;
  box-sizing: border-box;
  max-width: 1110px;
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

.pick-cnt {
  position: sticky;
  height: 70px;
  bottom: 15px;

  flex: none;
  max-width: 1110px;
  margin-bottom: 20px;

  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
}

.pick-btn {
  padding: 20px;
  box-shadow: 0px 4px 10px 2px rgba(0, 0, 0, 0.75);
  min-width: 200px;
  border-radius: 10px;
  color: #3f2f06;
  background-color: #efc354;
  border: 1px solid #b49f31;
  font-size: 16px;
}

.pick-btn:active {
  background-color: #c8a242;
  border-color: rgb(173, 163, 140);
}
</style>
