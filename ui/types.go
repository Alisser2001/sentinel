package ui

import (
	"time"

	"sentinel/model"
)

// Messages

type tickMsg time.Time

type dataMsg struct {
	records     []model.ProcRec
	tasks       int
	running     int
	l1, l5, l15 float64
	uptime      float64
}

type statusMsg struct {
	text    string
	isError bool
}

// UI Modes

type uiMode int

const (
	normalMode uiMode = iota
	filterMode
	confirmKillMode
	confirmNiceMode
	helpMode
	settingsMode
	editThresholdCPU
	editThresholdMEM
	addWebhookMode
	confirmDeleteWebhook
	selectWebhookMode
)
