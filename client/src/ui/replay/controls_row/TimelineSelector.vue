<!--

Displays the current draft's event history as a scrollable list.

Clicking on an individual entry will jump to that point in the event stream.
In synchronized mode, shows one entry per pick. In temporal mode, each event
gets its own entry.

-->

<template>
  <div class="_timeline-selector">
    <div class="title">Timeline</div>

    <div class="list-cnt">
      <template v-for="entry in listEntries">
        <div
            v-if="entry.type == 'synchronized-header'"
            :key="`round_${entry.round}`"
            class="sync-header"
            >
          Pack {{ entry.round }}
        </div>

        <div
            v-else-if="entry.type == 'synchronized-pick'"
            :key="entry.eventId"
            class="sync-pick"
            :class="{
              selected: entry.eventId == currentEventId
            }"
            @click="onSyncPickClicked(entry.eventId)"
            >
          Pick {{ entry.pick + 1 }}
        </div>

        <div
            v-else-if="entry.type == 'temporal-pick'"
            :key="entry.eventId"
            class="temp-pick"
            :class="{
              selected: entry.eventId == currentEventId
            }"
            @click="onTempPickClicked(entry.eventId, entry.seatId)"
            >
          <div class="event-id">{{ entry.eventId }}</div>
          <div class="temp-pick-message">
            <span class="temp-pick-pack">
              p{{ entry.round }}p{{ entry.pick + 1 }}
            </span>
            {{ entry.playerName }}
            picked
            <span class="picked-card-name">{{ entry.cardName }}</span>
          </div>
        </div>

        <div
            v-else-if="entry.type == 'temporal-other'"
            :key="entry.eventId"
            class="temp-other"
            >
          <span class="event-id">{{ entry.eventId }}</span>
          Nothing picked
        </div>
      </template>
    </div>

    <div class="synchronized-cnt">
      <input
          type="checkbox"
          id="synchronize-checkbox"
          v-model="synchronizeTimeline">
      <label
          for="synchronize-checkbox"
          class="synchronize-label"
          >
        Synchronize timeline
      </label>
    </div>

  </div>
</template>

<script lang="ts">
import Vue from 'vue';
import { CoreState } from '../../../state/store';
import { navTo } from '../../../router/url_manipulation';
import { isPickEvent, getPickAction } from '../../../state/util/isPickEvent';
import { find } from '../../../util/collection';
import { TimelineEvent } from '../../../draft/TimelineEvent';


export default Vue.extend({
  computed: {
    state(): CoreState {
      return this.$tstore.state;
    },

    synchronizeTimeline: {
      get() {
        return this.$tstore.state.timeMode == 'synchronized';
      },

      set(value) {
        this.$tstore.commit('setTimeMode', value ? 'synchronized' : 'original');
        navTo(this.$tstore, this.$route, this.$router, {});
      }
    },

    listEntries(): ListEntry[] {
      if (this.state.timeMode == 'synchronized') {
        return this.computeSynchronizedList();
      } else {
        return this.computeTemporalList();
      }
    },

    currentEventId(): number {
      const event = this.state.events[this.state.eventPos];
      return event != undefined ? event.id : -1;
    }
  },

  methods: {
    onSyncPickClicked(eventId: number) {
      const index = find(this.state.events, { id: eventId });
      if (index != -1) {
        navTo(this.$tstore, this.$route, this.$router, {
          eventIndex: index,
        });
      } else {
        console.warn(`Can't find event ID`, eventId);
      }
    },

    onTempPickClicked(eventId: number, seatId: number) {
      const index = find(this.state.events, { id: eventId });
      if (index != -1) {
        navTo(this.$tstore, this.$route, this.$router, {
          eventIndex: index,
          selection: {
            type: 'seat',
            id: seatId,
          }
        });
      } else {
        console.warn(`Can't find event ID`, eventId);
      }
    },

    computeSynchronizedList(): ListEntry[] {
      let currentRound = 0;
      let currentPick = -1;
      const entries = [] as ListEntry[];

      for (let event of this.state.events) {
        if (event.round != currentRound) {
          entries.push({
            type: 'synchronized-header',
            round: event.round,
          });
          currentRound++;
          currentPick = -1;
        }

        if (event.pick != currentPick && isPickEvent(event)) {
          entries.push({
            type: 'synchronized-pick',
            pick: event.pick,
            eventId: event.id,
          });
          currentPick = event.pick;
        }
      }

      return entries;
    },

    computeTemporalList() {
      const entries = [] as ListEntry[];
      for (let event of this.state.events) {
        const pickAction = getPickAction(event);

        if (pickAction != null) {
          entries.push({
            type: 'temporal-pick',
            eventId: event.id,
            seatId: event.associatedSeat,
            round: event.round,
            pick: event.pick,
            playerName:
                this.state.draft.seats[event.associatedSeat].player.name,
            cardName: pickAction.cardName,
          });
        } else {
          entries.push({
            type: 'temporal-other',
            eventId: event.id,
          });
        }
      }
      return entries;
    },
  },
});

type ListEntry =
    | SynchronizedHeaderEntry
    | SynchronizedPickEntry
    | TemporalPickEntry
    | TemporalOtherEntry
    ;

interface SynchronizedHeaderEntry {
  type: 'synchronized-header',
  round: number,
}

interface SynchronizedPickEntry {
  type: 'synchronized-pick',
  pick: number,
  eventId: number,
}

interface TemporalPickEntry {
  type: 'temporal-pick',
  eventId: number,
  round: number,
  pick: number,
  playerName: string,
  cardName: string,
  seatId: number,
}

interface TemporalOtherEntry {
  type: 'temporal-other',
  eventId: number,
}

</script>

<style scoped>

.title {
  font-size: 14px;
  margin-top: 8px;
  margin-bottom: 8px;
  text-align: center;
}

._timeline-selector {
  display: flex;
  flex-direction: column;
}

.list-cnt {
  flex: 1 0 0;
  overflow-y: scroll;
  padding-top: 10px;
  padding-bottom: 15px;
  user-select: none;
  cursor: default;
  border-top: 1px solid #eaeaea;
  border-bottom: 1px solid #eaeaea;
}

.selected {
  background-color: #fff6cb;
}

.sync-header {
  font-size: 14px;
  color: #a2310d;
  margin-bottom: 4px;
  padding-left: 10px;
}

.sync-header:not(:first-child) {
  margin-top: 10px;
}

.sync-pick {
  font-size: 14px;
  padding: 3px 0;
  padding-left: 10px;
}

.synchronized-cnt {
  padding: 10px 10px;
}

.temp-pick, .temp-other {
  font-size: 14px;
  padding: 3px 0 3px 5px;
}

.temp-pick {
  display: flex;
}

.event-id {
  display: inline-block;
  min-width: 25px;
  flex: 0 0 auto;
}

.event-id, .temp-pick-pack, .temp-other {
  color: #aaa;
}

.picked-card-name {
  color: #1560d4;
}

.temp-pick-message {
  margin-left: 5px;
}

</style>
