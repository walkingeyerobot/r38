import { FAKE_DATA_03 } from '../fake_data/FAKE_DATA_03';

export function getServerPayload() {
  if (window.DraftString != undefined) {
    console.log('Found server payload, loading!');
    return JSON.parse(window.DraftString);
  } else {
    console.log(`Couldn't find server payload, falling back to default...`);
    return FAKE_DATA_03;
  }
}
