package importer

import (
	"encoding/json"
	"fmt"
	"os"
	"poi-service/internal/audit"
	"poi-service/internal/model"
	"poi-service/internal/store"
	"strings"
	"time"
)

type GeoJSONFeatureCollection struct {
	Type     string           `json:"type"`
	Features []GeoJSONFeature `json:"features"`
}

type GeoJSONFeature struct {
	Type       string                 `json:"type"`
	Geometry   GeoJSONGeometry        `json:"geometry"`
	Properties map[string]interface{} `json:"properties"`
}

type GeoJSONGeometry struct {
	Type        string      `json:"type"`
	Coordinates interface{} `json:"coordinates"`
}

func ImportGeoJSONFile(filePath, operator, ip, ua string) (*ImportProgress, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var fc GeoJSONFeatureCollection
	if err := json.Unmarshal(data, &fc); err != nil {
		return nil, fmt.Errorf("failed to parse GeoJSON: %w", err)
	}

	if fc.Type != "FeatureCollection" {
		return nil, fmt.Errorf("invalid GeoJSON type: expected FeatureCollection, got %s", fc.Type)
	}

	progress := &ImportProgress{
		Total:  len(fc.Features),
		Status: StatusRunning,
	}

	s := store.GetStore()

	for idx, feature := range fc.Features {
		poi, parseErr := parseGeoJSONFeature(&feature)
		if parseErr != nil {
			progress.Errors++
			continue
		}

		existing := findExistingPOI(poi, s)
		isUpdate := existing != nil

		if isUpdate {
			poi.POIId = existing.POIId
			poi.CreatedAt = existing.CreatedAt
			poi.UpdatedAt = time.Now()
			progress.Updated++
		} else {
			poi.POIId = generatePOIID()
			poi.CreatedAt = time.Now()
			poi.UpdatedAt = time.Now()
			poi.Source = model.SourcePartner
			progress.Created++
		}

		s.AddPOI(poi)
		audit.GetAuditStore().LogImport(poi.POIId, operator, ip, ua, isUpdate)

		progress.Processed = idx + 1
	}

	progress.Status = StatusCompleted
	return progress, nil
}

func parseGeoJSONFeature(feature *GeoJSONFeature) (*model.POI, error) {
	poi := &model.POI{}

	lng, lat, err := extractCoordinates(feature.Geometry)
	if err != nil {
		return nil, err
	}
	poi.Lat = lat
	poi.Lng = lng

	props := feature.Properties

	getStr := func(key string) string {
		if v, ok := props[key]; ok {
			switch val := v.(type) {
			case string:
				return strings.TrimSpace(val)
			case float64:
				return fmt.Sprintf("%g", val)
			case bool:
				return fmt.Sprintf("%t", val)
			}
		}
		return ""
	}

	getFloat := func(key string) float64 {
		if v, ok := props[key]; ok {
			switch val := v.(type) {
			case float64:
				return val
			case string:
				if f, err := parseFloat(val); err == nil {
					return f
				}
			}
		}
		return 0
	}

	poi.Name.ZhCN = getFirstNonEmpty(
		getStr("name_zh"),
		getStr("name"),
		getStr("title"),
		getStr("Name"),
		getStr("NAME"),
	)
	poi.Name.EnUS = getFirstNonEmpty(
		getStr("name_en"),
		getStr("name:en"),
		getStr("english_name"),
	)
	poi.Name.JaJP = getFirstNonEmpty(
		getStr("name_ja"),
		getStr("name:ja"),
	)
	poi.Name.KoKR = getFirstNonEmpty(
		getStr("name_ko"),
		getStr("name:ko"),
	)

	if poi.Name.ZhCN == "" && poi.Name.EnUS == "" {
		poi.Name.ZhCN = fmt.Sprintf("POI_%s", generatePOIID())
	}

	poi.Category.L1 = getFirstNonEmpty(
		getStr("category_l1"),
		getStr("category"),
		getStr("type"),
		getStr("amenity"),
		getStr("shop"),
		getStr("tourism"),
	)
	poi.Category.L2 = getFirstNonEmpty(
		getStr("category_l2"),
		getStr("subcategory"),
		getStr("subtype"),
	)
	poi.Category.L3 = getStr("category_l3")

	poi.Address = getFirstNonEmpty(
		getStr("address"),
		getStr("addr:full"),
		getStr("street_address"),
	)
	poi.City = getFirstNonEmpty(
		getStr("city"),
		getStr("addr:city"),
	)
	poi.District = getFirstNonEmpty(
		getStr("district"),
		getStr("addr:district"),
		getStr("county"),
	)
	poi.Phone = getFirstNonEmpty(
		getStr("phone"),
		getStr("tel"),
		getStr("contact:phone"),
	)

	poi.Rating = getFloat("rating")

	tagsStr := getStr("tags")
	if tagsStr != "" {
		poi.Tags = strings.Split(tagsStr, "|")
	} else {
		if tagList, ok := props["tags"].([]interface{}); ok {
			for _, t := range tagList {
				if s, ok := t.(string); ok {
					poi.Tags = append(poi.Tags, s)
				}
			}
		}
	}

	statusStr := getStr("status")
	if statusStr == "" {
		poi.Status = model.StatusActive
	} else {
		poi.Status = model.POIStatus(statusStr)
	}

	sourceStr := getStr("source")
	if sourceStr != "" {
		poi.Source = model.POISource(sourceStr)
	}

	bhStr := getStr("business_hours")
	if bhStr != "" {
		poi.BusinessHours = parseBusinessHours(bhStr)
	}

	return poi, nil
}

func extractCoordinates(geom GeoJSONGeometry) (float64, float64, error) {
	switch geom.Type {
	case "Point":
		coords, ok := geom.Coordinates.([]interface{})
		if !ok || len(coords) < 2 {
			return 0, 0, fmt.Errorf("invalid Point coordinates")
		}
		lng, ok1 := coords[0].(float64)
		lat, ok2 := coords[1].(float64)
		if !ok1 || !ok2 {
			return 0, 0, fmt.Errorf("invalid coordinate values")
		}
		return lng, lat, nil

	default:
		return extractFirstPoint(geom.Coordinates)
	}
}

func extractFirstPoint(coords interface{}) (float64, float64, error) {
	switch c := coords.(type) {
	case []interface{}:
		if len(c) == 0 {
			return 0, 0, fmt.Errorf("empty coordinates")
		}
		if len(c) >= 2 {
			if lng, ok1 := c[0].(float64); ok1 {
				if lat, ok2 := c[1].(float64); ok2 {
					return lng, lat, nil
				}
			}
		}
		return extractFirstPoint(c[0])
	default:
		return 0, 0, fmt.Errorf("unsupported coordinate format")
	}
}

func getFirstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}
