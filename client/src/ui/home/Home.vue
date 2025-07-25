<!-- eslint-disable vue/multi-word-component-names -->
<template>
  <div class="_home">
    <div class="header">
      <div class="header-left title">R38</div>
      <div class="header-center"></div>
      <div class="header-right">
        <a v-if="!loggedIn" class="login-btn" href="/auth/discord/login"> Log in </a>
        <a v-if="loggedIn" href="/prefs">
          <img class="user-img" :src="userPic" :alt="userName" />
        </a>
      </div>
    </div>

    <div class="admin-link" v-if="admin"><a href="/tagwriter/new_rfids">Tag writer</a></div>

    <div v-if="admin && listFetchStatus == 'error'" class="status-msg">
      {{ error }}
    </div>
    <div class="drafts-cnt">
      <div v-if="listFetchStatus == 'fetching'" class="status-msg">Loading...</div>
      <div v-if="listFetchStatus == 'missing'" class="status-msg">No drafts</div>

      <DraftListItem2
        v-for="descriptor in joinableDrafts"
        class="list-item"
        :key="descriptor.id"
        :descriptor="descriptor"
        v-on:refreshDraftList="refreshDraftList"
      />

      <DraftListItem2
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
import { defineComponent } from "vue";
import DraftListItem2 from "./DraftListItem2.vue";

import { authStore } from "@/state/AuthStore";

import { type HomeDraftDescriptor, ROUTE_DRAFT_LIST } from "@/rest/api/draftlist/draftlist";
import { fetchEndpoint } from "@/fetch/fetchEndpoint";
import type { FetchStatus } from "../infra/FetchStatus";
import { AxiosError } from "axios";

export default defineComponent({
  components: {
    DraftListItem2,
  },

  data() {
    return {
      drafts: [] as HomeDraftDescriptor[],
      listFetchStatus: "missing" as FetchStatus,
      error: "",
    };
  },

  async created() {
    await this.refreshDraftList();
  },

  computed: {
    joinableDrafts(): HomeDraftDescriptor[] {
      return this.drafts.filter(
        (value) => value.status == "joinable" || value.status == "reserved",
      );
    },

    otherDrafts(): HomeDraftDescriptor[] {
      return this.drafts
        .filter((value) => value.status != "joinable" && value.status != "reserved")
        .sort((a, b) => b.id - a.id);
    },

    loggedIn(): boolean {
      return authStore.user?.id != 0;
    },

    admin(): boolean {
      return authStore.user?.id === 1;
    },

    userPic(): string | undefined {
      return authStore.user?.picture;
    },

    userName(): string | undefined {
      return authStore.user?.name;
    },
  },

  methods: {
    async refreshDraftList() {
      this.listFetchStatus = "fetching";
      try {
        const response = await fetchEndpoint(ROUTE_DRAFT_LIST, {
          as: authStore.user?.id,
        });
        this.drafts = response?.drafts ?? [];
        this.listFetchStatus = this.drafts.length ? "loaded" : "missing";
      } catch (e) {
        this.drafts = [];
        this.listFetchStatus = "error";
        if (e instanceof AxiosError) {
          this.error = e.response?.data.error ?? "";
        }
      }
    },
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

.header-left,
.header-right {
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
  background: #7187dd;
  border-radius: 4px;
  text-decoration: none;
  padding: 0 20px;
}

.admin-link {
  margin-top: 50px;
  margin-left: 30px;
  margin-bottom: 20px;
}

.drafts-cnt {
  margin-top: 50px;
  max-width: 400px;
}

.list-item,
.status-msg {
  margin-left: 30px;
  margin-right: 30px;
  /* border-top: 1px solid #e0e0e0; */
  margin-bottom: 20px;
}

/* .list-item:last-child {
  border-bottom: 1px solid #e0e0e0;
} */

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
