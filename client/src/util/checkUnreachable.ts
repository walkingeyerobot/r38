/**
 * Typechecks that [caseValue] is `never`
 *
 * Used to ensure that switch statements are exhaustive. This function should
 * never be executed in compiled code (since it will cause the compile to fail),
 * but if it does, it will throw an error to that effect.
 *
 * This function is useful for forcing the compiler to check that a switch
 * statement is exhaustive, since the type of the case value in the default
 * section will be `never`:
 *
 * ```
 *  switch (myCaseValue) {
 *    case 'foo':
 *      // ...
 *    case 'bar':
 *      // ...
 *    default:
 *      checkUnreachable(myCaseValue);
 *  }
 * ```
 */
export function checkUnreachable(caseValue: never) {
  throw new Error(`Unexpected case value: ${JSON.stringify(caseValue)}`);
}
