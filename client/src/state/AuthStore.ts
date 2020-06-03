import { rootStore } from './store';
import { vuexModule } from './vuex/vuexModule';

/**
 * Stores the logged-in user's credentials (if present)
 */
export const authStore = vuexModule(
  rootStore,
  'auth',
  {

    user: null,

  } as AuthenticatedUserState,
  {
    mutations: {
      setUser(state: AuthenticatedUserState, user: AuthenticatedUser) {
        state.user = user;
      }
    },

    getters: {},

    actions: {},
  },
);

export type AuthStore = typeof authStore;

interface AuthenticatedUserState {
  user: AuthenticatedUser | null;
}

export interface AuthenticatedUser {
  id: number;
  name: string;
  picture: string;
}
