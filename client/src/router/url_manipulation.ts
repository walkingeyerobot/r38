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
  route: Route,
) {
  const parsedUrl = parseUrl(route);
  applyUrl(store, parsedUrl);
}

function parseUrl(route: Route) {
  const parsedUrl: ParsedUrl = {
    draftId: parseInt(route.params['draftId']),
  };

  const rawParams  = route.params['param'] || '';
  const params = rawParams.split('/');

  for (let i = 0; i < params.length; i++) {

    const param = params[i];
    i++;
    const value = parseInt(params[i]);
    if (value == NaN) {
      console.error('Invalid value:', value);
      continue;
    }

    switch (param) {
      case 's':
        parsedUrl.timeMode = 'synchronized';
        parsedUrl.eventIndex = value;
        break;
      case 't':
        parsedUrl.timeMode = 'original';
        parsedUrl.eventIndex = value;
        break;
      case 'pack':
        parsedUrl.selection = {
          type: 'pack',
          id: value,
        }
        break;
      case 'seat':
        parsedUrl.selection = {
          type: 'seat',
          id: value,
        }
        break;
      case '':
        // No params, ignore this stub
        break;
      default:
        console.warn('Unrecognized URL param:', param);
        break;
    }
  }

  return parsedUrl;
}

function applyUrl(store: Store<RootState>, parse: ParsedUrl) {
  if (store.state.draftId != parse.draftId) {
    store.commit('setDraftId', parse.draftId);
  }

  if (parse.timeMode != undefined
      && parse.timeMode != store.state.timeMode) {
    store.commit('setTimeMode', parse.timeMode);
  }

  if (parse.eventIndex != undefined
      && parse.eventIndex >= 0
      && parse.eventIndex <= store.state.events.length) {
    store.commit('goTo', parse.eventIndex);
  }

  if (parse.selection != undefined
      && (parse.selection.type != store.state.selection?.type
          || parse.selection.id != store.state.selection.id)) {
    store.commit('setSelection', parse.selection);
  }
}

interface ParsedUrl {
  draftId: number,
  timeMode?: TimeMode,
  eventIndex?: number,
  selection?: SelectedView,
}
