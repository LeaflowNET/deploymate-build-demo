import { defineComponent } from "vue";
import { flushPromises, mount } from "@vue/test-utils";
import { afterEach, describe, expect, it, vi } from "vitest";
import { useTasks } from "./useTasks";
const emptyPage = {
  tasks: [],
  total: 0,
  stats: { todo: 0, doing: 0, done: 0 },
};
const info = {
  name: "Taskmate",
  version: "test",
  commit: "test",
  message: "test",
  startedAt: new Date().toISOString(),
};
function json(value: unknown, status = 200) {
  return new Response(JSON.stringify(value), {
    status,
    headers: { "Content-Type": "application/json" },
  });
}
function setup() {
  let instance: ReturnType<typeof useTasks> | undefined;
  const wrapper = mount(
    defineComponent({
      setup() {
        instance = useTasks();
        return () => null;
      },
    }),
  );
  if (!instance) throw new Error("component did not mount");
  return { state: instance, wrapper };
}
afterEach(() => {
  vi.unstubAllGlobals();
  vi.useRealTimers();
});
describe("task workspace", () => {
  it("preserves the draft and exposes an error when saving fails", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
        if (init?.method === "POST")
          return json({ error: "storage_error" }, 500);
        return json(String(input) === "/api/info" ? info : emptyPage);
      }),
    );
    const { state, wrapper } = setup();
    await flushPromises();
    state.open();
    state.draft.title = "Keep my input";
    state.draft.description = "Do not lose this";
    expect(await state.save()).toBe(false);
    expect(state.draft.title).toBe("Keep my input");
    expect(state.draft.description).toBe("Do not lose this");
    expect(state.formError.value).toBe("storage_error");
    expect(state.busy.value).toBe(false);
    wrapper.unmount();
  });
  it("ignores stale list responses after filters change", async () => {
    let resolveOld: ((response: Response) => void) | undefined;
    let lists = 0;
    vi.stubGlobal(
      "fetch",
      vi.fn((input: RequestInfo | URL) => {
        if (String(input) === "/api/info") return Promise.resolve(json(info));
        lists++;
        if (lists === 1)
          return new Promise<Response>((resolve) => {
            resolveOld = resolve;
          });
        return Promise.resolve(
          json({
            ...emptyPage,
            total: 4,
            stats: { todo: 4, doing: 0, done: 0 },
          }),
        );
      }),
    );
    const { state, wrapper } = setup();
    state.status.value = "todo";
    await flushPromises();
    expect(state.total.value).toBe(4);
    if (!resolveOld) throw new Error("old request missing");
    resolveOld(json({ ...emptyPage, total: 99 }));
    await flushPromises();
    expect(state.total.value).toBe(4);
    wrapper.unmount();
  });
  it("blocks duplicate submissions while a save is in flight", async () => {
    let finish: ((response: Response) => void) | undefined;
    let writes = 0;
    vi.stubGlobal(
      "fetch",
      vi.fn((input: RequestInfo | URL, init?: RequestInit) => {
        if (init?.method === "POST") {
          writes++;
          return new Promise<Response>((resolve) => {
            finish = resolve;
          });
        }
        return Promise.resolve(
          json(String(input) === "/api/info" ? info : emptyPage),
        );
      }),
    );
    const { state, wrapper } = setup();
    await flushPromises();
    state.open();
    state.draft.title = "Submit once";
    const first = state.save();
    expect(await state.save()).toBe(false);
    expect(writes).toBe(1);
    if (!finish) throw new Error("save request missing");
    finish(json({ id: "new" }, 201));
    expect(await first).toBe(true);
    wrapper.unmount();
  });
});
