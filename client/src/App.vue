<template>
  <div id="app">
    <router-view />
  </div>
</template>

<script lang="ts">
import Vue from 'vue'
import { authStore } from './state/AuthStore';

export default Vue.extend({
  created() {
    this.loadAuthInfo();
  },

  methods: {
    loadAuthInfo() {
      if (window.UserInfo != undefined) {
        const parsed = JSON.parse(window.UserInfo) as SourceUserInfo;
        authStore.setUser({
          id: parsed.userId,
          name: parsed.name,
          picture: parsed.picture || FALLBACK_USER_PICTURE,
        });
      }
    },
  },
});

interface SourceUserInfo {
  name: string;
  picture: string;
  userId: number;
}

const FALLBACK_USER_PICTURE =
    `https://cdn.discordapp.com/avatars/117108584017428481/f91aadd54de1929aaad167cabc99bdb1.png`;
</script>

<style>
html, body {
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
