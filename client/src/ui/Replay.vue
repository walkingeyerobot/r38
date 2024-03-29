<template>
  <div
      class="_replay"
      @mousedown.capture="onCaptureMouseDown"
      @mousedown="onBubbleMouseDown"
      >
    <template v-if="status == 'loaded'">
      <ReplayMobile
          v-if="formatStore.layout == 'mobile'"
          :showDraftPicker="showDraftPicker"
          />
      <ReplayDesktop v-else :showDraftPicker="showDraftPicker" />
    </template>
  </div>
</template>

<script lang="ts">
import { defineComponent } from 'vue';
import ReplayDesktop from './replay/ReplayDesktop.vue';
import ReplayMobile from './replay/ReplayMobile.vue';

import { rootStore } from '../state/store';
import { authStore } from '../state/AuthStore';
import { formatStore, FormatStore } from '../state/FormatStore';
import { replayStore } from '../state/ReplayStore';
import { draftStore, DraftStore } from '../state/DraftStore';
import { applyReplayUrlState, parseDraftUrl, pushDraftUrlFromState } from '../router/url_manipulation';
import { globalClickTracker } from './infra/globalClickTracker';
import { getPlayerSeat } from '../state/util/userIsSeated';
import { tuple } from '../util/tuple';
import { fetchEndpoint } from '../fetch/fetchEndpoint';
import { routeDraft } from '../rest/api/draft/draft';
import { FetchStatus } from './infra/FetchStatus';
import { isAuthedUserSelected } from './replay/isAuthedUserSelected';


export default defineComponent({
  components: {
    ReplayMobile,
    ReplayDesktop,
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
    formatStore(): FormatStore {
      return formatStore;
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

    getDefaultSeatSelection(): number {
      let seat = getPlayerSeat(authStore.user?.id, draftStore.currentState);
      return seat?.position || 0;
    },
  },
});

</script>

<style scoped>
._replay {
  height: 100%;
}
</style>
