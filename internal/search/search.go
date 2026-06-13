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
	POIId    string  `json:"poi_id"`
	Name     string  `json:"name"`
	Category string  `json:"category"`
	Address  string  `json:"address"`
	City     string  `json:"city"`
	Rating   float64 `json:"rating"`
	MatchType string `json:"match_type"`
}

type SearchQuery struct {
	Q         string
	City      string
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
	centerHash := geohash.Encode(q.Lat, q.Lng, precision)
	buckets := geohash.SurroundingBuckets(centerHash)

	seen := make(map[string]bool)
	var results []NearbyResult

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
				if !strings.HasPrefix(poi.Category.L1, q.Category) &&
					!strings.HasPrefix(poi.Category.L2, q.Category) &&
					!strings.HasPrefix(poi.Category.L3, q.Category) {
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

	for _, poi := range allPOIs {
		if poi.Status != model.StatusActive {
			continue
		}

		if q.City != "" && poi.City != q.City {
			continue
		}

		matchType := ""
		zhName := strings.ToLower(poi.Name.ZhCN)
		enName := strings.ToLower(poi.Name.EnUS)

		if strings.Contains(zhName, query) || zhName == query {
			matchType = "chinese"
		} else if strings.Contains(enName, query) || enName == query {
			matchType = "english"
		} else if utils.FuzzyMatch(zhName, query, maxEditDistance) {
			matchType = "fuzzy_chinese"
		} else if utils.FuzzyMatch(enName, query, maxEditDistance) {
			matchType = "fuzzy_english"
		} else if isPinyin && poi.PinyinName != "" {
			if strings.Contains(poi.PinyinName, query) || poi.PinyinName == query {
				matchType = "pinyin"
			} else if utils.FuzzyMatch(poi.PinyinName, query, maxEditDistance) {
				matchType = "fuzzy_pinyin"
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
			})

			catKey := poi.Category.L2
			if catKey == "" {
				catKey = poi.Category.L1
			}
			categoryCount[catKey]++
		}
	}

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
