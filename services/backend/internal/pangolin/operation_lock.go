package pangolin

import "net/http"

// Consultas e progresso não aguardam downloads; mutações concorrentes falham cedo.
func (s *service) withOperationLock(fn endpoint) endpoint {
	return func(w http.ResponseWriter, r *http.Request, c *client) error {
		if r.Method == http.MethodGet {
			return fn(w, r, c)
		}
		if !s.mutex.TryLock() {
			http.Error(w, "outra operação Pangolin está em andamento; aguarde o estado atual antes de repetir", http.StatusConflict)
			return nil
		}
		defer s.mutex.Unlock()
		return fn(w, r, c)
	}
}
