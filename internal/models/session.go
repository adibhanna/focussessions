package models

import (
	"time"
)

type Session struct {
	ID             string    `json:"id"`
	StartTime      time.Time `json:"start_time"`
	EndTime        time.Time `json:"end_time"`
	Duration       int       `json:"duration"` // in minutes
	Completed      bool      `json:"completed"`
	Date           string    `json:"date"`  // YYYY-MM-DD format
	Week           int       `json:"week"`  // Week number of the year
	Month          string    `json:"month"` // YYYY-MM format
	Year           int       `json:"year"`
	Active         bool      `json:"active"`          // Is this session currently active
	ElapsedSeconds int       `json:"elapsed_seconds"` // Seconds elapsed so far
	Paused         bool      `json:"paused"`          // Is the session paused
}

type Config struct {
	SessionDuration  int `json:"session_duration"`   // Default session duration in minutes
	DailySessionGoal int `json:"daily_session_goal"` // Number of sessions goal per day
	WorkStartHour    int `json:"work_start_hour"`    // Start hour (24h format)
	WorkEndHour      int `json:"work_end_hour"`      // End hour (24h format)
}

func DefaultConfig() Config {
	return Config{
		SessionDuration:  60,
		DailySessionGoal: 8,
		WorkStartHour:    8,
		WorkEndHour:      16,
	}
}

type DayStats struct {
	Date          string    `json:"date"`
	SessionsCount int       `json:"sessions_count"`
	TotalMinutes  int       `json:"total_minutes"`
	Sessions      []Session `json:"sessions"`
}

type WeekStats struct {
	Week          int        `json:"week"`
	Year          int        `json:"year"`
	SessionsCount int        `json:"sessions_count"`
	TotalMinutes  int        `json:"total_minutes"`
	DailyStats    []DayStats `json:"daily_stats"`
}

type MonthStats struct {
	Month         string      `json:"month"`
	Year          int         `json:"year"`
	SessionsCount int         `json:"sessions_count"`
	TotalMinutes  int         `json:"total_minutes"`
	WeeklyStats   []WeekStats `json:"weekly_stats"`
}

type YearStats struct {
	Year          int          `json:"year"`
	SessionsCount int          `json:"sessions_count"`
	TotalMinutes  int          `json:"total_minutes"`
	MonthlyStats  []MonthStats `json:"monthly_stats"`
}
