export function markPerformance(name: string) {
  if (!import.meta.env.DEV || typeof performance === "undefined" || typeof performance.mark !== "function") {
    return () => {};
  }

  const start = `${name}:start`;
  const end = `${name}:end`;
  performance.mark(start);

  return () => {
    performance.mark(end);
    if (typeof performance.measure === "function") {
      performance.measure(name, start, end);
    }
  };
}
