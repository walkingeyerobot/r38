import { rootStore } from "./store";
import { vuexModule } from "./vuex/vuexModule";

/**
 * Stores the logged-in user's credentials (if present)
 */
export const authStore = vuexModule(
  rootStore,
  "auth",
  {
    user: null,
  } as AuthenticatedUserState,
  {
    mutations: {
      setUser(state: AuthenticatedUserState, user: AuthenticatedUser) {
        state.user = user;
      },

      setUserError(state: AuthenticatedUserState) {
        state.user = "error";
      },
    },

    getters: {
      userId(state: AuthenticatedUserState) {
        if (!state.user || state.user === "error") {
          return 0;
        }
        return state.user.id;
      },
      userName(state: AuthenticatedUserState) {
        if (!state.user || state.user === "error") {
          return undefined;
        }
        return state.user.name;
      },
      userPicture(state: AuthenticatedUserState) {
        if (!state.user || state.user === "error") {
          return undefined;
        }
        return state.user.picture;
      },
      userMtgoName(state: AuthenticatedUserState) {
        if (!state.user || state.user === "error") {
          return undefined;
        }
        return state.user.mtgoName;
      },
    },

    actions: {},
  },
);

export type AuthStore = typeof authStore;

interface AuthenticatedUserState {
  user: AuthenticatedUser | null | "error";
}

export interface AuthenticatedUser {
  id: number;
  name: string;
  picture: string;
  mtgoName: string;
  isImpersonated: boolean;
}
