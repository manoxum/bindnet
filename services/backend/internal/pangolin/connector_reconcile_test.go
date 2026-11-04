package pangolin

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestConnectorAutostartRestoresSavedSitesAcrossAccounts(t *testing.T) {
	s := testService(t)
	if _, err := s.db.Exec(`INSERT INTO pangolin_account_sites(account_id,site_id,name,newt_id,secret) VALUES('other',7,'Outro','other-id','other-secret')`); err != nil {
		t.Fatal(err)
	}
	started := map[string]string{}
	s.worker = testWorker(t, func(w http.ResponseWriter, r *http.Request) {
		var body map[string]string
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		if r.Method != "POST" || body["endpoint"] != "https://app.pangolin.net" {
			t.Error("arranque deve usar a rota específica e o endpoint guardado")
		}
		started[r.URL.Path] = body["newtId"] + ":" + body["secret"]
		w.WriteHeader(204)
	})
	if err := s.reconcileConnectorsOnce(t.Context()); err != nil {
		t.Fatal(err)
	}
	if len(started) != 2 || started["/tunnels/test/7/start"] != "newt-id:private-secret" || started["/tunnels/other/7/start"] != "other-id:other-secret" {
		t.Fatalf("conectores guardados devem iniciar automaticamente com isolamento por conta; iniciados: %d", len(started))
	}
}

func TestConnectorAutostartRetriesFailureWithoutBlockingOtherAccounts(t *testing.T) {
	s := testService(t)
	if _, err := s.db.Exec(`INSERT INTO pangolin_account_sites(account_id,site_id,name,newt_id,secret) VALUES('other',7,'Outro','other-id','other-secret')`); err != nil {
		t.Fatal(err)
	}
	var failures, successes atomic.Int32
	s.worker = testWorker(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/tunnels/other/7/start" && failures.Add(1) == 1 {
			w.WriteHeader(503)
			return
		}
		successes.Add(1)
		w.WriteHeader(204)
	})
	if err := s.reconcileConnectorsOnce(t.Context()); err == nil || successes.Load() != 1 {
		t.Fatal("falha temporária deve permitir recuperar os outros conectores")
	}
	if err := s.reconcileConnectorsOnce(t.Context()); err != nil || successes.Load() != 3 {
		t.Fatal("a passagem seguinte deve repetir o arranque que falhou")
	}
}

func TestConnectorAutostartPreservesOperatorStopAcrossRestart(t *testing.T) {
	for _, workerStatus := range []int{204, 503} {
		t.Run(http.StatusText(workerStatus), func(t *testing.T) {
			s := testService(t)
			var starts atomic.Int32
			s.worker = testWorker(t, func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/tunnels/test/7/start" {
					starts.Add(1)
				}
				w.WriteHeader(workerStatus)
			})
			r := httptest.NewRequest("POST", "/", nil)
			r.SetPathValue("id", "7")
			c := newClient(Config{ID: "test", Endpoint: "https://app.pangolin.net"})
			err := s.stopConnector(httptest.NewRecorder(), r, c)
			if (err != nil) != (workerStatus == 503) {
				t.Fatalf("resultado inesperado da parada: %v", err)
			}
			// Nova instância simula reinício do backend com o mesmo banco.
			restarted := &service{db: s.db, worker: s.worker}
			if err := restarted.reconcileConnectorsOnce(t.Context()); err != nil || starts.Load() != 0 {
				t.Fatal("Parar deve impedir o arranque automático, mesmo após falha do worker")
			}
			_ = s.startConnector(httptest.NewRecorder(), r, c)
			_ = restarted.reconcileConnectorsOnce(t.Context())
			if starts.Load() != 2 {
				t.Fatal("Iniciar deve restaurar o arranque automático, mesmo após falha do worker")
			}
		})
	}
}

func TestConnectorAutostartStartsImmediatelyAndHonorsCancellation(t *testing.T) {
	s := testService(t)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	started := make(chan struct{}, 1)
	s.worker = testWorker(t, func(w http.ResponseWriter, r *http.Request) {
		started <- struct{}{}
		w.WriteHeader(204)
	})
	done := make(chan struct{})
	go func() {
		s.StartConnectorReconciliationLoop(ctx)
		close(done)
	}()
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("o arranque deve ocorrer antes do primeiro intervalo")
	}
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("cancelamento deve encerrar o loop sem aguardar o intervalo")
	}
}

func TestConnectorAutostartWaitsForConcurrentMutation(t *testing.T) {
	s := testService(t)
	var starts atomic.Int32
	s.worker = testWorker(t, func(w http.ResponseWriter, _ *http.Request) {
		starts.Add(1)
		w.WriteHeader(204)
	})
	s.mutex.Lock()
	err := s.reconcileConnectorsOnce(t.Context())
	if _, stopErr := s.db.Exec(`UPDATE pangolin_account_sites SET desired_running=false`); stopErr != nil {
		t.Error(stopErr)
	}
	s.mutex.Unlock()
	if err != nil || starts.Load() != 0 {
		t.Fatal("recuperação não deve concorrer com mutações do painel")
	}
	if err := s.reconcileConnectorsOnce(t.Context()); err != nil || starts.Load() != 0 {
		t.Fatal("parada concorrente deve prevalecer na passagem seguinte")
	}
}
