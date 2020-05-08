import Vue from 'vue';

import App from './App.vue';
import router from './router';
import { rootStore } from './state/store';

Vue.config.productionTip = false;

new Vue({
  router,
  store: rootStore,
  render: h => h(App),
}).$mount('#app');
