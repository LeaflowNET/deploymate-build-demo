import {
  computed,
  onBeforeUnmount,
  onMounted,
  reactive,
  ref,
  watch,
} from "vue";
export type Status = "todo" | "doing" | "done";
export type Priority = "low" | "medium" | "high";
export interface Task {
  id: string;
  title: string;
  description: string;
  status: Status;
  priority: Priority;
  createdAt: string;
  updatedAt: string;
}
export interface Draft {
  title: string;
  description: string;
  status: Status;
  priority: Priority;
}
interface Stats {
  todo: number;
  doing: number;
  done: number;
}
interface Page {
  tasks: Task[];
  total: number;
  stats: Stats;
}
interface Info {
  name: string;
  version: string;
  commit: string;
  message: string;
  startedAt: string;
}
function object(v: unknown): v is Record<string, unknown> {
  return typeof v === "object" && v !== null;
}
function isStatus(v: unknown): v is Status {
  return v === "todo" || v === "doing" || v === "done";
}
function isPriority(v: unknown): v is Priority {
  return v === "low" || v === "medium" || v === "high";
}
function isTask(v: unknown): v is Task {
  return (
    object(v) &&
    typeof v.id === "string" &&
    typeof v.title === "string" &&
    typeof v.description === "string" &&
    isStatus(v.status) &&
    isPriority(v.priority) &&
    typeof v.createdAt === "string" &&
    typeof v.updatedAt === "string"
  );
}
function isPage(v: unknown): v is Page {
  return (
    object(v) &&
    Array.isArray(v.tasks) &&
    v.tasks.every(isTask) &&
    typeof v.total === "number" &&
    object(v.stats) &&
    typeof v.stats.todo === "number" &&
    typeof v.stats.doing === "number" &&
    typeof v.stats.done === "number"
  );
}
function isInfo(v: unknown): v is Info {
  return (
    object(v) &&
    typeof v.name === "string" &&
    typeof v.version === "string" &&
    typeof v.commit === "string" &&
    typeof v.message === "string" &&
    typeof v.startedAt === "string"
  );
}
async function request(path: string, init?: RequestInit): Promise<unknown> {
  let response: Response;
  try {
    response = await fetch(path, init);
  } catch (e) {
    if (e instanceof DOMException && e.name === "AbortError") throw e;
    throw new Error("network");
  }
  if (response.status === 204) return null;
  const data: unknown = await response.json();
  if (!response.ok)
    throw new Error(
      object(data) && typeof data.error === "string" ? data.error : "generic",
    );
  return data;
}
export function useTasks() {
  const tasks = ref<Task[]>([]),
    stats = ref<Stats>({ todo: 0, doing: 0, done: 0 }),
    total = ref(0),
    info = ref<Info | null>(null);
  const status = ref(""),
    priority = ref(""),
    query = ref(""),
    offset = ref(0),
    loading = ref(true),
    error = ref(""),
    formError = ref(""),
    notice = ref(""),
    busy = ref(false);
  const draft = reactive<Draft>({
    title: "",
    description: "",
    status: "todo",
    priority: "medium",
  });
  const editing = ref<Task | null>(null),
    deleting = ref<Task | null>(null);
  const limit = 20;
  let controller: AbortController | undefined,
    sequence = 0,
    timer: ReturnType<typeof setTimeout> | undefined;
  const filtered = computed(() =>
    Boolean(status.value || priority.value || query.value.trim()),
  );
  async function load() {
    controller?.abort();
    controller = new AbortController();
    const current = ++sequence;
    loading.value = true;
    error.value = "";
    try {
      const params = new URLSearchParams({
        q: query.value.trim(),
        status: status.value,
        priority: priority.value,
        limit: String(limit),
        offset: String(offset.value),
      });
      const data = await request("/api/tasks?" + params, {
        signal: controller.signal,
      });
      if (current !== sequence) return;
      if (!isPage(data)) throw new Error("generic");
      if (data.total > 0 && offset.value >= data.total) {
        offset.value = Math.floor((data.total - 1) / limit) * limit;
        return;
      }
      tasks.value = data.tasks;
      stats.value = data.stats;
      total.value = data.total;
    } catch (e) {
      if (
        current === sequence &&
        !(e instanceof DOMException && e.name === "AbortError")
      )
        error.value = e instanceof Error ? e.message : "generic";
    } finally {
      if (current === sequence) loading.value = false;
    }
  }
  function open(task?: Task) {
    editing.value = task ?? null;
    Object.assign(
      draft,
      task
        ? {
            title: task.title,
            description: task.description,
            status: task.status,
            priority: task.priority,
          }
        : { title: "", description: "", status: "todo", priority: "medium" },
    );
    formError.value = "";
    notice.value = "";
  }
  async function save() {
    if (busy.value) return false;
    busy.value = true;
    formError.value = "";
    notice.value = "";
    try {
      await request(
        "/api/tasks" + (editing.value ? "/" + editing.value.id : ""),
        {
          method: editing.value ? "PUT" : "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify(draft),
        },
      );
      notice.value = editing.value ? "saved" : "created";
      await load();
      return true;
    } catch (e) {
      formError.value = e instanceof Error ? e.message : "generic";
      return false;
    } finally {
      busy.value = false;
    }
  }
  async function advance(task: Task) {
    if (busy.value) return;
    busy.value = true;
    notice.value = "";
    try {
      const next: Status =
        task.status === "todo"
          ? "doing"
          : task.status === "doing"
            ? "done"
            : "todo";
      await request("/api/tasks/" + task.id, {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          title: task.title,
          description: task.description,
          status: next,
          priority: task.priority,
        }),
      });
      notice.value = "saved";
      await load();
    } catch (e) {
      error.value = e instanceof Error ? e.message : "generic";
    } finally {
      busy.value = false;
    }
  }
  async function remove() {
    if (!deleting.value || busy.value) return false;
    busy.value = true;
    formError.value = "";
    try {
      await request("/api/tasks/" + deleting.value.id, { method: "DELETE" });
      notice.value = "deleted";
      await load();
      return true;
    } catch (e) {
      formError.value = e instanceof Error ? e.message : "generic";
      return false;
    } finally {
      busy.value = false;
    }
  }
  function clear() {
    status.value = "";
    priority.value = "";
    query.value = "";
  }
  watch([status, priority], () => {
    if (offset.value !== 0) offset.value = 0;
    else void load();
  });
  watch(query, () => {
    clearTimeout(timer);
    timer = setTimeout(() => {
      if (offset.value !== 0) offset.value = 0;
      else void load();
    }, 250);
  });
  watch(offset, () => void load());
  onMounted(() => {
    void load();
    void request("/api/info")
      .then((v) => {
        if (isInfo(v)) info.value = v;
      })
      .catch(() => {
        info.value = null;
      });
  });
  onBeforeUnmount(() => {
    sequence++;
    controller?.abort();
    clearTimeout(timer);
  });
  return {
    tasks,
    stats,
    total,
    info,
    status,
    priority,
    query,
    offset,
    loading,
    error,
    formError,
    notice,
    busy,
    draft,
    editing,
    deleting,
    limit,
    filtered,
    load,
    open,
    save,
    advance,
    remove,
    clear,
  };
}
