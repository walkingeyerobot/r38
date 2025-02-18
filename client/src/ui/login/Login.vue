<!-- eslint-disable vue/multi-word-component-names -->
<template>
  <div class="_login">
    <div class="login-cnt">
      <div class="title">R38</div>
      <div class="content-cnt">
        <a class="login-button" href="/auth/discord/login">
          Log in with
          <img class="discord-logo" src="./discord_wordmark_white.png" />
        </a>
      </div>
    </div>
  </div>
</template>

<script lang="ts">
import { defineComponent } from "vue";

export default defineComponent({});
</script>

<style scoped>
._login {
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  user-select: none;
}

.login-cnt {
  width: 400px;
  height: 200px;

  padding: 25px;
  border-radius: 3px;
  border: 1px solid #f39ac1;
  box-shadow: 0 4px 15px rgba(0, 0, 0, 0.15);

  display: flex;
  flex-direction: column;
}

.title {
  text-align: center;
  font-size: 18px;
  color: #eb028c;
}

.content-cnt {
  flex: 1 0 0;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
}

.login-button {
  display: inline-block;
  background: #7289da;
  padding: 5px 25px;

  border-radius: 3px;

  color: #fff;
  font-size: 15px;
  font-weight: bold;
  text-decoration: none;
}

.login-button:hover {
  transform: translate3d(0, 1px, 0);
}

.discord-logo {
  height: 45px;
  vertical-align: middle;
}
</style>
