<template>
  <div
      class="_replay"
      @mousedown.capture="onCaptureMouseDown"
      @mousedown="onBubbleMouseDown"
      >
    <template v-if="status == 'loaded'">
      <ControlsRow />
      <div class="main">
        <PlayerSelector class="table" />
        <DraftPicker v-if="showDraftPicker" class="picker" />
        <CardGrid v-else class="grid" />
      </div>
    </template>
  </div>
</template>

<script lang="ts">
import Vue from 'vue';
import ControlsRow from './replay/ControlsRow.vue';
import PlayerSelector from './replay/PlayerSelector.vue';
import CardGrid from './replay/CardGrid.vue';
import DraftPicker from './replay/DraftPicker.vue';

import { rootStore } from '../state/store';
import { authStore } from '../state/AuthStore';
import { replayStore } from '../state/ReplayStore';
import { draftStore, DraftStore } from '../state/DraftStore';

import { SourceData } from '../parse/SourceData';
import { SelectedView } from '../state/selection';
import { applyReplayUrlState, pushDraftUrlFromState, parseDraftUrl } from '../router/url_manipulation';
import { globalClickTracker } from './infra/globalClickTracker';
import { getUserPosition } from '../state/util/userIsSeated';
import { tuple } from '../util/tuple';
import { fetchEndpoint } from '../fetch/fetchEndpoint';
import { routeDraft } from '../rest/api/draft/draft';
import { FetchStatus } from './infra/FetchStatus';
import { DraftState } from '../draft/DraftState';
import { TimelineEvent } from '../draft/TimelineEvent';
import { isAuthedUserSelected } from './replay/isAuthedUserSelected';


export default Vue.extend({
  components: {
    ControlsRow,
    PlayerSelector,
    DraftPicker,
    CardGrid,
  },

  data() {
    return {
      targetDraftId: -1,
      status: 'missing' as FetchStatus,
      isFreshBundle: false,
      unwatchDraftStore: null as null | (() => void),
    };
  },

  created() {
    this.unwatchDraftStore = rootStore.watch(
      (state) => tuple(draftStore.initialState, draftStore.events),
      (newProps, oldProps) => this.onDraftStoreChanged(),
    );

    this.applyCurrentRoute();
  },

  destroyed() {
    if (this.unwatchDraftStore) {
      this.unwatchDraftStore();
    }
  },

  watch: {
    $route(to, from) {
      this.applyCurrentRoute();
    },
  },

  computed: {
    draftStore(): DraftStore {
      return draftStore;
    },

    showDraftPicker(): boolean {
      return draftStore.isFilteredDraft
          && replayStore.eventPos == replayStore.events.length
          && isAuthedUserSelected(authStore, draftStore, replayStore);
    },
  },

  methods: {
    applyCurrentRoute() {
      const parsedUrl = parseDraftUrl(this.$route);
      if (parsedUrl.draftId != this.targetDraftId) {
        this.fetchDraft(parsedUrl.draftId);
      } else {
        applyReplayUrlState(replayStore, this.$route);
      }
    },

    async fetchDraft(draftId: number) {
      this.status = 'fetching';
      this.targetDraftId = draftId;

      // TODO: Handle errors
      const payload = await fetchEndpoint(routeDraft, {
        id: draftId,
        as: authStore.user?.id,
      });

      if (payload.draftId != this.targetDraftId) {
        return;
      }

      draftStore.loadDraft(payload);

      document.title = `${draftStore.draftName}`;

      this.isFreshBundle = true;
      this.status = 'loaded';

      // onDraftStoreChanged will fire afterwards
    },

    onDraftStoreChanged() {
      console.log('Draft state changed, resyncing replay');
      replayStore.sync();

      if (this.isFreshBundle) {
        console.log('Syncing state to URL...');
        if (replayStore.selection == null) {
          replayStore.setSelection({
            type: 'seat',
            id: this.getDefaultSeatSelection(),
          });
        }
        applyReplayUrlState(replayStore, this.$route);
      } else {
        console.log('Syncing URL to state...');
        pushDraftUrlFromState(this, draftStore, replayStore);
      }
      this.isFreshBundle = false;
    },

    onCaptureMouseDown() {
      globalClickTracker.onCaptureGlobalMouseDown();
    },

    onBubbleMouseDown(e: MouseEvent) {
      globalClickTracker.onBubbleGlobalMouseDown(e);
    },

    getDefaultSeatSelection() {
      let position =
          getUserPosition(authStore.user?.id, draftStore.currentState);
      if (position == -1) {
        position = 0;
      }
      return position;
    },
  },
});

</script>

<style scoped>
._replay {
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

.table {
  width: 300px;
  flex: 0 0 auto;
}

.grid, .picker {
  flex: 1;
}
</style>
