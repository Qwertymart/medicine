package features

import (
    "math"
    "ml-service/pkg/utils"
)

// XCorrFeatures вычисляет признаки кросс-корреляции FHR и UC
type XCorrFeatures struct {
    MaxAbs float64 `json:"maxabs"`
    Lag    float64 `json:"lag"`
}

// CalculateXCorrFeatures вычисляет признаки кросс-корреляции
func CalculateXCorrFeatures(fhr, uc []float64, fs float64, maxLagS float64) XCorrFeatures {
    if len(fhr) == 0 || len(uc) == 0 {
        return XCorrFeatures{
            MaxAbs: math.NaN(),
            Lag:    math.NaN(),
        }
    }
    
    // Z-score нормализация
    fhrMean := utils.Mean(fhr)
    fhrStd := utils.Std(fhr)
    ucMean := utils.Mean(uc)
    ucStd := utils.Std(uc)
    
    if fhrStd < 1e-6 || ucStd < 1e-6 {
        return XCorrFeatures{
            MaxAbs: math.NaN(),
            Lag:    math.NaN(),
        }
    }
    
    // Нормализация
    fhrNorm := make([]float64, len(fhr))
    ucNorm := make([]float64, len(uc))
    
    for i, v := range fhr {
        fhrNorm[i] = (v - fhrMean) / fhrStd
    }
    
    for i, v := range uc {
        ucNorm[i] = (v - ucMean) / ucStd
    }
    
    maxLag := int(maxLagS * fs)
    bestVal := -1.0
    bestLag := 0
    
    // Поиск максимальной корреляции по лагам
    for lag := -maxLag; lag <= maxLag; lag++ {
        var a, b []float64
        
        if lag >= 0 {
            // fhr смещен вперед
            if lag < len(fhrNorm) {
                a = fhrNorm[lag:]
                minLen := int(math.Min(float64(len(a)), float64(len(ucNorm))))
                if minLen > 0 {
                    a = a[:minLen]
                    b = ucNorm[:minLen]
                }
            }
        } else {
            // uc смещен вперед
            absLag := -lag
            if absLag < len(ucNorm) {
                b = ucNorm[absLag:]
                minLen := int(math.Min(float64(len(fhrNorm)), float64(len(b))))
                if minLen > 0 {
                    a = fhrNorm[:minLen]
                    b = b[:minLen]
                }
            }
        }
        
        if len(a) < int(5*fs) { // минимум 5 секунд данных
            continue
        }
        
        // Вычисляем корреляцию
        corr := 0.0
        for i := 0; i < len(a); i++ {
            corr += a[i] * b[i]
        }
        corr /= float64(len(a))
        
        if utils.Abs(corr) > utils.Abs(bestVal) {
            bestVal = corr
            bestLag = lag
        }
    }
    
    return XCorrFeatures{
        MaxAbs: utils.Abs(bestVal),
        Lag:    float64(bestLag) / fs,
    }
}