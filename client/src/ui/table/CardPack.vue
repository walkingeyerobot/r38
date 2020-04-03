<template>
  <div
      class="_card-pack"
      :class="{ selected: isSelected }"
      @click="onClick"
      >
    Pack {{ pack.id }}
  </div>
</template>

<script lang="ts">
import Vue from 'vue'
import { CardPack } from '../../draft/draft_types';

export default Vue.extend({
  props: {
    pack: {
      type: Object as () => CardPack
    },
  },

  computed: {
    isSelected(): boolean {
      const selection = this.$tstore.state.selection;
      return selection != null
          && selection.type == 'pack'
          && selection.id == this.pack.id;
    }
  },

  methods: {
    onClick() {
      this.$tstore.commit('setSelection', {
        type: 'pack',
        id: this.pack.id,
      });
    },
  },

});
</script>

<style scoped>
.selected {
  font-weight: bold
}
</style>
