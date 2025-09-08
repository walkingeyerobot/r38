import { ref } from "vue";
import { useSound } from "@vueuse/sound";
import beep from "../sfx/beep.mp3";
import error from "../sfx/error.mp3";

export type BoopPrompt = "none" | "missing-hardware" | "request-permission";

const CARD_UUID_PATTERN = /\w{8}-\w{4}-\w{4}-\w{4}-\w{12}/;

export class RfidHandler {
  private isNfcSupported = "NDEFReader" in window;
  private hasNfcPermission = ref<boolean>(false);
  private scanListener = (event: CustomEvent<string>) => this.onRfidScan(event);

  constructor(private readonly handleCardScanned: ((cardRfid: string) => void) | undefined) {}

  async start(scan = true) {
    // Tell iOS app to start scanning
    postMessage("scan");
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    (window as any).webkit?.messageHandlers?.scanner?.postMessage("scan");

    document.body.addEventListener("rfidScan", this.scanListener);

    if (this.isNfcSupported) {
      console.log("NFC reading supported!");

      const nfcPermissionStatus = await navigator.permissions.query({
        name: "nfc" as unknown as PermissionName,
      });

      this.hasNfcPermission.value = nfcPermissionStatus.state == "granted";
      console.log("Initial NFC permission status is", nfcPermissionStatus.state);
      nfcPermissionStatus.addEventListener("change", () => {
        console.log("NFC permission status changed to", nfcPermissionStatus.state);
        this.hasNfcPermission.value = nfcPermissionStatus.state == "granted";
      });

      if (this.hasNfcPermission.value && scan) {
        console.log("Have permission, preparing to scan!");
        this.scanForTag();
      }
    }
  }

  scanForTag() {
    const reader = new NDEFReader();
    reader.onreadingerror = () => {
      console.log("Cannot read data from the NFC tag. Try another one?");
    };
    reader.onreading = (e) => {
      console.log("NDEF message read.");
      console.log("Records:");
      for (const record of e.message.records) {
        console.log(record.id, record.recordType);
        if (record.recordType == "text") {
          const td = new TextDecoder(record.encoding);
          const text = td.decode(record.data);
          console.log("Text:", td.decode(record.data));
          if (CARD_UUID_PATTERN.test(text)) {
            console.log("It's a thing!");
            if (this.handleCardScanned) {
              this.handleCardScanned(text);
            }
          }
        }
      }
    };
    reader
      .scan()
      .then(() => {
        console.log("Scan started successfully.");
      })
      .catch((error) => {
        console.log(`Error! Scan failed to start: ${error}.`);
      });
  }

  async writeTag(card: string) {
    const reader = new NDEFReader();
    console.log(`Writing ${card}.`);
    try {
      await reader.write(card);
      console.log(`Wrote ${card}.`);
      useSound(beep).play();
    } catch (e) {
      console.log(`Write failed: ${e}`);
      useSound(error).play();
    }
  }

  stop() {
    document.body.removeEventListener("rfidScan", this.scanListener);
  }

  getPrompt(): BoopPrompt {
    const isAppleMobileOs = navigator.platform.substring(0, 2) == "iP";

    if (this.hasNfcPermission.value || isAppleMobileOs) {
      return "none" as const;
    } else if (!this.isNfcSupported) {
      return "missing-hardware" as const;
    } else {
      return "request-permission" as const;
    }
  }

  private async onRfidScan(event: CustomEvent<string>) {
    const cardRfid = decodeURIComponent(
      Array.prototype.map
        .call(atob(event.detail), (c) => `%${`00${c.charCodeAt(0).toString(16)}`.slice(-2)}`)
        .slice(3)
        .join(""),
    );

    if (this.handleCardScanned) {
      this.handleCardScanned(cardRfid);
    }
  }
}
