import { vuexModule } from "./vuex/vuexModule";
import { rootStore } from "./store";

export const formatStore = vuexModule(
  rootStore,
  "format",
  {
    layout: "desktop",
  } as FormatState,
  {
    mutations: {
      setLayout(state: FormatState, layout: LayoutFormFactor) {
        state.layout = layout;
      },
    },

    getters: {},

    actions: {},
  },
);

export interface FormatState {
  layout: LayoutFormFactor;
}

export type LayoutFormFactor = "mobile" | "desktop";

export type FormatStore = typeof formatStore;
