package httpapi

import (
	"net/http"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/timileyin42/zgnis-solar/internal/auth"
	"github.com/timileyin42/zgnis-solar/internal/domain"
	"github.com/timileyin42/zgnis-solar/internal/registry"
)

type emissionFactorResponse struct {
	ID            int64     `json:"id"`
	KgCO2PerKWh   float64   `json:"kg_co2_per_kwh"`
	Country       string    `json:"country"`
	Source        string    `json:"source"`
	EffectiveFrom time.Time `json:"effective_from"`
}

func toEmissionFactorResponse(f registry.EmissionFactor) emissionFactorResponse {
	return emissionFactorResponse{
		ID:            f.ID,
		KgCO2PerKWh:   f.KgCO2PerKWh,
		Country:       f.Country,
		Source:        f.Source,
		EffectiveFrom: f.EffectiveFrom,
	}
}

type emissionPointResponse struct {
	PeriodStart time.Time `json:"period_start"`
	EnergyKWh   float64   `json:"energy_kwh"`
	KgCO2       float64   `json:"kg_co2"`
}

func toEmissionsResponse(period string, series registry.EmissionsSeries) map[string]any {
	points := make([]emissionPointResponse, 0, len(series.Points))
	for _, p := range series.Points {
		points = append(points, emissionPointResponse{PeriodStart: p.PeriodStart, EnergyKWh: p.EnergyKWh, KgCO2: p.KgCO2})
	}
	return map[string]any{
		"unit_kg":                        "kg",
		"unit_tonnes":                    "t",
		"period":                         period,
		"emission_factor":                toEmissionFactorResponse(series.Factor),
		"points":                         points,
		"cumulative_lifetime_co2_tonnes": series.CumulativeTonnesCO2,
	}
}

func emissionsErrorToHTTP(err error) error {
	if err == registry.ErrNoEmissionFactor {
		return echo.NewHTTPError(http.StatusConflict, "no grid emission factor configured yet — POST /v1/config/emission-factor first")
	}
	return echo.NewHTTPError(http.StatusBadRequest, err.Error())
}

func (h *handlers) siteEmissions(c echo.Context) error {
	period, err := parsePeriod(c)
	if err != nil {
		return err
	}
	from, to, err := parseAnalyticsRange(c)
	if err != nil {
		return err
	}

	series, err := h.deps.Emissions.SiteEmissions(c.Request().Context(), c.Param("site_id"), period, from, to)
	if err != nil {
		return emissionsErrorToHTTP(err)
	}
	return c.JSON(http.StatusOK, toEmissionsResponse(period, series))
}

func (h *handlers) fleetEmissions(c echo.Context) error {
	period, err := parsePeriod(c)
	if err != nil {
		return err
	}
	from, to, err := parseAnalyticsRange(c)
	if err != nil {
		return err
	}
	var cohortID *string
	if v := c.QueryParam("cohort_id"); v != "" {
		cohortID = &v
	}

	series, err := h.deps.Emissions.FleetEmissions(c.Request().Context(), cohortID, period, from, to)
	if err != nil {
		return emissionsErrorToHTTP(err)
	}
	return c.JSON(http.StatusOK, toEmissionsResponse(period, series))
}

type setEmissionFactorRequest struct {
	KgCO2PerKWh   float64   `json:"kg_co2_per_kwh"`
	Country       string    `json:"country"`
	Source        string    `json:"source"`
	EffectiveFrom time.Time `json:"effective_from"`
}

func (h *handlers) setEmissionFactor(c echo.Context) error {
	claims, _ := auth.GetClaims(c)

	var req setEmissionFactorRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	factor, err := h.deps.Emissions.Set(c.Request().Context(), claims.UserID, registry.SetEmissionFactorInput{
		KgCO2PerKWh:   req.KgCO2PerKWh,
		Country:       req.Country,
		Source:        req.Source,
		EffectiveFrom: req.EffectiveFrom,
	})
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	return c.JSON(http.StatusCreated, toEmissionFactorResponse(factor))
}

func (h *handlers) getEmissionFactor(c echo.Context) error {
	ctx := c.Request().Context()

	if c.QueryParam("history") == "true" {
		claims, _ := auth.GetClaims(c)
		if claims == nil {
			return echo.NewHTTPError(http.StatusUnauthorized, "missing claims")
		}
		// history=true is operator-only; enforced here rather than at the
		// route level since the same endpoint serves the current-factor
		// lookup to any authenticated role.
		if claims.Role != domain.RoleOperator {
			return echo.NewHTTPError(http.StatusForbidden, "history requires operator role")
		}
		history, err := h.deps.Emissions.History(ctx, c.QueryParam("country"), parseLimit(c))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}
		out := make([]emissionFactorResponse, 0, len(history))
		for _, f := range history {
			out = append(out, toEmissionFactorResponse(f))
		}
		return c.JSON(http.StatusOK, map[string]any{"items": out})
	}

	factor, err := h.deps.Emissions.Current(ctx, c.QueryParam("country"))
	if err != nil {
		return emissionsErrorToHTTP(err)
	}
	return c.JSON(http.StatusOK, toEmissionFactorResponse(factor))
}
