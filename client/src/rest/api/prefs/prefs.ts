import { endpoint } from '../../endpoint';

export const routePrefs = endpoint({
  route: '/api/prefs/',
  method: 'get',
  queryVars: {},
  response: {} as {
    prefs: UserPrefDescriptor[];
  },
});

export const routeSetPref = endpoint({
  route: '/api/setpref/',
  method: 'post',
  bodyVars: {
    format: '' as string,
    elig: false as boolean,
  },
  response: {},
});

export interface UserPrefDescriptor {
  format: string;
  elig: boolean;
}

export const routeSetMtgoName = endpoint({
  route: '/api/setmtgoname/',
  method: 'post',
  bodyVars: {
    mtgoName: '' as string,
  },
  response: {},
});
