<!-- eslint-disable vue/no-mutating-props -->
<template>
  <div v-if="stats">
    <div class="stat">Completed drafts: {{ stats.completedDrafts }}</div>
    <div class="stat">Active drafts: {{ stats.activeDrafts }}</div>
  </div>
</template>

<script lang="ts">
import { defineComponent } from "vue";
import { fetchEndpoint } from "@/fetch/fetchEndpoint";
import { authStore } from "@/state/AuthStore.ts";
import { ROUTE_USER_STATS, type SourceUserStats } from "@/rest/api/userstats/userstats.ts";

export default defineComponent({
  data() {
    return {
      stats: undefined as SourceUserStats | undefined,
    };
  },

  async created() {
    this.stats = await fetchEndpoint(ROUTE_USER_STATS, {
      as: authStore.userId,
    });
  },
});
</script>

<style scoped>
.stat {
  margin-left: 30px;
  margin-right: 30px;
  padding-top: 20px;
  padding-bottom: 20px;
  font-size: 25px;
}
</style>
