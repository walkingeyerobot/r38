import { ExtensibleError } from "./ExtensibleError";

export class NotFoundError extends ExtensibleError {
  constructor(message: string) {
    super(message);
  }
}
