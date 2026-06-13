package importer

import (
	"encoding/csv"
	"io"
	"os"
	"poi-service/internal/audit"
	"poi-service/internal/model"
	"poi-service/internal/store"
	"strconv"
	"strings"
	"sync"
	"time"
)

type ImportJobStatus string

const (
	StatusPending   ImportJobStatus = "pending"
	StatusRunning   ImportJobStatus = "running"
	StatusCompleted ImportJobStatus = "completed"
	StatusFailed    ImportJobStatus = "failed"
)

type ImportProgress struct {
	Total     int             `json:"total"`
	Processed int             `json:"processed"`
	Errors    int             `json:"errors"`
	Created   int             `json:"created"`
	Updated   int             `json:"updated"`
	Status    ImportJobStatus `json:"status"`
	ErrorFile string          `json:"error_file,omitempty"`
}

type ImportJob struct {
	ID        string
	FilePath  string
	Progress  ImportProgress
	Operator  string
	IP        string
	UA        string
	CreatedAt time.Time
}

type ImportStore struct {
	mu      sync.RWMutex
	jobs    map[string]*ImportJob
	jobList []string
}

var instance *ImportStore
var once sync.Once

func GetImportStore() *ImportStore {
	once.Do(func() {
		instance = &ImportStore{
			jobs:    make(map[string]*ImportJob),
			jobList: make([]string, 0),
		}
	})
	return instance
}

func (i *ImportStore) CreateJob(filePath, operator, ip, ua string) *ImportJob {
	i.mu.Lock()
	defer i.mu.Unlock()

	jobID := generateJobID()
	job := &ImportJob{
		ID:        jobID,
		FilePath:  filePath,
		Operator:  operator,
		IP:        ip,
		UA:        ua,
		CreatedAt: time.Now(),
		Progress: ImportProgress{
			Status: StatusPending,
		},
	}

	i.jobs[jobID] = job
	i.jobList = append(i.jobList, jobID)

	return job
}

func (i *ImportStore) GetJob(jobID string) (*ImportJob, bool) {
	i.mu.RLock()
	defer i.mu.RUnlock()
	job, ok := i.jobs[jobID]
	return job, ok
}

func (i *ImportStore) UpdateProgress(jobID string, progress ImportProgress) {
	i.mu.Lock()
	defer i.mu.Unlock()
	if job, ok := i.jobs[jobID]; ok {
		job.Progress = progress
	}
}

func (i *ImportStore) StartJobAsync(jobID string) {
	job, ok := i.GetJob(jobID)
	if !ok {
		return
	}

	go func() {
		job.Progress.Status = StatusRunning
		i.UpdateProgress(jobID, job.Progress)
		processImport(job, i)
	}()
}

func processImport(job *ImportJob, store_ *ImportStore) {
	file, err := os.Open(job.FilePath)
	if err != nil {
		job.Progress.Status = StatusFailed
		store_.UpdateProgress(job.ID, job.Progress)
		return
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.LazyQuotes = true
	reader.FieldsPerRecord = -1

	headers, err := reader.Read()
	if err != nil {
		job.Progress.Status = StatusFailed
		store_.UpdateProgress(job.ID, job.Progress)
		return
	}

	headerMap := make(map[string]int)
	for idx, h := range headers {
		headerMap[strings.TrimSpace(strings.ToLower(h))] = idx
	}

	errorFileName := "errors_" + job.ID + ".csv"
	errorFile, err := os.Create(errorFileName)
	if err != nil {
		job.Progress.Status = StatusFailed
		store_.UpdateProgress(job.ID, job.Progress)
		return
	}
	defer errorFile.Close()

	errorWriter := csv.NewWriter(errorFile)
	defer errorWriter.Flush()

	errorWriter.Write(append(headers, "error_message"))

	var allRecords [][]string
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			continue
		}
		allRecords = append(allRecords, record)
	}

	job.Progress.Total = len(allRecords)
	store_.UpdateProgress(job.ID, job.Progress)

	s := store.GetStore()

	for idx, record := range allRecords {
		poi, parseErr := parsePOIRecord(record, headerMap)
		if parseErr != nil {
			job.Progress.Errors++
			errorWriter.Write(append(record, parseErr.Error()))
			continue
		}

		existing := findExistingPOI(poi, s)
		isUpdate := existing != nil

		if isUpdate {
			poi.POIId = existing.POIId
			poi.CreatedAt = existing.CreatedAt
			poi.UpdatedAt = time.Now()
			job.Progress.Updated++
		} else {
			poi.POIId = generatePOIID()
			poi.CreatedAt = time.Now()
			poi.UpdatedAt = time.Now()
			poi.Source = model.SourcePartner
			job.Progress.Created++
		}

		s.AddPOI(poi)
		audit.GetAuditStore().LogImport(poi.POIId, job.Operator, job.IP, job.UA, isUpdate)

		job.Progress.Processed = idx + 1
		if idx%10 == 0 {
			store_.UpdateProgress(job.ID, job.Progress)
		}
	}

	job.Progress.Status = StatusCompleted
	job.Progress.ErrorFile = errorFileName
	store_.UpdateProgress(job.ID, job.Progress)
}

func parsePOIRecord(record []string, headerMap map[string]int) (*model.POI, error) {
	poi := &model.POI{}

	getField := func(name string) string {
		if idx, ok := headerMap[name]; ok && idx < len(record) {
			return strings.TrimSpace(record[idx])
		}
		return ""
	}

	poi.Name.ZhCN = getField("name_zh")
	poi.Name.EnUS = getField("name_en")
	poi.Name.JaJP = getField("name_ja")
	poi.Name.KoKR = getField("name_ko")

	if poi.Name.ZhCN == "" && poi.Name.EnUS == "" {
		poi.Name.ZhCN = getField("name")
	}

	poi.Category.L1 = getField("category_l1")
	poi.Category.L2 = getField("category_l2")
	poi.Category.L3 = getField("category_l3")

	if poi.Category.L1 == "" {
		poi.Category.L1 = getField("category")
	}

	poi.Address = getField("address")
	poi.City = getField("city")
	poi.District = getField("district")
	poi.Phone = getField("phone")

	latStr := getField("lat")
	lngStr := getField("lng")

	var err error
	poi.Lat, err = parseFloat(latStr)
	if err != nil {
		return nil, err
	}
	poi.Lng, err = parseFloat(lngStr)
	if err != nil {
		return nil, err
	}

	ratingStr := getField("rating")
	if ratingStr != "" {
		poi.Rating, _ = parseFloat(ratingStr)
	}

	tagsStr := getField("tags")
	if tagsStr != "" {
		poi.Tags = strings.Split(tagsStr, "|")
	}

	statusStr := getField("status")
	if statusStr == "" {
		poi.Status = model.StatusActive
	} else {
		poi.Status = model.POIStatus(statusStr)
	}

	sourceStr := getField("source")
	if sourceStr != "" {
		poi.Source = model.POISource(sourceStr)
	}

	bhStr := getField("business_hours")
	if bhStr != "" {
		poi.BusinessHours = parseBusinessHours(bhStr)
	}

	return poi, nil
}

func parseFloat(s string) (float64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, nil
	}
	s = strings.ReplaceAll(s, "\"", "")
	return strconv.ParseFloat(s, 64)
}

func parseBusinessHours(s string) model.BusinessHours {
	var bh model.BusinessHours
	days := strings.Split(s, ";")
	for _, day := range days {
		parts := strings.SplitN(strings.TrimSpace(day), ":", 2)
		if len(parts) != 2 {
			continue
		}
		dayName := strings.TrimSpace(parts[0])
		periodsStr := strings.TrimSpace(parts[1])

		var periods []model.BusinessHoursPeriod
		for _, p := range strings.Split(periodsStr, ",") {
			times := strings.Split(strings.TrimSpace(p), "-")
			if len(times) == 2 {
				periods = append(periods, model.BusinessHoursPeriod{
					Open:  strings.TrimSpace(times[0]),
					Close: strings.TrimSpace(times[1]),
				})
			}
		}

		switch strings.ToLower(dayName) {
		case "周一", "monday", "mon":
			bh.Monday = periods
		case "周二", "tuesday", "tue":
			bh.Tuesday = periods
		case "周三", "wednesday", "wed":
			bh.Wednesday = periods
		case "周四", "thursday", "thu":
			bh.Thursday = periods
		case "周五", "friday", "fri":
			bh.Friday = periods
		case "周六", "saturday", "sat":
			bh.Saturday = periods
		case "周日", "sunday", "sun":
			bh.Sunday = periods
		}
	}
	return bh
}

func findExistingPOI(poi *model.POI, s *store.Store) *model.POI {
	allPOIs := s.GetAllPOIs()
	for _, p := range allPOIs {
		nameMatch := (poi.Name.ZhCN != "" && p.Name.ZhCN == poi.Name.ZhCN) ||
			(poi.Name.EnUS != "" && strings.EqualFold(p.Name.EnUS, poi.Name.EnUS))

		if nameMatch && p.Lat == poi.Lat && p.Lng == poi.Lng {
			return p
		}
	}
	return nil
}

var (
	counterMu sync.Mutex
	counter   uint64
)

func generateJobID() string {
	return "job_" + time.Now().Format("20060102150405") + "_" + randomString(6)
}

func generatePOIID() string {
	return "poi_" + time.Now().Format("20060102150405.000000000") + "_" + randomString(8)
}

func randomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	counterMu.Lock()
	counter++
	seed := uint64(time.Now().UnixNano()) ^ counter
	counterMu.Unlock()
	for i := range b {
		seed = seed*6364136223846793005 + 1442695040888963407
		b[i] = letters[seed%uint64(len(letters))]
	}
	return string(b)
}
