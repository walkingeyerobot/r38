import Vue from 'vue';

import App from './App.vue';
import router from './router';
import { rootStore } from './state/store';
import AsyncComputed from "vue-async-computed";

Vue.config.productionTip = false;

Vue.use(AsyncComputed);

new Vue({
  router,
  store: rootStore,
  render: h => h(App),
}).$mount('#app');
