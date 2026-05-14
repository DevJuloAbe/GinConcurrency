package controller

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"time"

	"gingonic-concurrency/service"
	"gingonic-concurrency/utils"

	"github.com/gin-gonic/gin"
)

type UserController struct {
	userService *service.UserService
}

func NewUserController(userService *service.UserService) *UserController {
	return &UserController{userService: userService}
}

func (ctrl *UserController) SeedUsers(c *gin.Context) {
	req := service.DefaultSeedUsersRequest()

	if err := c.ShouldBindJSON(&req); err != nil && err != io.EOF {
		utils.Error(c, http.StatusBadRequest, "Invalid request body", err, nil)
		return
	}

	result, err := ctrl.userService.SeedUsers(c.Request.Context(), req)
	if err != nil {
		var validationErr *service.ValidationError
		if errors.As(err, &validationErr) {
			utils.Error(c, http.StatusBadRequest, validationErr.Message, nil, validationErr.Details)
			return
		}

		var insertErr *service.InsertError
		if errors.As(err, &insertErr) {
			utils.Error(c, http.StatusInternalServerError, "Failed to insert users", insertErr.Err, gin.H{
				"failed_at_row": insertErr.FailedAtRow,
			})
			return
		}

		utils.Error(c, http.StatusInternalServerError, "Failed to seed users", err, nil)
		return
	}

	utils.Success(c, http.StatusOK, gin.H{
		"message":    "Successfully inserted users",
		"total_rows": result.TotalRows,
		"batch_size": result.BatchSize,
		"game_id":    result.GameID,
		"workers":    result.Workers,
		"duration":   result.Duration,
	})
}

func (ctrl *UserController) FetchUsers(c *gin.Context) {
	req := service.DefaultFetchUsersRequest()

	if err := c.ShouldBindQuery(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "Invalid query parameters", err, nil)
		return
	}

	result, err := ctrl.userService.FetchUsers(req)
	if err != nil {
		var validationErr *service.ValidationError
		if errors.As(err, &validationErr) {
			utils.Error(c, http.StatusBadRequest, validationErr.Message, nil, validationErr.Details)
			return
		}

		utils.Error(c, http.StatusInternalServerError, "Failed to fetch users", err, nil)
		return
	}

	utils.Success(c, http.StatusOK, gin.H{
		"message":    "Successfully fetched users",
		"total_rows": result.TotalRows,
		"batch_size": result.BatchSize,
		"duration":   result.Duration,
		"users":      result.Users,
	})
}

func (ctrl *UserController) FetchUsersByChannel(c *gin.Context) {
	req := service.DefaultFetchUsersRequest()

	if err := c.ShouldBindQuery(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "Invalid query parameters", err, nil)
		return
	}

	batchCh, errCh, err := ctrl.userService.FetchUsersByChannel(c.Request.Context(), req)
	if err != nil {
		var validationErr *service.ValidationError
		if errors.As(err, &validationErr) {
			utils.Error(c, http.StatusBadRequest, validationErr.Message, nil, validationErr.Details)
			return
		}

		utils.Error(c, http.StatusInternalServerError, "Failed to start channel fetch", err, nil)
		return
	}

	start := time.Now()
	fetchedRows := 0
	encoder := json.NewEncoder(c.Writer)

	c.Header("Content-Type", "application/x-ndjson")
	c.Header("Cache-Control", "no-cache")
	c.Header("X-Accel-Buffering", "no")
	c.Status(http.StatusOK)

	if err := encoder.Encode(gin.H{
		"type":       "start",
		"message":    "Started channel user fetch",
		"total_rows": req.TotalRows,
		"batch_size": req.BatchSize,
		"workers":    req.Workers,
	}); err != nil {
		return
	}
	c.Writer.Flush()

	for batch := range batchCh {
		fetchedRows += batch.Count

		select {
		case <-c.Request.Context().Done():
			return
		default:
		}

		if err := encoder.Encode(gin.H{
			"type":  "batch",
			"batch": batch,
		}); err != nil {
			log.Printf("Failed to stream user batch: %v", err)
			return
		}
		c.Writer.Flush()
	}

	select {
	case <-c.Request.Context().Done():
		return
	default:
	}

	select {
	case err := <-errCh:
		if err != nil {
			_ = encoder.Encode(gin.H{
				"type":    "error",
				"message": "Failed to fetch users by channel",
				"error":   err.Error(),
			})
			c.Writer.Flush()
			return
		}
	default:
	}

	if err := encoder.Encode(gin.H{
		"type":       "summary",
		"message":    "Successfully fetched users by channel",
		"total_rows": fetchedRows,
		"batch_size": req.BatchSize,
		"workers":    req.Workers,
		"duration":   time.Since(start).String(),
	}); err != nil {
		log.Printf("Failed to stream user fetch summary: %v", err)
	}
	c.Writer.Flush()
}
