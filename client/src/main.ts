import { createApp } from 'vue';

import App from './App.vue';
import router from './router';
import { rootStore } from './state/store';


createApp(App)
    .use(router)
    .use(rootStore)
    .mount("#app");
