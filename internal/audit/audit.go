package audit

import (
	"sync"
	"time"
)

type AuditAction string

const (
	ActionCreate   AuditAction = "create"
	ActionUpdate   AuditAction = "update"
	ActionDelete   AuditAction = "delete"
	ActionApprove  AuditAction = "approve"
	ActionReject   AuditAction = "reject"
	ActionImport   AuditAction = "import"
)

type FieldChange struct {
	Field    string      `json:"field"`
	OldValue interface{} `json:"old_value"`
	NewValue interface{} `json:"new_value"`
}

type AuditLog struct {
	ID         string         `json:"id"`
	POIId      string         `json:"poi_id"`
	Operator   string         `json:"operator"`
	Action     AuditAction    `json:"action"`
	Changes    []FieldChange  `json:"changes"`
	Timestamp  time.Time      `json:"timestamp"`
	IPAddress  string         `json:"ip_address"`
	UserAgent  string         `json:"user_agent"`
}

type AuditStore struct {
	mu   sync.RWMutex
	logs []*AuditLog
}

var instance *AuditStore
var once sync.Once

func GetAuditStore() *AuditStore {
	once.Do(func() {
		instance = &AuditStore{
			logs: make([]*AuditLog, 0),
		}
	})
	return instance
}

func (a *AuditStore) LogCreate(poiId, operator, ip, ua string, newPOI interface{}) {
	a.addLog(&AuditLog{
		POIId:     poiId,
		Operator:  operator,
		Action:    ActionCreate,
		Changes: []FieldChange{{
			Field:    "poi",
			OldValue: nil,
			NewValue: newPOI,
		}},
		Timestamp: time.Now(),
		IPAddress: ip,
		UserAgent: ua,
	})
}

func (a *AuditStore) LogUpdate(poiId, operator, ip, ua string, changes []FieldChange) {
	if len(changes) == 0 {
		return
	}
	a.addLog(&AuditLog{
		POIId:     poiId,
		Operator:  operator,
		Action:    ActionUpdate,
		Changes:   changes,
		Timestamp: time.Now(),
		IPAddress: ip,
		UserAgent: ua,
	})
}

func (a *AuditStore) LogDelete(poiId, operator, ip, ua string, oldPOI interface{}) {
	a.addLog(&AuditLog{
		POIId:     poiId,
		Operator:  operator,
		Action:    ActionDelete,
		Changes: []FieldChange{{
			Field:    "poi",
			OldValue: oldPOI,
			NewValue: nil,
		}},
		Timestamp: time.Now(),
		IPAddress: ip,
		UserAgent: ua,
	})
}

func (a *AuditStore) LogApprove(poiId, operator, ip, ua string) {
	a.addLog(&AuditLog{
		POIId:     poiId,
		Operator:  operator,
		Action:    ActionApprove,
		Timestamp: time.Now(),
		IPAddress: ip,
		UserAgent: ua,
	})
}

func (a *AuditStore) LogReject(poiId, operator, ip, ua string, reason string) {
	a.addLog(&AuditLog{
		POIId:     poiId,
		Operator:  operator,
		Action:    ActionReject,
		Changes: []FieldChange{{
			Field:    "reason",
			OldValue: nil,
			NewValue: reason,
		}},
		Timestamp: time.Now(),
		IPAddress: ip,
		UserAgent: ua,
	})
}

func (a *AuditStore) LogImport(poiId, operator, ip, ua string, isUpdate bool) {
	action := ActionImport
	changes := []FieldChange{{
		Field:    "import_type",
		OldValue: nil,
		NewValue: map[string]bool{"is_update": isUpdate},
	}}
	a.addLog(&AuditLog{
		POIId:     poiId,
		Operator:  operator,
		Action:    action,
		Changes:   changes,
		Timestamp: time.Now(),
		IPAddress: ip,
		UserAgent: ua,
	})
}

func (a *AuditStore) addLog(log *AuditLog) {
	a.mu.Lock()
	defer a.mu.Unlock()
	log.ID = generateID()
	a.logs = append(a.logs, log)
}

func (a *AuditStore) GetLogs(poiId string, limit, offset int) ([]*AuditLog, int) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	var filtered []*AuditLog
	if poiId != "" {
		for _, log := range a.logs {
			if log.POIId == poiId {
				filtered = append(filtered, log)
			}
		}
	} else {
		filtered = make([]*AuditLog, len(a.logs))
		copy(filtered, a.logs)
	}

	total := len(filtered)
	start := offset
	end := offset + limit
	if start > total {
		return []*AuditLog{}, total
	}
	if end > total {
		end = total
	}

	return filtered[start:end], total
}

var (
	auditCounterMu sync.Mutex
	auditCounter   uint64
)

func generateID() string {
	return "audit_" + time.Now().Format("20060102150405.000000000") + "_" + randomString(6)
}

func randomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	auditCounterMu.Lock()
	auditCounter++
	seed := uint64(time.Now().UnixNano()) ^ auditCounter
	auditCounterMu.Unlock()
	for i := range b {
		seed = seed*6364136223846793005 + 1442695040888963407
		b[i] = letters[seed%uint64(len(letters))]
	}
	return string(b)
}
