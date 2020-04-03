import Vue from 'vue';

import App from './App.vue';
import router from './router';
import store from './state/store';

import './shims/shims-vuex';

Vue.config.productionTip = false;

new Vue({
  router,
  store,
  render: h => h(App),
}).$mount('#app');
