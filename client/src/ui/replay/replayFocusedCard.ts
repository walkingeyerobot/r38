import { type DraftCard } from "@/draft/DraftState";
import { ref } from "vue";

export const replayFocusedCard = ref<DraftCard | null>(null);
