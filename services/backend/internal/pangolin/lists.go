package pangolin

import (
	"context"
	"fmt"
	"net/http"
)

// A paginação evita omitir sites e recursos em organizações maiores.
func (c *client) sites(ctx context.Context) ([]Site, error) {
	result := []Site{}
	for page := 1; page <= 100; page++ {
		var data struct {
			Sites      []Site `json:"sites"`
			Pagination struct {
				Total jsonCount `json:"total"`
			} `json:"pagination"`
		}
		if err := c.call(ctx, http.MethodGet, c.orgPath(fmt.Sprintf("/sites?pageSize=100&page=%d", page)), nil, &data); err != nil {
			return nil, err
		}
		result = append(result, data.Sites...)
		if len(result) >= int(data.Pagination.Total) || len(data.Sites) == 0 {
			return result, nil
		}
	}
	return nil, fmt.Errorf("a organização excede o limite de 10000 sites")
}

func (c *client) resources(ctx context.Context) ([]Resource, error) {
	result := []Resource{}
	for page := 1; page <= 100; page++ {
		var data struct {
			Resources  []Resource `json:"resources"`
			Pagination struct {
				Total jsonCount `json:"total"`
			} `json:"pagination"`
		}
		if err := c.call(ctx, http.MethodGet, c.orgPath(fmt.Sprintf("/public-resources?pageSize=100&page=%d", page)), nil, &data); err != nil {
			return nil, err
		}
		result = append(result, data.Resources...)
		if len(result) >= int(data.Pagination.Total) || len(data.Resources) == 0 {
			return result, nil
		}
	}
	return nil, fmt.Errorf("a organização excede o limite de 10000 recursos")
}

func (c *client) domains(ctx context.Context) ([]Domain, error) {
	result := []Domain{}
	for offset := 0; offset < 10000; offset += 100 {
		var data struct {
			Domains    []Domain `json:"domains"`
			Pagination struct {
				Total jsonCount `json:"total"`
			} `json:"pagination"`
		}
		if err := c.call(ctx, http.MethodGet, c.orgPath(fmt.Sprintf("/domains?limit=100&offset=%d", offset)), nil, &data); err != nil {
			return nil, err
		}
		result = append(result, data.Domains...)
		if len(result) >= int(data.Pagination.Total) || len(data.Domains) == 0 {
			return result, nil
		}
	}
	return nil, fmt.Errorf("a organização excede o limite de 10000 domínios")
}
