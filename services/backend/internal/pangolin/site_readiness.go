package pangolin

import (
	"context"
	"fmt"
	"strconv"
	"time"
)

// Criar o container não confirma o registro WireGuard nem a rota no Cloud.
func (c *client) waitSiteOnline(ctx context.Context, siteID int) error {
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	for {
		online, err := c.siteOnline(ctx, siteID)
		if err != nil {
			return err
		}
		if online {
			return nil
		}
		timer := time.NewTimer(2 * time.Second)
		select {
		case <-ctx.Done():
			timer.Stop()
			return fmt.Errorf("o túnel não ficou online no Pangolin dentro do prazo")
		case <-timer.C:
		}
	}
}

func (c *client) siteOnline(ctx context.Context, siteID int) (bool, error) {
	var site Site
	err := c.call(ctx, "GET", "/site/"+strconv.Itoa(siteID), nil, &site)
	return site.Online, err
}
