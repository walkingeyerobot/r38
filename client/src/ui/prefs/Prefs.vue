<template>
  <div class="_prefs">
    <div class="header">
      <img class="user-img" :src="authStore.user.picture">
      {{ authStore.user.name }}
    </div>
    <PrefsItem
        v-for="pref in prefs"
        class="list-item"
        :key="pref.name"
        :pref="pref"
        />
  </div>
</template>

<script lang="ts">
import Vue from 'vue';
import { routePrefs, UserPrefDescriptor } from '../../rest/api/prefs/prefs';
import { fetchEndpoint } from '../../fetch/fetchEndpoint';
import { AuthStore, authStore } from '../../state/AuthStore';
import PrefsItem from './PrefsItem.vue';

export default Vue.extend({
  components: {
    PrefsItem,
  },

  data() {
    return {
      prefs: [] as UserPrefDescriptor[]
    }
  },

  computed: {
    authStore(): AuthStore {
      return authStore;
    },
  },

  async created() {
    const response =
        await fetchEndpoint(routePrefs, {
          as: authStore.user?.id,
        });
    // TODO: catch and show error
    this.prefs = response.prefs;
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
  border-top: 1px solid #e0e0e0;
}
</style>