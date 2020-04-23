import { TimeMode, RootState } from '../state/store';
import { Store } from 'vuex';
import { SelectedView } from '../state/selection';
import VueRouter, { Route } from 'vue-router';

/**
 * Navigates the UI to a particular state, updating the URL to match.
 *
 * At the moment, the navigable state includes:
 * (a) Where in the event stream we are
 * (b) What timeline mode is being used
 * (c) Which pack/seat is currently selected
 *
 * @param store The VueX data store
 * @param route An instance of the Vue route (`this.$route`)
 * @param router An instance of the Vue router (`this.$router`)
 * @param to The desired UI state to move to
 */
export function navTo(
  store: Store<RootState>,
  route: Route,
  router: VueRouter,
  to: TargetState,
) {
  const url = generateReplayUrl(store.state, to);
  if (url != route.path) {
    router.push(url);
  }
}

export interface TargetState {
  timeMode?: TimeMode,
  eventIndex?: number,
  selection?: SelectedView
}


function generateReplayUrl(state: RootState, to: TargetState): string {
  const draftId = state.draftId;

  const timeMode = to.timeMode != undefined ? to.timeMode : state.timeMode;
  const eventIndex =
      to.eventIndex != undefined ? to.eventIndex : state.eventPos;
  const shortTimelineMode = timeMode == 'original' ? 't' : 's';

  const root = `/replay/${draftId}/${shortTimelineMode}/${eventIndex}`;

  const selection = to.selection ? to.selection : state.selection;
  switch (selection?.type) {
    case 'seat':
      return `${root}/seat/${selection.id}`;
    case 'pack':
      return `${root}/pack/${selection.id}`;
    case undefined:
      return root;
    default:
      throw new Error(`Unrecognized selection type ${selection?.type}`);
  }
}

/**
 * Reads the state specified in the current URL path and applies it to the
 * VueX data store.
 *
 * @param store The data store to apply changes to
 * @param params The URL params (`this.$route.params`)
 */
export function applyReplayUrlState(
  store: Store<RootState>,
  params: { [key: string]: string },
) {
  let timelineMode: TimeMode | null = null;
  switch (params['timelineMode']) {
    case 't':
      timelineMode = 'original';
      break;
    case 's':
      timelineMode = 'synchronized';
      break;
    default:
      console.error('Unrecognized timeline mode:', params['timelineMode']);
      break;
  }

  if (timelineMode != null && timelineMode != store.state.timeMode) {
    store.commit('setTimeMode', timelineMode);
  }

  const eventIndex = parseInt(params['eventIndex']);
  if (eventIndex == NaN
      || eventIndex < 0
      || eventIndex > store.state.events.length) {
    console.error('Invalid event index:', params['eventIndex']);
  } else if (eventIndex != store.state.eventPos) {
    store.commit('goTo', eventIndex);
  }

  const selection = parseSelection(store, params);
  if (selection != null
      && (store.state.selection == null
          || selection.type != store.state.selection.type
          || selection.id != store.state.selection.id)) {
    store.commit('setSelection', selection);
  }
}

function parseSelection(
  store: Store<RootState>,
  params: { [key: string]: string },
): SelectedView | null {
  let selectionType: 'pack' | 'seat';
  switch (params['selectionType']) {
    case 'pack':
    case 'seat':
      selectionType = params['selectionType'];
      break;
    case undefined:
      return null;
    default:
      console.error('Invalid selection type:', params['selectionType']);
      return null;
  }

  const locationId = parseInt(params['locationId']);
  if (locationId == NaN || !store.state.draft.packs.has(locationId)) {
    console.error('Invalid location ID:', params['locationId']);
    return null;
  }

  return {
    type: selectionType,
    id: locationId,
  };
}
