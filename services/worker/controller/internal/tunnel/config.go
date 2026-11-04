package tunnel

import "time"

func containerConfig(c credentials) map[string]any {
	return map[string]any{
		"Image": image,
		"Env": []string{
			"PANGOLIN_ENDPOINT=" + c.Endpoint,
			"NEWT_ID=" + c.NewtID,
			"NEWT_SECRET=" + c.Secret,
			"DISABLE_CLIENTS=true",
			"DISABLE_SSH=true",
			"HEALTH_FILE=/tmp/healthy",
		},
		"Labels": map[string]string{ownerLabel: "true"},
		"Healthcheck": map[string]any{
			"Test":        []string{"CMD-SHELL", "[ -f /tmp/healthy ]"},
			"Interval":    int64(30 * time.Second),
			"Timeout":     int64(3 * time.Second),
			"Retries":     3,
			"StartPeriod": int64(30 * time.Second),
		},
		"HostConfig": map[string]any{
			"NetworkMode":   "host",
			"RestartPolicy": map[string]string{"Name": "unless-stopped"},
			"CapDrop":       []string{"ALL"},
			"SecurityOpt":   []string{"no-new-privileges:true"},
			"LogConfig":     map[string]any{"Type": "json-file", "Config": map[string]string{"max-size": "10m", "max-file": "2"}},
		},
	}
}
