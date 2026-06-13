package stats

import (
	"poi-service/internal/model"
	"poi-service/internal/store"
)

type CategoryStats struct {
	L1Category string `json:"l1_category"`
	Count      int    `json:"count"`
	Percentage float64 `json:"percentage"`
}

type CityStats struct {
	City       string  `json:"city"`
	Count      int     `json:"count"`
	Density    float64 `json:"density_per_sqkm"`
	AreaSqKm   float64 `json:"area_sqkm"`
}

type SourceStats struct {
	Source     model.POISource `json:"source"`
	SourceName string          `json:"source_name"`
	Count      int             `json:"count"`
	Percentage float64         `json:"percentage"`
}

type OverallStats struct {
	TotalPOIs   int              `json:"total_pois"`
	Categories  []CategoryStats  `json:"categories"`
	Cities      []CityStats      `json:"cities"`
	Sources     []SourceStats    `json:"sources"`
}

var cityAreas = map[string]float64{
	"北京": 16410.54,
	"上海": 6340.50,
	"广州": 7434.40,
	"深圳": 1997.47,
	"杭州": 16853.57,
	"成都": 14335.00,
}

func GetOverallStats() *OverallStats {
	s := store.GetStore()
	total := s.Size()

	categoryStats := getCategoryStats(total)
	cityStats := getCityStats(total)
	sourceStats := getSourceStats(total)

	return &OverallStats{
		TotalPOIs:  total,
		Categories: categoryStats,
		Cities:     cityStats,
		Sources:    sourceStats,
	}
}

func getCategoryStats(total int) []CategoryStats {
	counts := store.GetStore().CountPOIsByCategory()
	var result []CategoryStats

	for l1, count := range counts {
		percentage := 0.0
		if total > 0 {
			percentage = float64(count) / float64(total) * 100
		}
		result = append(result, CategoryStats{
			L1Category: l1,
			Count:      count,
			Percentage: percentage,
		})
	}

	return result
}

func getCityStats(total int) []CityStats {
	counts := store.GetStore().CountPOIsByCity()
	var result []CityStats

	for city, count := range counts {
		area := cityAreas[city]
		if area == 0 {
			area = 1000.0
		}
		density := float64(count) / area

		result = append(result, CityStats{
			City:     city,
			Count:    count,
			Density:  density,
			AreaSqKm: area,
		})
	}

	return result
}

func getSourceStats(total int) []SourceStats {
	counts := store.GetStore().CountPOIsBySource()
	var result []SourceStats

	sourceNames := map[model.POISource]string{
		model.SourceManual:     "手工录入",
		model.SourceCrawler:    "爬虫抓取",
		model.SourcePartner:    "合作方提供",
		model.SourceUserSubmit: "用户提交",
	}

	for source, count := range counts {
		percentage := 0.0
		if total > 0 {
			percentage = float64(count) / float64(total) * 100
		}
		name := sourceNames[source]
		if name == "" {
			name = string(source)
		}
		result = append(result, SourceStats{
			Source:     source,
			SourceName: name,
			Count:      count,
			Percentage: percentage,
		})
	}

	return result
}
