<template>
  <div class="_draft-list-item">
    <component :is="draftUrl ? 'RouterLink' : 'div'" class="main-section" :to="draftUrl">
      <div class="draft-name">
        {{ descriptor.name }}
      </div>
      <div class="draft-subtitle">
        {{ draftSubtitle }}
      </div>
    </component>
    <a v-if="isShufflable" class="shuffle-section" :href="`/shuffler/${descriptor.id}`">Shuffle</a>
    <a v-if="isAdminUser" class="shuffle-section" :href="`/draftpacks/${descriptor.id}`">Packs</a>
    <a v-if="isAdminUser" class="shuffle-section" href="#" @click.stop="onArchiveClicked()"
      >Archive</a
    >
    <a
      v-if="isAdminUser && !descriptor.inPerson"
      class="shuffle-section"
      href="#"
      @click.stop="onToggleInPersonClicked()"
      >Make in-person</a
    >
    <a
      v-if="isAdminUser && descriptor.inPerson"
      class="shuffle-section"
      href="#"
      @click.stop="onToggleInPersonClicked()"
      >Make online</a
    >
  </div>
</template>

<script setup lang="ts">
import type { HomeDraftDescriptor } from "@/rest/api/draftlist/draftlist";
import { authStore } from "@/state/AuthStore";
import { computed } from "vue";
import { useRoute } from "vue-router";
import { fetchEndpoint } from "@/fetch/fetchEndpoint.ts";
import { ROUTE_ARCHIVE_DRAFT } from "@/rest/api/archive/archive.ts";
import { ROUTE_TOGGLE_IN_PERSON } from "@/rest/api/toggleinperson/toggleinperson.ts";

const route = useRoute();

const { descriptor } = defineProps<{
  descriptor: HomeDraftDescriptor;
}>();

const isAdminUser = computed(() => {
  return authStore.userId === 1;
});

const isShufflable = computed(() => {
  return descriptor.inPerson && !descriptor.finished && isAdminUser.value;
});

const draftUrl = computed(() => {
  if (descriptor.finished) {
    return wrapUrl(`/draft/${descriptor.id}`);
  }
  switch (descriptor.status) {
    case "joinable":
    case "reserved":
      return wrapUrl(`/draft/${descriptor.id}/join`);
    case "member":
      if (descriptor.inPerson) {
        if (descriptor.availableSeats > 0) {
          return wrapUrl(`/draft/${descriptor.id}/join`);
        } else {
          return wrapUrl(`/draft/${descriptor.id}/pick`);
        }
      } else {
        return wrapUrl(`/draft/${descriptor.id}/replay`);
      }
    case "spectator":
      return wrapUrl(`/draft/${descriptor.id}/replay`);
    case "closed":
    default:
      return undefined;
  }
});

const draftSubtitle = computed(() => {
  if (descriptor.finished) {
    return `Finished`;
  }
  switch (descriptor.status) {
    case "joinable":
      return `${descriptor.availableSeats} seats available`;
    case "reserved":
      return `Spot reserved`;
    case "member":
      return `Joined`;
    case "spectator":
      return `Spectatable`;
    case "closed":
      return `Closed`;
    default:
      return undefined;
  }
});

async function onArchiveClicked() {
  const _response = await fetchEndpoint(ROUTE_ARCHIVE_DRAFT, {
    id: String(descriptor.id),
  });
  location.reload();
}

async function onToggleInPersonClicked() {
  const _response = await fetchEndpoint(ROUTE_TOGGLE_IN_PERSON, {
    id: String(descriptor.id),
  });
  location.reload();
}

function wrapUrl(url: string) {
  if (route.query["as"] != undefined) {
    return url + `?as=${route.query["as"]}`;
  } else {
    return url;
  }
}
</script>

<style scoped>
._draft-list-item {
  background-color: #f0f0f0;
  border-radius: 4px;
  display: flex;
  align-items: center;
}

.main-section,
.shuffle-section {
  padding: 20px;
}

.main-section {
  flex: 1;
  color: black;
  text-decoration: none;
}

a.main-section:hover .draft-name {
  text-decoration: underline;
}

.draft-subtitle {
  color: #717171;
  margin-top: 5px;
}
</style>
