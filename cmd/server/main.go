package main

import (
	"fmt"
	"net/http"
	"os"
	"poi-service/internal/audit"
	"poi-service/internal/importer"
	"poi-service/internal/model"
	"poi-service/internal/poi"
	"poi-service/internal/review"
	"poi-service/internal/search"
	"poi-service/internal/stats"
	"poi-service/internal/store"
	"poi-service/internal/utils"
	"strconv"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	initTestData()

	r := gin.Default()

	r.Use(cors.New(cors.Config{
		AllowAllOrigins:  true,
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"*"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
	}))

	api := r.Group("/api")
	{
		poiGroup := api.Group("/poi")
		{
			poiGroup.GET("/nearby", handleNearbySearch)
			poiGroup.GET("/search", handleKeywordSearch)
			poiGroup.GET("/:id", handleGetPOI)
			poiGroup.POST("", handleCreatePOI)
			poiGroup.PUT("/:id", handleUpdatePOI)
			poiGroup.DELETE("/:id", handleDeletePOI)
			poiGroup.POST("/submit", handleSubmitPOI)
			poiGroup.POST("/check-duplicate", handleCheckDuplicate)
			poiGroup.GET("/quality/issues", handleQualityCheck)
		}

		reviewGroup := api.Group("/review")
		{
			reviewGroup.GET("/pending", handleGetPendingReviews)
			reviewGroup.POST("/:id/approve", handleApprovePOI)
			reviewGroup.POST("/:id/reject", handleRejectPOI)
		}

		importGroup := api.Group("/import")
		{
			importGroup.POST("/start", handleStartImport)
			importGroup.POST("/geojson", handleImportGeoJSON)
			importGroup.GET("/callback", handleImportProgress)
		}

		api.GET("/stats", handleGetStats)

		auditGroup := api.Group("/audit")
		{
			auditGroup.GET("/logs", handleGetAuditLogs)
		}
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	fmt.Printf("POI Service starting on 0.0.0.0:%s\n", port)
	fmt.Printf("Total POIs in memory: %d\n", getTotalPOIs())
	r.Run("0.0.0.0:" + port)
}

func getTotalPOIs() int {
	return store.GetStore().Size()
}

func getOperator(c *gin.Context) string {
	return c.GetHeader("X-User")
}

func getClientIP(c *gin.Context) string {
	ip := c.GetHeader("X-Forwarded-For")
	if ip == "" {
		ip = c.ClientIP()
	}
	return ip
}

func getUserAgent(c *gin.Context) string {
	return c.GetHeader("User-Agent")
}

func handleNearbySearch(c *gin.Context) {
	lat, err1 := strconv.ParseFloat(c.Query("lat"), 64)
	lng, err2 := strconv.ParseFloat(c.Query("lng"), 64)
	radius, err3 := strconv.ParseFloat(c.Query("radius"), 64)
	category := utils.DecodeQueryParam(c.Query("category"))
	limit, err4 := strconv.Atoi(c.Query("limit"))

	if err1 != nil || lat < -90 || lat > 90 {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "invalid lat: must be between -90 and 90",
		})
		return
	}
	if err2 != nil || lng < -180 || lng > 180 {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "invalid lng: must be between -180 and 180",
		})
		return
	}
	if err3 != nil || radius <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "invalid radius: must be a positive number",
		})
		return
	}
	if radius > 500000 {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "invalid radius: maximum 500000 meters (500km)",
		})
		return
	}

	if err4 != nil || limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	results := search.NearbySearch(&search.NearbyQuery{
		Lat:      lat,
		Lng:      lng,
		Radius:   radius,
		Category: category,
		Limit:    limit,
	})

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    results,
		"count":   len(results),
		"radius":  radius,
		"center":  gin.H{"lat": lat, "lng": lng},
	})
}

func handleKeywordSearch(c *gin.Context) {
	q := utils.DecodeQueryParam(c.Query("q"))
	city := utils.DecodeQueryParam(c.Query("city"))
	category := utils.DecodeQueryParam(c.Query("category"))
	page, err1 := strconv.Atoi(c.Query("page"))
	pageSize, err2 := strconv.Atoi(c.Query("page_size"))

	if err1 != nil || page <= 0 {
		page = 1
	}
	if page > 1000 {
		page = 1000
	}
	if err2 != nil || pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	if q == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "query parameter 'q' is required",
		})
		return
	}

	results := search.KeywordSearch(&search.SearchQuery{
		Q:        q,
		City:     city,
		Category: category,
		Page:     page,
		PageSize: pageSize,
	})

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    results,
	})
}

func handleGetPOI(c *gin.Context) {
	id := c.Param("id")
	result, ok := poi.GetPOIDetail(id)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    404,
			"message": "POI not found",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    result,
	})
}

func handleCreatePOI(c *gin.Context) {
	var p model.POI
	if err := c.ShouldBindJSON(&p); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "Invalid request body: " + err.Error(),
		})
		return
	}

	operator := getOperator(c)
	ip := getClientIP(c)
	ua := getUserAgent(c)

	result, err := poi.CreatePOI(&p, operator, ip, ua)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "Create failed: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    result,
	})
}

func handleUpdatePOI(c *gin.Context) {
	id := c.Param("id")
	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "Invalid request body: " + err.Error(),
		})
		return
	}

	operator := getOperator(c)
	ip := getClientIP(c)
	ua := getUserAgent(c)

	result, ok, msg := poi.UpdatePOI(id, updates, operator, ip, ua)
	if !ok {
		if msg == "POI not found" {
			c.JSON(http.StatusNotFound, gin.H{
				"code":    404,
				"message": msg,
			})
		} else {
			c.JSON(http.StatusConflict, gin.H{
				"code":    409,
				"message": msg,
				"data":    result,
			})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    result,
	})
}

func handleDeletePOI(c *gin.Context) {
	id := c.Param("id")
	operator := getOperator(c)
	ip := getClientIP(c)
	ua := getUserAgent(c)

	ok := poi.DeletePOI(id, operator, ip, ua)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    404,
			"message": "POI not found",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
	})
}

func handleSubmitPOI(c *gin.Context) {
	var p model.POI
	if err := c.ShouldBindJSON(&p); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "Invalid request body: " + err.Error(),
		})
		return
	}

	submitter := getOperator(c)
	if submitter == "" {
		submitter = "anonymous"
	}

	submission := review.GetReviewStore().SubmitPOI(&p, submitter)

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    submission,
	})
}

func handleCheckDuplicate(c *gin.Context) {
	var p model.POI
	if err := c.ShouldBindJSON(&p); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "Invalid request body: " + err.Error(),
		})
		return
	}

	result := poi.CheckDuplicates(&p)

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    result,
	})
}

func handleQualityCheck(c *gin.Context) {
	issues := poi.CheckDataQuality()

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    issues,
		"count":   len(issues),
	})
}

func handleGetPendingReviews(c *gin.Context) {
	limit, _ := strconv.Atoi(c.Query("limit"))
	offset, _ := strconv.Atoi(c.Query("offset"))

	if limit <= 0 {
		limit = 20
	}

	results, total := review.GetReviewStore().GetPending(limit, offset)

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    results,
		"total":   total,
	})
}

func handleApprovePOI(c *gin.Context) {
	id := c.Param("id")
	reviewer := getOperator(c)
	ip := getClientIP(c)
	ua := getUserAgent(c)

	if reviewer == "" {
		reviewer = "admin"
	}

	approvedPOI, ok := review.GetReviewStore().Approve(id, reviewer)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    404,
			"message": "Submission not found or already reviewed",
		})
		return
	}

	audit.GetAuditStore().LogApprove(approvedPOI.POIId, reviewer, ip, ua)

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    approvedPOI,
	})
}

func handleRejectPOI(c *gin.Context) {
	id := c.Param("id")
	reviewer := getOperator(c)
	ip := getClientIP(c)
	ua := getUserAgent(c)

	if reviewer == "" {
		reviewer = "admin"
	}

	var req struct {
		Reason string `json:"reason"`
	}
	c.ShouldBindJSON(&req)

	ok := review.GetReviewStore().Reject(id, reviewer, req.Reason)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    404,
			"message": "Submission not found or already reviewed",
		})
		return
	}

	audit.GetAuditStore().LogReject(id, reviewer, ip, ua, req.Reason)

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
	})
}

func handleStartImport(c *gin.Context) {
	var req struct {
		FilePath string `json:"file_path"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "Invalid request body: " + err.Error(),
		})
		return
	}

	operator := getOperator(c)
	ip := getClientIP(c)
	ua := getUserAgent(c)

	if operator == "" {
		operator = "admin"
	}

	job := importer.GetImportStore().CreateJob(req.FilePath, operator, ip, ua)
	importer.GetImportStore().StartJobAsync(job.ID)

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"job_id": job.ID,
			"status": job.Progress.Status,
		},
	})
}

func handleImportProgress(c *gin.Context) {
	jobID := c.Query("job_id")

	job, ok := importer.GetImportStore().GetJob(jobID)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    404,
			"message": "Job not found",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    job.Progress,
	})
}

func handleImportGeoJSON(c *gin.Context) {
	var req struct {
		FilePath string `json:"file_path"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "Invalid request body: " + err.Error(),
		})
		return
	}

	operator := getOperator(c)
	ip := getClientIP(c)
	ua := getUserAgent(c)

	if operator == "" {
		operator = "admin"
	}

	progress, err := importer.ImportGeoJSONFile(req.FilePath, operator, ip, ua)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "Import failed: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    progress,
	})
}

func handleGetStats(c *gin.Context) {
	s := stats.GetOverallStats()

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    s,
	})
}

func handleGetAuditLogs(c *gin.Context) {
	poiID := c.Query("poi_id")
	limit, _ := strconv.Atoi(c.Query("limit"))
	offset, _ := strconv.Atoi(c.Query("offset"))

	if limit <= 0 {
		limit = 20
	}

	logs, total := audit.GetAuditStore().GetLogs(poiID, limit, offset)

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    logs,
		"total":   total,
	})
}

func initTestData() {
	csvFile := "data/beijing_test_data.csv"
	geojsonFile := "data/beijing_test_data.geojson"

	if _, err := os.Stat(csvFile); err == nil {
		fmt.Println("Loading CSV test data from", csvFile)
		job := importer.GetImportStore().CreateJob(csvFile, "system", "127.0.0.1", "init")
		job.Progress.Status = importer.StatusRunning
		importer.GetImportStore().UpdateProgress(job.ID, job.Progress)

		processImportSync(job)

		maxWait := 30 * time.Second
		start := time.Now()
		for time.Since(start) < maxWait {
			currentJob, _ := importer.GetImportStore().GetJob(job.ID)
			if currentJob.Progress.Status == importer.StatusCompleted || currentJob.Progress.Status == importer.StatusFailed {
				break
			}
			time.Sleep(100 * time.Millisecond)
		}

		job, _ = importer.GetImportStore().GetJob(job.ID)
		fmt.Printf("CSV import %s: %d created, %d updated, %d errors\n",
			job.Progress.Status, job.Progress.Created, job.Progress.Updated, job.Progress.Errors)
	} else {
		fmt.Println("CSV test data file not found, skipping CSV import")
	}

	if _, err := os.Stat(geojsonFile); err == nil {
		fmt.Println("Loading GeoJSON test data from", geojsonFile)
		progress, err := importer.ImportGeoJSONFile(geojsonFile, "system", "127.0.0.1", "init")
		if err != nil {
			fmt.Printf("GeoJSON import failed: %v\n", err)
		} else {
			fmt.Printf("GeoJSON import %s: %d created, %d updated, %d errors\n",
				progress.Status, progress.Created, progress.Updated, progress.Errors)
		}
	} else {
		fmt.Println("GeoJSON test data file not found, skipping GeoJSON import")
	}
}

func processImportSync(job *importer.ImportJob) {
	go func() {
		importer.GetImportStore().StartJobAsync(job.ID)
	}()
}
