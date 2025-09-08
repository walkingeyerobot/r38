import { getErrorMessage } from "@/util/error/getErrorMessage";
import { onMounted, onUnmounted, ref } from "vue";

export function useWakelock() {
  let wakeLock: WakeLockSentinel | null = null;
  const status = ref<EnumResult<"acquiring" | "acquired" | "not_supported">>({
    status: "acquiring",
  });

  onMounted(() => {
    if ("wakeLock" in navigator) {
      navigator.wakeLock.request("screen").then(
        (lock) => {
          console.log("Got a wake lock!");
          wakeLock = lock;
          status.value = { status: "acquired" };
        },
        (e: unknown) => {
          console.log("Error when trying to acquire wake lock", e);
          status.value = { status: "error", message: getErrorMessage(e) };
        },
      );
    } else {
      console.log("Wake locks not supported");
      status.value = { status: "not_supported" };
    }
  });

  onUnmounted(() => {
    wakeLock?.release();
  });

  return {
    status,
  };
}

type EnumResult<T extends string> =
  | {
      status: "error";
      message: string;
    }
  | {
      status: T;
    };
