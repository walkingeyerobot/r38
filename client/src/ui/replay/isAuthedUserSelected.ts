import { AuthStore } from '../../state/AuthStore';
import { DraftStore } from '../../state/DraftStore';
import { getPlayerSeat } from '../../state/util/userIsSeated';
import { ReplayStore } from '../../state/ReplayStore';

export function isAuthedUserSelected(
    authStore: AuthStore,
    draftStore: DraftStore,
    replayStore: ReplayStore,
) {
  const authedSeat = getPlayerSeat(authStore.user?.id, draftStore.currentState);

  return authedSeat != null
      && replayStore.selection?.type == 'seat'
      && replayStore.selection?.id == authedSeat.position;
}
