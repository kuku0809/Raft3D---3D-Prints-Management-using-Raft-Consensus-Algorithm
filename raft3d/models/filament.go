package models

type Filament struct {
	ID                    string `json:"id"`                          // unique identifier (string or int as string)
	Type                  string `json:"type"`                        // options: PLA, PETG, ABS, TPU
	Color                 string `json:"color"`                       // e.g., red, blue, black
	TotalWeightInGrams    int    `json:"totalWeightInGrams"`      // total filament weight
	RemainingWeightInGrams int   `json:"remainingWeightInGrams"`  // remaining filament weight
}
