package handler

import (
	"fmt"
	"net"

	"github.com/gin-gonic/gin"
	"github.com/jwalitptl/admin-api/internal/model"
	"github.com/jwalitptl/admin-api/internal/service/region"
)

type BaseHandler struct {
	RegionSvc     *region.Service
	DefaultConfig *model.RegionConfig
}

func (h *BaseHandler) GetRegionConfig(c *gin.Context) *model.RegionConfig {
	if config, exists := c.Get("region_config"); exists {
		return config.(*model.RegionConfig)
	}
	return h.DefaultConfig
}

func (h *BaseHandler) ValidateRegionCompliance(c *gin.Context) error {
	config := h.GetRegionConfig(c)

	// Check GDPR compliance
	if config.Region.GDPR {
		if err := h.validateGDPRCompliance(c); err != nil {
			return err
		}
	}

	// Check HIPAA compliance
	if config.Region.HIPAA {
		if err := h.validateHIPAACompliance(c); err != nil {
			return err
		}
	}

	return nil
}

func (h *BaseHandler) validateGDPRCompliance(c *gin.Context) error {
	// Check for required consent
	if consent := c.GetHeader("X-GDPR-Consent"); consent == "" {
		return fmt.Errorf("GDPR consent is required")
	}

	// Check for data processing agreement
	if dpa := c.GetHeader("X-DPA-Version"); dpa == "" {
		return fmt.Errorf("Data Processing Agreement acceptance is required")
	}

	return nil
}

func (h *BaseHandler) validateHIPAACompliance(c *gin.Context) error {
	// Check for BAA (Business Associate Agreement)
	if baa := c.GetHeader("X-BAA-Version"); baa == "" {
		return fmt.Errorf("Business Associate Agreement is required")
	}

	// Check for required security headers
	if auth := c.GetHeader("Authorization"); auth == "" {
		return fmt.Errorf("secure authentication is required")
	}

	// Verify IP is from allowed range
	config := h.GetRegionConfig(c)
	clientIP := c.ClientIP()
	if !h.isIPAllowed(clientIP, config.SecurityConfig.AllowedIPRanges) {
		return fmt.Errorf("access denied from this IP address")
	}

	return nil
}

func (h *BaseHandler) isIPAllowed(ip string, allowedRanges []string) bool {
	if len(allowedRanges) == 0 {
		return true
	}

	clientIP := net.ParseIP(ip)
	if clientIP == nil {
		return false
	}

	for _, r := range allowedRanges {
		_, ipNet, err := net.ParseCIDR(r)
		if err != nil {
			continue
		}
		if ipNet.Contains(clientIP) {
			return true
		}
	}
	return false
}
