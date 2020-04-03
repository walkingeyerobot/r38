
import Vue from 'vue';
import { Store } from 'vuex';
import { RootState } from '../state/store';

// Type the $tstore properly, which is the same as $store but properly typed.
// Unfortunately you cannot override the Store<any> type.
declare module 'vue/types/vue' {
  interface Vue {
    $tstore: Store<RootState>;
  }
}

// Set $tstore to be a getter that simply returns $store.
Object.defineProperty(Vue.prototype, "$tstore", {
  get: function() {
    return this.$store as Store<RootState>;
  },
  enumerable: true,
});
