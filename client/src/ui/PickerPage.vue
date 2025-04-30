<!--

The primary page for making picks during an active draft

-->
<template>
  <div class="_picker-page">
    <div class="container">
      <template v-if="loaded">
        <div class="header">
          {{ draftStore.draftName }}
        </div>

        <TableSeating
          v-if="nextPick.kind == 'NotSeated' || nextPick.kind == 'WaitingToFill'"
          class="table-seating"
        />

        <template v-else>
          <div class="player-list">
            <component
              :is="isDevMode ? 'a' : 'div'"
              v-for="seat in draftStore.currentState.seats"
              :key="seat.position"
              class="seated-player"
              :href="`${route.path}?as=${seat.player.id}`"
            >
              <img
                class="player-icon"
                :src="seat.player.iconUrl"
                :title="seat.player.name"
                :class="{ active: seat.player.id == authStore.user?.id }"
              />
            </component>
          </div>

          <div class="pick-count" v-if="nextPick.kind == 'ActivePosition'">
            <div class="pc-round">
              <div class="pc-round-label">Pack</div>
              <div class="pc-round-count">{{ nextPick.round }}</div>
            </div>
            <div class="pc-pick">
              <div class="pc-pick-label">Pick</div>
              <div class="pc-pick-count">{{ nextPick.pick + 1 }}</div>
            </div>
          </div>
          <div class="pick-count-error" v-else-if="nextPick.kind == 'Error'">
            {{ nextPick.message }}
          </div>

          <div class="active-pack">
            <template v-if="activePack">
              <div class="active-pack-label">Current pack</div>
              <div class="active-pack-cnt">
                <div
                  v-for="card in activePack"
                  :key="card.id"
                  class="card-slate"
                  @click="onCardClick(card)"
                >
                  <div class="card-name">{{ card.definition.name }}</div>
                  <div class="card-cost">
                    <ManaSymbol
                      v-for="(msymb, i) in card.definition.mana_cost"
                      :key="i"
                      :code="msymb"
                      class="mana-symbol"
                    />
                  </div>
                </div>
              </div>
            </template>
            <template v-else><em>Waiting for pack</em></template>
          </div>

          <div v-if="previousPicks" class="previous-pick" @click="onPreviousPickClick">
            <div class="active-pack-label">Previous pick</div>

            <div v-for="card in previousPicks" :key="card.id" class="card-slate previous">
              <div class="card-name">{{ card.definition.name }}</div>
              <div class="card-cost">
                <ManaSymbol
                  v-for="(msymb, i) in card.definition.mana_cost"
                  :key="i"
                  :code="msymb"
                  class="mana-symbol"
                />
              </div>
            </div>
          </div>
        </template>

        <CardDetailDialog
          v-if="activeDialog?.kind == 'card-detail'"
          :card="draftStore.getCard(activeDialog.cardId)"
          class="fullscreen-page"
          @close="activeDialog = null"
          @pick="onPickCard"
        />
        <PreviousPickDialog
          v-else-if="activeDialog?.kind == 'previous-pick'"
          :cards="activeDialog.cards"
          class="fullscreen-page"
          @close="activeDialog = null"
          @undo="onUndoPick"
        />
        <LoadingDialog
          v-else-if="activeDialog?.kind == 'loading-spinner'"
          class="fullscreen-page"
        />
        <DismissableDialog
          v-else-if="activeDialog?.kind == 'error-dialog'"
          class="fullscreen-page"
          @close="activeDialog = null"
        >
          <template #header>Flagrant System Error</template>
          <div class="error-msg">{{ activeDialog.message }}</div>
          <div v-if="activeDialog.escapedMessage" class="error-msg-escaped">
            {{ activeDialog.escapedMessage }}
          </div>
        </DismissableDialog>
      </template>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from "vue";
import { useRoute } from "vue-router";
import { useSound } from "@vueuse/sound";
import axios from "axios";

import PreviousPickDialog from "@/ui/picker/PreviousPickDialog.vue";
import DismissableDialog from "@/ui/picker/DismissableDialog.vue";
import CardDetailDialog from "@/ui/picker/CardDetailDialog.vue";
import LoadingDialog from "@/ui/picker/LoadingDialog.vue";
import ManaSymbol from "@/ui/shared/mana/ManaSymbol.vue";

import { fetchEndpoint, fetchEndpointEv } from "@/fetch/fetchEndpoint";
import { ROUTE_DRAFT } from "@/rest/api/draft/draft";
import { ROUTE_UNDO_PICK } from "@/rest/api/undopick/undopick";
import { ROUTE_PICK } from "@/rest/api/pick/pick";
import { ROUTE_PICK_RFID } from "@/rest/api/pickrfid/pickrfid";
import { authStore } from "@/state/AuthStore";
import { draftStore, type DraftStore } from "@/state/DraftStore";
import { isPickEvent } from "@/state/util/isPickEvent";
import type { DraftCard, DraftState } from "@/draft/DraftState";
import { PickerSounds } from "@/ui/picker/PickerSounds";
import TableSeating from "@/ui/picker/TableSeating.vue";

const route = useRoute();

const loaded = ref(false);
const activeDialog = ref<ActiveDialog | null>(null);
const draftId = parseInt(route.params["draftId"] as string);
const activeRequest = ref<boolean>(false);
const isDevMode = import.meta.env.DEV;

onMounted(() => {
  console.log("---- Draft ID is", draftId);

  document.body.addEventListener("rfidScan", onRfidScan);

  fetchDraft();
});

onUnmounted(() => {
  document.body.removeEventListener("rfidScan", onRfidScan);
});

const playerSeat = computed(() => {
  if (authStore.user?.id == null) {
    return null;
  }
  return findPlayerSeat(authStore.user.id, draftStore.currentState);
});

const activePack = computed(() => {
  const seat = playerSeat.value;
  if (seat == null) {
    return null;
  }

  const firstQueuedPack = seat.queuedPacks.packs[0];
  if (firstQueuedPack.round == seat.round && firstQueuedPack.cards.length > 0) {
    return firstQueuedPack.cards.map((id) => draftStore.getCard(id));
  } else {
    return null;
  }
});

const previousPicks = computed(() => {
  const seat = playerSeat.value;
  if (seat == null) {
    return null;
  }
  const pickEvent = getMostRecentPickEvent(draftStore, seat.position);
  if (pickEvent != null) {
    const pickedCards = [] as DraftCard[];
    for (const action of pickEvent.actions) {
      if (action.type == "move-card" && action.to == seat.picks.id) {
        pickedCards.push(draftStore.getCard(action.card));
      }
    }
    if (pickedCards.length > 0) {
      return pickedCards;
    }
  }
  return null;
});

const nextPick = computed<PickPosition>(() => {
  const userId = authStore.user?.id;

  if (userId == null) {
    return pickPositionError("User must be authed");
  }

  const seat = findPlayerSeat(userId, draftStore.currentState);

  if (seat == null) {
    return { kind: "NotSeated" };
  }

  const pickEvent = getMostRecentPickEvent(draftStore, seat.position);

  let nextPick: PickPosition;
  if (pickEvent == null) {
    if (draftStore.hasSeatsAvailable) {
      nextPick = { kind: "WaitingToFill" };
    } else {
      nextPick = { kind: "ActivePosition", round: 1, pick: 0 };
    }
  } else {
    nextPick = incrementPick(pickEvent.round, pickEvent.pick);
  }

  return nextPick;
});

const sounds = computed(() => {
  return {
    scan: PickerSounds.getScanSoundForSeat(playerSeat.value?.position),
    error: PickerSounds.getErrorSoundForSeat(playerSeat.value?.position),
  };
});

const scanSound = useSound(sounds.value.scan);
const errorSound = useSound(sounds.value.error);

function onCardClick(card: DraftCard) {
  activeDialog.value = {
    kind: "card-detail",
    cardId: card.id,
  };
}

function onPreviousPickClick() {
  if (previousPicks.value != null) {
    activeDialog.value = {
      kind: "previous-pick",
      cards: previousPicks.value,
    };
    scanSound.play();
  }
}

async function onRfidScan(event: CustomEvent<string>) {
  if (activeRequest.value) {
    return;
  }
  activeRequest.value = true;
  activeDialog.value = {
    kind: "loading-spinner",
    message: "Picking scanned card...",
  };

  const cardRfid = decodeURIComponent(
    Array.prototype.map
      .call(atob(event.detail), (c) => `%${`00${c.charCodeAt(0).toString(16)}`.slice(-2)}`)
      .slice(3)
      .join(""),
  );

  const [response, e] = await fetchEndpointEv(ROUTE_PICK_RFID, {
    draftId: draftStore.draftId,
    cardRfids: [cardRfid],
    xsrfToken: draftStore.pickXsrf,
    as: authStore.user?.id,
  });

  if (response) {
    draftStore.loadDraft(response);
    activeDialog.value = null;
    scanSound.play();
  } else {
    activeDialog.value = {
      kind: "error-dialog",
      message: `An error occurred while scanning card with ID "${cardRfid}"`,
      escapedMessage: getErrorMessage(e),
    };
    errorSound.play();
  }

  activeRequest.value = false;
}

async function onPickCard(card: DraftCard) {
  if (activeRequest.value) {
    return;
  }
  activeRequest.value = true;

  console.log("Pick card", card.id, card.definition.name);

  activeDialog.value = {
    kind: "loading-spinner",
    message: "Picking card...",
  };

  const [response, e] = await fetchEndpointEv(ROUTE_PICK, {
    cards: [card.id],
    xsrfToken: draftStore.pickXsrf,
    as: authStore.user?.id,
  });

  if (response) {
    draftStore.loadDraft(response);

    activeDialog.value = null;
    scanSound.play();
  } else {
    activeDialog.value = {
      kind: "error-dialog",
      message: `An error occurred while picking "${card.definition.name}"`,
      escapedMessage: getErrorMessage(e),
    };
    errorSound.play();
  }

  activeRequest.value = false;
}

async function onUndoPick(card: DraftCard) {
  activeDialog.value = {
    kind: "loading-spinner",
    message: "",
  };

  const [response, e] = await fetchEndpointEv(ROUTE_UNDO_PICK, {
    draftId: draftStore.draftId,
    xsrfToken: draftStore.pickXsrf,
    as: authStore.user?.id,
  });

  if (response) {
    draftStore.loadDraft(response);
    activeDialog.value = null;
  } else {
    activeDialog.value = {
      kind: "error-dialog",
      message: `An error occurred while undoing user ${authStore.user?.id}'s pick`,
      escapedMessage: getErrorMessage(e),
    };
    errorSound.play();
  }
}

async function fetchDraft() {
  const payload = await fetchEndpoint(ROUTE_DRAFT, {
    id: draftId.toString(),
    as: authStore.user?.id,
  });
  draftStore.loadDraft(payload);
  loaded.value = true;
}
</script>

<!-- Static definitions -->

<script lang="ts">
function findPlayerSeat(playerId: number, draft: DraftState) {
  for (const seat of draft.seats) {
    if (seat.player.id == playerId) {
      return seat;
    }
  }
  return null;
}

function getMostRecentPickEvent(store: DraftStore, seatId: number) {
  for (let i = store.events.length - 1; i >= 0; i--) {
    const event = store.events[i];
    if (event.associatedSeat == seatId && isPickEvent(event)) {
      return event;
    }
  }
}

function incrementPick(round: number, pick: number): PickPosition {
  // Remember that rounds are 1-indexed and picks are 0-indexed

  if (pick + 1 < CARDS_PER_PACK) {
    return {
      kind: "ActivePosition",
      round,
      pick: pick + 1,
    };
  } else if (round + 1 <= ROUNDS_PER_DRAFT) {
    return {
      kind: "ActivePosition",
      round: round + 1,
      pick: 0,
    };
  } else {
    return {
      kind: "DraftComplete",
    };
  }
}

function getErrorMessage(e: unknown): string {
  if (axios.isAxiosError(e)) {
    let message = `${axios.getUri(e.config)}\n\n${e.message}\n\n`;
    if (e.response) {
      message += `${e.response.status} ${e.response.statusText}\n\n`;
      message += JSON.stringify(e.response.data, undefined, 2);
    }
    return message;
  }
  if (e instanceof Error) {
    return e.message;
  } else {
    return JSON.stringify(e, undefined, 2);
  }
}

function pickPositionError(message: string): PickPosition {
  return {
    kind: "Error",
    message,
  };
}

type PickPosition =
  | {
      kind: "NotSeated";
    }
  | {
      kind: "WaitingToFill";
    }
  | {
      kind: "ActivePosition";
      round: number;
      pick: number;
    }
  | {
      kind: "DraftComplete";
    }
  | {
      kind: "Error";
      message: string;
    };

type ActiveDialog =
  | {
      kind: "card-detail";
      cardId: number;
    }
  | {
      kind: "previous-pick";
      cards: DraftCard[];
    }
  | {
      kind: "loading-spinner";
      message: string;
    }
  | {
      kind: "error-dialog";
      message: string;
      escapedMessage?: string;
    };

// TODO: Move these into a per-draft config object
const ROUNDS_PER_DRAFT = 3;
const CARDS_PER_PACK = 15;
</script>

<style scoped>
._picker-page {
  background: #333;
  color: white;

  height: 100%;

  overflow-y: auto;

  display: flex;
  flex-direction: column;
  align-items: center;
}

.container {
  display: flex;
  flex-direction: column;
  max-width: 450px;
  width: 100%;
}

.header {
  font-size: 25px;
  text-align: center;
  margin: 30px 0;
}

.player-list {
  display: flex;
  justify-content: center;
  padding: 7px 9px;
  width: fit-content;
  align-self: center;

  border-radius: 100px;
  background-color: #1b1b1b;
}

.player-icon {
  width: 40px;
  height: 40px;
  border-radius: 40px;
}

.player-icon.active {
  outline: 3px solid white;
}

.seated-player {
  display: flex;
}

.seated-player + .seated-player {
  margin-left: 5px;
}

.pick-count {
  aspect-ratio: 1 / 1;
  margin: 100px 40px 80px;
  flex-shrink: 0;

  position: relative;

  background-image: url("./picker/pick_medallion.svg");
  background-size: contain;
}

.pc-round,
.pc-pick {
  display: flex;
  justify-content: center;
  align-items: center;
  width: 0;
  height: 0;
}

.pc-round {
  position: absolute;
  left: 31%;
  top: 31%;
}

.pc-pick {
  position: absolute;
  left: 61%;
  top: 70%;
}

.pc-round-label,
.pc-pick-label {
  font-size: 20px;
}

.pc-round-count,
.pc-pick-count {
  font-size: 100px;
  margin-left: 5px;
}

.active-pack {
  margin: 20px 0px;
  padding: 0 20px;
}

.active-pack-label {
  margin-bottom: 10px;
  font-style: italic;
}

.active-pack-cnt {
  display: flex;
  flex-direction: column;
}

.card-slate {
  background: #1b1b1b;
  border-radius: 20px;
  padding: 20px;
  display: flex;
  align-items: center;
  cursor: pointer;
  user-select: none;
}

.card-slate:hover {
  outline: 1px solid #555;
}

.card-slate + .card-slate {
  margin-top: 10px;
}

.mana-symbol {
  width: 17px;
  height: 17px;
}

.mana-symbol + .mana-symbol {
  margin-left: 2.5px;
}

.card-name {
  color: white;
  flex: 1;
}

.previous-pick {
  padding: 0 20px;
  margin-top: 40px;
  margin-bottom: 50px;
}

.card-slate.previous {
  background-color: #3b3b3b;
}

.fullscreen-page {
  position: absolute;
  left: 0;
  top: 0;
  width: 100%;
  height: 100%;
}

.error-msg-escaped {
  margin-top: 40px;
  white-space: pre-wrap;
  font-family: "Courier New", Courier, monospace;
}
</style>
