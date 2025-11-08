<template>
  <div class="_join-page">
    <PickerHeader class="header" />
    <div class="content">
      <template v-if="loaded">
        <TableSeating class="table-seating" />
        <div class="footer">
          <template v-if="footerState == 'can-join'">
            <button class="footer-btn" :disabled="joinGuard.isRunning" @click="onJoinClick">
              Join
            </button>
          </template>

          <template v-else-if="footerState == 'waiting-to-start'">
            <div class="footer-msg">Waiting for seats to fill</div></template
          >

          <template v-else-if="footerState == 'ready-to-draft'">
            <div class="footer-msg">Take your seat</div>
            <button class="footer-btn" @click="onGoClick">Let's go</button>
          </template>

          <div v-if="footerError" class="footer-err">ERROR: {{ footerError }}</div>
        </div>
      </template>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from "vue";
import { useRoute, useRouter } from "vue-router";

import TableSeating from "@/ui/draft/join/TableSeating.vue";

import { fetchEndpoint } from "@/fetch/fetchEndpoint";
import { ROUTE_DRAFT } from "@/rest/api/draft/draft";
import { authStore } from "@/state/AuthStore";
import { draftStore } from "@/state/DraftStore";
import { ROUTE_JOIN_DRAFT } from "@/rest/api/join/join";
import PickerHeader from "@/ui/picker/PickerHeader.vue";
import { getErrorMessage } from "@/util/error/getErrorMessage";
import { ReentrantGuard } from "@/util/execution/ReentrantGuard";

const route = useRoute();
const router = useRouter();

const draftId = parseInt(route.params["draftId"] as string);

const loaded = ref(false);
let pollingId: number;

const fetchGuard = new ReentrantGuard<void>({ storeError: true });
const joinGuard = new ReentrantGuard<void>({ storeError: true });

onMounted(() => {
  fetchDraft();

  if (!import.meta.env.VITE_DISABLE_STATE_POLLING) {
    console.log("Starting polling...");
    pollingId = setInterval(fetchDraft, 5000) as unknown as number;
  }
});

onUnmounted(() => {
  clearInterval(pollingId);
});

const ownSeatPosition = computed(() => {
  for (const seat of draftStore.currentState.seats) {
    if (seat.player.id == authStore.userId) {
      return seat.position;
    }
  }
  return null;
});

const isSelfSeated = computed(() => {
  return ownSeatPosition.value != null;
});

const canJoin = computed(() => {
  return !isSelfSeated.value && draftStore.hasSeatsAvailable;
});

const footerState = computed(() => {
  if (draftStore.hasSeatsAvailable) {
    if (canJoin.value) {
      return "can-join";
    } else {
      return "waiting-to-start";
    }
  } else {
    if (isSelfSeated.value && draftStore.inPerson) {
      return "ready-to-draft";
    } else {
      return "ready-to-advance";
    }
  }
});

const footerError = computed(() => {
  return (
    (joinGuard.error && `Error while joining draft: ${getErrorMessage(joinGuard.error)}`) ??
    (fetchGuard.error && `Error while loading draft: ${getErrorMessage(fetchGuard.error)}`) ??
    null
  );
});

async function fetchDraft(force: boolean = false) {
  fetchGuard.runExclusive(async () => {
    const response = await fetchEndpoint(ROUTE_DRAFT, {
      id: draftId.toString(),
      as: authStore.userId,
    });

    draftStore.loadDraft(response);
    loaded.value = true;

    // Auto-advance to the drafting/spectating view if we're not in-person (for in-person drafts
    // we give the user a chance to see where they're sitting before advancing)
    if (isSelfSeated.value && !draftStore.inPerson) {
      router.replace(appendImpersonation(`/draft/${draftId}/replay`));
    }
  }, force);
}

async function onJoinClick() {
  joinGuard.runExclusive(async () => {
    await fetchEndpoint(ROUTE_JOIN_DRAFT, {
      id: draftStore.draftId,
      position: undefined,
      as: authStore.userId,
    });
    await fetchDraft(true);
  });
}

function onGoClick() {
  router.replace(appendImpersonation(`/draft/${draftId}/pick`));
}

function appendImpersonation(url: string) {
  if (route.query["as"] != undefined) {
    return url + `?as=${route.query["as"]}`;
  } else {
    return url;
  }
}
</script>

<style scoped>
._join-page {
  height: 100%;

  display: flex;
  flex-direction: column;

  color: white;
  background-color: #333;
}

.table-seating {
  margin-top: 30px;
}

.header {
  align-self: stretch;
}

.content {
  flex: 1;
  overflow-y: scroll;

  display: flex;
  flex-direction: column;
  align-items: center;
}

.footer {
  margin-top: 50px;
  align-self: stretch;
  display: flex;
  flex-direction: column;
  align-items: center;
}

.footer-btn {
  padding: 20px;
  min-width: 160px;
  font-size: 16px;
}

.footer-msg,
.footer-btn {
  margin-bottom: 20px;
}

.footer-err {
  color: rgb(255, 187, 187);
  max-width: 300px;
}
</style>
