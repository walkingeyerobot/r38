
/**
 * Our webpack is configured to allow all image files to be imported as modules
 *
 * The result of the import is just the path to the file in the final served
 * folder.
 *
 * Sadly, there's no way to force the TS compiler to check that the module
 * actually exists.
 */
declare module '*.png' {
  const value: string;
  export default value;
}
