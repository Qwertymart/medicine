package utils

import (
    "math"
    "sort"
)

func SafeFloat(v float64) float64 {
    if math.IsNaN(v) || math.IsInf(v, 0) {
        return 0.0
    }
    return v
}

// Percentile вычисляет процентиль массива
func Percentile(data []float64, p float64) float64 {
    if len(data) == 0 {
        return math.NaN()
    }

    sorted := make([]float64, len(data))
    copy(sorted, data)
    sort.Float64s(sorted)

    if p <= 0 {
        return sorted[0]
    }
    if p >= 100 {
        return sorted[len(sorted)-1]
    }

    n := float64(len(sorted) - 1)
    index := (p / 100.0) * n

    lower := int(math.Floor(index))
    upper := int(math.Ceil(index))

    if lower == upper {
        return sorted[lower]
    }

    weight := index - float64(lower)
    return sorted[lower]*(1-weight) + sorted[upper]*weight
}

// Mean вычисляет среднее значение
func Mean(data []float64) float64 {
    if len(data) == 0 {
        return math.NaN()
    }

    sum := 0.0
    for _, v := range data {
        sum += v
    }
    return sum / float64(len(data))
}

// Std вычисляет стандартное отклонение
func Std(data []float64) float64 {
    if len(data) <= 1 {
        return math.NaN()
    }

    mean := Mean(data)
    sumSquares := 0.0

    for _, v := range data {
        diff := v - mean
        sumSquares += diff * diff
    }

    return math.Sqrt(sumSquares / float64(len(data)-1))
}

// Min находит минимальное значение
func Min(data []float64) float64 {
    if len(data) == 0 {
        return math.NaN()
    }

    min := data[0]
    for _, v := range data[1:] {
        if v < min {
            min = v
        }
    }
    return min
}

// Max находит максимальное значение
func Max(data []float64) float64 {
    if len(data) == 0 {
        return math.NaN()
    }

    max := data[0]
    for _, v := range data[1:] {
        if v > max {
            max = v
        }
    }
    return max
}

// IQR вычисляет межквартильный размах
func IQR(data []float64) float64 {
    p75 := Percentile(data, 75)
    p25 := Percentile(data, 25)
    return p75 - p25
}

// Abs возвращает абсолютное значение
func Abs(x float64) float64 {
    if x < 0 {
        return -x
    }
    return x
}

// Diff вычисляет разности соседних элементов
func Diff(data []float64) []float64 {
    if len(data) <= 1 {
        return []float64{}
    }

    result := make([]float64, len(data)-1)
    for i := 1; i < len(data); i++ {
        result[i-1] = data[i] - data[i-1]
    }
    return result
}
