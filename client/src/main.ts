import { createApp } from "vue";
import { createPinia } from "pinia";

import App from "./App.vue";
import router from "./router";
import { rootStore } from "./state/store";

const app = createApp(App);

app.use(createPinia());
app.use(router);
app.use(rootStore);

// Wait until the router is ready before mounting the app, or components that
// attempt to read the route might get an empty response.
// See https://www.vuemastery.com/blog/vue-router-4-route-params-not-available-on-created-setup/
router.isReady().then(() => {
  app.mount("#app");
});
