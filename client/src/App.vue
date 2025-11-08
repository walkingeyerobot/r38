<template>
  <router-view v-if="status == 'ready'" />
  <div v-else-if="status == 'error'" class="error-msg">
    There was an error loading user info; please try refreshing
  </div>
</template>

<script lang="ts">
import { defineComponent } from "vue";

import { authStore } from "./state/AuthStore";
import { formatStore, type LayoutFormFactor } from "./state/FormatStore";

export default defineComponent({
  created() {},

  mounted() {
    this.initFormat();
  },

  computed: {
    status(): "init" | "ready" | "error" {
      const user = authStore.user;
      if (!user) {
        return "init";
      }
      if (user === "error") {
        return "error";
      }
      return "ready";
    },
  },

  methods: {
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
