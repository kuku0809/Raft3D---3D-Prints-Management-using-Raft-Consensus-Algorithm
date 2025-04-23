package models

type PrintJob struct {
	ID                 string `json:"id"`
	PrinterID          string `json:"printerID"`
	FilamentID         string `json:"filamentID"`
	FilePath           string `json:"filePath"`
	PrintWeightInGrams int    `json:"printWeightInGrams"`
	Status             string `json:"status"`
}

