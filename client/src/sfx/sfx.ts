import { useSound } from '@vueuse/sound'
import beep from './beep.mp3'
import error from './error.mp3'
import type { DraftPlayer } from "@/draft/DraftState.ts";

const scanSounds = [
  // change these to different sounds once we have them
  useSound(beep),
  useSound(beep),
  useSound(beep),
  useSound(beep),
  useSound(beep),
  useSound(beep),
  useSound(beep),
  useSound(beep),
];

const errorSounds = [
  // change these to different sounds once we have them
  useSound(error),
  useSound(error),
  useSound(error),
  useSound(error),
  useSound(error),
  useSound(error),
  useSound(error),
  useSound(error),
];

export function playScanSound(player: DraftPlayer) {
  scanSounds[player.scanSound].play();
}

export function playErrorSound(player: DraftPlayer) {
  errorSounds[player.errorSound].play();
}
