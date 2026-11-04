package pangolin

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestReadDoesNotWaitForPublication(t *testing.T) {
	s := testService(t)
	started := make(chan struct{})
	release := make(chan struct{})
	defer close(release)
	// Exercita o mesmo bloqueio usado pelos handlers de publicação e status.
	write := func(w http.ResponseWriter, r *http.Request, c *client) error {
		close(started)
		<-release
		w.WriteHeader(204)
		return nil
	}
	read := func(w http.ResponseWriter, r *http.Request, c *client) error { w.WriteHeader(204); return nil }
	first := httptest.NewRequest("POST", "/", nil)
	first.SetPathValue("accountId", "test")
	second := httptest.NewRequest("GET", "/", nil)
	second.SetPathValue("accountId", "test")
	done := make(chan struct{})
	go func() { s.withOperationLock(write)(httptest.NewRecorder(), first, newClient(Config{})) }()
	<-started
	go func() { s.withOperationLock(read)(httptest.NewRecorder(), second, newClient(Config{})); close(done) }()
	select {
	case <-done:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("leitura/status não pode aguardar download ou publicação")
	}
	conflict := httptest.NewRecorder()
	if err := s.withOperationLock(read)(conflict, first, newClient(Config{})); err != nil || conflict.Code != http.StatusConflict {
		t.Fatalf("outra mutação deve receber conflito imediato: %d, %v", conflict.Code, err)
	}
}
