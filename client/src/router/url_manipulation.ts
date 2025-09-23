import type { RouteLocationNormalizedLoaded, Router, RouteLocationNormalized } from "vue-router";
import type { SelectedView } from "@/state/selection";
import type { ReplayStore, TimeMode } from "@/state/ReplayStore";
import type { DraftStore } from "@/state/DraftStore";
import { checkNotNil } from "@/util/checkNotNil";
import { checkExhaustive } from "@/util/checkExhaustive";

interface RouteProvider {
  $route: RouteLocationNormalizedLoaded;
  $router: Router;
}

/**
 * Navigates to the draft url specified by [params].
 */
export function pushDraftUrl(component: RouteProvider, params: DraftUrlState) {
  const path = generateDraftUrl(params);
  const asUser = component.$route.query.as;
  const query = asUser != undefined ? { as: asUser } : {};

  if (path != component.$route.path) {
    component.$router.push({ path, query });
  }
}

/**
 * Navigates to the draft URL specified by [params]. Any missing params are
 * filled in by examining the current draft URL.
 */
export function pushDraftUrlRelative(component: RouteProvider, params: RelativeDraftUrlParams) {
  const parsed = parseDraftUrl(component.$route);
  parsed.timeMode = parsed.timeMode || "synchronized";
  const finalParams = Object.assign({}, parsed, params);

  pushDraftUrl(component, finalParams);
}

/**
 * Constructs a draft URL matching the current Vuex state, then navigates to it.
 */
export function pushDraftUrlFromState(
  vue: RouteProvider,
  draftStore: DraftStore,
  replayStore: ReplayStore,
) {
  const params: DraftUrlState = {
    draftId: draftStore.draftId,
    timeMode: replayStore.timeMode,
    eventIndex: replayStore.eventPos,
    selection: replayStore.selection || undefined,
  };
  pushDraftUrl(vue, params);
}

function generateDraftUrl(params: DraftUrlState) {
  let path = `/draft/${params.draftId}/replay/`;

  if (params.eventIndex != undefined) {
    const timeModeCode = checkNotNil(params.timeMode) == "synchronized" ? "s" : "t";
    path += `${timeModeCode}/${params.eventIndex}/`;
  }

  if (params.selection != undefined) {
    switch (params.selection.type) {
      case "seat":
        path += `seat/${params.selection.id}/`;
        break;
      case "pack":
        path += `pack/${params.selection.id}/`;
        break;
      default:
        checkExhaustive(params.selection);
    }
  }

  return path;
}

/**
 * Reads the state specified in the current URL path and applies it to the
 * VueX data store.
 *
 * @param store The data store to apply changes to
 * @param params The URL params (`this.$route.params`)
 */
export function applyReplayUrlState(replayStore: ReplayStore, route: RouteLocationNormalized) {
  const parsedUrl = parseDraftUrl(route);

  if (parsedUrl.eventIndex == undefined) {
    parsedUrl.eventIndex = replayStore.events.length;
  }
  if (parsedUrl.timeMode == undefined) {
    parsedUrl.timeMode = "synchronized";
  }

  applyUrl(replayStore, parsedUrl);
}

export function parseDraftUrl(route: RouteLocationNormalized) {
  const parsedUrl: DraftUrlState = {
    draftId: parseInt(route.params["draftId"] as string),
  };

  const params = route.params["param"];

  if (params != undefined) {
    for (let i = 0; i < params.length; i++) {
      const param = params[i];
      i++;

      if (i >= params.length) {
        break;
      }

      const value = parseInt(params[i]);
      if (Number.isNaN(value)) {
        console.error("Invalid value:", params[i]);
        continue;
      }

      switch (param) {
        case "s":
          parsedUrl.timeMode = "synchronized";
          parsedUrl.eventIndex = value;
          break;
        case "t":
          parsedUrl.timeMode = "original";
          parsedUrl.eventIndex = value;
          break;
        case "pack":
          parsedUrl.selection = {
            type: "pack",
            id: value,
          };
          break;
        case "seat":
          parsedUrl.selection = {
            type: "seat",
            id: value,
          };
          break;
        case "":
          // No params, ignore this stub
          break;
        default:
          console.warn("Unrecognized URL param:", param);
          break;
      }
    }
  }

  return parsedUrl;
}

function applyUrl(replayStore: ReplayStore, parse: DraftUrlState) {
  if (parse.timeMode != undefined && parse.timeMode != replayStore.timeMode) {
    replayStore.setTimeMode(parse.timeMode);
  }

  if (
    parse.eventIndex != undefined &&
    parse.eventIndex != replayStore.eventPos &&
    parse.eventIndex >= 0 &&
    parse.eventIndex <= replayStore.events.length
  ) {
    replayStore.goTo(parse.eventIndex);
  }

  if (
    parse.selection != undefined &&
    (parse.selection.type != replayStore.selection?.type ||
      parse.selection.id != replayStore.selection.id)
  ) {
    replayStore.setSelection(parse.selection);
  }
}

export interface DraftUrlState {
  draftId: number;
  timeMode?: TimeMode;
  eventIndex?: number;
  selection?: SelectedView;
}

type RelativeDraftUrlParams = Pick<DraftUrlState, Exclude<keyof DraftUrlState, "draftId">>;
