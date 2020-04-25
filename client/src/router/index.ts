import Vue from 'vue';
import VueRouter, { RouteConfig } from 'vue-router';
import DeckBuilder from '../ui/DeckBuilder.vue';
import Replay from '../ui/Replay.vue';

Vue.use(VueRouter);

// See https://github.com/pillarjs/path-to-regexp/ for route matching language

const routes = [
  {
    path: `/replay/:draftId(\\d+)/:param*`,
    component: Replay,
  },
  {
    path: '/deckbuilder/*',
    component: DeckBuilder,
  }
  // TODO: Figure out route splitting in the future
  // {
  //   path: '/about',
  //   name: 'About',
  //   // route level code-splitting
  //   // this generates a separate chunk (about.[hash].js) for this route
  //   // which is lazy-loaded when the route is visited.
  //   component: () =>
  //     import(/* webpackChunkName: 'about' */ '../ui/About.vue'),
  // },
] as RouteConfig[];

if (DEVELOPMENT) {
  routes.push({
    path: '/*',
    redirect: '/replay/0',
  })
}

const router = new VueRouter({
  mode: 'history',
  base: process.env.BASE_URL,
  routes,
});

export default router;
