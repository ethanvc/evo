package obs

import "github.com/ethanvc/evo/runstat"

func memoryStatSampler() (map[string]float64, map[string]float64) {
	info, err := runstat.GetMemory()
	if err != nil {
		return nil, nil
	}
	return map[string]float64{
		"memory_limit":   float64(info.MaxBytes),
		"memory_current": float64(info.UsedBytes),
	}, nil
}
