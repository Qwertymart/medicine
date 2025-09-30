package features


import (
   "fmt"
   "ml-service/pkg/utils"
)


// FeatureCalculator отвечает за расчет всех фичей
type FeatureCalculator struct {
   fs float64 // частота дискретизации
}


// NewFeatureCalculator создает новый калькулятор фичей
func NewFeatureCalculator(fs float64) *FeatureCalculator {
   return &FeatureCalculator{fs: fs}
}


// Calculate вычисляет все фичи для заданного окна
func (fc *FeatureCalculator) Calculate(fhr, uc []float64, windowSec int) map[string]float64 {
   windowSize := int(float64(windowSec) * fc.fs)
  
   // Берем данные из конца массива (последние windowSize точек)
   fhrWindow := fc.getLastWindow(fhr, windowSize)
   ucWindow := fc.getLastWindow(uc, windowSize)
  
   features := make(map[string]float64)
   prefix := fmt.Sprintf("f_%ds_", windowSec)
  
   // FHR фичи с проверкой на NaN
   fhrFeats := CalculateFHRFeatures(fhrWindow, fc.fs)
   features[prefix+"fhr_mean"] = utils.SafeFloat(fhrFeats.Mean)
   features[prefix+"fhr_std"] = utils.SafeFloat(fhrFeats.Std)
   features[prefix+"fhr_min"] = utils.SafeFloat(fhrFeats.Min)
   features[prefix+"fhr_max"] = utils.SafeFloat(fhrFeats.Max)
   features[prefix+"fhr_iqr"] = utils.SafeFloat(fhrFeats.IQR)
   features[prefix+"fhr_rmssd"] = utils.SafeFloat(fhrFeats.RMSSD)
   features[prefix+"fhr_abs_dev"] = utils.SafeFloat(fhrFeats.AbsDev)
   features[prefix+"fhr_brady_len"] = utils.SafeFloat(fhrFeats.BradyLen)
   features[prefix+"fhr_tachy_len"] = utils.SafeFloat(fhrFeats.TachyLen)
   features[prefix+"fhr_decel_cnt"] = utils.SafeFloat(float64(fhrFeats.DecelCnt))
  
   // UC фичи с проверкой на NaN
   ucFeats := CalculateUCFeatures(ucWindow, fc.fs)
   features[prefix+"uc_mean"] = utils.SafeFloat(ucFeats.Mean)
   features[prefix+"uc_std"] = utils.SafeFloat(ucFeats.Std)
   features[prefix+"uc_max"] = utils.SafeFloat(ucFeats.Max)
   features[prefix+"uc_iqr"] = utils.SafeFloat(ucFeats.IQR)
   features[prefix+"uc_peak_cnt"] = utils.SafeFloat(float64(ucFeats.PeakCnt))
   features[prefix+"uc_area"] = utils.SafeFloat(ucFeats.Area)
  
   // Кросс-корреляция с проверкой на NaN
   xcorrFeats := CalculateXCorrFeatures(fhrWindow, ucWindow, fc.fs, 60.0)
   features[prefix+"xcorr_maxabs"] = utils.SafeFloat(xcorrFeats.MaxAbs)
   features[prefix+"xcorr_lag"] = utils.SafeFloat(xcorrFeats.Lag)
  
   return features
}


// getLastWindow возвращает последние N точек из массива
func (fc *FeatureCalculator) getLastWindow(data []float64, windowSize int) []float64 {
   if len(data) <= windowSize {
       return data
   }
   return data[len(data)-windowSize:]
}


// CalculateAllFeatures вычисляет фичи для всех доступных окон
func (fc *FeatureCalculator) CalculateAllFeatures(fhr, uc []float64, duration int) map[string]float64 {
   features := make(map[string]float64)
   windows := []int{240, 600, 900}
  
   for _, window := range windows {
       if duration >= window {
           windowFeatures := fc.Calculate(fhr, uc, window)
           for k, v := range windowFeatures {
               features[k] = v
           }
       }
   }
  
   return features
}


// GetAvailableWindows возвращает список доступных окон на основе длительности данных
func (fc *FeatureCalculator) GetAvailableWindows(duration int) []string {
   var windows []string
  
   if duration >= 240 {
       windows = append(windows, "240s")
   }
   if duration >= 600 {
       windows = append(windows, "600s")
   }
   if duration >= 900 {
       windows = append(windows, "900s")
   }
  
   return windows
}
