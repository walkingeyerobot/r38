import type { AuthStore } from "@/state/AuthStore";
import type { DraftStore } from "@/state/DraftStore";
import type { ReplayStore } from "@/state/ReplayStore";
import { getPlayerSeat } from "@/state/util/userIsSeated";

export function isAuthedUserSelected(
  authStore: AuthStore,
  draftStore: DraftStore,
  replayStore: ReplayStore,
) {
  const authedSeat = getPlayerSeat(authStore.user?.id, draftStore.currentState);

  return (
    authedSeat != null &&
    replayStore.selection?.type == "seat" &&
    replayStore.selection?.id == authedSeat.position
  );
}
