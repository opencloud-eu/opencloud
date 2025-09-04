package loremipsum

import (
	"math"
	"strings"
)

func (li *LoremIpsum) gauss(mean, stdDev float64) float64 {
	x := li.rng.Float64()
	y := li.rng.Float64()
	z := math.Sqrt(-2*math.Log(x)) * math.Cos(2*math.Pi*y)
	return z*stdDev + mean
}

func (li *LoremIpsum) punctuate(sentence []string) string {
	count := len(sentence)
	if count > 4 {
		mean := math.Log(float64(count)) / math.Log(6.0)
		stdDev := mean / 6
		commas := int(li.gauss(mean, stdDev))
		for i := 1; i < commas; i++ {
			idx := int(float64(i) * float64(count) / (float64(commas) + 1))
			if idx > 0 && idx < (count-1) {
				sentence[idx] = sentence[idx] + ","
			}
		}
	}

	first := strings.Split(sentence[0], "")
	first[0] = strings.ToUpper(first[0])
	sentence[0] = strings.Join(first, "")

	lastIdx := count - 1
	sentence[lastIdx] = sentence[lastIdx] + "."

	return strings.Join(sentence, " ")
}
