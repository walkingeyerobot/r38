import type { nil } from "@/util/nil";
import beep from "../../sfx/beep.mp3";
import error from "../../sfx/error.mp3";

export const PickerSounds = {
  getScanSoundForSeat(seatId: number | nil) {
    return SCAN_SOUNDS[seatId ?? -1] ?? beep;
  },

  getErrorSoundForSeat(_seatId: number | nil) {
    return error;
  },

  defaults: {
    scanSound: beep,
    errorSound: error,
  },
};

// TODO: Store this as part of the draft config (and perhaps as a per-user
// setting).
const SCAN_SOUNDS = [beep, beep, beep, beep, beep, beep, beep, beep];
