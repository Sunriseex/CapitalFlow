export function runThemeRipple(
  trigger: HTMLElement,
  applyTheme: () => void,
) {
  const reducedMotion =
    "matchMedia" in window &&
    window.matchMedia("(prefers-reduced-motion: reduce)").matches;

  if (reducedMotion) {
    applyTheme();
    return;
  }

  const rect = trigger.getBoundingClientRect();
  const x = rect.left + rect.width / 2;
  const y = rect.top + rect.height / 2;
  const radius = Math.ceil(
    Math.hypot(
      Math.max(x, window.innerWidth - x),
      Math.max(y, window.innerHeight - y),
    ),
  );
  const root = document.documentElement;

  root.style.setProperty("--theme-ripple-x", `${x}px`);
  root.style.setProperty("--theme-ripple-y", `${y}px`);
  root.style.setProperty("--theme-ripple-radius", `${radius}px`);

  const viewTransitionDocument = document as Document & {
    startViewTransition?: (callback: () => void) => {
      ready: Promise<void>;
      finished: Promise<void>;
    };
  };

  if (typeof viewTransitionDocument.startViewTransition === "function") {
    root.classList.add("theme-view-transition");
    const transition = viewTransitionDocument.startViewTransition(applyTheme);
    void transition.finished.finally(() => {
      root.classList.remove("theme-view-transition");
    });
    return;
  }

  root.classList.remove("theme-ripple-fallback");
  applyTheme();
  window.requestAnimationFrame(() => {
    root.classList.add("theme-ripple-fallback");
    window.setTimeout(() => {
      root.classList.remove("theme-ripple-fallback");
    }, 620);
  });
}
