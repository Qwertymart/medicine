package models

type MedicalData struct {
	DeviceID string  `json:"device_id"`
	DataType string  `json:"data_type"`
	Value    float64 `json:"value"`
	Units    string  `json:"units"`
	TimeSec  float64 `json:"time_sec"`
}
