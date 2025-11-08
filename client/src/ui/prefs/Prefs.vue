<!-- eslint-disable vue/multi-word-component-names -->
<template>
  <div class="_prefs">
    <div class="header">
      <img class="user-img" :src="user.picture" />
      {{ user.name }}
    </div>
    <div class="list-item">
      <label for="mtgoName" class="mtgo-label"> MTGO name: </label>
      <input
        type="text"
        id="mtgoName"
        class="mtgo-input"
        v-model="user.mtgoName"
        @change="onMtgoNameChanged"
        @input="confirmed = false"
        @keyup.enter="($event.target! as HTMLInputElement).blur()"
      />
      <span class="confirm" :hidden="!confirmed"> ✓ </span>
    </div>
    <PrefsItem v-for="pref in prefs" class="list-item" :key="pref.name" :pref="pref" />
  </div>
</template>

<script lang="ts">
import { defineComponent } from "vue";
import PrefsItem from "./PrefsItem.vue";

import { type UserPrefDescriptor, ROUTE_PREFS, routeSetPref } from "@/rest/api/prefs/prefs";
import { fetchEndpoint } from "@/fetch/fetchEndpoint";
import { authStore, type AuthenticatedUser, type AuthStore } from "@/state/AuthStore";

export default defineComponent({
  components: {
    PrefsItem,
  },

  data() {
    return {
      prefs: [] as UserPrefDescriptor[],
      confirmed: false,
    };
  },

  computed: {
    authStore(): AuthStore {
      return authStore;
    },

    user(): AuthenticatedUser {
      const user = authStore.user;
      if (!user || user === "error") {
        return {
          id: -1,
          mtgoName: "Unknown user",
          name: "Unknown user",
          picture: "__unknown_image",
          isImpersonated: false,
        };
      }
      return user;
    },
  },

  async created() {
    const response = await fetchEndpoint(ROUTE_PREFS, {
      as: authStore.userId,
    });
    // TODO: catch and show error
    this.prefs = response.prefs;
  },

  methods: {
    async onMtgoNameChanged() {
      if (authStore.user) {
        await fetchEndpoint(routeSetPref, { pref: undefined, mtgoName: authStore.userMtgoName });
        this.confirmed = true;
      }
    },
  },
});
</script>

<style scoped>
._prefs {
  padding-top: 20px;
}

.header {
  font-size: 40px;
  padding-left: 30px;
}

.user-img {
  vertical-align: baseline;
  width: 28px;
  margin-right: 10px;
  border-radius: 20px;
}

.list-item {
  margin-left: 30px;
  margin-right: 30px;
  padding-top: 20px;
  padding-bottom: 20px;
  border-top: 1px solid #e0e0e0;
  font-size: 25px;
}

.mtgo-input {
  font-size: 25px;
}

.confirm {
  padding-left: 0.3em;
  color: #42b983;
}
</style>
