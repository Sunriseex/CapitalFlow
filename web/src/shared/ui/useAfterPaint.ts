import { useEffect, useState } from "react";

export function useAfterPaint() {
  const [ready, setReady] = useState(false);

  useEffect(() => {
    let frame = 0;
    const run = () => setReady(true);

    if (typeof window.requestAnimationFrame === "function") {
      frame = window.requestAnimationFrame(() => {
        frame = window.requestAnimationFrame(run);
      });
    } else {
      const timeout = window.setTimeout(run, 0);
      return () => window.clearTimeout(timeout);
    }

    return () => {
      if (frame) {
        window.cancelAnimationFrame(frame);
      }
    };
  }, []);

  return ready;
}
