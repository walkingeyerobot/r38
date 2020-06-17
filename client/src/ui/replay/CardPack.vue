<template>
  <div
      class="_card-pack"
      :class="{ selected: isSelected }"
      @click.stop="onClick"
      >
    {{ pack.id }}
  </div>
</template>

<script lang="ts">
import Vue from 'vue'
import { CardPack } from '../../draft/DraftState';
import { navTo } from '../../router/url_manipulation';

import { draftStore } from '../../state/DraftStore';
import { replayStore } from '../../state/ReplayStore';

export default Vue.extend({
  props: {
    pack: {
      type: Object as () => CardPack
    },
  },

  computed: {
    isSelected(): boolean {
      const selection = replayStore.selection;
      return selection != null
          && selection.type == 'pack'
          && selection.id == this.pack.id;
    }
  },

  methods: {
    onClick() {
      navTo(draftStore, replayStore, this.$route, this.$router, {
        selection: {
          type: 'pack',
          id: this.pack.id,
        },
      });
    },
  },

});
</script>

<style scoped>
._card-pack {
  width: 25px;
  height: 35px;
  background-color: #A24E30;
  border-radius: 2px;
  display: flex;
  align-items: center;
  justify-content: center;

  font-size: 14px;
  color: white;

  background-image: url('./card_back.png');
  background-size: cover;
}

.selected {
  box-shadow: 0px 0px 5px 2px #3206DE;
}
</style>
