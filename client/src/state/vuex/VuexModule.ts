import { Module } from 'vuex';

/**
 * This function doens't do anything, it just exists to enforce that the object
 * passed into it is a valid Vuex module definition.
 */
export function VuexModule<S, R>(definition: Module<S, R>): Module<S, R> {
  return definition;
}
