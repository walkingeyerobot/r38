/**
 * Tracks clicks in order to know whether to close any popovers or other
 * dismissable UI elements.
 *
 * Any UI that should be dismissed if the user clicks off from it (such as a
 * popover menu) should register themselves here. When the user clicks on the
 * dismissable UI, call onCaptureLocalMouseDown() to let the tracker know that
 * nothing should be dismissed.
 */
class GlobalClickTracker {
  private _clickWasHandled: boolean = false;
  private _listeners: UnhandledClickListener[] = [];

  onCaptureGlobalMouseDown() {
    this._clickWasHandled = false;
  }

  onBubbleGlobalMouseDown(e: MouseEvent) {
    if (!this._clickWasHandled) {
      for (const listener of this._listeners) {
        listener(e);
      }
    }
  }

  onCaptureLocalMouseDown() {
    this._clickWasHandled = true;
  }

  registerUnhandledClickListener(listener: UnhandledClickListener) {
    this._listeners.push(listener);
  }

  unregisterUnhandledClickListener(listener: UnhandledClickListener) {
    const index = this._listeners.indexOf(listener);
    if (index == -1) {
      throw new Error(`Cannot find listener ${listener}`);
    }
    this._listeners.splice(index, 1);
  }
}

export type UnhandledClickListener = (e: MouseEvent) => void;

export const globalClickTracker = new GlobalClickTracker();
