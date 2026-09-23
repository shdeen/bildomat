package params

import (
	"math"
	"strconv"
	"strings"
)

// SizeBounds contains constraints for free-form image dimensions.
//   - MaxRatio: the maximum ratio between the longer and shorter edges
//   - MinEdge: the minimum shorter edge
//   - MaxEdge: the maximum longer edge
//   - MinPx: the minimum pixel count
//   - MaxPx: the maximum pixel count
//   - EdgeIncrem: the required increment for each edge
//   - LongEdge: the longer edge used to derive dimensions from an aspect ratio
type SizeBounds struct {
	MaxRatio   float64 `json:"maxRatio,omitempty"`
	MinEdge    int     `json:"minEdge,omitempty"`
	MaxEdge    int     `json:"maxEdge,omitempty"`
	MinPx      int     `json:"minPx,omitempty"`
	MaxPx      int     `json:"maxPx,omitempty"`
	EdgeIncrem int     `json:"edgeIncrem,omitempty"`
	LongEdge   int     `json:"longEdge,omitempty"`
}

// parseRatio takes an A:B aspect ratio and returns its positive finite quotient and whether parsing
// succeeded.
func parseRatio(ratio string) (float64, bool) {
	parts := strings.SplitN(ratio, ":", 2)
	if len(parts) != 2 {
		return 0, false
	}

	aspectNumerator, e1 := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)

	aspectDenominator, e2 := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
	if e1 != nil || e2 != nil || !(aspectNumerator > 0) || !(aspectDenominator > 0) || math.IsInf(aspectNumerator, 0) || math.IsInf(aspectDenominator, 0) {
		return 0, false
	}

	r := aspectNumerator / aspectDenominator
	if !(r > 0) || math.IsInf(r, 0) {
		return 0, false
	}

	return r, true
}

// ParseDimensions takes WxH dimensions and returns positive width and height values and whether
// parsing succeeded.
func ParseDimensions(size string) (parsedWidth, parsedHeight int, isValid bool) {
	parts := strings.SplitN(strings.ToLower(size), "x", 2)
	if len(parts) != 2 {
		return 0, 0, false
	}

	width, e1 := strconv.Atoi(strings.TrimSpace(parts[0]))

	height, e2 := strconv.Atoi(strings.TrimSpace(parts[1]))
	if e1 != nil || e2 != nil || width <= 0 || height <= 0 {
		return 0, 0, false
	}

	return width, height, true
}

// fmtWH takes width and height values and returns WxH dimensions.
func fmtWH(width, height int) string { return strconv.Itoa(width) + "x" + strconv.Itoa(height) }

// suffixNum takes text and a required suffix and returns the positive integer before that suffix
// and whether parsing succeeded.
func suffixNum(text, suffix string) (int, bool) {
	if !strings.HasSuffix(text, suffix) {
		return 0, false
	}

	n, err := strconv.Atoi(strings.TrimSuffix(text, suffix))
	if err != nil || n <= 0 {
		return 0, false
	}

	return n, true
}

// aspectNearest takes an aspect-ratio value and allowed values and returns the exact member or
// nearest parseable ratio. Its boolean result reports whether the input was already a member.
func aspectNearest(input string, allowed []string) (string, bool) {
	trimmedInput := strings.TrimSpace(input)
	if member, found := allowedMember(trimmedInput, allowed); found {
		return member, true
	}

	inputRatio, ok := parseRatio(trimmedInput)
	if !ok {
		return "", false
	}

	nearest, nearestDist := "", math.MaxFloat64

	for _, candidate := range allowed {
		candidateRatio, parseable := parseRatio(candidate)
		if !parseable {
			continue
		}

		if dist := math.Abs(candidateRatio - inputRatio); dist < nearestDist {
			nearestDist, nearest = dist, candidate
		}
	}

	return nearest, false
}

// snapInt takes an integer and allowed values and returns the nearest allowed value. A tie returns
// the first matching value.
func snapInt(sec int, allowed []int) int {
	nearest, distance := 0, math.MaxFloat64
	for _, candidate := range allowed {
		if candidateDistance := math.Abs(float64(sec) - float64(candidate)); candidateDistance < distance {
			nearest, distance = candidate, candidateDistance
		}
	}

	return nearest
}

// nearestRes takes a height and allowed resolution labels and returns the label with the nearest
// representative height.
func nearestRes(height int, allowed []string) string {
	nearest, distance := "", math.MaxFloat64

	for _, candidate := range allowed {
		candidateHeight, valid := repHeight(strings.ToLower(strings.TrimSpace(candidate)))
		if !valid {
			continue
		}

		if candidateDistance := math.Abs(float64(height) - float64(candidateHeight)); candidateDistance < distance {
			nearest, distance = candidate, candidateDistance
		}
	}

	return nearest
}

// The resolution labels repHeight maps to a representative height.
//   - labelSuffixP: the suffix of a height label such as 720p; the number before it is the height
//   - labelSuffixK: the suffix of a K label such as 2k; the number before it counts K units
//   - heightPerK: the representative height of one K unit, which the K rule multiplies by the
//     label's count; for 4K and 8K it yields the published heights
const (
	labelSuffixP = "p"
	labelSuffixK = "k"
	heightPerK   = 540
)

// repHeight takes a resolution label, dimensions, or positive integer and returns its
// representative height and whether the value is recognized.
func repHeight(size string) (int, bool) {
	if n, ok := suffixNum(size, labelSuffixP); ok {
		return n, true
	}

	if n, ok := suffixNum(size, labelSuffixK); ok {
		if n > math.MaxInt/heightPerK {
			return 0, false // an overflowing digit count is outside the mapper's domain
		}

		return n * heightPerK, true
	}

	if width, height, ok := ParseDimensions(size); ok {
		return min(width, height), true
	}

	if n, err := strconv.Atoi(size); err == nil && n > 0 {
		return n, true
	}

	if i := strings.LastIndex(size, "_"); i >= 0 && i+1 < len(size) {
		return repHeight(size[i+1:])
	}

	return 0, false
}

// PickSize selects declared dimensions by orientation, nearest short edge, nearest supplied ratio,
// and declaration order. If the requested orientation has no candidate, it considers both
// orientations.
func PickSize(sizes []string, landscape bool, tierH int, requestedRatio Nullable[float64]) string {
	var sizeCands []string

	for _, size := range sizes {
		if width, height, ok := ParseDimensions(size); ok && (width >= height) == landscape {
			sizeCands = append(sizeCands, size)
		}
	}

	if len(sizeCands) == 0 {
		for _, size := range sizes {
			if _, _, ok := ParseDimensions(size); ok {
				sizeCands = append(sizeCands, size)
			}
		}
	}

	if len(sizeCands) == 0 {
		return ""
	}

	nearest, edgeDistance, ratioDistance := "", math.MaxFloat64, math.MaxFloat64
	ratio, hasRatio := requestedRatio.ValIf()

	for _, size := range sizeCands {
		width, height, _ := ParseDimensions(size)
		candidateEdgeDistance := math.Abs(float64(tierH) - float64(min(width, height)))

		candidateRatioDistance := 0.0
		if hasRatio {
			candidateRatioDistance = math.Abs(float64(width)/float64(height) - ratio)
		}

		if candidateEdgeDistance < edgeDistance || (candidateEdgeDistance == edgeDistance && candidateRatioDistance < ratioDistance) {
			nearest, edgeDistance, ratioDistance = size, candidateEdgeDistance, candidateRatioDistance
		}
	}

	return nearest
}

// nearestSize takes allowed dimensions and an aspect ratio and returns the dimensions with the
// nearest ratio. A tie returns the first matching value.
func nearestSize(sizes []string, ratio float64) string {
	nearest, nearestDist := "", math.MaxFloat64

	for _, size := range sizes {
		width, height, ok := ParseDimensions(size)
		if !ok {
			continue
		}

		if d := math.Abs(float64(width)/float64(height) - ratio); d < nearestDist {
			nearestDist, nearest = d, size
		}
	}

	return nearest
}

// deriveSize takes size bounds and an aspect ratio and returns constrained dimensions based on the
// configured long edge. It returns an empty string when no long edge is configured.
func deriveSize(bounds SizeBounds, ratio float64) string {
	if bounds.LongEdge == 0 {
		return ""
	}

	width, height := bounds.LongEdge, roundToIncrem(float64(bounds.LongEdge)/ratio, bounds.EdgeIncrem)
	if ratio < 1 {
		width, height = roundToIncrem(float64(bounds.LongEdge)*ratio, bounds.EdgeIncrem), bounds.LongEdge
	}

	cw, ch := clampFree(bounds, width, height)
	if cw == 0 || ch == 0 {
		return ""
	}

	return fmtWH(cw, ch)
}

// clampFree takes size bounds and dimensions and returns dimensions adjusted to the configured
// ratio, edge, pixel-count, and increment constraints.
func clampFree(bounds SizeBounds, width, height int) (adjustedWidth, adjustedHeight int) {
	floatWidth, floatHeight := capRatio(bounds, float64(width), float64(height))
	floatWidth, floatHeight = scaleTo(floatWidth, floatHeight, scaleLongFactor(bounds, floatWidth, floatHeight))
	floatWidth, floatHeight = scaleTo(floatWidth, floatHeight, scalePxFactor(bounds, floatWidth*floatHeight))

	return fixRounded(bounds, roundToIncrem(floatWidth, bounds.EdgeIncrem), roundToIncrem(floatHeight, bounds.EdgeIncrem))
}

// fixRounded chooses whole increments satisfying every constraint near the rounded request.
func fixRounded(bounds SizeBounds, width, height int) (adjustedWidth, adjustedHeight int) {
	units, step, usable := bounds.incrementBounds()
	if !usable {
		return 0, 0
	}

	if bounds.validDimensions(width, height) {
		return width, height
	}

	shortEdge, longEdge := units.searchNearestDimensions(float64(min(width, height))/float64(step), float64(max(width, height))/float64(step))

	adjustedWidth, adjustedHeight = longEdge*step, shortEdge*step
	if width < height {
		adjustedWidth, adjustedHeight = adjustedHeight, adjustedWidth
	}

	if !bounds.validDimensions(adjustedWidth, adjustedHeight) {
		return 0, 0
	}

	return adjustedWidth, adjustedHeight
}

// incrementBounds expresses the constraints in whole edge increments. Pixel limits use division so
// neither the step square nor the area can overflow.
func (bounds SizeBounds) incrementBounds() (units SizeBounds, step int, usable bool) {
	step = max(bounds.EdgeIncrem, 1)

	units = SizeBounds{MinEdge: (max(bounds.MinEdge, 1)-1)/step + 1, MaxEdge: math.MaxInt / step, MaxRatio: bounds.MaxRatio}
	if bounds.MaxEdge > 0 {
		units.MaxEdge = bounds.MaxEdge / step
	}

	if bounds.MinPx > 0 {
		units.MinPx = (bounds.MinPx-1)/step/step + 1
	}

	if bounds.MaxPx > 0 {
		units.MaxPx = bounds.MaxPx / step / step
		if units.MaxPx == 0 || units.MinPx > units.MaxPx {
			return units, step, false
		}
	}

	usable = units.MinEdge <= units.MaxEdge && (units.MaxRatio == 0 || units.MaxRatio >= 1) && !math.IsInf(units.MaxRatio, 0)

	return units, step, usable
}

// searchNearestDimensions searches a finite interval of shorter edges. A nearby supported pair
// narrows that interval; longForShort provides the complete admissible longer range.
func (bounds SizeBounds) searchNearestDimensions(requestedShort, requestedLong float64) (shortEdge, longEdge int) {
	shortMaximum := max(bounds.MinEdge, boundedInt(math.Ceil(requestedShort)), ceilSquareRoot(bounds.MinPx))
	if bounds.MaxRatio > 0 {
		shortMaximum = max(shortMaximum, boundedInt(math.Ceil(requestedLong/bounds.MaxRatio)))
	}

	shortMaximum = min(shortMaximum, bounds.MaxEdge)
	if bounds.MaxPx > 0 {
		shortMaximum = min(shortMaximum, floorSquareRoot(bounds.MaxPx))
	}

	shortEdge, longEdge, distance := bounds.probeInitialDimensions(requestedShort, requestedLong, shortMaximum)

	firstShort := bounds.MinEdge
	if !math.IsInf(distance, 1) {
		firstShort = max(firstShort, boundedInt(math.Ceil(requestedShort-distance)))
		shortMaximum = min(shortMaximum, boundedInt(math.Floor(requestedShort+distance)))
	}

	for candidateShort := firstShort; candidateShort <= shortMaximum; candidateShort++ {
		candidateLong := bounds.longForShort(candidateShort, requestedLong)
		if candidateLong != 0 {
			if closerDimensions(requestedShort, requestedLong, candidateShort, candidateLong, shortEdge, longEdge) {
				shortEdge, longEdge = candidateShort, candidateLong
			}
		}

		if candidateShort == shortMaximum {
			break
		} // Do not overflow the final increment.
	}

	return shortEdge, longEdge
}

// probeInitialDimensions compares the requested shorter edge with the two search limits.
func (bounds SizeBounds) probeInitialDimensions(requestedShort, requestedLong float64, shortMaximum int) (shortEdge, longEdge int, distance float64) {
	distance = math.Inf(1)

	initialShort := min(max(boundedInt(math.Round(requestedShort)), bounds.MinEdge), shortMaximum)
	for _, candidateShort := range []int{initialShort, bounds.MinEdge, shortMaximum} {
		candidateLong := bounds.longForShort(candidateShort, requestedLong)
		if candidateLong == 0 {
			continue
		}

		candidateDistance := math.Hypot(float64(candidateShort)-requestedShort, float64(candidateLong)-requestedLong)
		if closerDimensions(requestedShort, requestedLong, candidateShort, candidateLong, shortEdge, longEdge) {
			shortEdge, longEdge, distance = candidateShort, candidateLong, candidateDistance
		}
	}

	return shortEdge, longEdge, distance
}

// closerDimensions orders usable dimensions by distance, then a balanced shape, then preservation
// of the requested longer edge on an exact tie.
func closerDimensions(requestedShort, requestedLong float64, shortEdge, longEdge, chosenShort, chosenLong int) bool {
	if chosenShort == 0 {
		return true
	}

	distance := math.Hypot(float64(shortEdge)-requestedShort, float64(longEdge)-requestedLong)

	chosenDistance := math.Hypot(float64(chosenShort)-requestedShort, float64(chosenLong)-requestedLong)
	if distance != chosenDistance {
		return distance < chosenDistance
	}

	if longEdge-shortEdge != chosenLong-chosenShort {
		return longEdge-shortEdge < chosenLong-chosenShort
	}

	return longEdge > chosenLong
}

// longForShort returns the nearest supported longer edge in increment units.
func (bounds SizeBounds) longForShort(shortEdge int, requestedLong float64) int {
	if shortEdge < bounds.MinEdge || shortEdge > bounds.MaxEdge {
		return 0
	}

	lower, upper := shortEdge, bounds.MaxEdge
	if bounds.MinPx > 0 {
		lower = max(lower, (bounds.MinPx-1)/shortEdge+1)
	}

	if bounds.MaxPx > 0 {
		upper = min(upper, bounds.MaxPx/shortEdge)
	}

	if bounds.MaxRatio > 0 {
		upper = min(upper, boundedInt(math.Floor(float64(shortEdge)*bounds.MaxRatio)))
	}

	if lower > upper {
		return 0
	}

	return min(max(boundedInt(math.Round(requestedLong)), lower), upper)
}

// validDimensions checks every declared constraint without multiplying the edges. Its callers first
// validate the declaration through incrementBounds.
func (bounds SizeBounds) validDimensions(width, height int) bool {
	if width <= 0 || height <= 0 || min(width, height) < bounds.MinEdge {
		return false
	}

	if bounds.MaxEdge > 0 && max(width, height) > bounds.MaxEdge {
		return false
	}

	if bounds.MaxPx > 0 && width > bounds.MaxPx/height {
		return false
	}

	if bounds.MinPx > 0 && width <= (bounds.MinPx-1)/height {
		return false
	}

	if bounds.MaxRatio > 0 && float64(max(width, height))/float64(min(width, height)) > bounds.MaxRatio {
		return false
	}

	return bounds.EdgeIncrem <= 0 || (width%bounds.EdgeIncrem == 0 && height%bounds.EdgeIncrem == 0)
}

// boundedInt converts a nonnegative edge estimate without overflowing an int.
func boundedInt(value float64) int {
	if value >= float64(math.MaxInt) {
		return math.MaxInt
	}

	if !(value > 0) {
		return 0
	}

	return int(value)
}

// floorSquareRoot returns the floor of the square root of a nonnegative count.
func floorSquareRoot(count int) int {
	if count <= 0 {
		return 0
	}

	root := int(math.Sqrt(float64(count)))
	for root > count/root {
		root--
	}

	for root+1 <= count/(root+1) {
		root++
	}

	return root
}

// ceilSquareRoot returns the smallest edge whose square reaches the count.
func ceilSquareRoot(count int) int {
	root := floorSquareRoot(count)
	if root*root < count {
		return root + 1
	}

	return root
}

// capRatio takes size bounds and floating-point dimensions and returns dimensions whose
// long-to-short ratio does not exceed the configured maximum.
func capRatio(bounds SizeBounds, floatWidth, floatHeight float64) (adjustedWidth, adjustedHeight float64) {
	if bounds.MaxRatio <= 0 {
		return floatWidth, floatHeight
	}

	if floatWidth/floatHeight > bounds.MaxRatio {
		return floatHeight * bounds.MaxRatio, floatHeight
	}

	if floatHeight/floatWidth > bounds.MaxRatio {
		return floatWidth, floatWidth * bounds.MaxRatio
	}

	return floatWidth, floatHeight
}

// scaleLongFactor takes size bounds and floating-point dimensions and returns the scale that brings
// the longer edge within the configured maximum. It returns one when no reduction is needed.
func scaleLongFactor(bounds SizeBounds, floatWidth, floatHeight float64) float64 {
	if m := math.Max(floatWidth, floatHeight); bounds.MaxEdge > 0 && m > float64(bounds.MaxEdge) {
		return float64(bounds.MaxEdge) / m
	}

	return 1
}

// scalePxFactor takes size bounds and a pixel count and returns the scale that brings the count
// within the configured range. It returns one when no scaling is needed.
func scalePxFactor(bounds SizeBounds, pixelCount float64) float64 {
	if bounds.MaxPx > 0 && pixelCount > float64(bounds.MaxPx) {
		return math.Sqrt(float64(bounds.MaxPx) / pixelCount)
	}

	if bounds.MinPx > 0 && pixelCount < float64(bounds.MinPx) {
		return math.Sqrt(float64(bounds.MinPx) / pixelCount)
	}

	return 1
}

// scaleTo takes floating-point dimensions and a scale and returns both dimensions multiplied by
// that scale.
func scaleTo(floatWidth, floatHeight, scale float64) (scaledWidth, scaledHeight float64) {
	return floatWidth * scale, floatHeight * scale
}

// roundToIncrem takes a value and increment and returns the nearest positive multiple of the
// increment. A non-positive increment rounds to the nearest positive integer.
func roundToIncrem(value float64, step int) int {
	step = max(step, 1)
	increments := max(boundedInt(math.Round(value/float64(step))), 1)
	increments = min(increments, math.MaxInt/step)

	return increments * step
}

// allowedMember returns the declared spelling of a case-insensitive member.
func allowedMember(input string, allowed []string) (string, bool) {
	for _, candidate := range allowed {
		if strings.EqualFold(input, candidate) {
			return candidate, true
		}
	}

	return "", false
}

// ratioFromSpec returns the ratio represented by dimensions or an aspect ratio.
func ratioFromSpec(sizeInput string) (float64, bool) {
	if width, height, valid := ParseDimensions(sizeInput); valid {
		return float64(width) / float64(height), true
	}

	return parseRatio(sizeInput)
}
