<!--

Represents one entry in the list of drafts that a user might view/join

-->

<template>
  <div
    class="_draft-list-item"
    :class="{
      viewable: isViewable,
    }"
    @click="onRootClick"
  >
    <div class="main">
      <div
        class="name"
        :class="{
          viewable: isViewable,
          closed: descriptor.status == 'closed',
        }"
      >
        {{ descriptor.name }}
      </div>
      <div v-if="descriptor.availableSeats > 0 && isJoinable" class="seats">
        {{ descriptor.availableSeats }} seats available
      </div>
    </div>
    <button
      v-if="!descriptor.inPerson && isJoinable"
      class="join-btn"
      @click.stop="onJoinClicked(descriptor.id)"
      :disabled="joinFetchStatus == 'fetching'"
    >
      Join
    </button>
    <span v-if="descriptor.inPerson && isJoinable" class="join-seats">
      <button
        v-for="n in 8"
        :key="n"
        class="join-btn"
        @click.stop="onJoinSeatClicked(descriptor.id, n - 1)"
        :disabled="joinFetchStatus == 'fetching'"
      >
        {{ n }}
      </button>
    </span>
    <button
      v-if="isSkippable"
      class="join-btn"
      @click.stop="onSkipClicked(descriptor.id)"
      :disabled="joinFetchStatus == 'fetching'"
    >
      Skip
    </button>
    <button v-if="admin" class="join-btn" @click.stop="onArchiveClicked()">Archive</button>
    <button v-if="isShufflable" class="join-btn" @click.stop="onShuffleClicked()">Shuffle</button>
  </div>
</template>

<script lang="ts">
import { defineComponent } from "vue";
import type { HomeDraftDescriptor } from "@/rest/api/draftlist/draftlist";
import type { FetchStatus } from "../infra/FetchStatus";
import { fetchEndpoint } from "@/fetch/fetchEndpoint";
import { pushDraftUrl } from "@/router/url_manipulation";
import { ROUTE_JOIN_DRAFT } from "@/rest/api/join/join";
import { ROUTE_SKIP_DRAFT } from "@/rest/api/skip/skip";
import { authStore } from "@/state/AuthStore.ts";
import { ROUTE_ARCHIVE_DRAFT } from "@/rest/api/archive/archive.ts";

export default defineComponent({
  props: {
    descriptor: {
      type: Object as () => HomeDraftDescriptor,
      required: true,
    },
  },

  data() {
    return {
      joinFetchStatus: "missing" as FetchStatus,
    };
  },

  computed: {
    isJoinable(): boolean {
      return this.descriptor.status == "joinable" || this.descriptor.status == "reserved";
    },

    isSkippable(): boolean {
      return this.descriptor.status == "reserved";
    },

    isShufflable(): boolean {
      return this.descriptor.inPerson && !this.descriptor.finished && authStore.userId === 1;
    },

    isViewable(): boolean {
      return (
        this.descriptor.status != "joinable" &&
        this.descriptor.status != "reserved" &&
        this.descriptor.status != "closed"
      );
    },

    admin(): boolean {
      return authStore.userId === 1;
    },
  },

  methods: {
    onRootClick() {
      if (this.isViewable) {
        let path = `/draft/${this.descriptor.id}`;
        if (this.$route.query.as != undefined) {
          path += `/?as=${this.$route.query.as}`;
        }
        this.$router.push(path);
      }
    },

    async onArchiveClicked() {
      const _response = await fetchEndpoint(ROUTE_ARCHIVE_DRAFT, {
        id: String(this.descriptor.id),
      });
      location.reload();
    },

    onShuffleClicked() {
      this.$router.push(`/shuffle/${this.descriptor.id}`);
    },

    async onJoinClicked(draftId: number) {
      this.joinFetchStatus = "fetching";
      // TODO: Error handling
      const _response = await fetchEndpoint(ROUTE_JOIN_DRAFT, { id: draftId, position: undefined });
      this.joinFetchStatus = "loaded";
      pushDraftUrl(this, { draftId });
    },

    async onJoinSeatClicked(draftId: number, position: number) {
      this.joinFetchStatus = "fetching";
      // TODO: Error handling
      const _response = await fetchEndpoint(ROUTE_JOIN_DRAFT, { id: draftId, position });
      this.joinFetchStatus = "loaded";
      pushDraftUrl(this, { draftId });
    },

    async onSkipClicked(draftId: number) {
      this.joinFetchStatus = "fetching";
      // TODO: Error handling
      const _response = await fetchEndpoint(ROUTE_SKIP_DRAFT, { id: draftId });
      this.joinFetchStatus = "loaded";
      this.$emit("refreshDraftList");
    },
  },
});
</script>

<style scoped>
._draft-list-item {
  display: flex;
  flex-direction: row;
  min-height: 55px;
  padding: 15px 0;
  align-items: center;

  font-size: 16px;
}

._draft-list-item.viewable {
  cursor: pointer;
}

.main {
  flex: 1 0 0;
}

.name.closed {
  color: #777;
}

._draft-list-item:hover .name.viewable {
  text-decoration: underline;
}

.seats {
  margin-top: 5px;
  font-size: 14px;
}

.join-seats {
  display: grid;
  grid-template-columns: 25% 25% 25% 25%;
}

.join-btn {
  margin-left: 10px;
  padding: 5px 15px;

  font-size: 100%;
  font-family: inherit;
  border: 1px solid #c54818;
  border-radius: 5px;
  background: white;
  color: #c54818;
}

.join-btn:focus {
  outline: none;
}

.join-btn:active {
  background: #c54818;
  color: white;
}
</style>
