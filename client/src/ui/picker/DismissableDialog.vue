<template>
  <div class="_dismissable-dialog">
    <div class="scroller">
      <div class="content">
        <div v-if="$slots.header" class="header">
          <slot name="header"></slot>
        </div>
        <slot></slot>
      </div>
    </div>
    <button class="close-btn" @click="$emit('close')">
      <img class="close-img" src="../shared/icons/close_light_sharp.svg" />
    </button>
  </div>
</template>

<script setup lang="ts">
defineSlots<{
  header?(): void;
  default(): void;
}>();

defineEmits<{
  (e: "close"): void;
}>();
</script>

<style scoped>
._dismissable-dialog {
  position: relative;
  box-sizing: border-box;

  background: #333;
  color: white;
}

.scroller {
  overflow-y: auto;
  max-height: 100%;

  display: flex;
  flex-direction: column;
  align-items: center;
}

.content {
  padding: 20px;
  padding-bottom: 120px;

  display: flex;
  flex-direction: column;

  max-width: 450px;
  width: 100%;
  box-sizing: border-box;
}

.header {
  font-size: 20px;
  margin-top: 10px;
  margin-bottom: 20px;
  text-align: center;
}

.close-btn {
  position: absolute;
  width: 60px;
  height: 60px;
  bottom: 30px;
  left: calc(50% - 30px);

  border-radius: 100%;
  border: 2px solid #ddd;

  background: #141414;

  box-shadow: 0px 2px 6px rgba(0, 0, 0, 0.5);

  display: flex;
  padding: 10px;
  box-sizing: border-box;
}

.close-btn:active {
  filter: brightness(0.5);
}

.close-img {
  width: 100%;
  height: 100%;
}
</style>
