package nsfContext_test

import (
	"nsf/internal/db"
	"nsf/internal/nsfContext"
	"nsf/logger"
	"path/filepath"
	"testing"

	loggergoUtil "github.com/Alonza0314/logger-go/v2/util"
)

func TestServiceRecovery(t *testing.T) {
	previousPath := db.NSF_DB_PATH
	db.NSF_DB_PATH = filepath.Join(t.TempDir(), "services.db")
	defer func() { db.NSF_DB_PATH = previousPath }()
	log := logger.NewNsfLogger(loggergoUtil.LogLevelString("error"))
	ctx := nsfContext.NewNsfContext(&nsfContext.NsfContextIE{DbType: "bbolt", NsfLogger: &log})
	if ctx == nil {
		t.Fatal("failed to initialize context")
	}
	defer func() {
		if ctx != nil {
			ctx.Release()
		}
	}()

	if _, _, _, err := ctx.DiscoverService("missing"); err == nil {
		t.Fatal("discovery of an unknown service should fail")
	}
	if err := ctx.RegisterServices("NF1", "127.0.0.1", 8080, []string{"first", "second"}); err != nil {
		t.Fatal(err)
	}
	ctx.Release()
	ctx = nsfContext.NewNsfContext(&nsfContext.NsfContextIE{DbType: "bbolt", NsfLogger: &log})
	if ctx == nil {
		t.Fatal("failed to reopen context")
	}
	for _, service := range []string{"first", "second"} {
		nf, addr, port, err := ctx.DiscoverService(service)
		if err != nil || nf != "NF1" || addr != "127.0.0.1" || port != 8080 {
			t.Fatalf("service %s did not survive restart: (%s, %s, %d, %v)", service, nf, addr, port, err)
		}
	}
	if err := ctx.DeregisterServices("NF2", []string{"first"}); err == nil {
		t.Fatal("another NF must not deregister the service")
	}
	if _, _, _, err := ctx.DiscoverService("first"); err != nil {
		t.Fatalf("rejected deregistration removed the service: %v", err)
	}
	if err := ctx.DeregisterServices("NF1", []string{"first"}); err != nil {
		t.Fatal(err)
	}
	ctx.Release()
	ctx = nsfContext.NewNsfContext(&nsfContext.NsfContextIE{DbType: "bbolt", NsfLogger: &log})
	if ctx == nil {
		t.Fatal("failed to reopen context after deregistration")
	}
	if _, _, _, err := ctx.DiscoverService("first"); err == nil {
		t.Fatal("deregistered service reappeared after restart")
	}
	if _, _, _, err := ctx.DiscoverService("second"); err != nil {
		t.Fatalf("deregistration removed an unrelated service: %v", err)
	}
	if err := ctx.Save("broken", "invalid JSON"); err != nil {
		t.Fatal(err)
	}
	if _, _, _, err := ctx.DiscoverService("broken"); err == nil {
		t.Fatal("discovery should report corrupt service data")
	}
}
