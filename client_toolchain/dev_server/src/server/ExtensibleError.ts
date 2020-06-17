export class ExtensibleError extends Error {
  constructor(message: string) {
    super(message);
    this.name = this.constructor.name;
    this.message = message || this.name;
  }
};
