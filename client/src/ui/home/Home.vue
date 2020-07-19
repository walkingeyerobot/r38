<template>
  <div class="_home">
    <div class="title">R38</div>

    <div class="drafts-cnt">

      <div v-if="listFetchStatus == 'fetching'" class="loading-msg">Loading...</div>

      <div v-if="joinableDrafts.length > 0" class="joinable-cnt" >
        <div
            v-for="draft in joinableDrafts"
            :key="draft.id"
            class="joinable-entry"
            >
          <div class="joinable-text">
            <div class="joinable-title">{{ draft.name }}</div>
            <div class="joinable-seats">
              {{ draft.availableSeats }}
              seats available
            </div>
          </div>
          <button
              class="joinable-btn"
              @click="onJoinClicked(draft.id)"
              :disabled="joinFetchStatus == 'fetching'"
              >
            Join
          </button>
        </div>
      </div>

      <div class="other-cnt">
        <div
            v-for="draft in otherDrafts"
            :key="draft.id"
            class="other-entry"
            >
          <router-link
              v-if="draft.status != 'closed'"
              :to="`/draft/${draft.id}`"
              >
            {{ draft.name }}
          </router-link>
          <template v-else>
            {{ draft.name }}
          </template>
        </div>
      </div>

    </div>
  </div>
</template>

<script lang="ts">
import Vue from 'vue';
import { authStore } from '../../state/AuthStore';

import { HomeDraftDescriptor, routeDraftlist } from '../../rest/api/draftlist/draftlist'
import { fetchEndpoint } from '../../fetch/fetchEndpoint';
import { FetchStatus } from '../infra/FetchStatus';
import { ROUTE_JOIN_DRAFT } from '../../rest/api/join/join';

export default Vue.extend({
  data() {
    return {
      drafts: [] as HomeDraftDescriptor[],
      listFetchStatus: 'missing' as FetchStatus,
      joinFetchStatus: 'missing' as FetchStatus,
    };
  },

  async created() {
    this.listFetchStatus = 'fetching';
    const response =
        await fetchEndpoint(routeDraftlist, {
          as: authStore.user?.id,
        });
    this.listFetchStatus = 'loaded';
    // TODO: catch and show error
    this.drafts = response.drafts;
  },

  computed: {
    joinableDrafts(): HomeDraftDescriptor[] {
      return this.drafts.filter(value => value.status == 'joinable');
    },

    otherDrafts(): HomeDraftDescriptor[] {
      return this.drafts
          .filter(value => value.status != 'joinable')
          .sort((a, b) => b.id - a.id);
    },
  },

  methods: {
    async onJoinClicked(draftId: number) {
      this.joinFetchStatus = 'fetching';
      // TODO: Error handling
      const response = await fetchEndpoint(ROUTE_JOIN_DRAFT, { id: draftId });
      this.joinFetchStatus = 'loaded';
      this.$router.push(`/replay/${draftId}`);
    },
  },
});

</script>

<style scoped>
.title {
  margin-top: 20px;
  text-align: center;
}

.drafts-cnt {
  margin-top: 50px;
  margin-left: 30px;
  width: 400px;
}

.joinable-cnt {
  margin-bottom: 20px;
}

.joinable-entry, .other-entry {
  padding: 14px 15px;
}

.joinable-entry {
  display: flex;
  flex-direction: row;

  border: 1px solid #C7C7C7;
  border-radius: 4px;
  margin-bottom: 10px;
}

.joinable-text {
  flex: 1 0 auto;
}

.joinable-seats {
  margin-top: 5px;
  font-size: 14px;
}

.joinable-btn {
  align-self: center;
  margin-left: 10px;
  padding: 5px 15px;

  font-size: 100%;
  font-family: inherit;
  border: 1px solid #c54818;
  border-radius: 5px;
  color: #c54818;
}

.joinable-btn:focus {
  outline: none;
}

.joinable-btn:active {
  background: #f9f9f9;
  color: #8c320f;
  border-color: #8c320f;
}

.other-entry {
  padding-top: 5px;
  padding-bottom: 5px;
}
</style>
