// Command fiberlineage 是「古籍纸张纤维谱系比对台」的服务入口：
// 默认启动 HTTP API 服务；--smoke-test 模式下在临时库上跑迁移 + 自检后退出（用于健康门禁）。
package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"task215-fiberlineage/internal/httpapi"
	"task215-fiberlineage/internal/service"
	"task215-fiberlineage/internal/store"
)

func main() {
	var (
		dbPath    string
		addr      string
		smokeTest bool
	)
	flag.StringVar(&dbPath, "db", "fiberlineage.db", "SQLite 数据库文件路径")
	flag.StringVar(&addr, "addr", ":8080", "HTTP 服务监听地址")
	flag.BoolVar(&smokeTest, "smoke-test", false, "在临时库上执行迁移+自检后退出，返回 0 表示健康")
	flag.Parse()

	if smokeTest {
		if err := runSmokeTest(); err != nil {
			log.Fatalf("smoke-test failed: %v", err)
		}
		fmt.Println("smoke-test OK")
		return
	}

	st, err := store.Open(dbPath)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer st.Close()
	if err := st.Migrate(); err != nil {
		log.Fatalf("migrate: %v", err)
	}
	svc := service.New(st)
	srv := &http.Server{
		Addr:              addr,
		Handler:           httpapi.NewServer(svc).Handler(),
		ReadHeaderTimeout: 10 * time.Second,
	}
	log.Printf("fiberlineage listening on %s", addr)
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("serve: %v", err)
	}
}

// runSmokeTest 在临时 SQLite 上执行迁移、跑一次自检，确保服务层闭环健康。
func runSmokeTest() error {
	tmp, err := os.MkdirTemp("", "fiberlineage-smoke-")
	if err != nil {
		return fmt.Errorf("mkdtemp: %w", err)
	}
	defer os.RemoveAll(tmp)
	db := filepath.Join(tmp, "smoke.db")
	st, err := store.Open(db)
	if err != nil {
		return fmt.Errorf("open: %w", err)
	}
	defer st.Close()
	if err := st.Migrate(); err != nil {
		return fmt.Errorf("migrate: %w", err)
	}
	svc := service.New(st)
	rep := svc.SelfCheck()
	if !rep.OK {
		return fmt.Errorf("self-check reported %d issues", len(rep.Issues))
	}
	return nil
}
