import { createApp } from "vue";
import { createPinia } from "pinia";

import App from "./App.vue";
import router from "./router";
import { rootStore } from "./state/store";

const app = createApp(App);

app.use(createPinia());
app.use(router);
app.use(rootStore);

router.isReady().then(() => {
  app.mount("#app");
});
