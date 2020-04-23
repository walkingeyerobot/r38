<template>
  <div class="_card-grid">

    <div
        v-if="selectedPack"
        class="selected-pack"
        >
      <div
          v-for="card in selectedPack"
          :key="card.id"
          class="card"
          @click="onPackCardClicked(card.id)"
          >
        <img
            class="card-img"
            :class="getSelectionClass(card.id)"
            :title="card.definition.name"
            :src="getImageSrc(card)"
            >
      </div>
    </div>

    <div
        class="pool-label"
        v-if="selectedSeat && selectedPack"
        >
      Drafted
    </div>
    <div
        v-if="selectedSeat"
        class="player-grid"
        >
      <div
          v-for="card in selectedSeat.player.picks.cards"
          :key="card.id"
          class="card"
          @click="onPoolCardClicked(card.id)"
          >
        <img
            class="card-img"
            :title="card.definition.name"
            :src="getImageSrc(card)"
            >
      </div>
    </div>

  </div>
</template>

<script lang="ts">
import Vue from 'vue'
import { DraftCard, DraftPlayer, DraftSeat, CardPack, CardContainer } from '../../draft/DraftState'
import { TimelineEvent } from '../../draft/TimelineEvent';
import { SelectedView } from '../../state/selection';
import { checkNotNil } from '../../util/checkNotNil';
import { navTo } from '../../router/url_manipulation';
import { Store } from 'vuex';
import { RootState } from '../../state/store';

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

    selectedPack(): DraftCard[] | null {
      let pack: CardContainer | null = null;

      if (this.selection == null) {
        return null;
      } else if (this.selection.type == 'pack') {
        pack =
            checkNotNil(this.$tstore.state.draft.packs.get(this.selection.id));
      } else {
        const player = this.$tstore.state.draft.seats[this.selection.id];
        if (player.queuedPacks.length > 0) {
          pack = player.queuedPacks[0];
        }
      }

      if (pack != null) {
        return pack.cards
            .concat()
            .sort((a, b) => a.sourcePackIndex - b.sourcePackIndex);
      } else {
        return null;
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
    },

    getImageSrc(card: DraftCard): string {
      if (process.env.NODE_ENV == 'development') {
        return `http://api.scryfall.com/cards/${card.definition.set}/`
            + `${card.definition.collector_number}?format=image&version=normal`;
      } else {
        return `/proxy/${card.definition.set}/`
            + `${card.definition.collector_number}`;
      }
    },

    onPackCardClicked(cardId: number) {
      this.jumpToPick(cardId, 'future');
    },

    onPoolCardClicked(cardId: number) {
      this.jumpToPick(cardId, 'past');
    },

    jumpToPick(cardId: number, direction: 'future' | 'past') {
      const pick =
          findPick(
              cardId,
              this.$tstore.state.eventPos,
              this.$tstore.state.events,
              direction);
      if (pick != null) {
        const adjustedIndex =
            maybeAdjustToStartOfEpoch(this.$tstore, pick.index);

        navTo(this.$tstore, this.$route, this.$router, {
          eventIndex: adjustedIndex,
          selection: {
            type: 'seat',
            id: pick.seat,
          },
        });
      }
    }
  },
});

function maybeAdjustToStartOfEpoch(store: Store<RootState>, index: number) {
  let newIndex = index;
  const event = store.state.events[index];
  if (event != undefined && store.state.timeMode == 'synchronized') {
    for (let i = index; i >= 0; i--) {
      if (store.state.events[i].roundEpoch != event.roundEpoch) {
        break;
      }
      newIndex = i;
    }
  }
  return newIndex;
}

function findPick(
  cardId: number,
  currentIndex: number,
  events: TimelineEvent[],
  direction: 'future' | 'past',
) {
  if (direction == 'future') {
    for (let i = currentIndex; i < events.length; i++) {
      const event = events[i];
      if (containsPick(event, cardId)) {
        return {
          index: i,
          seat: event.associatedSeat,
        };
      }
    }
  } else {
    for (let i = currentIndex; i >= 0; i--) {
      const event = events[i];
      if (containsPick(event, cardId)) {
        return {
          index: i,
          seat: event.associatedSeat,
        };
      }
    }
  }
  return null;
}

function containsPick(event: TimelineEvent, cardId: number) {
  for (let action of event.actions) {
    if (action.type == 'move-card' && action.card == cardId) {
      return true;
    }
  }
  return false;
}

</script>

<style scoped>

._card-grid {
  padding-top: 10px;
  padding-left: 10px;
  overflow-y: scroll;
}

.player-grid, .selected-pack {
  display: flex;
  flex-direction: row;
  flex-wrap: wrap;
  padding: 10px;
}

.pool-label {
  border-top: 1px solid #EAEAEA;
  padding-top: 10px;
  margin-left: 10px;
  margin-right: 10px;
  margin-bottom: 5px;
}

.card {
  margin: 0 10px 10px 0;
  cursor: pointer;
  position: relative;
  display: flex; /* so we perfectly wrap the enclosed image */
}

.card::before {
  content: '';
  position: absolute;
  left: 0;
  top: 0;
  width: 100%;
  height: 100%;
  box-shadow: 0 0 4px 1.2px rgba(0, 0, 0, 0.7);
  opacity: 0;
  transition: opacity 110ms cubic-bezier(0.33, 1, 0.68, 1);
  border-radius: 9px;
}

.card:hover::before {
  opacity: 1;
}

/* native is 745 × 1040 */
.card-img {
  width: 200px;
  height: 279px;
  background: #AAA;
  border-radius: 9px;
  overflow: hidden;
}

.action-picked {
  outline: 5px solid #00F;
}

.action-returned {
  outline: 5px solid #F00;
}

</style>
