<template>
  <div class="_home">
    <div class="header">
      <div class="header-left title">R38</div>
      <div class="header-center"></div>
      <div class="header-right">
        <a v-if="!loggedIn" class="login-btn" href="/auth/discord/login">
          Log in
        </a>
        <a v-if="loggedIn" href="/prefs">
          <img class="user-img" :src="userPic">
        </a>
      </div>
    </div>

    <div class="drafts-cnt">

      <div v-if="listFetchStatus == 'fetching'" class="loading-msg">Loading...</div>

      <DraftListItem
          v-for="descriptor in joinableDrafts"
          class="list-item"
          :key="descriptor.id"
          :descriptor="descriptor"
          v-on:refreshDraftList="refreshDraftList"
          />

      <DraftListItem
          v-for="descriptor in otherDrafts"
          class="list-item"
          :key="descriptor.id"
          :descriptor="descriptor"
          v-on:refreshDraftList="refreshDraftList"
          />

    </div>
  </div>
</template>

<script lang="ts">
import { defineComponent } from 'vue';
import DraftListItem from './DraftListItem.vue';

import { authStore } from '../../state/AuthStore';

import { HomeDraftDescriptor, routeDraftlist } from '../../rest/api/draftlist/draftlist'
import { fetchEndpoint } from '../../fetch/fetchEndpoint';
import { FetchStatus } from '../infra/FetchStatus';


export default defineComponent({
  components: {
    DraftListItem,
  },

  data() {
    return {
      drafts: [] as HomeDraftDescriptor[],
      listFetchStatus: 'missing' as FetchStatus,
    };
  },

  async created() {
    await this.refreshDraftList();
  },

  computed: {
    joinableDrafts(): HomeDraftDescriptor[] {
      return this.drafts.filter(value => value.status == 'joinable' || value.status == 'reserved');
    },

    otherDrafts(): HomeDraftDescriptor[] {
      return this.drafts
          .filter(value => value.status != 'joinable' && value.status != 'reserved')
          .sort((a, b) => b.id - a.id);
    },

    loggedIn(): boolean {
      return authStore.user?.id != 0;
    },

    userPic(): string | undefined {
      return authStore.user?.picture
    }
  },

  methods: {
    async refreshDraftList() {
        this.listFetchStatus = 'fetching';
        const response =
            await fetchEndpoint(routeDraftlist, {
              as: authStore.user?.id,
            });
        this.listFetchStatus = 'loaded';
        // TODO: catch and show error
        this.drafts = response.drafts;
      }
    },
});

</script>

<style scoped>
._home {
  padding-top: 20px;
}

.header {
  display: flex;
  flex-direction: row;
  align-items: center;
}

.header-left, .header-right {
  flex: 1 0 0;
}

.header-left {
  font-size: 40px;
  padding-left: 30px;
}

.header-right {
  display: flex;
  flex-direction: row;
  justify-content: flex-end;
  margin-right: 20px;
}

.login-btn {
  display: flex;
  height: 34px;
  flex-direction: row;
  align-items: center;
  margin-right: 30px;

  font-size: 14px;
  font-weight: bold;
  color: white;
  background: #7187DD;
  border-radius: 4px;
  text-decoration: none;
  padding: 0 20px;
}

.drafts-cnt {
  margin-top: 50px;
  max-width: 400px;
}

.list-item {
  margin-left: 30px;
  margin-right: 30px;
  border-top: 1px solid #e0e0e0;
}

.list-item:last-child {
  border-bottom: 1px solid #e0e0e0;
}

.user-img {
  width: 28px;
  margin-left: 10px;
  border-radius: 20px;
}

@media only screen and (max-width: 768px) {
  .header {
    height: 55px;
  }

  .login-btn {
    height: 40px;
  }

  .header-left {
    font-size: 46px;
  }

  .drafts-cnt {
    max-width: unset;
  }
}
</style>
