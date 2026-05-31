package evogin

import (
	"maps"
	"sync/atomic"
)

var patternConfigMap atomic.Value

func getPatternConfigMap() map[string]*PatternConfig {
	if v, ok := patternConfigMap.Load().(map[string]*PatternConfig); ok {
		return v
	}
	return nil
}

func getPatternConfig(method string, pattern string) *PatternConfig {
	m := getPatternConfigMap()
	if m == nil {
		return nil
	}
	return m[patternKey(method, pattern)]
}

func SetPatternConfig(configPattern string, config *PatternConfig) {
	m := getPatternConfigMap()
	m = maps.Clone(m)
	if config != nil {
		m[configPattern] = config
	} else {
		delete(m, configPattern)
	}
	patternConfigMap.Store(m)
}

func patternKey(method string, pattern string) string {
	return method + " " + pattern
}

type PatternConfig struct {
	pattern           string
	IgnoreResponseLog bool
}

func (conf *PatternConfig) GetIgnoreResponseLog() bool {
	if conf == nil {
		return false
	}
	return conf.IgnoreResponseLog
}
