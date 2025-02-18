<template>
  <div id="app">
    <router-view />
  </div>
</template>

<script lang="ts">
import { defineComponent } from "vue";
import DefaultAvatar from "./ui/shared/avatars/default_avatar.png";

import { authStore } from "./state/AuthStore";
import { formatStore, type LayoutFormFactor } from "./state/FormatStore";

export default defineComponent({
  created() {
    this.loadAuthInfo();
    this.initFormat();
  },

  methods: {
    loadAuthInfo() {
      if (window.UserInfo != undefined) {
        const parsed = JSON.parse(window.UserInfo) as SourceUserInfo;
        authStore.setUser({
          id: parsed.userId,
          name: parsed.name,
          picture: parsed.picture || DefaultAvatar,
          mtgoName: parsed.mtgoName,
        });
      }
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

interface SourceUserInfo {
  name: string;
  picture: string;
  userId: number;
  mtgoName: string;
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
