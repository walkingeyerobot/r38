import {
  createRouter,
  createWebHistory,
  type LocationQueryValue,
  type RouteRecordNameGeneric,
} from "vue-router";
import { authStore } from "@/state/AuthStore.ts";
import { ROUTE_USER_INFO, type SourceUserInfo } from "@/rest/api/userinfo/userinfo.ts";
import { fetchEndpoint } from "@/fetch/fetchEndpoint.ts";
import DefaultAvatar from "@/ui/shared/avatars/default_avatar.png";

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
      path: `/draft/:draftId(\\d+)/`,
      redirect: { name: "replay" },
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
      name: "draftpacks",
      component: () => import("../ui/DraftPacks.vue"),
      props: true,
    },
  ],
});

const requiresAuth: RouteRecordNameGeneric[] = ["prefs", "join", "picker", "replay", "deckbuilder"];

router.beforeEach(async (to, _from) => {
  const user = await loadAuthInfo(to.query["as"]);
  if (requiresAuth.includes(to.name) && user !== "error" && user?.id === 0) {
    return { name: "login" };
  }
  return true;
});

async function loadAuthInfo(as: LocationQueryValue | LocationQueryValue[]) {
  if (authStore.user) {
    return authStore.user;
  }

  let result: SourceUserInfo;

  const asPlayer = parseInt(unwrapQuery(as) ?? "NaN");
  const asPlayerId = isNaN(asPlayer) ? undefined : asPlayer;

  try {
    result = await fetchEndpoint(ROUTE_USER_INFO, { as: asPlayerId });
  } catch (e) {
    console.error("Error fetching user info:", e);
    authStore.setUserError();
    return "error";
  }

  const user = {
    id: result.userId,
    name: result.name,
    picture: result.picture || DefaultAvatar,
    mtgoName: result.mtgoName,
    isImpersonated: asPlayerId != undefined,
  };
  authStore.setUser(user);
  return user;
}

function unwrapQuery<T>(value: T | T[]): T {
  if (value instanceof Array) {
    return value[0];
  } else {
    return value;
  }
}

export default router;
