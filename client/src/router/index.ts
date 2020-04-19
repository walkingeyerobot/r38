import Vue from 'vue';
import VueRouter, { RouteConfig } from 'vue-router';
import DeckBuilderScreen from '../ui/DeckBuilderScreen.vue';
import Home from '../ui/Home.vue';

Vue.use(VueRouter);

const routes = [
  {
    path: '/replay/:draftId/',
    component: Home,
  },
  {
    path: '/replay/:draftId/:timelineMode/:eventIndex/',
    component: Home,
  },
  {
    path:
        '/replay/:draftId/:timelineMode/:eventIndex/:selectionType/:locationId',
    component: Home,
  },
  {
    path: '/deckbuilder/*',
    name: 'DeckBuilderScreen',
    component: DeckBuilderScreen,
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
