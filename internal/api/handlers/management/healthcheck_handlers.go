package management

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/router-for-me/CLIProxyAPI/v6/internal/config"
)

func (h *Handler) GetHealthCheck(c *gin.Context) {
	if h == nil || h.healthChecker == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "health check unavailable"})
		return
	}
	c.JSON(http.StatusOK, h.healthChecker.Snapshot())
}

func (h *Handler) RunHealthCheck(c *gin.Context) {
	if h == nil || h.healthChecker == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "health check unavailable"})
		return
	}
	run, err := h.healthChecker.StartRun("manual")
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"status": "started", "run": run})
}

func (h *Handler) StopHealthCheck(c *gin.Context) {
	if h == nil || h.healthChecker == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "health check unavailable"})
		return
	}
	run, err := h.healthChecker.StopRun()
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"status": "stopping", "run": run})
}

func (h *Handler) GetHealthCheckEnabled(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"enabled": h.cfg.HealthCheck.Enabled})
}

func (h *Handler) PutHealthCheckEnabled(c *gin.Context) {
	h.updateBoolField(c, func(v bool) { h.cfg.HealthCheck.Enabled = v })
}

func (h *Handler) GetHealthCheckInterval(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"interval-seconds": h.cfg.HealthCheck.IntervalSeconds})
}

func (h *Handler) PutHealthCheckInterval(c *gin.Context) {
	var body struct {
		Value *int `json:"value"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.Value == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	value := *body.Value
	if value <= 0 {
		value = 1800
	}
	h.cfg.HealthCheck.IntervalSeconds = value
	h.persist(c)
}

func (h *Handler) GetHealthCheckParallelism(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"parallelism": h.cfg.HealthCheck.Parallelism})
}

func (h *Handler) PutHealthCheckParallelism(c *gin.Context) {
	var body struct {
		Value *int `json:"value"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.Value == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	value := *body.Value
	if value <= 0 {
		value = 4
	}
	if value > 64 {
		value = 64
	}
	h.cfg.HealthCheck.Parallelism = value
	h.persist(c)
}

func (h *Handler) GetHealthCheckUnauthorizedAction(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"unauthorized-action": h.cfg.HealthCheck.UnauthorizedAction})
}

func (h *Handler) PutHealthCheckUnauthorizedAction(c *gin.Context) {
	h.updateHealthCheckAction(c, "unauthorized", func(v string) { h.cfg.HealthCheck.UnauthorizedAction = v })
}

func (h *Handler) GetHealthCheckZeroQuotaAction(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"zero-quota-action": h.cfg.HealthCheck.ZeroQuotaAction})
}

func (h *Handler) PutHealthCheckZeroQuotaAction(c *gin.Context) {
	h.updateHealthCheckAction(c, "zero_quota", func(v string) { h.cfg.HealthCheck.ZeroQuotaAction = v })
}

func (h *Handler) GetHealthCheckProviderPolicies(c *gin.Context) {
	if h.cfg.HealthCheck.ProviderPolicies == nil {
		c.JSON(http.StatusOK, gin.H{"provider-policies": map[string]config.HealthCheckProviderPolicy{}})
		return
	}
	c.JSON(http.StatusOK, gin.H{"provider-policies": h.cfg.HealthCheck.ProviderPolicies})
}

func (h *Handler) PutHealthCheckProviderPolicies(c *gin.Context) {
	var body struct {
		Value map[string]config.HealthCheckProviderPolicy `json:"value"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	h.cfg.HealthCheck.ProviderPolicies = body.Value
	h.cfg.SanitizeHealthCheck()
	h.persist(c)
}

func (h *Handler) GetHealthCheckNotifications(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"notifications": h.cfg.HealthCheck.Notifications})
}

func (h *Handler) PutHealthCheckNotifications(c *gin.Context) {
	var body struct {
		Value config.HealthCheckNotificationsConfig `json:"value"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	h.cfg.HealthCheck.Notifications = body.Value
	h.cfg.SanitizeHealthCheck()
	h.persist(c)
}

func (h *Handler) updateHealthCheckAction(c *gin.Context, kind string, set func(string)) {
	var body struct {
		Value *string `json:"value"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.Value == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	value := strings.ToLower(strings.TrimSpace(*body.Value))
	switch kind {
	case "unauthorized":
		if value != "delete" && value != "disable" && value != "ignore" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid action"})
			return
		}
	case "zero_quota":
		if value != "disable" && value != "ignore" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid action"})
			return
		}
	}
	set(value)
	h.persist(c)
}
