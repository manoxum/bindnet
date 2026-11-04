package pangolin

import (
	"context"
	"net/http"
	"sync"
	"time"
)

type publicationProgress struct {
	State      string    `json:"state"`
	Stage      string    `json:"stage"`
	Message    string    `json:"message"`
	StartedAt  time.Time `json:"startedAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
	SiteID     int       `json:"siteId,omitempty"`
	ResourceID int       `json:"resourceId,omitempty"`
}
type publicationTracker struct {
	mutex    sync.RWMutex
	accounts map[string]publicationProgress
}

func (p *publicationTracker) stage(account, stage, message string, siteID, resourceID int) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	if p.accounts == nil {
		p.accounts = map[string]publicationProgress{}
	}
	value := p.accounts[account]
	if stage == "validating" {
		value = publicationProgress{StartedAt: time.Now()}
	}
	value.State = "running"
	value.Stage = stage
	value.Message = message
	value.UpdatedAt = time.Now()
	if siteID > 0 {
		value.SiteID = siteID
	}
	if resourceID > 0 {
		value.ResourceID = resourceID
	}
	p.accounts[account] = value
}
func (p *publicationTracker) finish(account string, err error) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	value := p.accounts[account]
	value.State = "complete"
	value.Stage = "complete"
	value.Message = "Publicação concluída"
	if err != nil {
		value.State = "failed"
		value.Stage = "failed"
		value.Message = err.Error()
	}
	value.UpdatedAt = time.Now()
	p.accounts[account] = value
}
func (s *service) publicationStatus(w http.ResponseWriter, r *http.Request, c *client) error {
	s.progress.mutex.RLock()
	value, ok := s.progress.accounts[c.config.ID]
	s.progress.mutex.RUnlock()
	if !ok {
		value = publicationProgress{State: "idle", Message: "Nenhuma publicação em andamento neste processo"}
	}
	writeJSON(w, value)
	return nil
}

// Chave de contexto permite que as etapas internas atualizem o mesmo progresso.
type progressKey struct{}
type progressCallback func(string, string, int, int)

func reportProgress(ctx context.Context, stage, message string, siteID, resourceID int) {
	if fn, ok := ctx.Value(progressKey{}).(progressCallback); ok {
		fn(stage, message, siteID, resourceID)
	}
}
