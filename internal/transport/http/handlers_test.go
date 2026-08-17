package http

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/require"

	"github.com/cry048/design-review-platform/internal/domain/annotation"
	domainaudit "github.com/cry048/design-review-platform/internal/domain/audit"
	memaudit "github.com/cry048/design-review-platform/internal/platform/audit"
	"github.com/cry048/design-review-platform/internal/platform/notify"
	"github.com/cry048/design-review-platform/internal/platform/storage"
	"github.com/cry048/design-review-platform/internal/repository/memory"
	"github.com/cry048/design-review-platform/internal/service"
)

// fixedClock for deterministic tests.
type fixedClock struct{ t time.Time }

func (f fixedClock) Now() time.Time { return f.t }

// counterIDs returns deterministic IDs in test order.
type counterIDs struct {
	n int
}

func (c *counterIDs) New() string {
	c.n++
	return "gen-id-" + itoaInt(c.n)
}

func itoaInt(i int) string {
	if i == 0 {
		return "0"
	}
	var b []byte
	for i > 0 {
		b = append([]byte{byte('0' + i%10)}, b...)
		i /= 10
	}
	return string(b)
}

func newTestHandlers(t *testing.T) (*Handlers, *memory.ProjectRepo, *memory.AnnotationRepo, *memory.ReviewRepo, *memaudit.MemoryLogger) {
	t.Helper()
	projs := memory.NewProjectRepo()
	annos := memory.NewAnnotationRepo()
	revs := memory.NewReviewRepo()
	runner := memory.NewTxRunner()
	audit := memaudit.NewMemoryLogger()
	store, err := storage.New(storage.DefaultLimits(t.TempDir()))
	if err != nil {
		t.Fatal(err)
	}
	notif := notify.New()
	clock := fixedClock{t: time.Date(2026, 8, 17, 10, 0, 0, 0, time.UTC)}
	ids := &counterIDs{}

	auditAdapt := testAuditAdapter{audit}
	notifAdapt := testNotifierAdapter{notif}
	storeAdapt := testStoreAdapter{store}

	projSvc := &service.ProjectService{Projects: projs, Audit: auditAdapt, Clock: clock, IDs: ids}
	annoSvc := &service.AnnotationService{
		Projects: projs, Annotations: annos, Reviews: revs, Audit: auditAdapt, Notifier: notifAdapt,
		Storage: store, Clock: clock, IDs: ids, AttachmentValidator: storeAdapt, TxRunner: runner,
	}
	revSvc := &service.ReviewService{
		Projects: projs, Annotations: annos, Reviews: revs, Audit: auditAdapt,
		Clock: clock, IDs: ids, TxRunner: runner,
	}
	auditSvc := &service.AuditService{Logger: audit}
	notifSvc := &service.NotifierService{N: notif}
	exportSvc := &service.ExportService{Annotations: annos, Reviews: revs, Projects: projs}
	return &Handlers{
		Projects: projSvc, Annotations: annoSvc, Reviews: revSvc,
		Notifier: notifSvc, Audit: auditSvc, Export: exportSvc,
		Validator: validator.New(),
	}, projs, annos, revs, audit
}

// adapters
type testAuditAdapter struct{ l *memaudit.MemoryLogger }

func (a testAuditAdapter) Log(ctx context.Context, action, actorID, entityType, entityID, before, after string) error {
	return a.l.Log(ctx, domainaudit.Entry{ActorID: actorID, Action: action, EntityType: entityType, EntityID: entityID, Before: before, After: after, At: time.Now()})
}

type testNotifierAdapter struct{ n *notify.Notifier }

func (n testNotifierAdapter) NotifyAnnotationAssigned(ctx context.Context, a *annotation.Annotation) error {
	return n.n.NotifyAnnotationAssigned(ctx, a)
}
func (n testNotifierAdapter) NotifyAnnotationResolved(ctx context.Context, a *annotation.Annotation) error {
	return n.n.NotifyAnnotationResolved(ctx, a)
}
func (n testNotifierAdapter) NotifyAnnotationReopened(ctx context.Context, a *annotation.Annotation) error {
	return n.n.NotifyAnnotationReopened(ctx, a)
}
func (n testNotifierAdapter) NotifyDueApproaching(ctx context.Context, a *annotation.Annotation, when time.Time) error {
	return n.n.NotifyDueApproaching(ctx, a, when)
}

type testStoreAdapter struct{ s *storage.Store }

func (s testStoreAdapter) ValidateUpload(mediaType string, size int64) error {
	return s.s.ValidateUpload(mediaType, size)
}

// we need to import annotation package for adapters above.
func newTestRouter(h *Handlers) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	// minimal middleware for tests: request id + recovery
	r.Use(func(c *gin.Context) {
		c.Set("request_id", "test-req")
		c.Next()
	})
	r.Use(func(c *gin.Context) {
		c.Set("actor_id", "test-user")
		c.Next()
	})
	registerRoutesForTest(r, h)
	return r
}

func registerRoutesForTest(r *gin.Engine, h *Handlers) {
	r.GET("/healthz", h.Healthz)
	v1 := r.Group("/api/v1")
	{
		v1.POST("/projects", h.CreateProject)
		v1.GET("/projects", h.ListProjects)
		v1.GET("/projects/:id", h.GetProject)
		v1.POST("/projects/:id/archive", h.ArchiveProject)
		v1.GET("/projects/:id/boards", h.ListBoards)
		v1.POST("/boards", h.CreateBoard)
		v1.GET("/boards/:id", h.GetBoard)
		v1.POST("/boards/:id/close", h.CloseBoard)
		v1.GET("/boards/:id/versions", h.ListVersions)
		v1.POST("/versions", h.CreateVersion)
		v1.GET("/versions/:id", h.GetVersion)
		v1.POST("/versions/:id/publish", h.PublishVersion)
		v1.POST("/members", h.AddMember)
		v1.GET("/members", h.ListMembers)
		v1.DELETE("/members/:id", h.RemoveMember)
		v1.POST("/annotations", h.CreateAnnotation)
		v1.GET("/annotations", h.ListAnnotations)
		v1.GET("/annotations/search", h.SearchAnnotations)
		v1.GET("/annotations/my-todos", h.ListMyTodos)
		v1.GET("/annotations/:id", h.GetAnnotation)
		v1.POST("/annotations/:id/replies", h.AddReply)
		v1.PATCH("/annotations/:id/assignee", h.SetAssignee)
		v1.PATCH("/annotations/:id/due", h.SetDue)
		v1.PATCH("/annotations/:id/priority", h.SetPriority)
		v1.POST("/annotations/:id/request-review", h.RequestReview)
		v1.POST("/annotations/:id/resolve", h.ResolveAnnotation)
		v1.POST("/annotations/:id/reopen", h.ReopenAnnotation)
		v1.POST("/annotations/:id/close", h.CloseAnnotation)
		v1.POST("/annotations/:id/attachments", h.AddAttachment)
		v1.POST("/annotations/migrate-anchors", h.MigrateAnchors)
		v1.POST("/reviews/rounds", h.CreateRound)
		v1.GET("/reviews/rounds/:id", h.GetRound)
		v1.GET("/reviews/boards/:boardID/rounds", h.ListRounds)
		v1.POST("/reviews/rounds/:id/snapshots", h.CreateSnapshot)
		v1.POST("/reviews/rounds/:id/conclusion", h.SetConclusion)
		v1.POST("/reviews/rounds/:id/close", h.CloseRound)
		v1.GET("/notifications", h.ListNotifications)
		v1.POST("/notifications/:id/read", h.MarkNotificationRead)
		v1.GET("/notifications/unread-count", h.UnreadCount)
		v1.GET("/audit", h.ListAudit)
		v1.GET("/export/annotations.csv", h.ExportAnnotations)
		v1.GET("/export/reviews/:id/minutes.csv", h.ExportReviewMinutes)
	}
}

func doJSON(t *testing.T, r http.Handler, method, path string, body interface{}) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			t.Fatal(err)
		}
	}
	req := httptest.NewRequest(method, path, &buf)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func setupProjectBoardVersion(t *testing.T, r http.Handler) (projectID, boardID, versionID string) {
	t.Helper()
	w := doJSON(t, r, "POST", "/api/v1/projects", map[string]any{"id": "p-test", "name": "Test"})
	require.Equal(t, 201, w.Code)
	projectID = "p-test"
	w = doJSON(t, r, "POST", "/api/v1/boards", map[string]any{"id": "b-test", "project_id": projectID, "name": "Board", "width": 1440, "height": 1024})
	require.Equal(t, 201, w.Code)
	boardID = "b-test"
	w = doJSON(t, r, "POST", "/api/v1/versions", map[string]any{"id": "v-test", "board_id": boardID, "number": 1, "label": "L1", "preview_key": "p.png"})
	require.Equal(t, 201, w.Code)
	versionID = "v-test"
	w = doJSON(t, r, "POST", "/api/v1/versions/v-test/publish", nil)
	require.Equal(t, 200, w.Code)
	return
}

func TestHealthz(t *testing.T) {
	h, _, _, _, _ := newTestHandlers(t)
	r := newTestRouter(h)
	w := doJSON(t, r, "GET", "/healthz", nil)
	require.Equal(t, 200, w.Code)
}

func TestCreateProjectValidation(t *testing.T) {
	h, _, _, _, _ := newTestHandlers(t)
	r := newTestRouter(h)
	w := doJSON(t, r, "POST", "/api/v1/projects", map[string]any{"name": ""})
	require.Equal(t, 400, w.Code)
	body := w.Body.String()
	require.Contains(t, body, "VALIDATION")
}

func TestCreateBoardInvalidSize(t *testing.T) {
	h, _, _, _, _ := newTestHandlers(t)
	r := newTestRouter(h)
	w := doJSON(t, r, "POST", "/api/v1/boards", map[string]any{"project_id": "p1", "name": "b", "width": 0, "height": 100})
	require.Equal(t, 400, w.Code)
}

func TestCreateAnnotationRejectsClosedBoard(t *testing.T) {
	h, _, _, _, _ := newTestHandlers(t)
	r := newTestRouter(h)
	pid, bid, vid := setupProjectBoardVersion(t, r)
	// close board
	w := doJSON(t, r, "POST", "/api/v1/boards/"+bid+"/close", nil)
	require.Equal(t, 200, w.Code)
	// try to create annotation
	w = doJSON(t, r, "POST", "/api/v1/annotations", map[string]any{
		"id": "a-test", "project_id": pid, "board_id": bid, "version_id": vid,
		"title": "t", "point": map[string]any{"x": 10, "y": 10},
	})
	require.Equal(t, 409, w.Code)
	require.Contains(t, w.Body.String(), "画板已关闭")
}

func TestCreateAnnotationRejectsArchivedProject(t *testing.T) {
	h, _, _, _, _ := newTestHandlers(t)
	r := newTestRouter(h)
	pid, bid, vid := setupProjectBoardVersion(t, r)
	// archive project
	w := doJSON(t, r, "POST", "/api/v1/projects/"+pid+"/archive", nil)
	require.Equal(t, 200, w.Code)
	w = doJSON(t, r, "POST", "/api/v1/annotations", map[string]any{
		"project_id": pid, "board_id": bid, "version_id": vid,
		"title": "t", "point": map[string]any{"x": 10, "y": 10},
	})
	require.Equal(t, 409, w.Code)
	require.Contains(t, w.Body.String(), "项目已归档")
}

func TestCreateAnnotationRejectsNonPublishedVersion(t *testing.T) {
	h, _, _, _, _ := newTestHandlers(t)
	r := newTestRouter(h)
	pid, bid, _ := setupProjectBoardVersion(t, r)
	// create a draft version
	w := doJSON(t, r, "POST", "/api/v1/versions", map[string]any{"id": "v-draft", "board_id": bid, "number": 2, "preview_key": "p2.png"})
	require.Equal(t, 201, w.Code)
	// annotation on draft should fail
	w = doJSON(t, r, "POST", "/api/v1/annotations", map[string]any{
		"project_id": pid, "board_id": bid, "version_id": "v-draft",
		"title": "t", "point": map[string]any{"x": 10, "y": 10},
	})
	require.Equal(t, 400, w.Code)
	require.Contains(t, w.Body.String(), "non-published")
}

func TestCreateAnnotationRejectsAnchorOutsideBounds(t *testing.T) {
	h, _, _, _, _ := newTestHandlers(t)
	r := newTestRouter(h)
	pid, bid, vid := setupProjectBoardVersion(t, r)
	w := doJSON(t, r, "POST", "/api/v1/annotations", map[string]any{
		"project_id": pid, "board_id": bid, "version_id": vid,
		"title": "t", "point": map[string]any{"x": 99999, "y": 10},
	})
	require.Equal(t, 400, w.Code)
}

func TestAnnotationWorkflowResolveReopen(t *testing.T) {
	h, _, _, _, _ := newTestHandlers(t)
	r := newTestRouter(h)
	pid, bid, vid := setupProjectBoardVersion(t, r)
	w := doJSON(t, r, "POST", "/api/v1/annotations", map[string]any{
		"id": "a1", "project_id": pid, "board_id": bid, "version_id": vid,
		"title": "t", "body": "b", "priority": "high",
		"point": map[string]any{"x": 100, "y": 100},
	})
	require.Equal(t, 201, w.Code)
	// add reply
	w = doJSON(t, r, "POST", "/api/v1/annotations/a1/replies", map[string]any{"body": "已修复"})
	require.Equal(t, 201, w.Code)
	// request review
	w = doJSON(t, r, "POST", "/api/v1/annotations/a1/request-review", nil)
	require.Equal(t, 200, w.Code)
	// resolve
	w = doJSON(t, r, "POST", "/api/v1/annotations/a1/resolve", nil)
	require.Equal(t, 200, w.Code)
	// reopen via 复核
	w = doJSON(t, r, "POST", "/api/v1/annotations/a1/reopen", map[string]any{"reason": "复核重新打开"})
	require.Equal(t, 200, w.Code)
	// verify status is open
	w = doJSON(t, r, "GET", "/api/v1/annotations/a1", nil)
	require.Equal(t, 200, w.Code)
	var resp struct {
		Data struct {
			Status string `json:"status"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Equal(t, "open", resp.Data.Status)
}

func TestResolveWithoutReplyFails(t *testing.T) {
	h, _, _, _, _ := newTestHandlers(t)
	r := newTestRouter(h)
	pid, bid, vid := setupProjectBoardVersion(t, r)
	_ = doJSON(t, r, "POST", "/api/v1/annotations", map[string]any{
		"id": "a1", "project_id": pid, "board_id": bid, "version_id": vid,
		"title": "t", "point": map[string]any{"x": 100, "y": 100},
	})
	// cannot request-review directly from open (allowed) then resolve (fails without reply)
	_ = doJSON(t, r, "POST", "/api/v1/annotations/a1/request-review", nil)
	w := doJSON(t, r, "POST", "/api/v1/annotations/a1/resolve", nil)
	require.Equal(t, 400, w.Code)
	require.Contains(t, w.Body.String(), "回复")
}

func TestReopenUnresolvedFails(t *testing.T) {
	h, _, _, _, _ := newTestHandlers(t)
	r := newTestRouter(h)
	_, _, vid := setupProjectBoardVersion(t, r)
	_ = doJSON(t, r, "POST", "/api/v1/annotations", map[string]any{
		"id": "a1", "project_id": "p-test", "board_id": "b-test", "version_id": vid,
		"title": "t", "point": map[string]any{"x": 100, "y": 100},
	})
	w := doJSON(t, r, "POST", "/api/v1/annotations/a1/reopen", map[string]any{"reason": "x"})
	require.Equal(t, 400, w.Code)
	require.Contains(t, w.Body.String(), "INVALID_TRANSITION")
}

func TestMigrateAnchorsNoSilentDrop(t *testing.T) {
	h, _, _, _, _ := newTestHandlers(t)
	r := newTestRouter(h)
	pid, bid, v1 := setupProjectBoardVersion(t, r)
	// create annotation on v1 while v1 is still the published version
	w := doJSON(t, r, "POST", "/api/v1/annotations", map[string]any{
		"id": "a1", "project_id": pid, "board_id": bid, "version_id": v1,
		"title": "t", "point": map[string]any{"x": 10, "y": 10},
	})
	require.Equal(t, 201, w.Code, "body=%s", w.Body.String())
	// now create v2 published (v1 superseded)
	_ = doJSON(t, r, "POST", "/api/v1/versions", map[string]any{"id": "v2", "board_id": bid, "number": 2, "preview_key": "p2.png"})
	w = doJSON(t, r, "POST", "/api/v1/versions/v2/publish", nil)
	require.Equal(t, 200, w.Code, "body=%s", w.Body.String())
	// migrate v1 -> v2
	w = doJSON(t, r, "POST", "/api/v1/annotations/migrate-anchors", map[string]any{
		"from_version_id": v1, "to_version_id": "v2", "reason": "版本切换",
	})
	require.Equal(t, 200, w.Code, "body=%s", w.Body.String())
	require.Contains(t, w.Body.String(), "migrated_count")
	// annotation a1 now points to v2 (JSON serialised as VersionID)
	w = doJSON(t, r, "GET", "/api/v1/annotations/a1", nil)
	require.Equal(t, 200, w.Code)
	body := w.Body.String()
	require.Contains(t, body, `"VersionID":"v2"`)
}

func TestMigrateAnchorsRejectsSameVersion(t *testing.T) {
	h, _, _, _, _ := newTestHandlers(t)
	r := newTestRouter(h)
	_, _, v1 := setupProjectBoardVersion(t, r)
	w := doJSON(t, r, "POST", "/api/v1/annotations/migrate-anchors", map[string]any{
		"from_version_id": v1, "to_version_id": v1,
	})
	require.Equal(t, 400, w.Code)
}

func TestPaginationDefaults(t *testing.T) {
	h, _, _, _, _ := newTestHandlers(t)
	r := newTestRouter(h)
	pid, bid, vid := setupProjectBoardVersion(t, r)
	for i := 0; i < 5; i++ {
		_ = doJSON(t, r, "POST", "/api/v1/annotations", map[string]any{
			"project_id": pid, "board_id": bid, "version_id": vid,
			"title": "t", "point": map[string]any{"x": 10, "y": 10},
		})
	}
	w := doJSON(t, r, "GET", "/api/v1/annotations?page=1&page_size=2", nil)
	require.Equal(t, 200, w.Code)
	var resp struct {
		Total int   `json:"total"`
		Page  int   `json:"page"`
		Size  int   `json:"page_size"`
		Data  []any `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Equal(t, 5, resp.Total)
	require.Equal(t, 1, resp.Page)
	require.Equal(t, 2, resp.Size)
	require.Len(t, resp.Data, 2)
}

func TestSortWhitelist(t *testing.T) {
	h, _, _, _, _ := newTestHandlers(t)
	r := newTestRouter(h)
	pid, bid, vid := setupProjectBoardVersion(t, r)
	_ = doJSON(t, r, "POST", "/api/v1/annotations", map[string]any{
		"project_id": pid, "board_id": bid, "version_id": vid,
		"title": "t", "priority": "high", "point": map[string]any{"x": 10, "y": 10},
	})
	// sort by priority allowed
	w := doJSON(t, r, "GET", "/api/v1/annotations?sort_by=priority&order=desc", nil)
	require.Equal(t, 200, w.Code)
	// sort by malicious field should not 500 (falls back to created_at)
	w = doJSON(t, r, "GET", "/api/v1/annotations?sort_by=password", nil)
	require.Equal(t, 200, w.Code)
}

func TestAttachmentValidationFails(t *testing.T) {
	h, _, _, _, _ := newTestHandlers(t)
	r := newTestRouter(h)
	pid, bid, vid := setupProjectBoardVersion(t, r)
	_ = doJSON(t, r, "POST", "/api/v1/annotations", map[string]any{
		"id": "a1", "project_id": pid, "board_id": bid, "version_id": vid,
		"title": "t", "point": map[string]any{"x": 10, "y": 10},
	})
	// forbidden type
	w := doJSON(t, r, "POST", "/api/v1/annotations/a1/attachments", map[string]any{
		"filename": "x.exe", "media_type": "application/x-msdownload", "size": 100, "storage_key": "k",
	})
	require.Equal(t, 400, w.Code)
	// size too large
	w = doJSON(t, r, "POST", "/api/v1/annotations/a1/attachments", map[string]any{
		"filename": "x.png", "media_type": "image/png", "size": 1000000000, "storage_key": "k",
	})
	require.Equal(t, 400, w.Code)
}

func TestReviewRoundWorkflow(t *testing.T) {
	h, _, _, _, _ := newTestHandlers(t)
	r := newTestRouter(h)
	pid, bid, vid := setupProjectBoardVersion(t, r)
	w := doJSON(t, r, "POST", "/api/v1/reviews/rounds", map[string]any{
		"id": "rd1", "project_id": pid, "board_id": bid, "title": "评审一轮",
	})
	require.Equal(t, 201, w.Code)
	w = doJSON(t, r, "POST", "/api/v1/reviews/rounds/rd1/snapshots", map[string]any{
		"project_id": pid, "version_id": vid,
	})
	require.Equal(t, 201, w.Code)
	w = doJSON(t, r, "POST", "/api/v1/reviews/rounds/rd1/conclusion", map[string]any{
		"conclusion": "通过", "recommendation": "approve_with_conditions",
	})
	require.Equal(t, 200, w.Code)
	w = doJSON(t, r, "POST", "/api/v1/reviews/rounds/rd1/close", nil)
	require.Equal(t, 200, w.Code)
	// operations after close should fail
	w = doJSON(t, r, "POST", "/api/v1/reviews/rounds/rd1/snapshots", map[string]any{
		"project_id": pid, "version_id": vid,
	})
	require.Equal(t, 409, w.Code)
}

func TestExportAnnotationsCSV(t *testing.T) {
	h, _, _, _, _ := newTestHandlers(t)
	r := newTestRouter(h)
	pid, bid, vid := setupProjectBoardVersion(t, r)
	_ = doJSON(t, r, "POST", "/api/v1/annotations", map[string]any{
		"id": "a1", "project_id": pid, "board_id": bid, "version_id": vid,
		"title": "t", "point": map[string]any{"x": 10, "y": 10},
	})
	w := doJSON(t, r, "GET", "/api/v1/export/annotations.csv", nil)
	require.Equal(t, 200, w.Code)
	require.Contains(t, w.Body.String(), "id,title,status")
	require.Contains(t, w.Body.String(), "a1")
}

func TestAuditTrailCaptured(t *testing.T) {
	h, _, _, _, audit := newTestHandlers(t)
	r := newTestRouter(h)
	_, _, _ = setupProjectBoardVersion(t, r)
	// audit log should contain entries
	entries := audit.All()
	if len(entries) == 0 {
		t.Fatal("audit log should have entries")
	}
	found := false
	for _, e := range entries {
		if strings.Contains(e.Action, "project.create") {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("audit log should contain project.create entry")
	}
}

func TestNotificationOnAssignee(t *testing.T) {
	h, _, _, _, _ := newTestHandlers(t)
	r := newTestRouter(h)
	pid, bid, vid := setupProjectBoardVersion(t, r)
	// create with actor=designer-1 so they get the notification on assignment
	w := doJSON(t, r, "POST", "/api/v1/annotations", map[string]any{
		"id": "a1", "project_id": pid, "board_id": bid, "version_id": vid,
		"title": "t", "point": map[string]any{"x": 10, "y": 10},
		"assignee_id": "test-user",
	})
	require.Equal(t, 201, w.Code, "body=%s", w.Body.String())
	// default actor is test-user so they should have a notification
	w = doJSON(t, r, "GET", "/api/v1/notifications", nil)
	require.Equal(t, 200, w.Code)
	require.Contains(t, w.Body.String(), "批注已指派给你")
}

func TestVersionPublishSupersedesPrevious(t *testing.T) {
	h, _, _, _, _ := newTestHandlers(t)
	r := newTestRouter(h)
	_, bid, v1 := setupProjectBoardVersion(t, r)
	// create v2
	_ = doJSON(t, r, "POST", "/api/v1/versions", map[string]any{
		"id": "v2", "board_id": bid, "number": 2, "preview_key": "p2.png",
	})
	w := doJSON(t, r, "POST", "/api/v1/versions/v2/publish", nil)
	require.Equal(t, 200, w.Code)
	// v1 should now be superseded
	w = doJSON(t, r, "GET", "/api/v1/versions/"+v1, nil)
	require.Equal(t, 200, w.Code)
	require.Contains(t, w.Body.String(), `"Status":"superseded"`)
}

func TestSearchEmptyQueryFails(t *testing.T) {
	h, _, _, _, _ := newTestHandlers(t)
	r := newTestRouter(h)
	w := doJSON(t, r, "GET", "/api/v1/annotations/search", nil)
	require.Equal(t, 400, w.Code)
}
