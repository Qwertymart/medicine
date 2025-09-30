package models

// import "encoding/json"

type MLRequest struct {
    CardID           string                 `json:"card_id"`
    TSec             int                    `json:"t_sec"`
    FsHz             int                    `json:"fs_hz"`
    AvailableWindows []string               `json:"available_windows"`
    Features         map[string]float64     `json:"features"`
}

type MLResponse struct {
    OK        bool                   `json:"ok"`
    CardID    string                 `json:"card_id"`
    TSec      int                    `json:"t_sec"`
    Ran       []string               `json:"ran"`
    Missing   map[string][]string    `json:"missing"`
    Notes     []string               `json:"notes"`
    Result    map[string]interface{} `json:"result"`
    UI      map[string]interface{} `json:"ui,omitempty"` 

}

// TrendResult - результат модели тренда
type TrendResult struct {
    Class string             `json:"class"`
    Proba map[string]float64 `json:"proba"`
}

// RiskResult - результат модели риска
type RiskResult struct {
    Proba float64 `json:"proba"`
    Thr   float64 `json:"thr"`
    Pred  int     `json:"pred"`
}