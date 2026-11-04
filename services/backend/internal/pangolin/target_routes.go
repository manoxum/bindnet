package pangolin

import (
	"fmt"
	"net/http"
)

func (s *service) listTargets(w http.ResponseWriter, r *http.Request, c *client) error {
	id, err := pathID(r)
	if err != nil {
		return err
	}
	ids, err := s.resourceIDs(r.Context(), c.config.ID)
	if err != nil {
		return fmt.Errorf("não foi possível consultar os recursos geridos")
	}
	if _, ok := ids[id]; !ok {
		http.Error(w, "recurso não gerido pelo Bindnet", 403)
		return nil
	}
	targets, err := c.targets(r, id)
	if err != nil {
		return err
	}
	writeJSON(w, targets)
	return nil
}

func (c *client) targets(r *http.Request, id int) ([]Target, error) {
	targets := []Target{}
	for offset := 0; offset < 10000; offset += 100 {
		var data struct {
			Targets    []Target `json:"targets"`
			Pagination struct {
				Total jsonCount `json:"total"`
			} `json:"pagination"`
		}
		path := fmt.Sprintf("/public-resource/%d/targets?limit=100&offset=%d", id, offset)
		if err := c.call(r.Context(), "GET", path, nil, &data); err != nil {
			return nil, err
		}
		targets = append(targets, data.Targets...)
		if len(targets) >= int(data.Pagination.Total) || len(data.Targets) == 0 {
			return targets, nil
		}
	}
	return nil, fmt.Errorf("o recurso excede o limite de 10000 destinos")
}
