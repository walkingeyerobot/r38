<!-- eslint-disable vue/no-mutating-props -->
<template>
  <div v-if="stats">
    <div class="stat">Completed drafts: {{ stats.completedDrafts }}</div>
    <div class="stat">Active drafts: {{ stats.activeDrafts }}</div>
    <div class="stat">Drafted colors</div>
    <svg class="stat" ref="draftedColors"></svg>
  </div>
</template>

<script setup lang="ts">
import * as d3 from "d3";
import { nextTick, onMounted, ref, useTemplateRef } from "vue";
import { fetchEndpoint } from "@/fetch/fetchEndpoint";
import { authStore } from "@/state/AuthStore.ts";
import { ROUTE_USER_STATS, type SourceUserStats } from "@/rest/api/userstats/userstats.ts";
import MANA_W from "@/ui/shared/mana/W.svg";
import MANA_U from "@/ui/shared/mana/U.svg";
import MANA_B from "@/ui/shared/mana/B.svg";
import MANA_R from "@/ui/shared/mana/R.svg";
import MANA_G from "@/ui/shared/mana/G.svg";
import MANA_C from "@/ui/shared/mana/C.svg";

const stats = ref<SourceUserStats | undefined>(undefined);
const draftedColors = useTemplateRef<HTMLElement>("draftedColors");

onMounted(async () => {
  stats.value = await fetchEndpoint(ROUTE_USER_STATS, {
    as: authStore.userId,
  });

  await nextTick();

  const width = 300;
  const height = 300;
  const draftedColorsSvg = d3.select(draftedColors.value);
  draftedColorsSvg
    .attr("width", width)
    .attr("height", height)
    .attr("viewBox", [-width / 2, -height / 2, width, height]);

  const pie = d3.pie().sort(null);
  const arcs = pie(stats.value.draftedColors);
  const arc = d3
    .arc<d3.PieArcDatum<unknown>>()
    .innerRadius(0)
    .outerRadius(Math.min(width, height) / 2 - 1);
  const labelArc = d3
    .arc<d3.PieArcDatum<unknown>>()
    .innerRadius(Math.min(width, height) / 4)
    .outerRadius(Math.min(width, height) / 2 - 1);

  draftedColorsSvg
    .append("g")
    .attr("stroke", "#ccc")
    .selectAll()
    .data(arcs)
    .join("path")
    .attr("fill", (_: unknown, i: number) => ["#fff", "#aaf", "#666", "#faa", "#afa", "#ddd"][i])
    .attr("d", arc);

  draftedColorsSvg
    .append("g")
    .selectAll()
    .data(arcs)
    .join("image")
    .attr("transform", (d) => `translate(${labelArc.centroid(d).map((n) => n - 7.5)})`)
    .attr("width", 15)
    .attr("height", 15)
    .attr("href", (_: unknown, i: number) => [MANA_W, MANA_U, MANA_B, MANA_R, MANA_G, MANA_C][i]);
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
