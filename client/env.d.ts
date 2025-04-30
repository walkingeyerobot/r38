/// <reference types="vite/client" />

// The latest version of vuex messed up their package.json definition in a way
// that breaks their typing exports (how did no one catch this?). So we have
// to manually export their typings here.
declare module "vuex" {
  export * from "vuex/types/index.d.ts";
  export * from "vuex/types/helpers.d.ts";
  export * from "vuex/types/logger.d.ts";
  export * from "vuex/types/vue.d.ts";
}

// Declare the special RFID event that our enclosing apps will inject if
// someone scans a card
interface HTMLElementEventMap {
  rfidScan: CustomEvent<string>;
}
