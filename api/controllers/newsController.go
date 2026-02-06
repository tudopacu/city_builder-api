package controllers

import (
	"API/api/services"
	"github.com/gin-gonic/gin"
	"net/http"
	"strconv"
)

func GetNews(c *gin.Context) {
	// Parse page parameter (default: 1)
	page := 1
	if pageParam := c.Query("page"); pageParam != "" {
		parsedPage, err := strconv.Atoi(pageParam)
		if err != nil || parsedPage < 1 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid page parameter"})
			return
		}
		page = parsedPage
	}

	// Parse pageSize parameter (default: 10, max: 100)
	pageSize := 10
	if pageSizeParam := c.Query("pageSize"); pageSizeParam != "" {
		parsedPageSize, err := strconv.Atoi(pageSizeParam)
		if err != nil || parsedPageSize < 1 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid pageSize parameter"})
			return
		}
		if parsedPageSize > 100 {
			parsedPageSize = 100
		}
		pageSize = parsedPageSize
	}

	news, totalCount, err := services.GetNews(page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Calculate total pages
	totalPages := (totalCount + int64(pageSize) - 1) / int64(pageSize)

	c.JSON(http.StatusOK, gin.H{
		"news":       news,
		"page":       page,
		"pageSize":   pageSize,
		"totalCount": totalCount,
		"totalPages": totalPages,
	})
}
