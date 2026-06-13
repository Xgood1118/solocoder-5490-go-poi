package geohash

import (
	"strings"
)

const base32 = "0123456789bcdefghjkmnpqrstuvwxyz"

var bits = []int{16, 8, 4, 2, 1}

func Encode(lat, lng float64, precision int) string {
	var latMin, latMax = -90.0, 90.0
	var lngMin, lngMax = -180.0, 180.0

	var result strings.Builder
	result.Grow(precision)

	var isEven = true
	var bit, ch int

	for i := 0; i < precision; {
		if isEven {
			mid := (lngMin + lngMax) / 2
			if lng >= mid {
				ch |= bits[bit]
				lngMin = mid
			} else {
				lngMax = mid
			}
		} else {
			mid := (latMin + latMax) / 2
			if lat >= mid {
				ch |= bits[bit]
				latMin = mid
			} else {
				latMax = mid
			}
		}
		isEven = !isEven
		bit++
		if bit == 5 {
			result.WriteByte(base32[ch])
			ch = 0
			bit = 0
			i++
		}
	}
	return result.String()
}

func Neighbors(hash string) []string {
	result := make([]string, 0, 8)
	north := adjacent(hash, "n")
	south := adjacent(hash, "s")
	east := adjacent(hash, "e")
	west := adjacent(hash, "w")

	if north != "" {
		result = append(result, north)
	}
	if south != "" {
		result = append(result, south)
	}
	if east != "" {
		result = append(result, east)
	}
	if west != "" {
		result = append(result, west)
	}
	if north != "" && east != "" {
		ne := adjacent(north, "e")
		if ne != "" {
			result = append(result, ne)
		}
	}
	if north != "" && west != "" {
		nw := adjacent(north, "w")
		if nw != "" {
			result = append(result, nw)
		}
	}
	if south != "" && east != "" {
		se := adjacent(south, "e")
		if se != "" {
			result = append(result, se)
		}
	}
	if south != "" && west != "" {
		sw := adjacent(south, "w")
		if sw != "" {
			result = append(result, sw)
		}
	}
	return result
}

func SurroundingBuckets(hash string) []string {
	result := make([]string, 0, 9)
	result = append(result, hash)
	result = append(result, Neighbors(hash)...)
	return result
}

var neighborMap = map[string]map[string]string{
	"n": {
		"0": "p0r21436x8zb9dcf5h7kjnmqesgutwvy",
		"1": "bc01fg45238967deuvhjyznpkmstqrwx",
	},
	"s": {
		"0": "14365h7k9dcfesgujnmqp0r2twvyx8zb",
		"1": "238967debc01fg45kmstqrwxuvhjyznp",
	},
	"e": {
		"0": "bc01fg45238967deuvhjyznpkmstqrwx",
		"1": "p0r21436x8zb9dcf5h7kjnmqesgutwvy",
	},
	"w": {
		"0": "238967debc01fg45kmstqrwxuvhjyznp",
		"1": "14365h7k9dcfesgujnmqp0r2twvyx8zb",
	},
}

var borderMap = map[string]map[string]string{
	"n": {"0": "prxz", "1": "bcfguvyz"},
	"s": {"0": "028b", "1": "0145hjnp"},
	"e": {"0": "bcfguvyz", "1": "prxz"},
	"w": {"0": "0145hjnp", "1": "028b"},
}

func adjacent(hash, direction string) string {
	if len(hash) == 0 {
		return ""
	}
	lastChar := string(hash[len(hash)-1])
	parent := hash[:len(hash)-1]
	typ := "0"
	if len(hash)%2 == 1 {
		typ = "1"
	}

	borderSet := borderMap[direction][typ]
	if strings.ContainsRune(borderSet, rune(lastChar[0])) {
		if parent != "" {
			parent = adjacent(parent, direction)
			if parent == "" {
				return ""
			}
		} else {
			return ""
		}
	}

	neighborSet := neighborMap[direction][typ]
	idx := strings.Index(neighborSet, lastChar)
	if idx == -1 {
		return ""
	}
	return parent + string(base32[idx])
}

func DynamicPrecision(radiusMeters float64) int {
	switch {
	case radiusMeters <= 2000:
		return 6
	case radiusMeters <= 10000:
		return 5
	case radiusMeters <= 50000:
		return 4
	default:
		return 3
	}
}
