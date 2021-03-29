package backend

import (
	"math"
	"regexp"
	"strings"

	"gopkg.in/gorp.v2"
)

type classifier struct {
	classes      []int64
	classProbs   map[int64]float64
	featureProbs map[int64]map[string]float64
}

const (
	CertaintyThreshold float64 = 0.25
	SplitThreshold     int     = 6
	RegularizationTerm float64 = 0.01
)

var SplitPattern = regexp.MustCompile(`[\t\n\v\f\r /\-\(\)]+`)
var AsciiPattern = regexp.MustCompile(`^[[:ascii:]]*$`)
var TextPattern = regexp.MustCompile(`([[:alpha:]]+)`)

func extractFeatures(name string, amount string) []string {
	result := make([]string, 0)

	name = strings.ToLower(name)
	words := SplitPattern.Split(name, -1)

	for _, w := range words {
		if w == "" {
			continue
		} else {
			result = append(result, w)
		}

		if AsciiPattern.MatchString(w) {
			// Split long words in half (e.g. strawberry -> straw+berry).
			if len(w) >= SplitThreshold {
				result = append(result, w[:len(w)/2], w[len(w)/2:])
			}
		} else {
			// If not ASCII (e.g. Chinese), split into individual glyphs.
			result = append(result, strings.Split(w, "")...)
		}
	}

	if amount != "" {
		amount = strings.ToLower(amount)

		unit := TextPattern.FindString(amount)
		if unit != "" {
			result = append(result, "unit:"+unit)
		}
	}

	return result
}

func NewClassifier() *classifier {
	return &classifier{
		classes:      make([]int64, 0),
		classProbs:   make(map[int64]float64),
		featureProbs: make(map[int64]map[string]float64),
	}
}

func (c *classifier) Train(db *gorp.DbMap) {
	ingredients := []Ingredient{}

	_, err := db.Select(&ingredients, "SELECT * FROM ingredients WHERE class_id IS NOT NULL")
	if err != nil {
		panic(err)
	}

	allFeatures := make(map[string]int64)
	classCounts := make(map[int64]int64)
	classFeatures := make(map[int64]map[string]float64)
	classNumFeatures := make(map[int64]float64)

	// Count the occurence of features under different classes.
	for _, ingredient := range ingredients {
		words := extractFeatures(ingredient.Name, ingredient.Amount.String)
		class_id := ingredient.ClassId.Int64

		classCounts[class_id] += 1

		for _, feature := range words {
			if classFeatures[class_id] == nil {
				classFeatures[class_id] = make(map[string]float64)
			}
			classFeatures[class_id][feature] += 1
			classNumFeatures[class_id] += 1

			allFeatures[feature] += 1
		}
	}

	// Add 1 for every feature to prevent zero probabilities.
	for feature, _ := range allFeatures {
		for class_id, _ := range classFeatures {
			classFeatures[class_id][feature] += RegularizationTerm
			classNumFeatures[class_id] += RegularizationTerm
		}
	}

	// Now calculate the log probabilities.
	for class_id, count := range classCounts {
		c.classProbs[class_id] = math.Log(float64(count) / float64(len(ingredients)))

		c.featureProbs[class_id] = make(map[string]float64)
		for feature, fcount := range classFeatures[class_id] {
			c.featureProbs[class_id][feature] = math.Log(float64(fcount) / float64(classNumFeatures[class_id]))
		}
	}
}

func (c *classifier) Classify(name string, amount string) int64 {
	probs := make(map[int64]float64)
	for k, v := range c.classProbs {
		probs[k] = v
	}

	features := extractFeatures(name, amount)
	for _, feature := range features {
		for class_id, _ := range probs {
			probs[class_id] += c.featureProbs[class_id][feature]
		}
	}

	var best int64
	probAlpha := math.Inf(-1)
	probBeta := math.Inf(-1)
	for class_id, logp := range probs {
		if logp >= probAlpha {
			probBeta = probAlpha
			probAlpha = logp
			best = class_id
		} else if logp >= probBeta {
			probBeta = logp
		}
	}

	if probAlpha-probBeta < CertaintyThreshold {
		return -1
	} else {
		return best
	}
}
