package features

import (
    "math"
    "ml-service/pkg/utils"
)

// FHRFeatures вычисляет признаки для FHR данных
type FHRFeatures struct {
    Mean     float64 `json:"mean"`
    Std      float64 `json:"std"`
    Min      float64 `json:"min"`
    Max      float64 `json:"max"`
    IQR      float64 `json:"iqr"`
    RMSSD    float64 `json:"rmssd"`
    AbsDev   float64 `json:"abs_dev"`
    BradyLen float64 `json:"brady_len"`
    TachyLen float64 `json:"tachy_len"`
    DecelCnt int     `json:"decel_cnt"`
}

// CalculateFHRFeatures вычисляет все признаки FHR
func CalculateFHRFeatures(fhr []float64, fs float64) FHRFeatures {
    return FHRFeatures{
        Mean:     utils.Mean(fhr),
        Std:      utils.Std(fhr),
        Min:      utils.Min(fhr),
        Max:      utils.Max(fhr),
        IQR:      utils.IQR(fhr),
        RMSSD:    calculateRMSSD(fhr),
        AbsDev:   calculateAbsDev(fhr),
        BradyLen: calculateBradyLen(fhr, fs),
        TachyLen: calculateTachyLen(fhr, fs),
        DecelCnt: calculateDecelCnt(fhr, fs),
    }
}

// calculateRMSSD вычисляет RMSSD (Root Mean Square of Successive Differences)
func calculateRMSSD(fhr []float64) float64 {
    if len(fhr) <= 1 {
        return math.NaN()
    }

    diff := utils.Diff(fhr)
    sumSquares := 0.0

    for _, d := range diff {
        sumSquares += d * d
    }

    if len(diff) == 0 {
        return math.NaN()
    }

    return math.Sqrt(sumSquares / float64(len(diff)))
}

// calculateAbsDev вычисляет среднее абсолютное отклонение от медианы
func calculateAbsDev(fhr []float64) float64 {
    if len(fhr) == 0 {
        return math.NaN()
    }

    median := utils.Percentile(fhr, 50)
    sum := 0.0

    for _, v := range fhr {
        sum += utils.Abs(v - median)
    }

    return sum / float64(len(fhr))
}

// calculateBradyLen вычисляет суммарную длительность брадикардии (FHR < 110)
func calculateBradyLen(fhr []float64, fs float64) float64 {
    return calculateRunLength(fhr, fs, func(v float64) bool {
        return v < 110
    })
}

// calculateTachyLen вычисляет суммарную длительность тахикардии (FHR > 160)
func calculateTachyLen(fhr []float64, fs float64) float64 {
    return calculateRunLength(fhr, fs, func(v float64) bool {
        return v > 160
    })
}

// calculateRunLength вычисляет суммарную длительность состояний
func calculateRunLength(data []float64, fs float64, condition func(float64) bool) float64 {
    total := 0.0
    run := 0.0

    for _, v := range data {
        if condition(v) {
            run += 1.0
        } else {
            total += run
            run = 0.0
        }
    }

    if run > 0 {
        total += run
    }

    return total / fs // конвертируем в секунды
}

// calculateDecelCnt вычисляет количество децелераций
func calculateDecelCnt(fhr []float64, fs float64) int {
    if len(fhr) == 0 {
        return 0
    }

    threshold := utils.Percentile(fhr, 50) - 15
    minLen := int(7.5 * fs) // минимум 7.5 секунд

    count := 0
    run := 0

    for _, v := range fhr {
        if v < threshold {
            run++
        } else {
            if run >= minLen {
                count++
            }
            run = 0
        }
    }

    if run >= minLen {
        count++
    }

    return count
}
