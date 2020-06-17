<template>
  <div class="_home">
    <div class="title">R38</div>

    <div class="drafts-cnt">

      <div v-if="fetchStatus == 'loading'" class="loading-msg">Loading...</div>

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
import { HomeDraftDescriptor, routeDraftlist } from '../../rest/api/draftlist/draftlist'
import { fetchEndpoint } from '../../fetch/fetchEndpoint';

export default Vue.extend({
  data() {
    return {
      drafts: [] as HomeDraftDescriptor[],
      fetchStatus: 'missing' as 'missing' | 'loading' | 'error' | 'loaded'
    };
  },

  async created() {
    this.fetchStatus = 'loading';
    const response = await fetchEndpoint(routeDraftlist, {});
    this.fetchStatus = 'loaded';
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
    onJoinClicked(draftId: number) {
      console.log('Joining', draftId);
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
