<!-- eslint-disable vue/multi-word-component-names -->
<template>
  <div class="_replay" @mousedown.capture="onCaptureMouseDown" @mousedown="onBubbleMouseDown">
    <template v-if="status == 'loaded'">
      <ReplayMobile v-if="layout == 'mobile'" :showDraftPicker="showDraftPicker" />
      <ReplayDesktop v-else :showDraftPicker="showDraftPicker" />
    </template>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from "vue";
import ReplayDesktop from "./replay/ReplayDesktop.vue";
import ReplayMobile from "./replay/ReplayMobile.vue";

import { rootStore } from "@/state/store";
import { authStore } from "@/state/AuthStore";
import { formatStore } from "@/state/FormatStore";
import { replayStore } from "@/state/ReplayStore";
import { draftStore } from "@/state/DraftStore";
import {
  parseDraftUrl,
  applyReplayUrlState,
  pushDraftUrlFromState,
} from "@/router/url_manipulation";
import { globalClickTracker } from "./infra/globalClickTracker";
import { getPlayerSeat } from "@/state/util/userIsSeated";
import { tuple } from "@/util/tuple";
import { fetchEndpoint } from "@/fetch/fetchEndpoint";
import { ROUTE_DRAFT } from "@/rest/api/draft/draft";
import type { FetchStatus } from "./infra/FetchStatus";
import { isAuthedUserSelected } from "./replay/isAuthedUserSelected";
import { useRoute, useRouter } from "vue-router";

const targetDraftId = ref<number>(-1);
const status = ref<FetchStatus>("missing");
const isFreshBundle = ref<boolean>(false);
const unwatchDraftStore = ref<(() => void) | null>(null);

const route = useRoute();
const router = useRouter();

onMounted(() => {
  unwatchDraftStore.value = rootStore.watch(
    (_state) => tuple(draftStore.initialState, draftStore.events),
    (_newProps, _oldProps) => onDraftStoreChanged(),
  );
  applyCurrentRoute();
});

onUnmounted(() => {
  if (unwatchDraftStore.value) {
    unwatchDraftStore.value();
  }
});

watch(route, (_to, _from) => applyCurrentRoute());

const layout = computed(() => formatStore.layout);

const showDraftPicker = computed(
  () =>
    draftStore.isFilteredDraft &&
    replayStore.eventPos == replayStore.events.length &&
    isAuthedUserSelected(authStore, draftStore, replayStore),
);

function applyCurrentRoute() {
  const parsedUrl = parseDraftUrl(route);
  if (parsedUrl.draftId != targetDraftId.value) {
    fetchDraft(parsedUrl.draftId);
  } else {
    applyReplayUrlState(replayStore, route);
  }
}

async function fetchDraft(draftId: number) {
  status.value = "fetching";
  targetDraftId.value = draftId;

  // TODO: Handle errors
  const payload = await fetchEndpoint(ROUTE_DRAFT, {
    id: draftId.toString(),
    as: authStore.userId,
  });

  if (payload.draftId != targetDraftId.value) {
    return;
  }

  draftStore.loadDraft(payload);

  if (!draftStore.isComplete && authStore.userId === 0) {
    if (unwatchDraftStore.value) {
      unwatchDraftStore.value();
    }
    await router.replace({ name: "login" });
    return;
  }

  document.title = `${draftStore.draftName}`;

  isFreshBundle.value = true;
  status.value = "loaded";

  // onDraftStoreChanged will fire afterwards
}

function onDraftStoreChanged() {
  console.log("Draft state changed, resyncing replay");
  replayStore.sync();

  if (isFreshBundle.value) {
    console.log("Syncing state to URL...");
    if (replayStore.selection == null) {
      replayStore.setSelection({
        type: "seat",
        id: getDefaultSeatSelection(),
      });
    }
    applyReplayUrlState(replayStore, route);
  } else {
    console.log("Syncing URL to state...");
    pushDraftUrlFromState({ $route: route, $router: router }, draftStore, replayStore);
  }
  isFreshBundle.value = false;
}

function onCaptureMouseDown() {
  globalClickTracker.onCaptureGlobalMouseDown();
}

function onBubbleMouseDown(e: MouseEvent) {
  globalClickTracker.onBubbleGlobalMouseDown(e);
}

function getDefaultSeatSelection(): number {
  const seat = getPlayerSeat(authStore.userId, draftStore.currentState);
  return seat?.position || 0;
}
</script>

<style scoped>
._replay {
  height: 100%;
}
</style>
