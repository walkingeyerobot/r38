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
  </div>
</template>

<script setup lang="ts">
import type { HomeDraftDescriptor } from "@/rest/api/draftlist/draftlist";
import { authStore } from "@/state/AuthStore";
import { computed } from "vue";
import { useRoute } from "vue-router";

const route = useRoute();

const { descriptor } = defineProps<{
  descriptor: HomeDraftDescriptor;
}>();

const isAdminUser = computed(() => {
  return authStore.user?.id === 1;
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
    case "member":
      return wrapUrl(`/picker/${descriptor.id}`);
    case "spectator":
      return wrapUrl(`/draft/${descriptor.id}`);
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
      return `Reserved`;
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
