
/**
 * Given a type T, makes all properties of T whose types can be undefined
 * optional.
 *
 * In other works, converts `{ a: number | undefined }` to `{ a?: number }`.
 */
export type Optionalize<T> =
    Partial<Pick<T, KeysOfUndefProps<T>>>
        & Pick<T, Exclude<keyof T, KeysOfUndefProps<T>>>;

/** The keys of T whose properties can hold undefined values */
type KeysOfUndefProps<T> = {
  // We have to use Extract here because of how extends interacts with type
  // unions.
  // Alternately, we could write the following:
  // [P in keyof T]: undefined extends T[P] ? P : never
  // But I find Extract to me more intuitive
  [P in keyof T]: Extract<T[P], undefined> extends never ? never : P
}[keyof T];




interface ExampleObj {
  a: string,
  b: number,
  c: string,
  d: string | undefined,
  e: undefined,
}

type C4 = Optionalize<ExampleObj>;

let foo: C4;

let ex: { a?: string; b: string; };
