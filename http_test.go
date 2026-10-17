package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func perform(handler http.Handler, method, path, body string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	handler.ServeHTTP(w, r)
	return w
}
func fixture(t *testing.T) (*Store, http.Handler) {
	t.Helper()
	s, err := openStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return s, newHandler(s, http.NotFoundHandler())
}
func createTask(t *testing.T, h http.Handler, title string) Task {
	t.Helper()
	input := TaskInput{Title: title, Status: "todo", Priority: "high"}
	data, _ := json.Marshal(input)
	w := perform(h, "POST", "/api/tasks", string(data))
	if w.Code != 201 {
		t.Fatalf("create: %d %s", w.Code, w.Body)
	}
	var task Task
	if err := json.Unmarshal(w.Body.Bytes(), &task); err != nil {
		t.Fatal(err)
	}
	return task
}
func TestTaskLifecycleAndPersistence(t *testing.T) {
	s, h := fixture(t)
	task := createTask(t, h, "测试构建")
	if task.ID == "" || task.CreatedAt.IsZero() {
		t.Fatalf("missing identity: %+v", task)
	}
	w := perform(h, "PUT", "/api/tasks/"+task.ID, `{"title":"修改后的任务","description":"部署后检查","status":"done","priority":"low"}`)
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body)
	}
	reopened, err := openStore(filepath.Dir(s.path))
	if err != nil {
		t.Fatal(err)
	}
	page := reopened.list("", "done", "low", 0, 20)
	if page.Total != 1 || page.Tasks[0].Title != "修改后的任务" || !page.Tasks[0].CreatedAt.Equal(task.CreatedAt) {
		t.Fatalf("lost data: %+v", page)
	}
	if w = perform(h, "DELETE", "/api/tasks/"+task.ID, ""); w.Code != 204 {
		t.Fatal(w.Code)
	}
	reopened, err = openStore(filepath.Dir(s.path))
	if err != nil {
		t.Fatal(err)
	}
	if reopened.list("", "", "", 0, 20).Total != 0 {
		t.Fatal("delete not persisted")
	}
	if w = perform(h, "DELETE", "/api/tasks/"+task.ID, ""); w.Code != 404 {
		t.Fatal(w.Code)
	}
}
func TestValidationAndPagination(t *testing.T) {
	_, h := fixture(t)
	for _, body := range []string{`{}`, `{"title":" ","status":"todo","priority":"low"}`, `{"title":"x","status":"bad","priority":"low"}`, `{"title":"x","status":"todo","priority":"urgent"}`, `{"title":"x","status":"todo","priority":"low","unexpected":true}`, `{"title":"x","status":"todo","priority":"low"} {}`} {
		if w := perform(h, "POST", "/api/tasks", body); w.Code != 400 {
			t.Fatalf("accepted %s: %d", body, w.Code)
		}
	}
	for i := range 3 {
		createTask(t, h, fmt.Sprintf("Release %d", i))
	}
	w := perform(h, "GET", "/api/tasks?q=release&status=todo&priority=high&limit=1&offset=1", "")
	var page TaskPage
	if err := json.Unmarshal(w.Body.Bytes(), &page); err != nil {
		t.Fatal(err)
	}
	if w.Code != 200 || page.Total != 3 || len(page.Tasks) != 1 || page.Stats.Todo != 3 {
		t.Fatalf("bad page: %d %+v", w.Code, page)
	}
	for _, q := range []string{"limit=0", "limit=101", "offset=-1", "offset=999999999999999999999", "status=invalid", "priority=invalid"} {
		if w := perform(h, "GET", "/api/tasks?"+q, ""); w.Code != 400 {
			t.Fatal(q, w.Code)
		}
	}
	if w := perform(h, "GET", "/api/tasks?q=missing", ""); w.Code != 200 || !bytes.Contains(w.Body.Bytes(), []byte(`"tasks":[]`)) {
		t.Fatal(w.Code, w.Body)
	}
}
func TestFailedWriteDoesNotChangeMemory(t *testing.T) {
	s, _ := fixture(t)
	if err := os.Mkdir(s.path, 0700); err != nil {
		t.Fatal(err)
	}
	if _, err := s.put("", TaskInput{Title: "fail", Status: "todo", Priority: "low"}); err == nil {
		t.Fatal("expected storage error")
	}
	if s.list("", "", "", 0, 20).Total != 0 {
		t.Fatal("failed write changed memory")
	}
}
func TestConcurrentCreatesPersist(t *testing.T) {
	s, h := fixture(t)
	var wg sync.WaitGroup
	for i := range 25 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			w := perform(h, "POST", "/api/tasks", fmt.Sprintf(`{"title":"task %d","status":"todo","priority":"medium"}`, i))
			if w.Code != 201 {
				t.Errorf("create: %d %s", w.Code, w.Body)
			}
		}()
	}
	wg.Wait()
	reopened, err := openStore(filepath.Dir(s.path))
	if err != nil {
		t.Fatal(err)
	}
	if reopened.list("", "", "", 0, 100).Total != 25 {
		t.Fatal("concurrent create lost data")
	}
}
func TestCorruptStoreFailsStartup(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "tasks.json"), []byte("not json"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := openStore(dir); err == nil {
		t.Fatal("corrupt data was ignored")
	}
}
func TestRejectCrossOriginWrites(t *testing.T) {
	_, h := fixture(t)
	r := httptest.NewRequest("POST", "/api/tasks", bytes.NewBufferString(`{"title":"x","status":"todo","priority":"low"}`))
	r.Header.Set("Origin", "https://untrusted.example")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 403 {
		t.Fatal(w.Code)
	}
}
