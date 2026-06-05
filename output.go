package main

import (
	"fmt"
)

type LogFn func(format string, args ...any)

var logFn LogFn = func(format string, args ...any) {
	fmt.Printf(format+"\n", args...)
}

func out(format string, args ...any) {
	logFn(format, args...)
}

type ProgressUpdate struct {
	QuestID string
	Name    string
	Type    string
	Current float64
	Target  float64
	Status  string
}

var progressCh chan ProgressUpdate

func setProgressChannel(ch chan ProgressUpdate) {
	progressCh = ch
}

func sendProgress(id, name, taskType string, cur, target float64, status string) {
	if progressCh != nil {
		progressCh <- ProgressUpdate{
			QuestID: id,
			Name:    name,
			Type:    taskType,
			Current: cur,
			Target:  target,
			Status:  status,
		}
	}
}
