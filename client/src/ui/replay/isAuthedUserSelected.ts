import { AuthStore } from '../../state/AuthStore';
import { DraftStore } from '../../state/DraftStore';
import { getUserPosition } from '../../state/util/userIsSeated';
import { ReplayStore } from '../../state/ReplayStore';

export function isAuthedUserSelected(
    authStore: AuthStore,
    draftStore: DraftStore,
    replayStore: ReplayStore,
) {
  const activeSeatPosition =
          getUserPosition(authStore.user?.id, draftStore.currentState);

  return activeSeatPosition != -1
      && replayStore.selection?.type == 'seat'
      && replayStore.selection?.id == activeSeatPosition;
}
