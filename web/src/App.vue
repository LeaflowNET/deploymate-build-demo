<script setup lang="ts">
import { computed, onMounted, ref, watch } from "vue";
import { messages, type Locale, type MessageKey } from "./i18n";
import { useTasks, type Task } from "./useTasks";
const {
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
} = useTasks();
const locale = ref<Locale>("zh"),
  dark = ref(false);
const t = computed(() => messages[locale.value]);
const editDialog = ref<HTMLDialogElement | null>(null),
  deleteDialog = ref<HTMLDialogElement | null>(null);
const statusKeys = ["todo", "doing", "done"] as const;
const priorities = ["low", "medium", "high"] as const;
function isMessageKey(code: string): code is MessageKey {
  return Object.prototype.hasOwnProperty.call(t.value, code);
}
function message(code: string) {
  return isMessageKey(code) ? t.value[code] : t.value.generic;
}
function date(value: string) {
  return new Intl.DateTimeFormat(
    locale.value === "zh"
      ? "zh-CN"
      : locale.value === "yue"
        ? "zh-HK"
        : "en-GB",
    { dateStyle: "medium", timeStyle: "short" },
  ).format(new Date(value));
}
function startEdit(task?: Task) {
  open(task);
  editDialog.value?.showModal();
}
function startDelete(task: Task) {
  deleting.value = task;
  formError.value = "";
  deleteDialog.value?.showModal();
}
async function submit() {
  if (await save()) editDialog.value?.close();
}
async function confirmDelete() {
  if (await remove()) {
    deleteDialog.value?.close();
    deleting.value = null;
  }
}
watch(dark, (v) => {
  document.documentElement.dataset.theme = v ? "dark" : "light";
  localStorage.setItem("taskmate-theme", v ? "dark" : "light");
});
watch(locale, (v) => {
  document.documentElement.lang =
    v === "zh" ? "zh-CN" : v === "yue" ? "zh-HK" : "en";
  localStorage.setItem("taskmate-locale", v);
});
onMounted(() => {
  dark.value = localStorage.getItem("taskmate-theme") === "dark";
  const saved = localStorage.getItem("taskmate-locale");
  if (saved === "zh" || saved === "yue" || saved === "en") locale.value = saved;
});
</script>

<template>
  <div class="shell">
    <header class="topbar">
      <a class="brand" href="/" aria-label="Taskmate"
        ><span class="brand-mark">✓</span>Taskmate<span class="brand-dot"
          >.</span
        ></a
      >
      <div class="top-actions">
        <span class="workspace">{{ t.workspace }}</span
        ><select v-model="locale" :aria-label="t.language">
          <option value="zh">简体中文</option>
          <option value="yue">繁體粵語</option>
          <option value="en">English</option></select
        ><button
          class="icon-button theme-button"
          :aria-label="t.theme"
          :title="t.theme"
          @click="dark = !dark"
        >
          {{ dark ? "☀" : "☾" }}
        </button>
      </div>
    </header>
    <main>
      <div class="page-heading">
        <div>
          <div class="eyebrow">{{ t.breadcrumb }}</div>
          <h1>{{ t.board }}</h1>
          <p>{{ t.subtitle }}</p>
        </div>
        <button class="primary" @click="startEdit()">
          <span aria-hidden="true">＋</span>{{ t.newTask }}
        </button>
      </div>
      <div class="stats-grid">
        <button
          v-for="key in statusKeys"
          :key="key"
          class="stat"
          :class="{ selected: status === key }"
          :aria-pressed="status === key"
          @click="status = status === key ? '' : key"
        >
          <span class="stat-label"
            ><span class="status-dot" :class="key"></span>{{ t[key] }}</span
          ><strong>{{ stats[key] }}</strong
          ><span class="stat-hint">{{
            key === "todo" ? "01" : key === "doing" ? "02" : "03"
          }}</span>
        </button>
      </div>
      <section class="task-panel" :aria-label="t.all">
        <div class="panel-header">
          <h2>
            {{ t.all }} <span class="count">{{ total }}</span>
          </h2>
          <button
            class="text-button"
            :disabled="loading || busy"
            @click="load()"
          >
            ↻ {{ t.refresh }}
          </button>
        </div>
        <div class="toolbar">
          <label class="search"
            ><span aria-hidden="true">⌕</span
            ><input
              v-model="query"
              type="search"
              maxlength="200"
              :placeholder="t.search"
              :aria-label="t.search"
          /></label>
          <div class="filters">
            <select v-model="status" :aria-label="t.status">
              <option value="">{{ t.all }}</option>
              <option v-for="key in statusKeys" :key="key" :value="key">
                {{ t[key] }}
              </option></select
            ><select v-model="priority" :aria-label="t.priority">
              <option value="">{{ t.anyPriority }}</option>
              <option v-for="key in priorities" :key="key" :value="key">
                {{ t[key] }}
              </option>
            </select>
          </div>
        </div>
        <div v-if="notice" class="notice" role="status">
          ✓ {{ message(notice) }}
        </div>
        <div v-if="error" class="error-banner" role="alert">
          <span>{{ message(error) }}</span
          ><button class="text-button" @click="load()">{{ t.retry }}</button>
        </div>
        <div v-if="loading" class="empty" role="status">
          <span class="spinner"></span>
          <p>{{ t.loading }}</p>
        </div>
        <template v-else-if="!error">
          <div v-if="tasks.length === 0" class="empty">
            <div class="empty-symbol">✓</div>
            <h3>{{ filtered ? t.noMatch : t.empty }}</h3>
            <p>{{ filtered ? t.noMatchText : t.emptyText }}</p>
            <button v-if="filtered" class="secondary" @click="clear()">
              {{ t.clear }}</button
            ><button v-else class="primary" @click="startEdit()">
              ＋ {{ t.newTask }}
            </button>
          </div>
          <div v-else class="task-list">
            <article
              v-for="task in tasks"
              :key="task.id"
              class="task-row"
              :class="{ completed: task.status === 'done' }"
            >
              <button
                class="task-check"
                :class="task.status"
                :disabled="busy"
                :aria-label="
                  task.status === 'todo'
                    ? t.start
                    : task.status === 'doing'
                      ? t.finish
                      : t.reopen
                "
                :title="
                  task.status === 'todo'
                    ? t.start
                    : task.status === 'doing'
                      ? t.finish
                      : t.reopen
                "
                @click="advance(task)"
              >
                {{
                  task.status === "done"
                    ? "✓"
                    : task.status === "doing"
                      ? "•"
                      : ""
                }}
              </button>
              <div class="task-body">
                <button class="task-title" @click="startEdit(task)">
                  {{ task.title }}
                </button>
                <p v-if="task.description" class="task-description">
                  {{ task.description }}
                </p>
                <div class="task-meta">
                  <span class="badge" :class="task.status">{{
                    t[task.status]
                  }}</span
                  ><span class="priority" :class="task.priority"
                    ><span aria-hidden="true">⚑</span>
                    {{ t[task.priority] }}</span
                  ><time :datetime="task.updatedAt"
                    >{{ t.updated }} {{ date(task.updatedAt) }}</time
                  >
                </div>
              </div>
              <div class="row-actions">
                <button
                  class="icon-button"
                  :aria-label="t.edit"
                  :title="t.edit"
                  :disabled="busy"
                  @click="startEdit(task)"
                >
                  ✎</button
                ><button
                  class="icon-button danger"
                  :aria-label="t.delete"
                  :title="t.delete"
                  :disabled="busy"
                  @click="startDelete(task)"
                >
                  ×
                </button>
              </div>
            </article>
          </div>
          <div v-if="total > limit" class="pagination">
            <span
              >{{ offset + 1 }}–{{ Math.min(offset + limit, total) }} /
              {{ total }} {{ t.total }}</span
            >
            <div>
              <button
                class="secondary"
                :disabled="offset === 0 || busy"
                @click="offset = Math.max(0, offset - limit)"
              >
                {{ t.previous }}</button
              ><button
                class="secondary"
                :disabled="offset + limit >= total || busy"
                @click="offset += limit"
              >
                {{ t.next }}
              </button>
            </div>
          </div>
        </template>
      </section>
      <details class="deployment">
        <summary>
          <span class="live-dot"></span>{{ t.build
          }}<span class="mono">{{ info?.version ?? "—" }}</span>
        </summary>
        <div v-if="info" class="deployment-grid">
          <div>
            <span>{{ t.version }}</span
            ><code>{{ info.version }}</code>
          </div>
          <div>
            <span>{{ t.commit }}</span
            ><code>{{ info.commit }}</code>
          </div>
          <div>
            <span>{{ t.boot }}</span
            ><span>{{ date(info.startedAt) }}</span>
          </div>
          <p>{{ info.message }}</p>
        </div>
        <p v-else>{{ t.infoError }}</p>
      </details>
      <footer>
        <span>{{ t.storage }}</span
        ><span>{{ t.demo }}</span>
      </footer>
    </main>
    <dialog
      ref="editDialog"
      aria-labelledby="edit-heading"
      @cancel="busy && $event.preventDefault()"
    >
      <form @submit.prevent="submit">
        <div class="dialog-heading">
          <h2 id="edit-heading">{{ editing ? t.edit : t.newTask }}</h2>
          <button
            type="button"
            class="icon-button"
            :aria-label="t.cancel"
            :disabled="busy"
            @click="editDialog?.close()"
          >
            ×
          </button>
        </div>
        <label class="field"
          >{{ t.title
          }}<input
            v-model="draft.title"
            autofocus
            required
            maxlength="200"
            :placeholder="t.titlePlaceholder"
            :disabled="busy" /></label
        ><label class="field"
          >{{ t.description
          }}<textarea
            :aria-label="t.description"
            v-model="draft.description"
            rows="4"
            maxlength="2000"
            :placeholder="t.descriptionPlaceholder"
            :disabled="busy"
          ></textarea
          ><span class="char-count"
            >{{ draft.description.length }} / 2000</span
          ></label
        >
        <div class="form-grid">
          <label class="field"
            >{{ t.status
            }}<select
              :aria-label="t.status"
              v-model="draft.status"
              :disabled="busy"
            >
              <option v-for="key in statusKeys" :key="key" :value="key">
                {{ t[key] }}
              </option>
            </select></label
          ><label class="field"
            >{{ t.priority
            }}<select
              :aria-label="t.priority"
              v-model="draft.priority"
              :disabled="busy"
            >
              <option v-for="key in priorities" :key="key" :value="key">
                {{ t[key] }}
              </option>
            </select></label
          >
        </div>
        <p v-if="formError" class="form-error" role="alert">
          {{ message(formError) }}
        </p>
        <div class="dialog-actions">
          <button
            type="button"
            class="secondary"
            :disabled="busy"
            @click="editDialog?.close()"
          >
            {{ t.cancel }}</button
          ><button
            type="submit"
            class="primary"
            :disabled="busy || !draft.title.trim()"
          >
            {{ busy ? t.saving : t.save }}
          </button>
        </div>
      </form>
    </dialog>
    <dialog
      ref="deleteDialog"
      aria-labelledby="delete-heading"
      @cancel="busy && $event.preventDefault()"
    >
      <div class="dialog-heading">
        <h2 id="delete-heading">{{ t.deleteTitle }}</h2>
      </div>
      <p class="delete-name">{{ deleting?.title }}</p>
      <p>{{ t.deleteText }}</p>
      <p v-if="formError" class="form-error" role="alert">
        {{ message(formError) }}
      </p>
      <div class="dialog-actions">
        <button
          class="secondary"
          autofocus
          :disabled="busy"
          @click="deleteDialog?.close()"
        >
          {{ t.cancel }}</button
        ><button
          class="danger-button"
          :disabled="busy"
          @click="confirmDelete()"
        >
          {{ busy ? t.deleting : t.delete }}
        </button>
      </div>
    </dialog>
  </div>
</template>
