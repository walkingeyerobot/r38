import { createRouter, createWebHistory } from "vue-router";

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    {
      path: `/`,
      name: "root",
      component: () => import("../ui/home/Home.vue"),
    },
    {
      path: `/home`,
      name: "home",
      component: () => import("../ui/home/Home.vue"),
    },
    {
      path: `/login`,
      name: "login",
      component: () => import("../ui/login/Login.vue"),
    },
    {
      path: `/prefs`,
      name: "prefs",
      component: () => import("../ui/prefs/Prefs.vue"),
    },
    {
      path: `/draft/:draftId(\\d+)/join`,
      name: "join",
      component: () => import("../ui/draft/join/JoinPage.vue"),
    },
    {
      path: `/draft/:draftId(\\d+)/pick`,
      name: "picker",
      component: () => import("../ui/PickerPage.vue"),
    },
    {
      path: `/draft/:draftId(\\d+)/replay/:param*`,
      name: "replay",
      component: () => import("../ui/Replay.vue"),
    },
    {
      path: "/deckbuilder/:draftId(\\d+)",
      name: "deckbuilder",
      component: () => import("../ui/DeckBuilder.vue"),
    },
    {
      path: "/shuffler/:draftId(\\d+)",
      name: "shuffler",
      component: () => import("../ui/CubeShuffler.vue"),
      props: true,
    },
    {
      path: "/tagwriter/:set",
      name: "tagwriter",
      component: () => import("../ui/TagWriter.vue"),
      props: true,
    },
    {
      path: "/samplepack/:set",
      redirect: (to) => {
        return to.path + `/${Math.floor(Math.random() * Math.pow(2, 63))}`;
      },
    },
    {
      path: "/samplepack/:set/:seed",
      name: "samplepack",
      component: () => import("../ui/SamplePack.vue"),
      props: (route) => ({ set: route.params.set, seed: Number(route.params.seed) }),
    },
    {
      path: "/draftpacks/:id",
      name: "samplepack",
      component: () => import("../ui/DraftPacks.vue"),
      props: true,
    },
  ],
});

export default router;
