<template>
  <div class="_draft-picker">
    <div class="picks">
      <div v-if="availablePack" class="picks-cnt">
        <CardView
            v-for="(cardId, i) in availablePack.cards"
            :key="cardId"
            :card="draftStore.getCard(cardId)"
            class="card"
            :class="getCardCssClass(cardId)"
            :style="{'animation-delay': animationDelays[i] * -15000 + 'ms'}"
            @click.native="onPackCardClicked(cardId)"
            />
      </div>
      <div v-else class="no-picks">You don't have any picks (yet)</div>
    </div>
    <div class="pool">
      <div class="pool-title">Drafted cards</div>
      <div class="pool-cnt">
        <CardView
            v-for="cardId in currentPool"
            :key="cardId"
            :card="draftStore.getCard(cardId)"
            class="card"
            />
      </div>
    </div>
  </div>
</template>

<script lang="ts">
import Vue from 'vue';
import { authStore } from '../../state/AuthStore';
import { draftStore, DraftStore } from '../../state/DraftStore';
import { replayStore, ReplayStore } from '../../state/ReplayStore';

import CardView from './CardView.vue';
import { CardPack, DraftSeat, DraftCard } from '../../draft/DraftState';
import { TimelineEvent } from '../../draft/TimelineEvent';
import { checkNotNil } from '../../util/checkNotNil';
import { fetchEndpoint } from '../../fetch/fetchEndpoint';
import { routePick } from '../../rest/api/pick/pick';
import { delay } from '../../util/delay';
import { navTo } from '../../router/url_manipulation';

export default Vue.extend({

  components: {
    CardView
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
      return this.currentSeat.queuedPacks.packs[0] || null;
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
      return this.currentSeat.player.picks.cards;
    },
  },

  methods: {
    async onPackCardClicked(cardId: number) {
      if (this.submittingPick) {
        return;
      }
      this.submittingPick = true;
      this.pickedCardId = cardId;

      const card = draftStore.getCard(cardId);

      const start = Date.now();
      // TODO: Error handling
      const response = await fetchEndpoint(routePick, {
        cards: [cardId],
      });
      const elapsed = Date.now() - start;
      await delay(500 - elapsed);

      // TODO: Should probably just apply the returned state instead
      draftStore.pickCard({
        seatId: this.currentSeat.position,
        cardId: cardId,
      });

      this.submittingPick = false;
      this.pickedCardId = null;
    },

    getCardCssClass(cardId: number) {
      const isPicked = cardId == this.pickedCardId;
      return {
        'picked-fade': isPicked,
        'not-picked-fade': this.submittingPick && !isPicked,
      };
    },
  }
});
</script>

<style scoped>
._draft-picker {
  display: flex;
  flex-direction: column;
  background: #3c3c3c;
  overflow-y: scroll;
  align-items: center;
}

.picks-cnt {
  display: flex;
  flex-direction: row;
  flex-wrap: wrap;
  align-content: flex-start;
  padding: 15px;
  min-height: 400px;
  box-sizing: border-box;
}

.no-picks {
  display: flex;
  height: 400px;
  align-items: center;
  justify-content: center;
  color: #CCC;
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

.pool-title {
  text-align: center;
  color: #CCC;
}

.pool-cnt {
  display: flex;
  flex-direction: row;
  flex-wrap: wrap;
  padding: 15px;
}

.picks, .pool {
  box-sizing: border-box;
  max-width: 1110px;
  width: 100%;
}

</style>
