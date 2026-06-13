package search

import (
	"poi-service/internal/geohash"
	"poi-service/internal/model"
	"poi-service/internal/store"
	"poi-service/internal/utils"
	"sort"
	"strings"
)

type NearbyResult struct {
	POIId           string  `json:"poi_id"`
	Name            string  `json:"name"`
	Category        string  `json:"category"`
	Address         string  `json:"address"`
	Lat             float64 `json:"lat"`
	Lng             float64 `json:"lng"`
	Rating          float64 `json:"rating"`
	DistanceM       float64 `json:"distance_m"`
	WalkingDistance float64 `json:"walking_distance"`
	WalkingDuration int     `json:"walking_duration"`
}

type NearbyQuery struct {
	Lat      float64
	Lng      float64
	Radius   float64
	Category string
	Limit    int
}

type SearchResult struct {
	POIId     string  `json:"poi_id"`
	Name      string  `json:"name"`
	Category  string  `json:"category"`
	Address   string  `json:"address"`
	City      string  `json:"city"`
	Rating    float64 `json:"rating"`
	MatchType string  `json:"match_type"`
	Score     int     `json:"score"`
}

type SearchQuery struct {
	Q         string
	City      string
	Category  string
	Page      int
	PageSize  int
}

type CategoryAggregation struct {
	Category string `json:"category"`
	Count    int    `json:"count"`
}

type SearchResponse struct {
	Total       int                     `json:"total"`
	Page        int                     `json:"page"`
	PageSize    int                     `json:"page_size"`
	Results     []SearchResult          `json:"results"`
	Aggregations []CategoryAggregation  `json:"aggregations"`
}

const maxEditDistance = 1

func NearbySearch(q *NearbyQuery) []NearbyResult {
	precision := geohash.DynamicPrecision(q.Radius)
	if precision < 3 {
		precision = 3
	}
	centerHash := geohash.Encode(q.Lat, q.Lng, precision)
	buckets := geohash.SurroundingBuckets(centerHash)

	seen := make(map[string]bool)
	var results []NearbyResult

	extraRadius := q.Radius * 1.2
	extraPrecision := geohash.DynamicPrecision(extraRadius)
	if extraPrecision < precision && extraPrecision >= 3 {
		extraHash := geohash.Encode(q.Lat, q.Lng, extraPrecision)
		extraBuckets := geohash.SurroundingBuckets(extraHash)
		seenBuckets := make(map[string]bool)
		for _, b := range buckets {
			seenBuckets[b] = true
		}
		for _, b := range extraBuckets {
			if !seenBuckets[b] {
				buckets = append(buckets, b)
				seenBuckets[b] = true
			}
		}
	}

	for _, bucket := range buckets {
		pois := store.GetStore().GetPOIsByGeohash(bucket)
		for _, poi := range pois {
			if poi.Status != model.StatusActive {
				continue
			}
			if seen[poi.POIId] {
				continue
			}

			if q.Category != "" {
				if !matchCategory(poi.Category, q.Category) {
					continue
				}
			}

			dist := utils.Haversine(q.Lat, q.Lng, poi.Lat, poi.Lng)
			if dist <= q.Radius {
				seen[poi.POIId] = true
				walkingDist := utils.WalkingDistance(dist)
				walkingDur := utils.WalkingDuration(walkingDist)

				results = append(results, NearbyResult{
					POIId:           poi.POIId,
					Name:            poi.Name.ZhCN,
					Category:        poi.Category.L1 + "/" + poi.Category.L2,
					Address:         poi.Address,
					Lat:             poi.Lat,
					Lng:             poi.Lng,
					Rating:          poi.Rating,
					DistanceM:       dist,
					WalkingDistance: walkingDist,
					WalkingDuration: walkingDur,
				})
			}
		}
	}

	sort.Slice(results, func(i, j int) bool {
		if results[i].DistanceM == results[j].DistanceM {
			return results[i].POIId < results[j].POIId
		}
		return results[i].DistanceM < results[j].DistanceM
	})

	limit := q.Limit
	if limit <= 0 || limit > len(results) {
		limit = len(results)
	}

	return results[:limit]
}

func matchCategory(cat model.Category, query string) bool {
	query = strings.ToLower(strings.TrimSpace(query))

	l1 := strings.ToLower(cat.L1)
	l2 := strings.ToLower(cat.L2)
	l3 := strings.ToLower(cat.L3)
	full := l1 + "/" + l2 + "/" + l3

	if l1 == query || l2 == query || l3 == query {
		return true
	}

	if strings.Contains(l1, query) || strings.Contains(l2, query) || strings.Contains(l3, query) {
		return true
	}

	if strings.HasPrefix(full, query) || strings.HasPrefix(l1, query) ||
		strings.HasPrefix(l2, query) || strings.HasPrefix(l3, query) {
		return true
	}

	return false
}

func KeywordSearch(q *SearchQuery) *SearchResponse {
	query := strings.ToLower(strings.TrimSpace(q.Q))
	if query == "" {
		return &SearchResponse{
			Total:   0,
			Page:    q.Page,
			PageSize: q.PageSize,
			Results: []SearchResult{},
		}
	}

	isPinyin := utils.IsPinyin(query)
	allPOIs := store.GetStore().GetAllPOIs()

	var matches []SearchResult
	categoryCount := make(map[string]int)
	categoryFilter := strings.ToLower(strings.TrimSpace(q.Category))

	for _, poi := range allPOIs {
		if poi.Status != model.StatusActive {
			continue
		}

		if q.City != "" && !strings.Contains(strings.ToLower(poi.City), strings.ToLower(q.City)) {
			continue
		}

		if categoryFilter != "" && !matchCategory(poi.Category, categoryFilter) {
			continue
		}

		matchType := ""
		matchScore := 0
		zhName := strings.ToLower(poi.Name.ZhCN)
		enName := strings.ToLower(poi.Name.EnUS)
		address := strings.ToLower(poi.Address)
		tags := strings.ToLower(strings.Join(poi.Tags, ","))

		if zhName == query {
			matchType = "chinese_exact"
			matchScore = 100
		} else if strings.HasPrefix(zhName, query) {
			matchType = "chinese_prefix"
			matchScore = 80
		} else if strings.Contains(zhName, query) {
			matchType = "chinese"
			matchScore = 60
		} else if strings.Contains(address, query) {
			matchType = "address"
			matchScore = 40
		} else if strings.Contains(tags, query) {
			matchType = "tag"
			matchScore = 35
		} else if enName == query || strings.HasPrefix(enName, query) || strings.Contains(enName, query) {
			matchType = "english"
			matchScore = 50
		} else if utils.FuzzyMatch(zhName, query, maxEditDistance) {
			matchType = "fuzzy_chinese"
			matchScore = 25
		} else if utils.FuzzyMatch(enName, query, maxEditDistance) {
			matchType = "fuzzy_english"
			matchScore = 20
		} else if isPinyin && poi.PinyinName != "" {
			if poi.PinyinName == query || strings.HasPrefix(poi.PinyinName, query) || strings.Contains(poi.PinyinName, query) {
				matchType = "pinyin"
				matchScore = 45
			} else if utils.FuzzyMatch(poi.PinyinName, query, maxEditDistance) {
				matchType = "fuzzy_pinyin"
				matchScore = 15
			}
		}

		if matchType != "" {
			matches = append(matches, SearchResult{
				POIId:     poi.POIId,
				Name:      poi.Name.ZhCN,
				Category:  poi.Category.L1 + "/" + poi.Category.L2,
				Address:   poi.Address,
				City:      poi.City,
				Rating:    poi.Rating,
				MatchType: matchType,
				Score:     matchScore,
			})

			catKey := poi.Category.L2
			if catKey == "" {
				catKey = poi.Category.L1
			}
			categoryCount[catKey]++
		}
	}

	sort.Slice(matches, func(i, j int) bool {
		if matches[i].Score != matches[j].Score {
			return matches[i].Score > matches[j].Score
		}
		if matches[i].Rating != matches[j].Rating {
			return matches[i].Rating > matches[j].Rating
		}
		return matches[i].POIId < matches[j].POIId
	})

	total := len(matches)
	start := (q.Page - 1) * q.PageSize
	end := start + q.PageSize

	if start > total {
		start = total
	}
	if end > total {
		end = total
	}

	var aggregations []CategoryAggregation
	for cat, count := range categoryCount {
		mainCat := getMainCategory(q.Q)
		if !strings.Contains(cat, mainCat) {
			aggregations = append(aggregations, CategoryAggregation{
				Category: cat,
				Count:    count,
			})
		}
	}

	sort.Slice(aggregations, func(i, j int) bool {
		return aggregations[i].Count > aggregations[j].Count
	})

	return &SearchResponse{
		Total:        total,
		Page:         q.Page,
		PageSize:     q.PageSize,
		Results:      matches[start:end],
		Aggregations: aggregations,
	}
}

func getMainCategory(query string) string {
	query = strings.ToLower(query)
	switch {
	case strings.Contains(query, "咖啡") || strings.Contains(query, "coffee"):
		return "咖啡店"
	case strings.Contains(query, "餐厅") || strings.Contains(query, "restaurant"):
		return "中餐"
	case strings.Contains(query, "酒店") || strings.Contains(query, "hotel"):
		return "酒店"
	case strings.Contains(query, "购物") || strings.Contains(query, "shopping"):
		return "购物中心"
	default:
		return ""
	}
}
