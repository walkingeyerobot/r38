export function checkNotNil<T>(value: T | undefined | null): T {
  if (value == undefined) {
    throw new Error("Value cannot be nil");
  }
  return value;
}
