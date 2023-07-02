import { createApp } from 'vue';

import App from './App.vue';
import router from './router';
import { rootStore } from './state/store';


const app = createApp({
  App,
})
.use(router)
.use(rootStore);

app.mount("#app");