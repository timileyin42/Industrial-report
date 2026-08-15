package httpapi

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/timileyin42/zgnis-solar/internal/auth"
	"github.com/timileyin42/zgnis-solar/internal/domain"
	"github.com/timileyin42/zgnis-solar/internal/registry"
)

type registerDeviceRequest struct {
	DeviceID      string  `json:"device_id"`
	SiteID        string  `json:"site_id"`
	InstallNotes  *string `json:"install_notes,omitempty"`
	InverterBrand *string `json:"inverter_brand,omitempty"`
	InverterModel *string `json:"inverter_model,omitempty"`
}

func (h *handlers) registerDevice(c echo.Context) error {
	claims, _ := auth.GetClaims(c)

	var req registerDeviceRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	result, err := h.deps.Devices.Register(c.Request().Context(), claims.UserID, registry.RegisterDeviceInput{
		DeviceID:      req.DeviceID,
		SiteID:        req.SiteID,
		InstallNotes:  req.InstallNotes,
		InverterBrand: req.InverterBrand,
		InverterModel: req.InverterModel,
	})
	if err != nil {
		if err == registry.ErrUnknownSite {
			return echo.NewHTTPError(http.StatusBadRequest, "site does not exist")
		}
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	return c.JSON(http.StatusCreated, deviceSecretResponse{
		deviceResponse:    toDeviceResponse(result.Device),
		Secret:            result.Secret,
		BrokerSyncWarning: result.BrokerSyncWarning,
	})
}

func (h *handlers) getDevice(c echo.Context) error {
	device, err := h.deps.Devices.Get(c.Request().Context(), c.Param("device_id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "device not found")
	}
	return c.JSON(http.StatusOK, toDeviceResponse(device))
}

func (h *handlers) listDevices(c echo.Context) error {
	claims, _ := auth.GetClaims(c)

	var siteFilter *string
	if claims.Role == domain.RoleRestricted {
		if v := c.QueryParam("site_id"); v != "" && (claims.SiteID == nil || v != *claims.SiteID) {
			return echo.NewHTTPError(http.StatusForbidden, "site access denied")
		}
		siteFilter = claims.SiteID
	} else if v := c.QueryParam("site_id"); v != "" {
		siteFilter = &v
	}

	limit := parseLimit(c)
	devices, next, err := h.deps.Devices.List(c.Request().Context(), siteFilter, c.QueryParam("cursor"), limit)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	items := make([]deviceResponse, 0, len(devices))
	for _, d := range devices {
		items = append(items, toDeviceResponse(d))
	}
	return c.JSON(http.StatusOK, pageResponse[deviceResponse]{Items: items, NextCursor: next})
}

func (h *handlers) revokeDevice(c echo.Context) error {
	claims, _ := auth.GetClaims(c)
	device, err := h.deps.Devices.Revoke(c.Request().Context(), claims.UserID, c.Param("device_id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "device not found")
	}
	return c.JSON(http.StatusOK, toDeviceResponse(device))
}

func (h *handlers) rotateDeviceSecret(c echo.Context) error {
	claims, _ := auth.GetClaims(c)
	result, err := h.deps.Devices.RotateSecret(c.Request().Context(), claims.UserID, c.Param("device_id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "device not found")
	}
	return c.JSON(http.StatusOK, deviceSecretResponse{
		deviceResponse:    toDeviceResponse(result.Device),
		Secret:            result.Secret,
		BrokerSyncWarning: result.BrokerSyncWarning,
	})
}
