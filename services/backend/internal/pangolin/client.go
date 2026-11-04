package pangolin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

type client struct {
	config Config
	http   *http.Client
}

func newClient(c Config) *client {
	return &client{config: c, http: &http.Client{Timeout: 20 * time.Second, CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }}}
}

func (c *client) orgPath(path string) string {
	return "/org/" + url.PathEscape(c.config.OrgID) + path
}

// call não repete mutações: uma resposta perdida pode já ter criado o recurso.
func (c *client) call(ctx context.Context, method, path string, body, out any) error {
	var data []byte
	var err error
	if body != nil {
		data, err = json.Marshal(body)
		if err != nil {
			return err
		}
	}
	req, err := http.NewRequestWithContext(ctx, method, c.config.APIURL+"/v1"+path, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("não foi possível preparar a chamada ao Pangolin")
	}
	req.Header.Set("Authorization", "Bearer "+c.config.APIKey)
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("não foi possível contactar a API Pangolin; verifique o endereço, a rede e o TLS antes de repetir a operação")
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// A resposta remota pode conter segredos; não a repasse ao navegador/log.
		return &apiError{status: resp.StatusCode}
	}
	if resp.StatusCode == http.StatusNoContent {
		return nil
	}
	var envelope struct {
		Data    json.RawMessage `json:"data"`
		Success *bool           `json:"success"`
		Error   bool            `json:"error"`
	}
	if err = json.NewDecoder(io.LimitReader(resp.Body, 4<<20)).Decode(&envelope); err != nil {
		return fmt.Errorf("resposta inválida da API Pangolin")
	}
	if envelope.Error || (envelope.Success != nil && !*envelope.Success) {
		return fmt.Errorf("a API Pangolin recusou a operação")
	}
	if out == nil {
		return nil
	}
	if len(envelope.Data) == 0 || string(envelope.Data) == "null" {
		return fmt.Errorf("resposta Pangolin sem dados; verifique a versão da Integration API")
	}
	if err = json.Unmarshal(envelope.Data, out); err != nil {
		return fmt.Errorf("formato de resposta Pangolin incompatível")
	}
	return nil
}

type apiError struct{ status int }

func (e *apiError) Error() string {
	switch e.status {
	case 401:
		return "Pangolin recusou a autenticação (HTTP 401); verifique a chave da conta"
	case 403:
		return "Pangolin recusou esta operação (HTTP 403); verifique permissões e limites do plano"
	case 404:
		return "Pangolin não disponibiliza esta rota ou não encontrou o recurso (HTTP 404)"
	case 400, 409:
		return fmt.Sprintf("Pangolin recusou os dados ou encontrou um conflito (HTTP %d); verifique o domínio e o estado do recurso", e.status)
	case 429:
		return "Pangolin limitou a frequência das chamadas (HTTP 429); tente novamente mais tarde"
	}
	return fmt.Sprintf("Pangolin retornou HTTP %d", e.status)
}
func (c *client) remove(ctx context.Context, path string) error {
	err := c.call(ctx, "DELETE", path, nil, nil)
	if remote, ok := err.(*apiError); ok && remote.status == 404 {
		return nil
	}
	return err
}
