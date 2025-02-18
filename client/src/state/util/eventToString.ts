import type { TimelineEvent, TimelineAction } from "@/draft/TimelineEvent";

export function eventToString(event: TimelineEvent) {
  return (
    `[TimelineEvent id=${event.id}\n` +
    `  round=${event.round} roundEpoch=${event.roundEpoch} ` +
    `associatedSeat=${event.associatedSeat}\n` +
    `  type=${event.type}\n` +
    event.actions
      .map((action) => {
        return `  ${actionToString(action)}`;
      })
      .join("\n") +
    "]"
  );
}

function actionToString(action: TimelineAction) {
  switch (action.type) {
    case "move-card":
      return (
        `${action.type} ${action.subtype} ${action.card} ` +
        `"${action.cardName}" from=${action.from} to=${action.to}`
      );
    case "move-pack":
      return (
        `${action.type} ${action.subtype} pack=${action.pack} ` +
        `from=${action.from} to=${action.to}`
      );
    default:
      return action.type;
  }
}
