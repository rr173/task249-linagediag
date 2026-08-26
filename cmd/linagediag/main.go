// 数据仓库血缘列级截断诊断服务入口。
// 默认启动 HTTP 服务；传入 --smoke-test 则运行端到端自测（含关闭重开同库恢复校验）后退出。
package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"task249-linagediag/internal/httpapi"
	"task249-linagediag/internal/meta"
	"task249-linagediag/internal/service"
	"task249-linagediag/internal/store"
)

func main() {
	dbPath := flag.String("db", "linagediag.db", "SQLite 数据库文件路径")
	addr := flag.String("addr", ":8080", "HTTP 监听地址")
	smoke := flag.Bool("smoke-test", false, "运行自测场景后退出（含重启恢复校验）")
	flag.Parse()

	if *smoke {
		if err := runSmoke(); err != nil {
			log.Fatalf("smoke-test FAILED: %v", err)
		}
		fmt.Println("smoke-test OK")
		return
	}

	st, err := store.Open(*dbPath)
	if err != nil {
		log.Fatalf("open store: %v", err)
	}
	defer st.Close()

	svc := service.New(st)
	mux := http.NewServeMux()
	mux.Handle("/api/", httpapi.New(svc))
	log.Printf("linagediag listening on %s (db=%s)", *addr, *dbPath)
	if err := http.ListenAndServe(*addr, mux); err != nil {
		log.Fatalf("server: %v", err)
	}
}

// runSmoke 复现「上游列重命名 → 下游派生列未更新 → 定位断裂 → 修订/豁免 → 发布 → 封存」闭环，
// 并校验关闭重开同库后状态与快照仍持久化。
func runSmoke() error {
	dir, err := os.MkdirTemp("", "linagediag-smoke-")
	if err != nil {
		return err
	}
	dbPath := filepath.Join(dir, "smoke.db")
	defer os.RemoveAll(dir)

	st, err := store.Open(dbPath)
	if err != nil {
		return err
	}
	svc := service.New(st)

	batch, err := svc.ImportBatch("smoke-rename")
	if err != nil {
		return err
	}
	bid := batch.ID

	// 源表 v1：含 amount 列。
	if _, err := svc.Meta.IngestTable(bid, meta.TableSpec{
		QualifiedName: "db.s.src", SchemaVersion: "v1",
		Columns: []meta.ColumnSpec{
			{Name: "id", DataType: "bigint", IsPrimary: true},
			{Name: "amount", DataType: "decimal"},
		},
	}); err != nil {
		return err
	}
	// 源表 v2：amount 被重命名为 amt（最新版本）。
	if _, err := svc.Meta.IngestTable(bid, meta.TableSpec{
		QualifiedName: "db.s.src", SchemaVersion: "v2",
		Columns: []meta.ColumnSpec{
			{Name: "id", DataType: "bigint", IsPrimary: true},
			{Name: "amt", DataType: "decimal"},
		},
	}); err != nil {
		return err
	}
	// 目标表：仍引用 amount 列。
	if _, err := svc.Meta.IngestTable(bid, meta.TableSpec{
		QualifiedName: "db.s.tgt", SchemaVersion: "v1",
		Columns: []meta.ColumnSpec{
			{Name: "id", DataType: "bigint", IsPrimary: true},
			{Name: "amount", DataType: "decimal"},
		},
	}); err != nil {
		return err
	}

	// 变换声明：从 src.amount 派生 tgt.amount（但 src 最新版本已改名 → 断裂）。
	if _, err := svc.AddTransform(bid, 0, "db.s.src", "amount", "db.s.tgt", "amount", "tgt.amount = src.amount"); err != nil {
		return err
	}
	if err := svc.BuildLineage(bid); err != nil {
		return err
	}

	res, err := svc.RunScenario(bid)
	if err != nil {
		return err
	}
	if res.BrokenBefore == 0 {
		return fmt.Errorf("smoke: expected broken-before > 0, got 0")
	}
	if res.BrokenAfter != 0 {
		return fmt.Errorf("smoke: expected broken-after == 0, got %d", res.BrokenAfter)
	}
	if !res.Sealed {
		return fmt.Errorf("smoke: expected batch sealed")
	}
	if res.SnapshotVer < 1 {
		return fmt.Errorf("smoke: expected snapshot version >= 1")
	}

	// 重启恢复校验：关闭后重开同库，确认批次已封存且快照已持久化。
	if err := st.Close(); err != nil {
		return err
	}
	st2, err := store.Open(dbPath)
	if err != nil {
		return err
	}
	defer st2.Close()
	svc2 := service.New(st2)
	b2, err := svc2.Store.GetBatch(bid)
	if err != nil {
		return err
	}
	if b2.SealedAt == nil {
		return fmt.Errorf("smoke: batch not sealed after reopen")
	}
	snaps, err := svc2.Store.ListSnapshots(bid)
	if err != nil {
		return err
	}
	if len(snaps) == 0 {
		return fmt.Errorf("smoke: no snapshot persisted after reopen")
	}
	return nil
}
