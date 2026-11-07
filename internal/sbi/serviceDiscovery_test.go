package sbi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"nsf/internal/db"
	"nsf/logger"
	"nsf/model"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	loggergoUtil "github.com/Alonza0314/logger-go/v2/util"
	"github.com/gin-gonic/gin"
)

func TestDiscoveryAndRecovery(t *testing.T) {
	previousPath := db.NSF_DB_PATH
	db.NSF_DB_PATH = filepath.Join(t.TempDir(), "nsf.db")
	defer func() { db.NSF_DB_PATH = previousPath }()
	log := logger.NewNsfLogger(loggergoUtil.LogLevelString("error"))
	open := func() *Server {
		s := NewServer(&ServerIE{DbType: "bbolt", NsfLogger: &log})
		if s.NsfContext == nil {
			t.Fatal("failed to open service database")
		}
		s.router = gin.New()
		addService(s)
		return s
	}
	s := open()
	defer func() { s.Processor.Release() }()
	restart := func() {
		s.Processor.Release()
		s = open()
	}
	request := func(t *testing.T, method, path, body string, status int) *httptest.ResponseRecorder {
		t.Helper()
		r := httptest.NewRequest(method, path, strings.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		s.router.ServeHTTP(w, r)
		if w.Code != status {
			t.Fatalf("%s %s: got %d, want %d; body: %s", method, path, w.Code, status, w.Body.String())
		}
		return w
	}
	checkGroup := func(t *testing.T, want []model.NnsfDiscGetResponse) {
		t.Helper()
		services := make([]string, 0, len(want))
		for _, record := range want {
			services = append(services, record.Service)
		}
		body, err := json.Marshal(model.NnsfDiscGroupGetRequest{Services: services})
		if err != nil {
			t.Fatal(err)
		}
		w := request(t, http.MethodGet, "/sbi/nnsf-disc/group", string(body), http.StatusOK)
		var got model.NnsfDiscGroupGetResponse
		if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
			t.Fatal(err)
		}
		if !slices.Equal(got.Services, want) {
			t.Fatalf("group discovery: got %+v, want %+v", got.Services, want)
		}
	}

	request(t, http.MethodGet, "/sbi/nnsf-disc", `{"service":"missing"}`, http.StatusNotFound)
	request(t, http.MethodPost, "/sbi/nnsf-nfm",
		`{"nf_name":"NF1","sbi_address":"127.0.0.1","sbi_port":8080,"services":["first","second"]}`, http.StatusOK)
	request(t, http.MethodPost, "/sbi/nnsf-nfm",
		`{"nf_name":"NF2","sbi_address":"127.0.0.2","sbi_port":9090,"services":["third"]}`, http.StatusOK)
	want := []model.NnsfDiscGetResponse{
		{Service: "third", NfName: "NF2", SbiAddress: "127.0.0.2", SbiPort: 9090},
		{Service: "second", NfName: "NF1", SbiAddress: "127.0.0.1", SbiPort: 8080},
		{Service: "first", NfName: "NF1", SbiAddress: "127.0.0.1", SbiPort: 8080},
	}
	t.Run("GroupDiscoveryAcrossNFs", func(t *testing.T) { checkGroup(t, want) })
	restart()
	t.Run("GroupDiscoveryAfterRestart", func(t *testing.T) { checkGroup(t, want) })

	request(t, http.MethodDelete, "/sbi/nnsf-nfm", `{"nf_name":"NF1","services":["first"]}`, http.StatusNoContent)
	restart()
	t.Run("PartialDeregistrationAfterRestart", func(t *testing.T) {
		request(t, http.MethodGet, "/sbi/nnsf-disc", `{"service":"first"}`, http.StatusNotFound)
		checkGroup(t, want[:2])
	})

	request(t, http.MethodDelete, "/sbi/nnsf-nfm", `{"nf_name":"NF1","services":["second"]}`, http.StatusNoContent)
	request(t, http.MethodDelete, "/sbi/nnsf-nfm", `{"nf_name":"NF2","services":["third"]}`, http.StatusNoContent)
	restart()
	t.Run("AllDeregisteredAfterRestart", func(t *testing.T) {
		for _, service := range []string{"first", "second", "third"} {
			request(t, http.MethodGet, "/sbi/nnsf-disc", `{"service":"`+service+`"}`, http.StatusNotFound)
		}
	})
}
