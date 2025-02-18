/* eslint-disable @typescript-eslint/no-empty-object-type */
/* eslint-disable @typescript-eslint/no-explicit-any */
import { type ActionTree, type Module, type MutationPayload, Store } from "vuex";
import { deepCopy } from "../../util/deepCopy";

export function vuexModule<S extends object, D extends ModuleDef<S>>(
  rootStore: Store<any>,
  name: string,
  state: S,
  module: D,
): TypedModule<S, D> {
  const publicModule = {} as TypedModule<S, D>;

  const stateKeys = Object.keys(state) as (keyof S)[];
  const getterKeys = Object.keys(module.getters) as StringKeys<D["getters"]>[];
  const mutatorKeys = Object.keys(module.mutations) as StringKeys<D["mutations"]>[];
  const actionKeys = Object.keys(module.actions) as StringKeys<D["actions"]>[];

  // Attempt to salvage state from pre-existing module
  // If things have changed too much, this won't work
  if (rootStore.hasModule(name)) {
    // Make a deep copy of the current state, erasing any of the property
    // proxies that Vue uses.
    const cleanedState = deepCopy((rootStore.state as any)[name]);

    // Copy the existing state onto the new module's state definition
    for (const key of stateKeys) {
      const value = cleanedState[key];
      if (value !== undefined) {
        state[key] = value;
      }
    }
    rootStore.unregisterModule(name);
  }

  rootStore.registerModule(name, transformModuleDef(state, module));

  for (const stateKey of stateKeys) {
    Object.defineProperty(publicModule, stateKey, {
      get: () => (rootStore.state as any)[name][stateKey],
      enumerable: true,
    });
  }

  for (const getterKey of getterKeys) {
    Object.defineProperty(publicModule, getterKey, {
      get: () => rootStore.getters[`${name}/${getterKey}`],
      enumerable: true,
    });
  }

  for (const mutatorKey of mutatorKeys) {
    (publicModule as any)[mutatorKey] = function (payload: any) {
      rootStore.commit(`${name}/${mutatorKey}`, payload);
    };
  }

  for (const actionKey of actionKeys) {
    (publicModule as any)[actionKey] = function (payload: any) {
      rootStore.dispatch(`${name}/${actionKey}`, payload);
    };
  }

  (publicModule as any).subscribe = function <P extends MutationPayload>(
    fn: (mutation: P, state: S) => any,
  ) {
    rootStore.subscribe<P>((mutation, mutatedState) => {
      if (mutation.type.startsWith(name)) {
        fn(mutation, mutatedState[name]);
      }
    });
  };

  return publicModule;
}

function transformModuleDef<S>(state: S, module: ModuleDef<S>): Module<S, {}> {
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
  for (const key in actions) {
    out[key] = (context: { state: S }, payload?: any) => actions[key](context.state, payload);
  }
  return out;
}

type ModuleDef<S> = {
  getters: SimpleCollection<Getter<S>>;
  mutations: SimpleCollection<Mutation<S>>;
  actions: SimpleCollection<Action<S>>;
};

type TypedModule<S, D extends ModuleDef<S>> = Readonly<S> &
  PublicGetterCollection<S, D["getters"]> &
  PublicMutatorCollection<S, D["mutations"]> &
  PublicActionCollection<S, D["actions"]> &
  ModuleFuncs<S>;

type Getter<S> = (state: Readonly<S>) => any;
type Mutation<S> = (state: S, payload?: any) => void;
type Action<S> = (context: S, payload?: any) => any;

interface ModuleFuncs<S> {
  subscribe<P extends MutationPayload>(fn: (mutation: P, state: S) => any): void;
}

type SimpleCollection<T> = {
  [key: string]: T;
};

type PublicGetter<S, T extends (state: S) => any> = T extends (state: S) => infer R ? R : any;

type PublicGetterCollection<S, D extends SimpleCollection<Getter<S>>> = {
  [P in keyof D]: PublicGetter<S, D[P]>;
};

type PublicMutator<S, T extends Mutation<S>> = T extends (state: S) => void
  ? () => void
  : T extends (state: S, payload: infer P) => void
    ? (payload: P) => void
    : any;

type PublicMutatorCollection<S, D extends SimpleCollection<Mutation<S>>> = {
  [P in keyof D]: PublicMutator<S, D[P]>;
};

type PublicAction<S, T extends Action<S>> = T extends (state: S) => infer R
  ? () => Promise<R>
  : T extends (state: S, payload: infer P) => infer R
    ? (payload: P) => Promise<R>
    : any;

type PublicActionCollection<S, D extends SimpleCollection<Action<S>>> = {
  [P in keyof D]: PublicAction<S, D[P]>;
};

type StringKeys<T extends object> = Extract<keyof T, string>;
