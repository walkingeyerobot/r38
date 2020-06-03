import { ActionTree, Module, MutationPayload, Store } from 'vuex';
import { MixedCollection } from '../../util/MixedCollection';

export function vuexModule<S, D extends ModuleDef<S>>(
  rootStore: Store<any>,
  name: string,
  state: S,
  module: D,
): TypedModule<S, D> {
  const publicModule = {} as TypedModule<S, D>;

  const stateKeys = Object.keys(state) as (keyof S)[];
  const getterKeys = Object.keys(module.getters) as (keyof D['getters'])[];
  const mutatorKeys = Object.keys(module.mutations) as (keyof D['mutations'])[];
  const actionKeys = Object.keys(module.actions) as (keyof D['actions'])[];

  // Attempt to salvage state from pre-existing module
  // If things have changed too much, this won't work
  if (rootStore.hasModule(name)) {
    // Make a deep copy of the current state, erasing any of the property
    // proxies that Vue uses.
    // At the moment, this will break for all Maps and Sets
    const cleanedState = JSON.parse(JSON.stringify(rootStore.state));
    for (let key of stateKeys) {
      const value = cleanedState[key];
      if (value !== undefined) {
        state[key] = value;
      }
    }
    rootStore.unregisterModule(name);
  }

  rootStore.registerModule(name, transformModuleDef(state, module));

  for (let stateKey of stateKeys) {
    Object.defineProperty(publicModule, stateKey, {
      get: () => (rootStore.state as any)[name][stateKey],
      enumerable: true,
    });
  }

  for (let getterKey of getterKeys) {
    Object.defineProperty(publicModule, getterKey, {
      get: () => rootStore.getters[`${name}/${getterKey}`],
      enumerable: true,
    });
  }

  for (let mutatorKey of mutatorKeys) {
    (publicModule as any)[mutatorKey] = function(payload: any) {
      rootStore.commit(`${name}/${mutatorKey}`, payload);
    }
  }

  for (let actionKey of actionKeys) {
    (publicModule as any)[actionKey] = function(payload: any) {
      rootStore.dispatch(`${name}/${actionKey}`, payload);
    }
  }

  (publicModule as any).subscribe =
      function<P extends MutationPayload>(fn: (mutation: P, state: S) => any) {
        rootStore.subscribe<P>(((mutation, mutatedState) => {
          if (mutation.type.startsWith(name)) {
            fn(mutation, mutatedState[name]);
          }
        }));
      };

  return publicModule;
}

function transformModuleDef<S>(
  state: S,
  module: ModuleDef<S>,
): Module<S, {}> {
  const transformed = {
    namespaced: true,
    state: state,
    getters: module.getters,
    mutations: module.mutations,
    actions: transformActionHandlers(module.actions),
  };

  return transformed;
}

function transformActionHandlers<S>(actions: SimpleCollection<Action<S>>) {
  const out = {} as ActionTree<S, {}>;
  for (let key in actions) {
    out[key] =
        (context: { state: S }, payload?: any) =>
            actions[key](context.state, payload);
  }
  return out;
}

type ModuleDef<S> = {
  getters: SimpleCollection<Getter<S>>,
  mutations: SimpleCollection<Mutation<S>>,
  actions: SimpleCollection<Action<S>>,
}

type TypedModule<S, D extends ModuleDef<S>> =
    & Readonly<S>
    & PublicGetterCollection<S, D['getters']>
    & PublicMutatorCollection<S, D['mutations']>
    & PublicActionCollection<S, D['actions']>
    & Subscribe<S>
    ;

type Getter<S> = (state: Readonly<S>) => any;
type Mutation<S> = (state: S, payload?: any) => void;
type Action<S> = (context: S, payload?: any) => any;

interface Subscribe<S> {
  subscribe<P extends MutationPayload>(fn: (mutation: P, state: S) => any): void;
}

type SimpleCollection<T> = {
  [key: string]: T,
}

type PublicGetter<S, T extends (state: S) => any> =
    T extends (state: S) => infer R
        ? R
        : any;

type PublicGetterCollection<S, D extends SimpleCollection<Getter<S>>> = {
  [P in keyof D]: PublicGetter<S, D[P]>
}

type PublicMutator<S, T extends Mutation<S>> =
    T extends (state: S) => void ? () => void :
    T extends (state: S, payload: infer P) => void ? (payload: P) => void :
    any;

type PublicMutatorCollection<S, D extends SimpleCollection<Mutation<S>>> = {
  [P in keyof D]: PublicMutator<S, D[P]>
}

type PublicAction<S, T extends Action<S>> =
    T extends (state: S) => infer R ? () => Promise<R> :
    T extends (state: S, payload: infer P) => infer R ?
        (payload: P) => Promise<R> :
    any;

type PublicActionCollection<S, D extends SimpleCollection<Action<S>>> = {
  [P in keyof D]: PublicAction<S, D[P]>
}
