<!--

Displays a single mana symbol, with or without a shadow

-->

<template>
  <img :src="imageSrc" class="_mana-symbol" :class="{ _shadow: shadow }" />
</template>

<script lang="ts">
import MANA_W from "./W_small.png";
import MANA_U from "./U_small.png";
import MANA_B from "./B_small.png";
import MANA_R from "./R_small.png";
import MANA_G from "./G_small.png";

import MANA_WP from "./WP.svg";
import MANA_UP from "./UP.svg";
import MANA_BP from "./BP.svg";
import MANA_RP from "./RP.svg";
import MANA_GP from "./GP.svg";

import MANA_2W from "./2W.svg";
import MANA_2U from "./2U.svg";
import MANA_2B from "./2B.svg";
import MANA_2R from "./2R.svg";
import MANA_2G from "./2G.svg";

import MANA_WU from "./WU.svg";
import MANA_WB from "./WB.svg";
import MANA_UB from "./UB.svg";
import MANA_UR from "./UR.svg";
import MANA_BR from "./BR.svg";
import MANA_BG from "./BG.svg";
import MANA_RG from "./RG.svg";
import MANA_RW from "./RW.svg";
import MANA_GW from "./GW.svg";
import MANA_GU from "./GU.svg";

import MANA_0 from "./0.svg";
import MANA_1 from "./1.svg";
import MANA_2 from "./2.svg";
import MANA_3 from "./3.svg";
import MANA_4 from "./4.svg";
import MANA_5 from "./5.svg";
import MANA_6 from "./6.svg";
import MANA_7 from "./7.svg";
import MANA_8 from "./8.svg";
import MANA_9 from "./9.svg";
import MANA_10 from "./10.svg";
import MANA_11 from "./11.svg";
import MANA_12 from "./12.svg";
import MANA_13 from "./13.svg";
import MANA_14 from "./14.svg";
import MANA_15 from "./15.svg";
import MANA_16 from "./16.svg";
import MANA_17 from "./17.svg";
import MANA_18 from "./18.svg";
import MANA_19 from "./19.svg";
import MANA_20 from "./20.svg";

import MANA_X from "./X.svg";
import MANA_C from "./C.svg";
import MANA_S from "./S.svg";

import MANA_UNKNOWN from "./unknown.svg";

import { defineComponent } from "vue";
import { type SimpleCollection } from "../../../util/SimpleCollection";

export default defineComponent({
  props: {
    /**
     * Accepts the official comprehensive rules plaintext notation, but with
     * the brackets removed. For example, 'G' or '2/W' or 'W/U'.
     */
    code: {
      type: String,
      required: true,
    },

    /**
     * Whether to display a hard shadow underneath the icon (this treatment
     * is used in mana costs but not in text bodies).
     */
    shadow: {
      type: Boolean,
      required: false,
      default: true,
    },
  },

  computed: {
    imageSrc(): string {
      if (MANA_SYMBOL_PATHS[this.code] == undefined) {
        console.log("UNKNOWN SYMBOL:", this.code);
      }
      return MANA_SYMBOL_PATHS[this.code] || MANA_UNKNOWN;
    },
  },
});

const MANA_SYMBOL_PATHS = {
  W: MANA_W,
  U: MANA_U,
  B: MANA_B,
  R: MANA_R,
  G: MANA_G,

  "W/P": MANA_WP,
  "U/P": MANA_UP,
  "B/P": MANA_BP,
  "R/P": MANA_RP,
  "G/P": MANA_GP,

  "2/W": MANA_2W,
  "2/U": MANA_2U,
  "2/B": MANA_2B,
  "2/R": MANA_2R,
  "2/G": MANA_2G,

  "W/U": MANA_WU,
  "W/B": MANA_WB,
  "U/B": MANA_UB,
  "U/R": MANA_UR,
  "B/R": MANA_BR,
  "B/G": MANA_BG,
  "R/G": MANA_RG,
  "R/W": MANA_RW,
  "G/W": MANA_GW,
  "G/U": MANA_GU,

  "0": MANA_0,
  "1": MANA_1,
  "2": MANA_2,
  "3": MANA_3,
  "4": MANA_4,
  "5": MANA_5,
  "6": MANA_6,
  "7": MANA_7,
  "8": MANA_8,
  "9": MANA_9,
  "10": MANA_10,
  "11": MANA_11,
  "12": MANA_12,
  "13": MANA_13,
  "14": MANA_14,
  "15": MANA_15,
  "16": MANA_16,
  "17": MANA_17,
  "18": MANA_18,
  "19": MANA_19,
  "20": MANA_20,

  X: MANA_X,
  C: MANA_C,
  S: MANA_S,
} as SimpleCollection<string>;

// Random note: the font face for numbers in mana symbols is Plantin
</script>

<style scoped>
._mana-symbol {
  width: 13px;
  height: 13px;
  vertical-align: bottom;
}

._shadow {
  border-radius: 100px;
  box-shadow: -0.5px 1px 0 #444;
}
</style>
