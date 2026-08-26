// Package httpapi 暴露血缘诊断服务的 HTTP 接口。
// 所有接口以 /api 为前缀，返回 JSON；错误统一以 {error: msg} 形式回传。
package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"task249-linagediag/internal/meta"
	"task249-linagediag/internal/model"
	"task249-linagediag/internal/service"
)

// Handler 持有编排服务，按路径分发请求。
type Handler struct {
	svc *service.Service
	mux *http.ServeMux
}

// New 构造 HTTP 处理器。
func New(svc *service.Service) *Handler {
	h := &Handler{svc: svc}
	h.mux = http.NewServeMux()
	h.mux.HandleFunc("/api/health", h.dispatch)
	h.mux.HandleFunc("/api/batches", h.dispatch)
	h.mux.HandleFunc("/api/batches/{id}", h.dispatch)
	h.mux.HandleFunc("/api/batches/{id}/tables", h.dispatch)
	h.mux.HandleFunc("/api/batches/{id}/columns", h.dispatch)
	h.mux.HandleFunc("/api/batches/{id}/transforms", h.dispatch)
	h.mux.HandleFunc("/api/batches/{id}/jobs", h.dispatch)
	h.mux.HandleFunc("/api/batches/{id}/build", h.dispatch)
	h.mux.HandleFunc("/api/batches/{id}/diagnose", h.dispatch)
	h.mux.HandleFunc("/api/batches/{id}/lineage", h.dispatch)
	h.mux.HandleFunc("/api/batches/{id}/impact", h.dispatch)
	h.mux.HandleFunc("/api/batches/{id}/edges", h.dispatch)
	h.mux.HandleFunc("/api/batches/{id}/edges/{edge}/confirm", h.dispatch)
	h.mux.HandleFunc("/api/batches/{id}/edges/{edge}/exempt", h.dispatch)
	h.mux.HandleFunc("/api/batches/{id}/edges/{edge}/revise", h.dispatch)
	h.mux.HandleFunc("/api/batches/{id}/adjudications", h.dispatch)
	h.mux.HandleFunc("/api/batches/{id}/snapshots", h.dispatch)
	h.mux.HandleFunc("/api/batches/{id}/seal", h.dispatch)
	h.mux.HandleFunc("/api/batches/{id}/confirm-publish", h.dispatch)
	h.mux.HandleFunc("/api/batches/{id}/scenario", h.dispatch)
	return h
}

type errBody struct {
	Error string `json:"error"`
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.mux.ServeHTTP(w, r)
}

func (h *Handler) dispatch(w http.ResponseWriter, r *http.Request) {
	path := strings.Trim(r.URL.Path, "/")
	if !strings.HasPrefix(path, "api/") {
		http.NotFound(w, r)
		return
	}
	rest := strings.TrimPrefix(path, "api/")
	segs := strings.Split(rest, "/")
	switch {
	case rest == "health":
		h.health(w, r)
	case segs[0] == "batches" && len(segs) == 1:
		h.batchesRoot(w, r)
	case segs[0] == "batches" && len(segs) >= 2:
		h.batchSub(w, r, segs[1], segs[2:])
	default:
		http.NotFound(w, r)
	}
}

func (h *Handler) health(w http.ResponseWriter, r *http.Request) {
	h.writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// batchesRoot 处理 /api/batches：POST 新建批次，GET 列出全部批次。
func (h *Handler) batchesRoot(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		batches, err := h.svc.Store.ListBatches()
		if err != nil {
			h.writeErr(w, http.StatusBadRequest, err)
			return
		}
		h.writeJSON(w, http.StatusOK, batches)
		return
	}
	if r.Method != http.MethodPost {
		h.writeErr(w, http.StatusMethodNotAllowed, errors.New("method not allowed"))
		return
	}
	var body struct {
		Name string `json:"name"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	if body.Name == "" {
		body.Name = "unnamed"
	}
	b, err := h.svc.ImportBatch(body.Name)
	if err != nil {
		h.writeErr(w, http.StatusBadRequest, err)
		return
	}
	h.writeJSON(w, http.StatusCreated, b)
}

// batchSub 处理 /api/batches/{id}/... 子树。
func (h *Handler) batchSub(w http.ResponseWriter, r *http.Request, idSeg string, rem []string) {
	id, err := strconv.ParseInt(idSeg, 10, 64)
	if err != nil {
		h.writeErr(w, http.StatusBadRequest, errors.New("invalid batch id"))
		return
	}
	if len(rem) == 0 {
		if r.Method != http.MethodGet {
			h.writeErr(w, http.StatusMethodNotAllowed, errors.New("method not allowed"))
			return
		}
		b, err := h.svc.Store.GetBatch(id)
		if err != nil {
			h.writeErr(w, http.StatusNotFound, err)
			return
		}
		h.writeJSON(w, http.StatusOK, b)
		return
	}
	switch rem[0] {
	case "tables":
		h.batchTables(w, r, id)
	case "transforms":
		h.batchTransforms(w, r, id)
	case "jobs":
		h.batchJobs(w, r, id)
	case "build":
		h.needPost(w, r, func() error { return h.svc.BuildLineage(id) }, http.StatusOK, map[string]string{"result": "built"})
	case "diagnose":
		h.needGet(w, r, func() (interface{}, error) { return h.svc.Diagnose(id) })
	case "lineage":
		h.needGet(w, r, func() (interface{}, error) { return h.svc.LineageView(id) })
	case "impact":
		h.batchImpact(w, r, id)
	case "edges":
		if len(rem) == 1 {
			edges, err := h.svc.Store.ListEdges(id)
			if err != nil {
				h.writeErr(w, http.StatusBadRequest, err)
				return
			}
			h.writeJSON(w, http.StatusOK, edges)
			return
		}
		h.batchEdges(w, r, id, rem[1:])
	case "columns":
		h.batchColumns(w, r, id)
	case "adjudications":
		h.batchAdjudications(w, r, id)
	case "snapshots":
		h.batchSnapshots(w, r, id)
	case "seal":
		h.needPost(w, r, func() error { return h.svc.Seal(id) }, http.StatusOK, map[string]string{"result": "sealed"})
	case "confirm-publish":
		h.batchConfirmPublish(w, r, id)
	case "scenario":
		h.batchScenario(w, r, id)
	default:
		http.NotFound(w, r)
	}
}

func (h *Handler) batchTables(w http.ResponseWriter, r *http.Request, id int64) {
	if r.Method == http.MethodGet {
		tables, err := h.svc.Store.ListTables(id)
		if err != nil {
			h.writeErr(w, http.StatusBadRequest, err)
			return
		}
		h.writeJSON(w, http.StatusOK, tables)
		return
	}
	if r.Method != http.MethodPost {
		h.writeErr(w, http.StatusMethodNotAllowed, errors.New("method not allowed"))
		return
	}
	var body struct {
		QualifiedName string            `json:"qualified_name"`
		SchemaVersion string            `json:"schema_version"`
		Note          string            `json:"note"`
		Columns       []meta.ColumnSpec `json:"columns"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		h.writeErr(w, http.StatusBadRequest, err)
		return
	}
	res, err := h.svc.Meta.IngestTable(id, meta.TableSpec{
		QualifiedName: body.QualifiedName,
		SchemaVersion: body.SchemaVersion,
		Note:          body.Note,
		Columns:       body.Columns,
	})
	if err != nil {
		h.writeErr(w, http.StatusBadRequest, err)
		return
	}
	h.writeJSON(w, http.StatusCreated, res)
}

func (h *Handler) batchTransforms(w http.ResponseWriter, r *http.Request, id int64) {
	if r.Method != http.MethodPost {
		h.writeErr(w, http.StatusMethodNotAllowed, errors.New("method not allowed"))
		return
	}
	var body struct {
		JobID        int64  `json:"job_id"`
		SourceTable  string `json:"source_table"`
		SourceColumn string `json:"source_column"`
		TargetTable  string `json:"target_table"`
		TargetColumn string `json:"target_column"`
		Expression   string `json:"expression"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		h.writeErr(w, http.StatusBadRequest, err)
		return
	}
	t, err := h.svc.AddTransform(id, body.JobID, body.SourceTable, body.SourceColumn,
		body.TargetTable, body.TargetColumn, body.Expression)
	if err != nil {
		h.writeErr(w, http.StatusBadRequest, err)
		return
	}
	h.writeJSON(w, http.StatusCreated, t)
}

func (h *Handler) batchJobs(w http.ResponseWriter, r *http.Request, id int64) {
	if r.Method != http.MethodPost {
		h.writeErr(w, http.StatusMethodNotAllowed, errors.New("method not allowed"))
		return
	}
	var body struct {
		OutputTableID int64   `json:"output_table_id"`
		Name          string  `json:"name"`
		Summary       string  `json:"summary"`
		InputTableIDs []int64 `json:"input_table_ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		h.writeErr(w, http.StatusBadRequest, err)
		return
	}
	j, err := h.svc.AddJob(id, body.OutputTableID, body.Name, body.Summary, body.InputTableIDs)
	if err != nil {
		h.writeErr(w, http.StatusBadRequest, err)
		return
	}
	h.writeJSON(w, http.StatusCreated, j)
}

func (h *Handler) batchImpact(w http.ResponseWriter, r *http.Request, id int64) {
	if r.Method != http.MethodGet {
		h.writeErr(w, http.StatusMethodNotAllowed, errors.New("method not allowed"))
		return
	}
	table := r.URL.Query().Get("table")
	column := r.URL.Query().Get("column")
	if table == "" || column == "" {
		h.writeErr(w, http.StatusBadRequest, errors.New("table and column required"))
		return
	}
	refs, err := h.svc.Impact(id, table, column)
	if err != nil {
		h.writeErr(w, http.StatusBadRequest, err)
		return
	}
	h.writeJSON(w, http.StatusOK, refs)
}

func (h *Handler) batchEdges(w http.ResponseWriter, r *http.Request, id int64, rem []string) {
	if len(rem) < 2 {
		http.NotFound(w, r)
		return
	}
	edgeID, err := strconv.ParseInt(rem[0], 10, 64)
	if err != nil {
		h.writeErr(w, http.StatusBadRequest, errors.New("invalid edge id"))
		return
	}
	action := rem[1]
	if r.Method != http.MethodPost {
		h.writeErr(w, http.StatusMethodNotAllowed, errors.New("method not allowed"))
		return
	}
	var body struct {
		Note            string `json:"note"`
		NewSourceTable  string `json:"new_source_table"`
		NewSourceColumn string `json:"new_source_column"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	switch action {
	case "confirm":
		if err := h.svc.Confirm(id, edgeID, body.Note); err != nil {
			h.writeErr(w, http.StatusBadRequest, err)
			return
		}
	case "exempt":
		if err := h.svc.Exempt(id, edgeID, body.Note); err != nil {
			h.writeErr(w, http.StatusBadRequest, err)
			return
		}
	case "revise":
		if body.NewSourceTable == "" || body.NewSourceColumn == "" {
			h.writeErr(w, http.StatusBadRequest, errors.New("new_source_table and new_source_column required"))
			return
		}
		if err := h.svc.Revise(id, edgeID, body.NewSourceTable, body.NewSourceColumn, body.Note); err != nil {
			h.writeErr(w, http.StatusBadRequest, err)
			return
		}
	default:
		http.NotFound(w, r)
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]string{"result": action, "edge_id": strconv.FormatInt(edgeID, 10)})
}

func (h *Handler) batchSnapshots(w http.ResponseWriter, r *http.Request, id int64) {
	if r.Method == http.MethodGet {
		snaps, err := h.svc.Store.ListSnapshots(id)
		if err != nil {
			h.writeErr(w, http.StatusBadRequest, err)
			return
		}
		h.writeJSON(w, http.StatusOK, snaps)
		return
	}
	if r.Method != http.MethodPost {
		h.writeErr(w, http.StatusMethodNotAllowed, errors.New("method not allowed"))
		return
	}
	var body struct {
		Note string `json:"note"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	snap, err := h.svc.PublishSnapshot(id, body.Note)
	if err != nil {
		h.writeErr(w, http.StatusBadRequest, err)
		return
	}
	h.writeJSON(w, http.StatusCreated, snap)
}

func (h *Handler) batchConfirmPublish(w http.ResponseWriter, r *http.Request, id int64) {
	if r.Method != http.MethodPost {
		h.writeErr(w, http.StatusMethodNotAllowed, errors.New("method not allowed"))
		return
	}
	var body struct {
		EdgeID int64  `json:"edge_id"`
		Note   string `json:"note"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		h.writeErr(w, http.StatusBadRequest, err)
		return
	}
	snap, err := h.svc.ConfirmAndPublish(id, body.EdgeID, body.Note)
	if err != nil {
		h.writeErr(w, http.StatusBadRequest, err)
		return
	}
	h.writeJSON(w, http.StatusCreated, snap)
}

func (h *Handler) batchScenario(w http.ResponseWriter, r *http.Request, id int64) {
	if r.Method != http.MethodPost {
		h.writeErr(w, http.StatusMethodNotAllowed, errors.New("method not allowed"))
		return
	}
	res, err := h.svc.RunScenario(id)
	if err != nil {
		h.writeErr(w, http.StatusBadRequest, err)
		return
	}
	h.writeJSON(w, http.StatusOK, res)
}

func (h *Handler) batchColumns(w http.ResponseWriter, r *http.Request, id int64) {
	if r.Method != http.MethodGet {
		h.writeErr(w, http.StatusMethodNotAllowed, errors.New("method not allowed"))
		return
	}
	tidStr := r.URL.Query().Get("table_id")
	tid, err := strconv.ParseInt(tidStr, 10, 64)
	if err != nil || tid <= 0 {
		h.writeErr(w, http.StatusBadRequest, errors.New("table_id (positive int) required"))
		return
	}
	cols, err := h.svc.Store.ListColumns(tid)
	if err != nil {
		h.writeErr(w, http.StatusBadRequest, err)
		return
	}
	h.writeJSON(w, http.StatusOK, cols)
}

func (h *Handler) batchAdjudications(w http.ResponseWriter, r *http.Request, id int64) {
	if r.Method != http.MethodGet {
		h.writeErr(w, http.StatusMethodNotAllowed, errors.New("method not allowed"))
		return
	}
	ads, err := h.svc.Store.ListAdjudications(id)
	if err != nil {
		h.writeErr(w, http.StatusBadRequest, err)
		return
	}
	h.writeJSON(w, http.StatusOK, ads)
}

func (h *Handler) needPost(w http.ResponseWriter, r *http.Request, fn func() error, code int, ok interface{}) {
	if r.Method != http.MethodPost {
		h.writeErr(w, http.StatusMethodNotAllowed, errors.New("method not allowed"))
		return
	}
	if err := fn(); err != nil {
		h.writeErr(w, http.StatusBadRequest, err)
		return
	}
	h.writeJSON(w, code, ok)
}

func (h *Handler) needGet(w http.ResponseWriter, r *http.Request, fn func() (interface{}, error)) {
	if r.Method != http.MethodGet {
		h.writeErr(w, http.StatusMethodNotAllowed, errors.New("method not allowed"))
		return
	}
	v, err := fn()
	if err != nil {
		h.writeErr(w, http.StatusBadRequest, err)
		return
	}
	h.writeJSON(w, http.StatusOK, v)
}

func (h *Handler) writeJSON(w http.ResponseWriter, code int, v interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func (h *Handler) writeErr(w http.ResponseWriter, code int, err error) {
	code = mapErrStatus(code, err)
	h.writeJSON(w, code, errBody{Error: err.Error()})
}

func mapErrStatus(code int, err error) int {
	if errors.Is(err, model.ErrBatchNotFound) {
		return http.StatusNotFound
	}
	if errors.Is(err, model.ErrSealedMutation) || errors.Is(err, model.ErrInvalidTransition) ||
		errors.Is(err, model.ErrCycleDetected) {
		return http.StatusConflict
	}
	return code
}
