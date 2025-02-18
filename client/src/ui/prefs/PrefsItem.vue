<!-- eslint-disable vue/no-mutating-props -->
<template>
  <label class="_prefs-item" :for="pref.format">
    {{ pref.format }}
    <input
      :id="pref.format"
      type="checkbox"
      class="checkbox"
      v-model="pref.elig"
      @change="toggle"
    />
    <span class="styled-checkbox" />
  </label>
</template>

<script lang="ts">
import { defineComponent } from "vue";
import { routeSetPref, type UserPrefDescriptor } from "@/rest/api/prefs/prefs";
import { fetchEndpoint } from "@/fetch/fetchEndpoint";

export default defineComponent({
  props: {
    pref: {
      type: Object as () => UserPrefDescriptor,
      required: true,
    },
  },

  methods: {
    async toggle() {
      await fetchEndpoint(routeSetPref, {
        pref: { ...this.pref },
        mtgoName: undefined,
      });
    },
  },
});
</script>

<style scoped>
._prefs-item {
  display: block;
  cursor: pointer;
}

._prefs-item:hover {
  background: #eee;
}

.checkbox {
  clip-path: polygon(0 0);
}

.styled-checkbox {
  height: 20px;
  width: 40px;
  border-radius: 10px;
  background: #ccc;
  display: inline-block;
  position: relative;
  transition: background-color 250ms;
}

.checkbox:checked ~ .styled-checkbox {
  background: #c54818;
}

.styled-checkbox::after {
  content: "";
  position: absolute;
  left: 3px;
  top: 3px;
  height: 14px;
  width: 14px;
  border-radius: 7px;
  background: white;
  transition: transform 250ms;
}

.checkbox:checked ~ .styled-checkbox::after {
  transform: translateX(20px);
}
</style>
