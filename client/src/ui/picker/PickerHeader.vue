<template>
  <div class="_picker-header">
    <div class="header-left">
      <a class="back-link" href="/" @click.prevent="$router.push(appendImpersonation('/'))"
        >〈 Back</a
      >
    </div>
    <div class="title">{{ draftStore.draftName }}</div>
    <div class="header-right"></div>
  </div>
</template>

<script setup lang="ts">
import { draftStore } from "@/state/DraftStore";
import { useRoute } from "vue-router";

const route = useRoute();

function appendImpersonation(url: string) {
  if (route.query["as"] != undefined) {
    return url + `?as=${route.query["as"]}`;
  } else {
    return url;
  }
}
</script>

<style scoped>
._picker-header {
  font-size: 16px;
  text-align: center;
  padding: 15px 15px;
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
</style>
