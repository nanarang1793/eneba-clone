// Tests run in Node, which has no real browser localStorage (newer Node
// versions define a placeholder that doesn't store anything). The cart store
// saves itself there, so replace it with a simple in-memory version.
// defineProperty (not `globalThis.localStorage = …`) also replaces Node's
// built-in property, which a plain assignment can't overwrite.
const memory = new Map<string, string>();
const storage: Storage = {
  getItem: (k) => memory.get(k) ?? null,
  setItem: (k, v) => void memory.set(k, String(v)),
  removeItem: (k) => void memory.delete(k),
  clear: () => memory.clear(),
  key: (i) => [...memory.keys()][i] ?? null,
  get length() {
    return memory.size;
  },
};
Object.defineProperty(globalThis, "localStorage", { value: storage, configurable: true, writable: true });
