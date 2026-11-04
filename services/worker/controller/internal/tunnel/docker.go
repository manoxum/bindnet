// Package tunnel controla somente conectores Newt criados pelo Bindnet.
package tunnel

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"time"
)

const image = "fosrl/newt:1.19.0"
const ownerLabel = "net.bindnet.pangolin"

type dockerClient struct{ http *http.Client }

func newDockerClient() *dockerClient {
	return &dockerClient{http: &http.Client{Timeout: 80 * time.Second, Transport: &http.Transport{DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, "unix", "/var/run/docker.sock")
	}}}}
}

func (d *dockerClient) call(ctx context.Context, method, path string, body, out any) (int, error) {
	var data []byte
	var err error
	if body != nil {
		data, err = json.Marshal(body)
		if err != nil {
			return 0, err
		}
	}
	req, err := http.NewRequestWithContext(ctx, method, "http://docker"+path, bytes.NewReader(data))
	if err != nil {
		return 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := d.http.Do(req)
	if err != nil {
		return 0, fmt.Errorf("não foi possível contactar o Docker")
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return resp.StatusCode, fmt.Errorf("Docker retornou HTTP %d", resp.StatusCode)
	}
	if out != nil {
		err = json.NewDecoder(io.LimitReader(resp.Body, 2<<20)).Decode(out)
	}
	return resp.StatusCode, err
}

func (d *dockerClient) ensureImage(ctx context.Context) error {
	status, err := d.call(ctx, "GET", "/images/"+image+"/json", nil, nil)
	if err == nil {
		return nil
	}
	if status != 404 {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, "POST", "http://docker/images/create?fromImage="+url.QueryEscape(image), nil)
	if err != nil {
		return err
	}
	resp, err := d.http.Do(req)
	if err != nil {
		return fmt.Errorf("não foi possível baixar a imagem Newt")
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("Docker não conseguiu baixar a imagem Newt")
	}
	dec := json.NewDecoder(resp.Body)
	for {
		var item struct {
			Error string `json:"error"`
		}
		err := dec.Decode(&item)
		if err == io.EOF {
			return nil
		}
		if err != nil || item.Error != "" {
			return fmt.Errorf("falha ao baixar a imagem Newt")
		}
	}
}

type container struct {
	ID     string `json:"Id"`
	Config struct {
		Labels map[string]string `json:"Labels"`
	} `json:"Config"`
	State struct {
		Running bool   `json:"Running"`
		Status  string `json:"Status"`
		Health  struct {
			Status string `json:"Status"`
		} `json:"Health"`
	} `json:"State"`
}

func (d *dockerClient) inspect(ctx context.Context, name string) (container, error) {
	var c container
	status, err := d.call(ctx, "GET", "/containers/"+name+"/json", nil, &c)
	if status == 404 {
		return c, nil
	}
	if err != nil {
		return c, err
	}
	if c.Config.Labels[ownerLabel] != "true" {
		return container{}, fmt.Errorf("o container existente não pertence à integração Pangolin")
	}
	return c, nil
}
