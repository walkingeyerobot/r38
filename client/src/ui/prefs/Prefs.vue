<template>
  <div class="_prefs">
    <div class="header">
      <img class="user-img" :src="authStore.user.picture">
      {{ authStore.user.name }}
    </div>
    <div class="list-item">
      <label
          for="mtgoName"
          class="mtgo-label"
      >
        MTGO name:
      </label>
      <input
          type="text"
          id="mtgoName"
          class="mtgo-input"
          v-model="authStore.user.mtgoName"
          @change="onMtgoNameChanged"
          @input="confirmed = false"
          @keyup.enter="$event.target.blur()"
      />
      <span
          class="confirm"
          :hidden="!confirmed"
      >
        ✓
      </span>
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
import { routePrefs, routeSetPref, UserPrefDescriptor } from '../../rest/api/prefs/prefs';
import { fetchEndpoint } from '../../fetch/fetchEndpoint';
import { AuthStore, authStore } from '../../state/AuthStore';
import PrefsItem from './PrefsItem.vue';

export default Vue.extend({
  components: {
    PrefsItem,
  },

  data() {
    return {
      prefs: [] as UserPrefDescriptor[],
      confirmed: false,
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

  methods: {
    async onMtgoNameChanged() {
      if (authStore.user) {
        await fetchEndpoint(routeSetPref,
            {pref: undefined, mtgoName: authStore.user.mtgoName});
        this.confirmed = true;
      }
    }
  }
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