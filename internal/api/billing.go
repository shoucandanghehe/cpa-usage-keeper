package api

import (
	"errors"
	"net/http"
	"strconv"

	"cpa-usage-keeper/internal/repository"
	"cpa-usage-keeper/internal/repository/dto"
	"cpa-usage-keeper/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func registerBillingRoutes(router gin.IRoutes, provider service.BillingProvider) {
	if provider == nil {
		return
	}
	router.GET("/billing/accounts", func(c *gin.Context) {
		setNoStoreHeaders(c)
		accounts, err := provider.ListBillingAccounts(c.Request.Context())
		if err != nil {
			writeBillingError(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"accounts": accounts})
	})
	router.GET("/billing/bills", func(c *gin.Context) {
		setNoStoreHeaders(c)
		bills, err := provider.ListBillingBills(c.Request.Context(), c.Query("auth_index"))
		if err != nil {
			writeBillingError(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"bills": bills})
	})
	router.PUT("/billing/bills", func(c *gin.Context) {
		setNoStoreHeaders(c)
		var request dto.SaveBillingBillRequest
		if err := c.ShouldBindJSON(&request); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid billing request body"})
			return
		}
		bill, err := provider.SaveBillingBill(c.Request.Context(), request)
		if err != nil {
			writeBillingError(c, err)
			return
		}
		c.JSON(http.StatusOK, bill)
	})
	router.DELETE("/billing/bills/:id", func(c *gin.Context) {
		setNoStoreHeaders(c)
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil || id <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid bill id"})
			return
		}
		if err := provider.DeleteBillingBill(c.Request.Context(), id); err != nil {
			writeBillingError(c, err)
			return
		}
		c.Status(http.StatusNoContent)
	})
}

func writeBillingError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrBillingInvalidInput):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	case errors.Is(err, gorm.ErrRecordNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "billing account or bill not found"})
	case errors.Is(err, repository.ErrBillingOverlap):
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
	case errors.Is(err, service.ErrBillingIncomplete):
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
	default:
		writeInternalError(c, "billing operation failed", err)
	}
}
