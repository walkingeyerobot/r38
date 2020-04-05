<template>
  <div class="_card-grid">
    <div
        class="selected-player"
        v-if="selectedSeat"
        >
      {{ selectedSeat.player.name }}
    </div>

    <div
        v-if="selectedPack"
        class="selected-pack"
        >
      <div
          v-for="card in selectedPack.cards"
          :key="card.id"
          class="card"
          >
        <img
            class="card-img"
            :class="getSelectionClass(card.id)"
            :title="card.definition.name"
            :src="`/proxy/${card.definition.set}/${card.definition.collector_number}`"
            >
      </div>
    </div>

    <div
        v-if="selectedSeat"
        class="player-grid"
        >
      <div
          v-for="card in selectedSeat.player.picks.cards"
          :key="card.id"
          class="card"
          >
        <img
            class="card-img"
            :title="card.definition.name"
            :src="`/proxy/${card.definition.set}/${card.definition.collector_number}`"
            >
      </div>
    </div>

  </div>
</template>

<script lang="ts">
import Vue from 'vue'
import { DraftCard, DraftPlayer, DraftSeat, CardPack, CardContainer, TimelineEvent } from '../../draft/draft_types'
import { SelectedView } from '../../state/selection';
import { checkNotNil } from '../../util/checkNotNil';

export default Vue.extend({

  computed: {
    selection(): SelectedView | null {
      return this.$tstore.state.selection;
    },

    selectedSeat(): DraftSeat | null  {
      if (this.selection == null || this.selection.type == 'pack') {
        return null;
      } else {
        return this.$tstore.state.draft.seats[this.selection.id]
      }
      return null;
    },

    selectedPack(): CardContainer | null {
      if (this.selection == null) {
        return null;
      } else if (this.selection.type == 'pack') {
        return checkNotNil(
            this.$tstore.state.draft.packs.get(this.selection.id));
      } else {
        const player = this.$tstore.state.draft.seats[this.selection.id];
        if (player.queuedPacks.length > 0) {
          return player.queuedPacks[0];
        } else {
          return null;
        }
      }
    },

    nextEventForSeat(): TimelineEvent | null {
      if (this.selectedSeat == null) {
        return null;
      }
      const eventPos = this.$tstore.state.eventPos;
      const events = this.$tstore.state.events;
      for (let i = eventPos; i < events.length; i++) {
        const event = events[i];
        if (event.associatedSeat == this.selectedSeat.position) {
          return event;
        }
      }
      return null;
    },

  },

  methods: {
    getSelectionClass(cardId: number) {
      if (this.nextEventForSeat == null) {
        return undefined;
      }
      for (const action of this.nextEventForSeat.actions) {
        if (action.type == 'move-card' && action.card == cardId) {
          if (action.to == this.selectedSeat!.player.picks.id) {
            return 'action-picked'
          } else {
            return 'action-returned';
          }
        }
      }
      return undefined;
    }
  },

});
</script>

<style scoped>

._card-grid {
  border-top: 1px solid black;
}

.player-grid, .selected-pack {
  display: flex;
  flex-direction: row;
  flex-wrap: wrap;
  padding: 10px;
}

.selected-pack {
  border-bottom: 1px solid black;
}

.card {
  margin: 0 10px 10px 0;
}

/* native is 745 × 1040 */
.card-img {
  width: 200px;
  height: 279px;
  background: #AAA;
}

.action-picked {
  outline: 5px solid #00F;
}

.action-returned {
  outline: 5px solid #F00;
}

</style>
