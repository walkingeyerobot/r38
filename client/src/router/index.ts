import Vue from 'vue';
import VueRouter, { RouteConfig } from 'vue-router';

Vue.use(VueRouter);

// See https://github.com/pillarjs/path-to-regexp/ for route matching language

// We use chunk splitting so that we only load the source for the route we're
// looking at right now. That's what the webpackChunkName directive does below

const routes = [
  {
    path: `/`,
    component: () =>
        import(/* webpackChunkName: 'home' */ '../ui/home/Home.vue'),
  },
  {
    path: `/login`,
    component: () =>
        import(/* webpackChunkName: 'login' */ '../ui/login/Login.vue'),
  },
  {
    path: `/replay/:draftId(\\d+)/:param*`,
    component: () =>
        import(/* webpackChunkName: 'replay' */ '../ui/Replay.vue'),
  },
  {
    path: `/draft/:draftId(\\d+)/:param*`,
    component: () =>
        import(/* webpackChunkName: 'replay' */ '../ui/Replay.vue'),
  },
  {
    path: '/deckbuilder/:draftId(\\d+)',
    component: () =>
        import(/* webpackChunkName: 'deckbuilder' */ '../ui/DeckBuilder.vue'),
  },
] as RouteConfig[];

const router = new VueRouter({
  mode: 'history',
  base: process.env.BASE_URL,
  routes,
});

export default router;
