package service

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"gingonic-concurrency/model"
	"gingonic-concurrency/repository"
)

const (
	DefaultTotalRows   = 5000000
	DefaultBatchSize   = 1000
	DefaultSeedWorkers = 8
	MaxBatchSize       = 4000
	MaxSeedWorkers     = 64

	DefaultFetchTotalRows = 5000000
	DefaultFetchBatchSize = 10000
	DefaultFetchWorkers   = 128
	MaxFetchBatchSize     = 50000
	MaxFetchWorkers       = 1080
)

type SeedUsersRequest struct {
	TotalRows int  `json:"total_rows"`
	BatchSize int  `json:"batch_size"`
	GameID    uint `json:"game_id"`
	Workers   int  `json:"workers"`
}

type SeedUsersResult struct {
	TotalRows int
	BatchSize int
	GameID    uint
	Workers   int
	Duration  string
}

type FetchUsersRequest struct {
	TotalRows int `form:"total_rows"`
	BatchSize int `form:"batch_size"`
	Workers   int `form:"workers"`
}

type FetchUsersResult struct {
	Users     []model.User
	TotalRows int
	BatchSize int
	Duration  string
}

type FetchUsersBatch struct {
	WorkerID int          `json:"worker_id"`
	Start    int          `json:"start"`
	End      int          `json:"end"`
	Count    int          `json:"count"`
	Users    []model.User `json:"users"`
}

type seedUsersJob struct {
	Start int
	End   int
}

type fetchUsersJob struct {
	Start  int
	Offset int
	Limit  int
}

type ValidationError struct {
	Message string
	Details map[string]any
}

func (e *ValidationError) Error() string {
	return e.Message
}

type InsertError struct {
	FailedAtRow int
	Err         error
}

func (e *InsertError) Error() string {
	return e.Err.Error()
}

func (e *InsertError) Unwrap() error {
	return e.Err
}

type UserService struct {
	userRepository *repository.UserRepository
}

func NewUserService(userRepository *repository.UserRepository) *UserService {
	return &UserService{userRepository: userRepository}
}

func DefaultSeedUsersRequest() SeedUsersRequest {
	return SeedUsersRequest{
		TotalRows: DefaultTotalRows,
		BatchSize: DefaultBatchSize,
		GameID:    1,
		Workers:   DefaultSeedWorkers,
	}
}

func DefaultFetchUsersRequest() FetchUsersRequest {
	return FetchUsersRequest{
		TotalRows: DefaultFetchTotalRows,
		BatchSize: DefaultFetchBatchSize,
		Workers:   DefaultFetchWorkers,
	}
}

func (s *UserService) SeedUsers(ctx context.Context, req SeedUsersRequest) (*SeedUsersResult, error) {
	if err := validateSeedUsersRequest(req); err != nil {
		return nil, err
	}
	if ctx == nil {
		ctx = context.Background()
	}

	start := time.Now()
	otpExpiresAt := start.Add(10 * time.Minute)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	jobCh := make(chan seedUsersJob, req.Workers)
	errCh := make(chan error, 1)

	sendErr := func(err error) {
		select {
		case errCh <- err:
			cancel()
		default:
		}
	}

	var wg sync.WaitGroup
	for workerID := 1; workerID <= req.Workers; workerID++ {
		wg.Add(1)
		go s.seedUsersByWorker(ctx, workerID, req, otpExpiresAt, jobCh, sendErr, &wg)
	}

publishLoop:
	for i := 1; i <= req.TotalRows; i += req.BatchSize {
		end := i + req.BatchSize - 1
		if end > req.TotalRows {
			end = req.TotalRows
		}

		select {
		case <-ctx.Done():
			break publishLoop
		case jobCh <- seedUsersJob{Start: i, End: end}:
		}
	}
	close(jobCh)
	wg.Wait()

	select {
	case err := <-errCh:
		return nil, err
	default:
	}

	if err := ctx.Err(); err != nil {
		return nil, err
	}

	return &SeedUsersResult{
		TotalRows: req.TotalRows,
		BatchSize: req.BatchSize,
		GameID:    req.GameID,
		Workers:   req.Workers,
		Duration:  time.Since(start).String(),
	}, nil
}

func (s *UserService) seedUsersByWorker(
	ctx context.Context,
	workerID int,
	req SeedUsersRequest,
	otpExpiresAt time.Time,
	jobCh <-chan seedUsersJob,
	sendErr func(error),
	wg *sync.WaitGroup,
) {
	defer wg.Done()

	for job := range jobCh {
		select {
		case <-ctx.Done():
			return
		default:
		}

		users := buildSeedUsers(req, job, otpExpiresAt)

		if err := s.userRepository.CreateUsersWithContext(ctx, users); err != nil {
			sendErr(&InsertError{FailedAtRow: job.Start, Err: err})
			return
		}

		log.Printf("Worker %d inserted rows: %d to %d", workerID, job.Start, job.End)
	}
}

func buildSeedUsers(req SeedUsersRequest, job seedUsersJob, otpExpiresAt time.Time) []model.User {
	users := make([]model.User, 0, job.End-job.Start+1)

	for j := job.Start; j <= job.End; j++ {
		users = append(users, model.User{
			GameID:       req.GameID,
			IsVerified:   false,
			OTP:          fmt.Sprintf("%06d", j%1000000),
			OTPExpiresAt: otpExpiresAt,
			Name:         fmt.Sprintf("User %d", j),
			Password:     fmt.Sprintf("password%d", j),
			Phone:        fmt.Sprintf("123-456-789%d", j),
			Gender:       "Male",
			Address:      fmt.Sprintf("Address %d", j),
			Email:        fmt.Sprintf("userexample%d@gphi.com", j),
			Role:         "user",
		})
	}

	return users
}

func (s *UserService) FetchUsers(req FetchUsersRequest) (*FetchUsersResult, error) {

	if err := validateFetchUsersRequest(req); err != nil {
		return nil, err
	}

	start := time.Now()
	users := make([]model.User, 0, req.TotalRows)

	for offset := 0; len(users) < req.TotalRows; offset += req.BatchSize {
		limit := req.BatchSize
		remaining := req.TotalRows - len(users)
		if remaining < limit {
			limit = remaining
		}

		batch, err := s.userRepository.FetchUsers(limit, offset)
		if err != nil {
			return nil, err
		}
		if len(batch) == 0 {
			break
		}

		users = append(users, batch...)
		log.Printf("Fetched rows: %d to %d", offset+1, offset+len(batch))
	}

	return &FetchUsersResult{
		Users:     users,
		TotalRows: len(users),
		BatchSize: req.BatchSize,
		Duration:  time.Since(start).String(),
	}, nil
}

func (s *UserService) FetchUsersByChannel(ctx context.Context, req FetchUsersRequest) (<-chan FetchUsersBatch, <-chan error, error) {
	if err := validateFetchUsersRequest(req); err != nil {
		return nil, nil, err
	}

	ctx, cancel := context.WithCancel(ctx)
	jobCh := make(chan fetchUsersJob, req.Workers)
	errCh := make(chan error, 1)

	sendErr := func(err error) {
		select {
		case errCh <- err:
			cancel()
		default:
		}
	}

	go publishFetchJobs(ctx, jobCh, req.TotalRows, req.BatchSize)

	workerChannels := make([]<-chan FetchUsersBatch, 0, req.Workers)
	for workerID := 1; workerID <= req.Workers; workerID++ {
		workerChannels = append(workerChannels, s.fetchUserBatchesByWorker(ctx, workerID, jobCh, sendErr))
	}

	batchCh := mergeFetchUserBatchChannels(ctx, cancel, errCh, workerChannels...)

	return batchCh, errCh, nil
}

func publishFetchJobs(ctx context.Context, jobCh chan<- fetchUsersJob, totalRows int, batchSize int) {
	defer close(jobCh)

	for offset := 0; offset < totalRows; offset += batchSize {
		limit := batchSize
		remaining := totalRows - offset
		if remaining < limit {
			limit = remaining
		}

		job := fetchUsersJob{
			Start:  offset + 1,
			Offset: offset,
			Limit:  limit,
		}

		select {
		case <-ctx.Done():
			return
		case jobCh <- job:
		}
	}
}

func (s *UserService) fetchUserBatchesByWorker(
	ctx context.Context,
	workerID int,
	jobCh <-chan fetchUsersJob,
	sendErr func(error),
) <-chan FetchUsersBatch {
	batchCh := make(chan FetchUsersBatch, 1)

	go func() {
		defer close(batchCh)

		for job := range jobCh {
			select {
			case <-ctx.Done():
				return
			default:
			}

			users, err := s.userRepository.FetchUsersWithContext(ctx, job.Limit, job.Offset)
			if err != nil {
				sendErr(err)
				return
			}
			if len(users) == 0 {
				continue
			}

			batch := FetchUsersBatch{
				WorkerID: workerID,
				Start:    job.Start,
				End:      job.Offset + len(users),
				Count:    len(users),
				Users:    users,
			}

			select {
			case <-ctx.Done():
				return
			case batchCh <- batch:
				log.Printf("Worker %d fetched rows by channel: %d to %d", workerID, batch.Start, batch.End)
			}
		}
	}()

	return batchCh
}

func mergeFetchUserBatchChannels(
	ctx context.Context,
	cancel context.CancelFunc,
	errCh chan error,
	workerChannels ...<-chan FetchUsersBatch,
) <-chan FetchUsersBatch {
	mergedCh := make(chan FetchUsersBatch, len(workerChannels))
	var wg sync.WaitGroup

	for _, workerCh := range workerChannels {
		wg.Add(1)

		go func(workerCh <-chan FetchUsersBatch) {
			defer wg.Done()

			for batch := range workerCh {
				select {
				case <-ctx.Done():
					return
				case mergedCh <- batch:
				}
			}
		}(workerCh)
	}

	go func() {
		wg.Wait()
		cancel()
		close(mergedCh)
		close(errCh)
	}()

	return mergedCh
}

func validateSeedUsersRequest(req SeedUsersRequest) error {
	if req.TotalRows <= 0 {
		return &ValidationError{Message: "total_rows must be greater than 0"}
	}
	if req.BatchSize <= 0 {
		return &ValidationError{Message: "batch_size must be greater than 0"}
	}
	if req.BatchSize > MaxBatchSize {
		return &ValidationError{
			Message: "batch_size is too large for MySQL prepared statements",
			Details: map[string]any{
				"max_batch_size": MaxBatchSize,
			},
		}
	}
	if req.GameID == 0 {
		return &ValidationError{Message: "game_id must be greater than 0"}
	}
	if req.Workers <= 0 {
		return &ValidationError{Message: "workers must be greater than 0"}
	}
	if req.Workers > MaxSeedWorkers {
		return &ValidationError{
			Message: "workers is too large for seed requests",
			Details: map[string]any{
				"max_workers": MaxSeedWorkers,
			},
		}
	}

	return nil
}

func validateFetchUsersRequest(req FetchUsersRequest) error {
	if req.TotalRows <= 0 {
		return &ValidationError{Message: "total_rows must be greater than 0"}
	}
	if req.BatchSize <= 0 {
		return &ValidationError{Message: "batch_size must be greater than 0"}
	}
	if req.BatchSize > MaxFetchBatchSize {
		return &ValidationError{
			Message: "batch_size is too large for fetch requests",
			Details: map[string]any{
				"max_batch_size": MaxFetchBatchSize,
			},
		}
	}
	if req.Workers <= 0 {
		return &ValidationError{Message: "workers must be greater than 0"}
	}
	if req.Workers > MaxFetchWorkers {
		return &ValidationError{
			Message: "workers is too large for fetch requests",
			Details: map[string]any{
				"max_workers": MaxFetchWorkers,
			},
		}
	}

	return nil
}
