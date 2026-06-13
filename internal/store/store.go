package store

import (
	"poi-service/internal/geohash"
	"poi-service/internal/model"
	"poi-service/internal/utils"
	"sync"
)

type Store struct {
	mu            sync.RWMutex
	poiByID       map[string]*model.POI
	geohashBuckets map[string][]*model.POI
	cityIndex     map[string][]*model.POI
	addressIndex  map[string][]*model.POI
	categoryIndex map[string]map[string]map[string][]*model.POI
}

var instance *Store
var once sync.Once

func GetStore() *Store {
	once.Do(func() {
		instance = &Store{
			poiByID:        make(map[string]*model.POI),
			geohashBuckets: make(map[string][]*model.POI),
			cityIndex:      make(map[string][]*model.POI),
			addressIndex:   make(map[string][]*model.POI),
			categoryIndex:  make(map[string]map[string]map[string][]*model.POI),
		}
	})
	return instance
}

var geohashPrecisions = []int{3, 4, 5, 6}

func (s *Store) AddPOI(poi *model.POI) {
	s.mu.Lock()
	defer s.mu.Unlock()

	poi.Geohash6 = geohash.Encode(poi.Lat, poi.Lng, 6)
	poi.Geohash5 = geohash.Encode(poi.Lat, poi.Lng, 5)
	poi.PinyinName = utils.ToPinyin(poi.Name.ZhCN)

	if existing, ok := s.poiByID[poi.POIId]; ok {
		s.removeFromIndexes(existing)
	}

	s.poiByID[poi.POIId] = poi

	for _, prec := range geohashPrecisions {
		hash := geohash.Encode(poi.Lat, poi.Lng, prec)
		s.geohashBuckets[hash] = append(s.geohashBuckets[hash], poi)
	}

	if poi.City != "" {
		s.cityIndex[poi.City] = append(s.cityIndex[poi.City], poi)
	}

	if poi.Address != "" {
		s.addressIndex[poi.Address] = append(s.addressIndex[poi.Address], poi)
	}

	s.addToCategoryIndex(poi)
}

func (s *Store) removeFromIndexes(poi *model.POI) {
	for _, prec := range geohashPrecisions {
		hash := geohash.Encode(poi.Lat, poi.Lng, prec)
		s.geohashBuckets[hash] = removePOI(s.geohashBuckets[hash], poi)
	}
	s.cityIndex[poi.City] = removePOI(s.cityIndex[poi.City], poi)
	s.addressIndex[poi.Address] = removePOI(s.addressIndex[poi.Address], poi)
	s.removeFromCategoryIndex(poi)
}

func (s *Store) addToCategoryIndex(poi *model.POI) {
	l1 := poi.Category.L1
	l2 := poi.Category.L2
	l3 := poi.Category.L3

	if _, ok := s.categoryIndex[l1]; !ok {
		s.categoryIndex[l1] = make(map[string]map[string][]*model.POI)
	}
	if _, ok := s.categoryIndex[l1][l2]; !ok {
		s.categoryIndex[l1][l2] = make(map[string][]*model.POI)
	}
	s.categoryIndex[l1][l2][l3] = append(s.categoryIndex[l1][l2][l3], poi)
}

func (s *Store) removeFromCategoryIndex(poi *model.POI) {
	l1 := poi.Category.L1
	l2 := poi.Category.L2
	l3 := poi.Category.L3

	if l1Map, ok := s.categoryIndex[l1]; ok {
		if l2Map, ok := l1Map[l2]; ok {
			l2Map[l3] = removePOI(l2Map[l3], poi)
		}
	}
}

func removePOI(slice []*model.POI, poi *model.POI) []*model.POI {
	for i, p := range slice {
		if p.POIId == poi.POIId {
			return append(slice[:i], slice[i+1:]...)
		}
	}
	return slice
}

func (s *Store) GetPOIByID(id string) (*model.POI, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	poi, ok := s.poiByID[id]
	return poi, ok
}

func (s *Store) GetPOIsByGeohash(hash string) []*model.POI {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.geohashBuckets[hash]
}

func (s *Store) GetAllPOIs() []*model.POI {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]*model.POI, 0, len(s.poiByID))
	for _, poi := range s.poiByID {
		result = append(result, poi)
	}
	return result
}

func (s *Store) GetPOIsByCity(city string) []*model.POI {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.cityIndex[city]
}

func (s *Store) GetPOIsByAddress(address string) []*model.POI {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.addressIndex[address]
}

func (s *Store) GetPOIsByCategory(l1, l2, l3 string) []*model.POI {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*model.POI

	if l1 == "" {
		for _, l1Map := range s.categoryIndex {
			for _, l2Map := range l1Map {
				for _, pois := range l2Map {
					result = append(result, pois...)
				}
			}
		}
		return result
	}

	if l1Map, ok := s.categoryIndex[l1]; ok {
		if l2 == "" {
			for _, l2Map := range l1Map {
				for _, pois := range l2Map {
					result = append(result, pois...)
				}
			}
			return result
		}

		if l2Map, ok := l1Map[l2]; ok {
			if l3 == "" {
				for _, pois := range l2Map {
					result = append(result, pois...)
				}
				return result
			}
			return l2Map[l3]
		}
	}

	return result
}

func (s *Store) CountPOIsByCategory() map[string]int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make(map[string]int)
	for l1, l1Map := range s.categoryIndex {
		count := 0
		for _, l2Map := range l1Map {
			for _, pois := range l2Map {
				count += len(pois)
			}
		}
		result[l1] = count
	}
	return result
}

func (s *Store) CountPOIsByCity() map[string]int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make(map[string]int)
	for city, pois := range s.cityIndex {
		result[city] = len(pois)
	}
	return result
}

func (s *Store) CountPOIsBySource() map[model.POISource]int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make(map[model.POISource]int)
	for _, poi := range s.poiByID {
		result[poi.Source]++
	}
	return result
}

func (s *Store) DeletePOI(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	poi, ok := s.poiByID[id]
	if !ok {
		return false
	}

	s.removeFromIndexes(poi)
	delete(s.poiByID, id)
	return true
}

func (s *Store) Size() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.poiByID)
}
