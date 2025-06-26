<!--

The primary page for making picks during an active draft

-->
<template>
  <div class="_picker-page">
    <template v-if="loaded">
      <div class="scroll-outer">
        <div class="scroll-inner">
          <div class="header">
            <div class="header-left">
              <a class="back-link" href="/" @click.prevent="$router.push(appendImpersonation('/'))"
                >〈 Back</a
              >
            </div>
            <div class="title">{{ draftStore.draftName }}</div>
            <div class="header-right"></div>
          </div>

          <TableSeating
            v-if="nextPick.kind == 'NotSeated' || nextPick.kind == 'WaitingToFill'"
            class="table-seating"
          />

          <template v-else>
            <div class="player-list-bg"></div>
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

            <div class="boop-prompt" v-if="boopPrompt != 'none'">
              <template v-if="boopPrompt == 'request-permission'">
                <div>To scan cards, you must first enable booping.</div>
                <div>
                  <button class="boop-btn" @click="onRequestNfcPermission">Enable booping</button>
                </div>
              </template>
              <template v-else>
                <div>Your hardware doesn't appear to support card scanning 😔.</div>
                <div style="margin-top: 10px">
                  To participate in an in-person draft you'll need a device with an RFID reader
                  (such as a phone).
                </div>
              </template>
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
        </div>
      </div>
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
      <LoadingDialog v-else-if="activeDialog?.kind == 'loading-spinner'" class="fullscreen-page" />
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
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, reactive, ref } from "vue";
import { useRoute } from "vue-router";
import { useSound } from "@vueuse/sound";
import axios from "axios";

import PreviousPickDialog from "@/ui/picker/PreviousPickDialog.vue";
import DismissableDialog from "@/ui/picker/DismissableDialog.vue";
import CardDetailDialog from "@/ui/picker/CardDetailDialog.vue";
import LoadingDialog from "@/ui/picker/LoadingDialog.vue";
import ManaSymbol from "@/ui/shared/mana/ManaSymbol.vue";
import zone from "@/sfx/zone.mp3";

import { fetchEndpointEv } from "@/fetch/fetchEndpoint";
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

const isNfcSupported = "NDEFReader" in window;
const hasNfcPermission = ref<boolean>(false);

let fetchingDraft = false;
let pollingId: number;
let wakeLock: WakeLockSentinel | null = null;

const isDevMode = import.meta.env.DEV;
const devOptions: DevOptions = reactive({
  boopPrompt: null,
  pollState: false,
});

onMounted(() => {
  console.log("---- Draft ID is", draftId);

  // Tell iOS app to start scanning
  postMessage("scan");
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  (window as any).webkit?.messageHandlers?.scanner?.postMessage("scan");

  document.body.addEventListener("rfidScan", onRfidScan);

  fetchDraft();

  if (!isDevMode || devOptions.pollState) {
    pollingId = setInterval(onPollForState, 5000) as unknown as number;
  }

  if ("wakeLock" in navigator) {
    navigator.wakeLock.request("screen").then(
      (lock) => {
        console.log("Got a wake lock!");
        wakeLock = lock;
      },
      (e) => {
        console.log("Error when trying to acquire wake lock", e);
      },
    );
  } else {
    console.log("Wake locks not supported");
  }

  handleNfcScanning();
});

onUnmounted(() => {
  document.body.removeEventListener("rfidScan", onRfidScan);

  clearInterval(pollingId);

  wakeLock?.release();
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
  if (firstQueuedPack && firstQueuedPack.round == seat.round && firstQueuedPack.cards.length > 0) {
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

const boopPrompt = computed<BoopPrompt>(() => {
  if (isDevMode && devOptions.boopPrompt != null) {
    return devOptions.boopPrompt;
  }

  const isAppleMobileOs = navigator.platform.substring(0, 2) == "iP";

  if (
    !draftStore.inPerson ||
    hasNfcPermission.value ||
    isAppleMobileOs ||
    activePack.value != null
  ) {
    return "none" as const;
  } else if (!isNfcSupported) {
    return "missing-hardware" as const;
  } else {
    return "request-permission" as const;
  }
});

const sounds = computed(() => {
  return {
    scan: PickerSounds.getScanSoundForSeat(playerSeat.value?.position),
    error: PickerSounds.getErrorSoundForSeat(playerSeat.value?.position),
  };
});

const scanSound = useSound(sounds.value.scan);
const errorSound = useSound(sounds.value.error);
const zoneSound = useSound(zone);

function onPollForState() {
  if (activeDialog.value == null) {
    fetchDraft();
  }
}

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
  const cardRfid = decodeURIComponent(
    Array.prototype.map
      .call(atob(event.detail), (c) => `%${`00${c.charCodeAt(0).toString(16)}`.slice(-2)}`)
      .slice(3)
      .join(""),
  );

  await handleCardScanned(cardRfid);
}

async function handleCardScanned(cardRfid: string) {
  console.log("Attempting to pick scanned card with ID", cardRfid);

  if (activeRequest.value) {
    return;
  }
  activeRequest.value = true;
  activeDialog.value = {
    kind: "loading-spinner",
    message: "Picking scanned card...",
  };

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
    if (isZoneDraftViolation(e)) {
      activeDialog.value = {
        kind: "error-dialog",
        message: `Hold your horses!`,
        escapedMessage: "Wait for the next player to take their pack before picking.",
      };
      zoneSound.play();
    } else {
      activeDialog.value = {
        kind: "error-dialog",
        message: `An error occurred while scanning card with ID "${cardRfid}"`,
        escapedMessage: getErrorMessage(e),
      };
      errorSound.play();
    }
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
    draftId,
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
  if (fetchingDraft) {
    return;
  }
  fetchingDraft = true;
  const [payload, e] = await fetchEndpointEv(ROUTE_DRAFT, {
    id: draftId.toString(),
    as: authStore.user?.id,
  });
  fetchingDraft = false;

  if (payload) {
    draftStore.loadDraft(payload);
    loaded.value = true;
  } else {
    // TODO: Can't show an error dialog until loaded is true, woops
    console.error("Error while loading draft", e);
  }
}

async function handleNfcScanning() {
  if (isNfcSupported) {
    console.log("NFC reading supported!");

    const nfcPermissionStatus = await navigator.permissions.query({
      name: "nfc" as unknown as PermissionName,
    });

    hasNfcPermission.value = nfcPermissionStatus.state == "granted";
    nfcPermissionStatus.addEventListener("change", (e) => {
      console.log("NFC permission status changed to", nfcPermissionStatus.state);
      hasNfcPermission.value = nfcPermissionStatus.state == "granted";
    });

    if (hasNfcPermission.value) {
      console.log("Have permission, preparing to scan!");
      scanForTag();
    }
  }
}

function onRequestNfcPermission() {
  scanForTag();
}

function scanForTag() {
  const reader = new NDEFReader();
  reader.onreadingerror = () => {
    console.log("Cannot read data from the NFC tag. Try another one?");
  };
  reader.onreading = (e) => {
    console.log("NDEF message read.");
    console.log("Records:");
    for (const record of e.message.records) {
      console.log(record.id, record.recordType);
      if (record.recordType == "text") {
        const td = new TextDecoder(record.encoding);
        const text = td.decode(record.data);
        console.log("Text:", td.decode(record.data));
        if (CARD_UUID_PATTERN.test(text)) {
          console.log("It's a thing!");
          handleCardScanned(text);
        }
      }
    }
  };
  reader
    .scan()
    .then(() => {
      console.log("Scan started successfully.");
    })
    .catch((error) => {
      console.log(`Error! Scan failed to start: ${error}.`);
    });
}

function appendImpersonation(url: string) {
  if (route.query["as"] != undefined) {
    return url + `?as=${route.query["as"]}`;
  } else {
    return url;
  }
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

function isZoneDraftViolation(e: unknown): boolean {
  if (axios.isAxiosError(e)) {
    return e.response?.status === 400;
  }
  return false;
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

type BoopPrompt = "none" | "missing-hardware" | "request-permission";

interface DevOptions {
  boopPrompt: BoopPrompt | null;
  pollState: boolean;
}

// TODO: Move these into a per-draft config object
const ROUNDS_PER_DRAFT = 3;
const CARDS_PER_PACK = 15;

const CARD_UUID_PATTERN = /\w{8}-\w{4}-\w{4}-\w{4}-\w{12}/;
</script>

<style scoped>
._picker-page {
  position: relative;
  height: 100%;

  background: #333;
  color: white;
  overflow: hidden;
}

.scroll-outer {
  height: 100%;

  display: flex;
  flex-direction: column;
  align-items: center;
  overflow-y: auto;
}

.scroll-inner {
  display: flex;
  flex-direction: column;
  max-width: 450px;
  width: 100%;
  padding-bottom: 40px;
}

.header {
  font-size: 16px;
  text-align: center;
  padding: 20px 20px;
  display: flex;
  align-items: center;

  background-color: #1b1b1b;
}

.header-left,
.header-right {
  width: 80px;
  font-size: 16px;
}

.header-left {
  text-align: left;
}

.header-right {
  text-align: right;
}

.title {
  flex: 1;
}

.back-link {
  color: #ccc;
  text-decoration: none;
  padding: 10px;
  padding-left: 0px;
}
.back-link:hover {
  text-decoration: underline;
}

.table-seating {
  margin-top: 30px;
  padding-bottom: 50px;
}

.player-list-bg {
  width: 100%;
  height: 35px;
  background-color: #1b1b1b;
}

.player-list {
  display: flex;
  justify-content: center;
  padding: 7px 9px;
  width: fit-content;
  align-self: center;

  border-radius: 100px;
  background-color: #1b1b1b;

  margin-top: -30px;
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

.boop-prompt {
  padding: 20px;
  margin: 20px;
  background: #1b1b1b;
  border-radius: 2px;
  border: 1px solid #8b4d4d;
}

.boop-btn {
  padding: 10px;
  margin-top: 20px;
  width: 100%;
  box-sizing: border-box;
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
