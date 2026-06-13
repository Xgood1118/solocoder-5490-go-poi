package poi

import (
	"poi-service/internal/audit"
	"poi-service/internal/geohash"
	"poi-service/internal/model"
	"poi-service/internal/store"
	"poi-service/internal/utils"
	"sort"
	"strings"
	"sync"
	"time"
)

type POIDetailResponse struct {
	*model.POI
	RealTimeStatus   utils.BusinessStatus `json:"real_time_status"`
	NearbyRecommend  []NearbyPOI          `json:"nearby_recommend"`
	Reviews          []model.Review       `json:"reviews"`
	AdjacentPOIs     []AdjacentPOI        `json:"adjacent_pois"`
}

type NearbyPOI struct {
	POIId      string  `json:"poi_id"`
	Name       string  `json:"name"`
	Category   string  `json:"category"`
	DistanceM  float64 `json:"distance_m"`
	Rating     float64 `json:"rating"`
	Score      float64 `json:"-"`
}

type AdjacentPOI struct {
	POIId    string `json:"poi_id"`
	Name     string `json:"name"`
	Category string `json:"category"`
}

type DuplicateCheckResult struct {
	IsDuplicate bool         `json:"is_duplicate"`
	Duplicates  []Duplicate  `json:"duplicates,omitempty"`
}

type Duplicate struct {
	POIId     string  `json:"poi_id"`
	Name      string  `json:"name"`
	DistanceM float64 `json:"distance_m"`
}

type QualityIssue struct {
	POIId   string `json:"poi_id"`
	Name    string `json:"name"`
	Issue   string `json:"issue"`
	Severity string `json:"severity"`
}

var cityBounds = map[string][4]float64{
	"北京": {39.4, 115.4, 41.1, 117.5},
	"上海": {30.7, 120.9, 31.9, 122.1},
	"广州": {22.9, 112.9, 23.9, 114.1},
	"深圳": {22.4, 113.7, 22.9, 114.7},
}

func GetPOIDetail(id string) (*POIDetailResponse, bool) {
	poi, ok := store.GetStore().GetPOIByID(id)
	if !ok {
		return nil, false
	}

	now := time.Now()
	status := utils.GetBusinessStatus(poi.BusinessHours, now)

	nearby := getNearbyRecommend(poi)
	reviews := getMockReviews(id)
	adjacent := getAdjacentPOIs(poi)

	return &POIDetailResponse{
		POI:             poi,
		RealTimeStatus:  status,
		NearbyRecommend: nearby,
		Reviews:         reviews,
		AdjacentPOIs:    adjacent,
	}, true
}

func getNearbyRecommend(poi *model.POI) []NearbyPOI {
	radius := 2000.0
	precision := geohash.DynamicPrecision(radius)
	centerHash := geohash.Encode(poi.Lat, poi.Lng, precision)
	buckets := geohash.SurroundingBuckets(centerHash)

	seen := make(map[string]bool)
	var candidates []NearbyPOI

	for _, bucket := range buckets {
		pois := store.GetStore().GetPOIsByGeohash(bucket)
		for _, p := range pois {
			if p.POIId == poi.POIId || p.Status != model.StatusActive {
				continue
			}
			if seen[p.POIId] {
				continue
			}
			dist := utils.Haversine(poi.Lat, poi.Lng, p.Lat, p.Lng)
			if dist <= radius {
				seen[p.POIId] = true
				score := p.Rating*100 - dist/100
				candidates = append(candidates, NearbyPOI{
					POIId:     p.POIId,
					Name:      p.Name.ZhCN,
					Category:  p.Category.L1 + "/" + p.Category.L2,
					DistanceM: dist,
					Rating:    p.Rating,
					Score:     score,
				})
			}
		}
	}

	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].Score > candidates[j].Score
	})

	if len(candidates) > 10 {
		candidates = candidates[:10]
	}

	return candidates
}

func getMockReviews(poiId string) []model.Review {
	return []model.Review{
		{
			Id:        "rev_1",
			POIId:     poiId,
			User:      "美食达人",
			Rating:    4.5,
			Content:   "味道不错，环境也很好，值得推荐！",
			CreatedAt: time.Now().AddDate(0, 0, -3),
		},
		{
			Id:        "rev_2",
			POIId:     poiId,
			User:      "旅行者小王",
			Rating:    5.0,
			Content:   "服务态度很好，地理位置也很方便。",
			CreatedAt: time.Now().AddDate(0, 0, -7),
		},
		{
			Id:        "rev_3",
			POIId:     poiId,
			User:      "本地土著",
			Rating:    4.0,
			Content:   "价格有点小贵，但是品质确实不错。",
			CreatedAt: time.Now().AddDate(0, 0, -14),
		},
	}
}

func getAdjacentPOIs(poi *model.POI) []AdjacentPOI {
	if poi.Address == "" {
		return nil
	}

	pois := store.GetStore().GetPOIsByAddress(poi.Address)
	var result []AdjacentPOI

	for _, p := range pois {
		if p.POIId == poi.POIId {
			continue
		}
		result = append(result, AdjacentPOI{
			POIId:    p.POIId,
			Name:     p.Name.ZhCN,
			Category: p.Category.L1 + "/" + p.Category.L2,
		})
	}

	return result
}

func CreatePOI(poi *model.POI, operator, ip, ua string) (*model.POI, error) {
	poi.POIId = generatePOIID()
	poi.CreatedAt = time.Now()
	poi.UpdatedAt = time.Now()

	store.GetStore().AddPOI(poi)
	audit.GetAuditStore().LogCreate(poi.POIId, operator, ip, ua, poi)

	return poi, nil
}

func UpdatePOI(id string, updates map[string]interface{}, operator, ip, ua string) (*model.POI, bool, string) {
	poi, ok := store.GetStore().GetPOIByID(id)
	if !ok {
		return nil, false, "POI not found"
	}

	expectedVersion := int64(-1)
	if v, ok := updates["expected_version"].(float64); ok {
		expectedVersion = int64(v)
	} else if v, ok := updates["version"].(float64); ok {
		expectedVersion = int64(v)
	}

	var changes []audit.FieldChange
	oldPOI := *poi
	updatedPOI := *poi

	if name, ok := updates["name"].(map[string]interface{}); ok {
		oldName := updatedPOI.Name
		if zh, ok := name["zh_cn"].(string); ok {
			updatedPOI.Name.ZhCN = zh
		}
		if en, ok := name["en_us"].(string); ok {
			updatedPOI.Name.EnUS = en
		}
		if ja, ok := name["ja_jp"].(string); ok {
			updatedPOI.Name.JaJP = ja
		}
		if ko, ok := name["ko_kr"].(string); ok {
			updatedPOI.Name.KoKR = ko
		}
		changes = append(changes, audit.FieldChange{
			Field:    "name",
			OldValue: oldName,
			NewValue: updatedPOI.Name,
		})
	}

	if category, ok := updates["category"].(map[string]interface{}); ok {
		oldCat := updatedPOI.Category
		if l1, ok := category["l1"].(string); ok {
			updatedPOI.Category.L1 = l1
		}
		if l2, ok := category["l2"].(string); ok {
			updatedPOI.Category.L2 = l2
		}
		if l3, ok := category["l3"].(string); ok {
			updatedPOI.Category.L3 = l3
		}
		changes = append(changes, audit.FieldChange{
			Field:    "category",
			OldValue: oldCat,
			NewValue: updatedPOI.Category,
		})
	}

	if address, ok := updates["address"].(string); ok {
		changes = append(changes, audit.FieldChange{
			Field:    "address",
			OldValue: updatedPOI.Address,
			NewValue: address,
		})
		updatedPOI.Address = address
	}

	if city, ok := updates["city"].(string); ok {
		changes = append(changes, audit.FieldChange{
			Field:    "city",
			OldValue: updatedPOI.City,
			NewValue: city,
		})
		updatedPOI.City = city
	}

	if district, ok := updates["district"].(string); ok {
		changes = append(changes, audit.FieldChange{
			Field:    "district",
			OldValue: updatedPOI.District,
			NewValue: district,
		})
		updatedPOI.District = district
	}

	if lat, ok := updates["lat"].(float64); ok {
		changes = append(changes, audit.FieldChange{
			Field:    "lat",
			OldValue: updatedPOI.Lat,
			NewValue: lat,
		})
		updatedPOI.Lat = lat
	}

	if lng, ok := updates["lng"].(float64); ok {
		changes = append(changes, audit.FieldChange{
			Field:    "lng",
			OldValue: updatedPOI.Lng,
			NewValue: lng,
		})
		updatedPOI.Lng = lng
	}

	if phone, ok := updates["phone"].(string); ok {
		changes = append(changes, audit.FieldChange{
			Field:    "phone",
			OldValue: updatedPOI.Phone,
			NewValue: phone,
		})
		updatedPOI.Phone = phone
	}

	if bh, ok := updates["business_hours"].(model.BusinessHours); ok {
		changes = append(changes, audit.FieldChange{
			Field:    "business_hours",
			OldValue: oldPOI.BusinessHours,
			NewValue: bh,
		})
		updatedPOI.BusinessHours = bh
	}

	if rating, ok := updates["rating"].(float64); ok {
		changes = append(changes, audit.FieldChange{
			Field:    "rating",
			OldValue: updatedPOI.Rating,
			NewValue: rating,
		})
		updatedPOI.Rating = rating
	}

	if tags, ok := updates["tags"].([]string); ok {
		changes = append(changes, audit.FieldChange{
			Field:    "tags",
			OldValue: updatedPOI.Tags,
			NewValue: tags,
		})
		updatedPOI.Tags = tags
	}

	if status, ok := updates["status"].(model.POIStatus); ok {
		changes = append(changes, audit.FieldChange{
			Field:    "status",
			OldValue: updatedPOI.Status,
			NewValue: status,
		})
		updatedPOI.Status = status
	}

	updatedPOI.UpdatedAt = time.Now()

	result, ok, msg := store.GetStore().UpdatePOIWithVersion(&updatedPOI, expectedVersion)
	if !ok {
		return result, false, msg
	}

	audit.GetAuditStore().LogUpdate(updatedPOI.POIId, operator, ip, ua, changes)

	return result, true, ""
}

func DeletePOI(id, operator, ip, ua string) bool {
	poi, ok := store.GetStore().GetPOIByID(id)
	if !ok {
		return false
	}

	oldPOI := *poi
	success := store.GetStore().DeletePOI(id)
	if success {
		audit.GetAuditStore().LogDelete(id, operator, ip, ua, oldPOI)
	}
	return success
}

func CheckDuplicates(poi *model.POI) *DuplicateCheckResult {
	allPOIs := store.GetStore().GetAllPOIs()
	var duplicates []Duplicate

	for _, p := range allPOIs {
		if p.POIId == poi.POIId {
			continue
		}

		nameMatch := strings.EqualFold(p.Name.ZhCN, poi.Name.ZhCN) ||
			strings.EqualFold(p.Name.EnUS, poi.Name.EnUS)

		if !nameMatch {
			continue
		}

		dist := utils.Haversine(poi.Lat, poi.Lng, p.Lat, p.Lng)
		if dist < 50 {
			duplicates = append(duplicates, Duplicate{
				POIId:     p.POIId,
				Name:      p.Name.ZhCN,
				DistanceM: dist,
			})
		}
	}

	return &DuplicateCheckResult{
		IsDuplicate: len(duplicates) > 0,
		Duplicates:  duplicates,
	}
}

func CheckDataQuality() []QualityIssue {
	allPOIs := store.GetStore().GetAllPOIs()
	var issues []QualityIssue

	for _, poi := range allPOIs {
		if poi.Address == "" {
			issues = append(issues, QualityIssue{
				POIId:    poi.POIId,
				Name:     poi.Name.ZhCN,
				Issue:    "缺少详细地址",
				Severity: "high",
			})
		}

		if poi.Lat == 0 || poi.Lng == 0 {
			issues = append(issues, QualityIssue{
				POIId:    poi.POIId,
				Name:     poi.Name.ZhCN,
				Issue:    "缺少坐标信息",
				Severity: "high",
			})
		}

		if bounds, ok := cityBounds[poi.City]; ok && poi.Lat != 0 && poi.Lng != 0 {
			minLat, minLng, maxLat, maxLng := bounds[0], bounds[1], bounds[2], bounds[3]
			if poi.Lat < minLat || poi.Lat > maxLat || poi.Lng < minLng || poi.Lng > maxLng {
				issues = append(issues, QualityIssue{
					POIId:    poi.POIId,
					Name:     poi.Name.ZhCN,
					Issue:    "坐标超出城市范围",
					Severity: "high",
				})
			}
		}

		if poi.City == "" {
			issues = append(issues, QualityIssue{
				POIId:    poi.POIId,
				Name:     poi.Name.ZhCN,
				Issue:    "缺少城市信息",
				Severity: "medium",
			})
		}

		if poi.Phone == "" {
			issues = append(issues, QualityIssue{
				POIId:    poi.POIId,
				Name:     poi.Name.ZhCN,
				Issue:    "缺少联系电话",
				Severity: "low",
			})
		}
	}

	return issues
}

var (
	poiCounterMu sync.Mutex
	poiCounter   uint64
)

func generatePOIID() string {
	return "poi_" + time.Now().Format("20060102150405.000000000") + "_" + randomString(8)
}

func randomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	poiCounterMu.Lock()
	poiCounter++
	seed := uint64(time.Now().UnixNano()) ^ poiCounter
	poiCounterMu.Unlock()
	for i := range b {
		seed = seed*6364136223846793005 + 1442695040888963407
		b[i] = letters[seed%uint64(len(letters))]
	}
	return string(b)
}
