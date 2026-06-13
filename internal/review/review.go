package review

import (
	"poi-service/internal/model"
	"poi-service/internal/store"
	"sync"
	"time"
)

type ReviewStatus string

const (
	ReviewPending  ReviewStatus = "pending"
	ReviewApproved ReviewStatus = "approved"
	ReviewRejected ReviewStatus = "rejected"
)

type Submission struct {
	ID            string              `json:"id"`
	POI           *model.POI          `json:"poi"`
	Submitter     string              `json:"submitter"`
	SubmitTime    time.Time           `json:"submit_time"`
	Status        ReviewStatus        `json:"status"`
	Reviewer      string              `json:"reviewer,omitempty"`
	ReviewTime    time.Time           `json:"review_time,omitempty"`
	RejectReason  string              `json:"reject_reason,omitempty"`
}

type ReviewStore struct {
	mu          sync.RWMutex
	submissions map[string]*Submission
	pendingList []string
}

var instance *ReviewStore
var once sync.Once

func GetReviewStore() *ReviewStore {
	once.Do(func() {
		instance = &ReviewStore{
			submissions: make(map[string]*Submission),
			pendingList: make([]string, 0),
		}
	})
	return instance
}

func (r *ReviewStore) SubmitPOI(poi *model.POI, submitter string) *Submission {
	r.mu.Lock()
	defer r.mu.Unlock()

	poi.Status = model.StatusPendingReview
	poi.CreatedAt = time.Now()
	poi.UpdatedAt = time.Now()
	poi.Source = model.SourceUserSubmit

	sub := &Submission{
		ID:         generateSubID(),
		POI:        poi,
		Submitter:  submitter,
		SubmitTime: time.Now(),
		Status:     ReviewPending,
	}

	r.submissions[sub.ID] = sub
	r.pendingList = append(r.pendingList, sub.ID)

	return sub
}

func (r *ReviewStore) Approve(subID, reviewer string) (*model.POI, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	sub, ok := r.submissions[subID]
	if !ok || sub.Status != ReviewPending {
		return nil, false
	}

	sub.Status = ReviewApproved
	sub.Reviewer = reviewer
	sub.ReviewTime = time.Now()
	sub.POI.Status = model.StatusActive
	sub.POI.UpdatedAt = time.Now()

	store.GetStore().AddPOI(sub.POI)

	r.removeFromPending(subID)

	return sub.POI, true
}

func (r *ReviewStore) Reject(subID, reviewer, reason string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	sub, ok := r.submissions[subID]
	if !ok || sub.Status != ReviewPending {
		return false
	}

	sub.Status = ReviewRejected
	sub.Reviewer = reviewer
	sub.ReviewTime = time.Now()
	sub.RejectReason = reason
	sub.POI.Status = model.StatusClosed

	r.removeFromPending(subID)

	return true
}

func (r *ReviewStore) GetPending(limit, offset int) ([]*Submission, int) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	total := len(r.pendingList)
	start := offset
	end := offset + limit
	if start > total {
		return []*Submission{}, total
	}
	if end > total {
		end = total
	}

	result := make([]*Submission, 0, end-start)
	for i := start; i < end; i++ {
		if sub, ok := r.submissions[r.pendingList[i]]; ok {
			result = append(result, sub)
		}
	}

	return result, total
}

func (r *ReviewStore) GetByID(subID string) (*Submission, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	sub, ok := r.submissions[subID]
	return sub, ok
}

func (r *ReviewStore) GetBySubmitter(submitter string, limit, offset int) ([]*Submission, int) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var filtered []*Submission
	for _, sub := range r.submissions {
		if sub.Submitter == submitter {
			filtered = append(filtered, sub)
		}
	}

	total := len(filtered)
	start := offset
	end := offset + limit
	if start > total {
		return []*Submission{}, total
	}
	if end > total {
		end = total
	}

	return filtered[start:end], total
}

func (r *ReviewStore) removeFromPending(subID string) {
	for i, id := range r.pendingList {
		if id == subID {
			r.pendingList = append(r.pendingList[:i], r.pendingList[i+1:]...)
			break
		}
	}
}

var (
	reviewCounterMu sync.Mutex
	reviewCounter   uint64
)

func generateSubID() string {
	return "sub_" + time.Now().Format("20060102150405.000000000") + "_" + randomString(6)
}

func randomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	reviewCounterMu.Lock()
	reviewCounter++
	seed := uint64(time.Now().UnixNano()) ^ reviewCounter
	reviewCounterMu.Unlock()
	for i := range b {
		seed = seed*6364136223846793005 + 1442695040888963407
		b[i] = letters[seed%uint64(len(letters))]
	}
	return string(b)
}
