<template>
  <router-view v-if="status == 'ready'" />
  <div v-else-if="status == 'error'" class="error-msg">
    There was an error loading user info; please try refreshing
  </div>
</template>

<script lang="ts">
import { defineComponent } from "vue";
import DefaultAvatar from "./ui/shared/avatars/default_avatar.png";

import { authStore } from "./state/AuthStore";
import { formatStore, type LayoutFormFactor } from "./state/FormatStore";
import { fetchEndpoint } from "./fetch/fetchEndpoint";
import { ROUTE_USER_INFO, type SourceUserInfo } from "./rest/api/userinfo/userinfo";

export default defineComponent({
  created() {},

  mounted() {
    this.loadAuthInfo();
    this.initFormat();
  },

  data() {
    return {
      status: "init" as "init" | "ready" | "error",
    };
  },

  methods: {
    async loadAuthInfo() {
      let result: SourceUserInfo;

      const asPlayer = parseInt(unwrapQuery(this.$route.query.as) ?? "NaN");
      const asPlayerId = isNaN(asPlayer) ? undefined : asPlayer;

      try {
        result = await fetchEndpoint(ROUTE_USER_INFO, { as: asPlayerId });
        this.status = "ready";
      } catch (e) {
        console.error("Error fetching user info:", e);
        this.status = "error";
        return;
      }

      authStore.setUser({
        id: result.userId,
        name: result.name,
        picture: result.picture || DefaultAvatar,
        mtgoName: result.mtgoName,
      });
    },

    initFormat() {
      this.updateFormFactor();

      window.addEventListener("resize", () => {
        this.updateFormFactor();
      });
    },

    updateFormFactor() {
      const layout = getLayoutFormFactor();
      if (layout != formatStore.layout) {
        formatStore.setLayout(layout);
      }
    },
  },
});

function getLayoutFormFactor(): LayoutFormFactor {
  return window.innerWidth >= 768 ? "desktop" : "mobile";
}

function unwrapQuery<T>(value: T | T[]): T {
  if (value instanceof Array) {
    return value[0];
  } else {
    return value;
  }
}
</script>

<style>
html,
body {
  height: 100%;
}

body {
  padding: 0;
  margin: 0;
}

#app {
  height: 100%;

  font-family: Arial, sans-serif;
  -webkit-font-smoothing: antialiased;
  -moz-osx-font-smoothing: grayscale;
  color: #2c3e50;
}

#nav {
  padding: 30px;
}

#nav a {
  font-weight: bold;
  color: #2c3e50;
}

#nav a.router-link-exact-active {
  color: #42b983;
}
</style>
