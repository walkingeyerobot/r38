import { devReplayData } from '../fake_data/devReplayData';


export function getServerPayload() {
  if (window.DraftString != undefined) {
    console.log('Found server payload, loading!');
    return JSON.parse(window.DraftString);
  } else if (devReplayData != null) {
    console.log(`Couldn't find server payload, falling back to default...`);
    return devReplayData;
  } else {
    throw new Error(`No server payload found, but in a production build!`);
  }
}
