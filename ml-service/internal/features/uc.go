package features

import (
    "ml-service/pkg/utils"
)

// UCFeatures вычисляет признаки для UC данных
type UCFeatures struct {
    Mean     float64 `json:"mean"`
    Std      float64 `json:"std"`
    Max      float64 `json:"max"`
    IQR      float64 `json:"iqr"`
    PeakCnt  int     `json:"peak_cnt"`
    Area     float64 `json:"area"`
}

// CalculateUCFeatures вычисляет все признаки UC
func CalculateUCFeatures(uc []float64, fs float64) UCFeatures {
    peakCnt, area := calculateUCPeaksAndArea(uc, fs)
    
    return UCFeatures{
        Mean:    utils.Mean(uc),
        Std:     utils.Std(uc),
        Max:     utils.Max(uc),
        IQR:     utils.IQR(uc),
        PeakCnt: peakCnt,
        Area:    area,
    }
}

// calculateUCPeaksAndArea вычисляет количество пиков и площадь под кривой UC
func calculateUCPeaksAndArea(uc []float64, fs float64) (int, float64) {
    if len(uc) == 0 {
        return 0, 0.0
    }
    
    p10 := utils.Percentile(uc, 10)
    p90 := utils.Percentile(uc, 90)
    threshold := p10 + 0.5*(p90-p10)
    minLen := int(5 * fs) // минимум 5 секунд
    
    count := 0
    run := 0
    area := 0.0
    
    for _, v := range uc {
        if v > threshold {
            run++
            area += (v - threshold)
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
    
    return count, area / fs // конвертируем в секунды
}