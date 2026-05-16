package models

import "time"

type SessionStatus string

const SessionStatusPending SessionStatus = "Pending"
const SessionStatusActive SessionStatus = "Active"
const SessionStatusAwaitingUserInput SessionStatus = "Awaiting User Input"
const SessionStatusInactive SessionStatus = "Inactive"

type Session struct {
	Id             string         `json:"id"`
	Intent         string         `json:"intent"`
	Status         SessionStatus  `json:"status"`
	Created        time.Time      `json:"created"`
	History        []*HistoryItem `json:"history"`
	HistoryVersion int            `json:"historyVersion"`
	Framework      string         `json:"framework"`
}

type HistoryItem struct {
	Role    string `json:"role"`
	Content string `json:"content"`
	Input   string `json:"input,omitempty"`
}
