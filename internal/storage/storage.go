package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"focussessions/internal/models"
)

type Storage struct {
	dataDir string
}

func New() (*Storage, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	dataDir := filepath.Join(homeDir, ".focussessions")
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, err
	}

	return &Storage{dataDir: dataDir}, nil
}

func (s *Storage) sessionsFile() string {
	return filepath.Join(s.dataDir, "sessions.json")
}

func (s *Storage) configFile() string {
	return filepath.Join(s.dataDir, "config.json")
}

func (s *Storage) SaveSession(session models.Session) error {
	sessions, err := s.GetAllSessions()
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	// Check if this is an update to an existing session
	found := false
	for i, existingSession := range sessions {
		if existingSession.ID == session.ID {
			sessions[i] = session
			found = true
			break
		}
	}

	if !found {
		sessions = append(sessions, session)
	}

	data, err := json.MarshalIndent(sessions, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(s.sessionsFile(), data, 0644)
}

func (s *Storage) GetActiveSession() (*models.Session, error) {
	sessions, err := s.GetAllSessions()
	if err != nil {
		return nil, err
	}

	for _, session := range sessions {
		if session.Active && !session.Completed {
			return &session, nil
		}
	}

	return nil, nil
}

func (s *Storage) DeactivateAllSessions() error {
	sessions, err := s.GetAllSessions()
	if err != nil {
		return err
	}

	for i := range sessions {
		sessions[i].Active = false
	}

	data, err := json.MarshalIndent(sessions, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(s.sessionsFile(), data, 0644)
}

func (s *Storage) GetAllSessions() ([]models.Session, error) {
	data, err := os.ReadFile(s.sessionsFile())
	if err != nil {
		if os.IsNotExist(err) {
			return []models.Session{}, nil
		}
		return nil, err
	}

	var sessions []models.Session
	if err := json.Unmarshal(data, &sessions); err != nil {
		return nil, err
	}

	return sessions, nil
}

func (s *Storage) GetTodaySessions() ([]models.Session, error) {
	today := time.Now().Format("2006-01-02")
	return s.GetSessionsByDate(today)
}

func (s *Storage) GetSessionsByDate(date string) ([]models.Session, error) {
	allSessions, err := s.GetAllSessions()
	if err != nil {
		return nil, err
	}

	var sessions []models.Session
	for _, session := range allSessions {
		if session.Date == date {
			sessions = append(sessions, session)
		}
	}

	return sessions, nil
}

func (s *Storage) GetWeekSessions(year int, week int) ([]models.Session, error) {
	allSessions, err := s.GetAllSessions()
	if err != nil {
		return nil, err
	}

	var sessions []models.Session
	for _, session := range allSessions {
		if session.Year == year && session.Week == week {
			sessions = append(sessions, session)
		}
	}

	return sessions, nil
}

func (s *Storage) GetMonthSessions(year int, month int) ([]models.Session, error) {
	allSessions, err := s.GetAllSessions()
	if err != nil {
		return nil, err
	}

	monthStr := fmt.Sprintf("%04d-%02d", year, month)
	var sessions []models.Session
	for _, session := range allSessions {
		if session.Month == monthStr {
			sessions = append(sessions, session)
		}
	}

	return sessions, nil
}

func (s *Storage) GetYearSessions(year int) ([]models.Session, error) {
	allSessions, err := s.GetAllSessions()
	if err != nil {
		return nil, err
	}

	var sessions []models.Session
	for _, session := range allSessions {
		if session.Year == year {
			sessions = append(sessions, session)
		}
	}

	return sessions, nil
}

func (s *Storage) GetConfig() (models.Config, error) {
	data, err := os.ReadFile(s.configFile())
	if err != nil {
		if os.IsNotExist(err) {
			config := models.DefaultConfig()
			if err := s.SaveConfig(config); err != nil {
				return config, err
			}
			return config, nil
		}
		return models.Config{}, err
	}

	var config models.Config
	if err := json.Unmarshal(data, &config); err != nil {
		return models.Config{}, err
	}

	return config, nil
}

func (s *Storage) SaveConfig(config models.Config) error {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(s.configFile(), data, 0644)
}

func (s *Storage) GetDayStats(date string) (models.DayStats, error) {
	sessions, err := s.GetSessionsByDate(date)
	if err != nil {
		return models.DayStats{}, err
	}

	completedCount := 0
	totalMinutes := 0
	for _, session := range sessions {
		if session.Completed {
			completedCount++
			// Use actual time spent, fallback to planned duration
			actualMinutes := session.ElapsedSeconds / 60
			if actualMinutes == 0 && !session.EndTime.IsZero() && !session.StartTime.IsZero() {
				actualMinutes = int(session.EndTime.Sub(session.StartTime).Minutes())
			}
			if actualMinutes == 0 {
				actualMinutes = session.Duration
			}
			totalMinutes += actualMinutes
		}
	}

	stats := models.DayStats{
		Date:          date,
		SessionsCount: completedCount,
		Sessions:      sessions,
		TotalMinutes:  totalMinutes,
	}

	return stats, nil
}

func (s *Storage) GetWeekStats(year int, week int) (models.WeekStats, error) {
	sessions, err := s.GetWeekSessions(year, week)
	if err != nil {
		return models.WeekStats{}, err
	}

	completedCount := 0
	totalMinutes := 0
	dateMap := make(map[string][]models.Session)

	for _, session := range sessions {
		if session.Completed {
			completedCount++
			// Use actual time spent
			actualMinutes := session.ElapsedSeconds / 60
			if actualMinutes == 0 && !session.EndTime.IsZero() && !session.StartTime.IsZero() {
				actualMinutes = int(session.EndTime.Sub(session.StartTime).Minutes())
			}
			if actualMinutes == 0 {
				actualMinutes = session.Duration
			}
			totalMinutes += actualMinutes
			dateMap[session.Date] = append(dateMap[session.Date], session)
		}
	}

	stats := models.WeekStats{
		Week:          week,
		Year:          year,
		SessionsCount: completedCount,
		TotalMinutes:  totalMinutes,
	}

	for date, dateSessions := range dateMap {
		dayStats := models.DayStats{
			Date:          date,
			SessionsCount: len(dateSessions),
			Sessions:      dateSessions,
		}
		for _, s := range dateSessions {
			// Use actual time spent for daily stats too
			actualMinutes := s.ElapsedSeconds / 60
			if actualMinutes == 0 && !s.EndTime.IsZero() && !s.StartTime.IsZero() {
				actualMinutes = int(s.EndTime.Sub(s.StartTime).Minutes())
			}
			if actualMinutes == 0 {
				actualMinutes = s.Duration
			}
			dayStats.TotalMinutes += actualMinutes
		}
		stats.DailyStats = append(stats.DailyStats, dayStats)
	}

	return stats, nil
}

func (s *Storage) GetMonthStats(year int, month int) (models.MonthStats, error) {
	sessions, err := s.GetMonthSessions(year, month)
	if err != nil {
		return models.MonthStats{}, err
	}

	monthStr := fmt.Sprintf("%04d-%02d", year, month)
	completedCount := 0
	totalMinutes := 0
	weekMap := make(map[int][]models.Session)

	for _, session := range sessions {
		if session.Completed {
			completedCount++
			// Use actual time spent
			actualMinutes := session.ElapsedSeconds / 60
			if actualMinutes == 0 && !session.EndTime.IsZero() && !session.StartTime.IsZero() {
				actualMinutes = int(session.EndTime.Sub(session.StartTime).Minutes())
			}
			if actualMinutes == 0 {
				actualMinutes = session.Duration
			}
			totalMinutes += actualMinutes
			weekMap[session.Week] = append(weekMap[session.Week], session)
		}
	}

	stats := models.MonthStats{
		Month:         monthStr,
		Year:          year,
		SessionsCount: completedCount,
		TotalMinutes:  totalMinutes,
	}

	for week, weekSessions := range weekMap {
		weekStats := models.WeekStats{
			Week:          week,
			Year:          year,
			SessionsCount: len(weekSessions),
		}
		for _, s := range weekSessions {
			// Use actual time spent for weekly stats in month view too
			actualMinutes := s.ElapsedSeconds / 60
			if actualMinutes == 0 && !s.EndTime.IsZero() && !s.StartTime.IsZero() {
				actualMinutes = int(s.EndTime.Sub(s.StartTime).Minutes())
			}
			if actualMinutes == 0 {
				actualMinutes = s.Duration
			}
			weekStats.TotalMinutes += actualMinutes
		}
		stats.WeeklyStats = append(stats.WeeklyStats, weekStats)
	}

	return stats, nil
}

func (s *Storage) GetYearStats(year int) (models.YearStats, error) {
	sessions, err := s.GetYearSessions(year)
	if err != nil {
		return models.YearStats{}, err
	}

	completedCount := 0
	totalMinutes := 0
	monthMap := make(map[int][]models.Session)

	for _, session := range sessions {
		if session.Completed {
			completedCount++
			// Use actual time spent
			actualMinutes := session.ElapsedSeconds / 60
			if actualMinutes == 0 && !session.EndTime.IsZero() && !session.StartTime.IsZero() {
				actualMinutes = int(session.EndTime.Sub(session.StartTime).Minutes())
			}
			if actualMinutes == 0 {
				actualMinutes = session.Duration
			}
			totalMinutes += actualMinutes

			// Extract month from session.Month (YYYY-MM format)
			var month int
			fmt.Sscanf(session.Month, "%4d-%02d", &year, &month)
			monthMap[month] = append(monthMap[month], session)
		}
	}

	stats := models.YearStats{
		Year:          year,
		SessionsCount: completedCount,
		TotalMinutes:  totalMinutes,
	}

	// Generate monthly stats for each month that has sessions
	for month := range monthMap {
		monthStats, err := s.GetMonthStats(year, month)
		if err != nil {
			continue // Skip this month if there's an error
		}
		stats.MonthlyStats = append(stats.MonthlyStats, monthStats)
	}

	return stats, nil
}

func (s *Storage) ResetAllData() error {
	// Remove sessions file
	if err := os.Remove(s.sessionsFile()); err != nil && !os.IsNotExist(err) {
		return err
	}

	// Remove config file
	if err := os.Remove(s.configFile()); err != nil && !os.IsNotExist(err) {
		return err
	}

	return nil
}

func (s *Storage) IsFirstTime() bool {
	// Check if config file exists
	if _, err := os.Stat(s.configFile()); os.IsNotExist(err) {
		return true
	}
	return false
}

func (s *Storage) ExportAllStats() (string, error) {
	allSessions, err := s.GetAllSessions()
	if err != nil {
		return "", err
	}

	now := time.Now()
	report := fmt.Sprintf("Focus Sessions - Statistics Report\n")
	report += fmt.Sprintf("Generated: %s\n", now.Format("January 2, 2006 3:04 PM"))
	report += fmt.Sprintf("=====================================\n\n")

	// Overall statistics
	totalSessions := 0
	completedSessions := 0
	totalMinutes := 0

	for _, session := range allSessions {
		totalSessions++
		if session.Completed {
			completedSessions++
			actualMinutes := session.ElapsedSeconds / 60
			if actualMinutes == 0 && !session.EndTime.IsZero() && !session.StartTime.IsZero() {
				actualMinutes = int(session.EndTime.Sub(session.StartTime).Minutes())
			}
			if actualMinutes == 0 {
				actualMinutes = session.Duration
			}
			totalMinutes += actualMinutes
		}
	}

	report += fmt.Sprintf("OVERALL STATISTICS\n")
	report += fmt.Sprintf("------------------\n")
	report += fmt.Sprintf("Total Sessions: %d\n", totalSessions)
	report += fmt.Sprintf("Completed Sessions: %d\n", completedSessions)

	hours := totalMinutes / 60
	mins := totalMinutes % 60
	if hours > 0 {
		report += fmt.Sprintf("Total Focus Time: %dh %dm\n", hours, mins)
	} else {
		report += fmt.Sprintf("Total Focus Time: %dm\n", mins)
	}

	if completedSessions > 0 {
		avgMinutes := totalMinutes / completedSessions
		report += fmt.Sprintf("Average Session Duration: %d minutes\n", avgMinutes)
	}
	report += fmt.Sprintf("\n")

	// Year Statistics
	yearMap := make(map[int]models.YearStats)
	for _, session := range allSessions {
		if session.Completed {
			if _, exists := yearMap[session.Year]; !exists {
				yearStats, _ := s.GetYearStats(session.Year)
				yearMap[session.Year] = yearStats
			}
		}
	}

	for year, yearStats := range yearMap {
		report += fmt.Sprintf("YEAR %d\n", year)
		report += fmt.Sprintf("--------\n")
		report += fmt.Sprintf("Sessions: %d\n", yearStats.SessionsCount)

		hours := yearStats.TotalMinutes / 60
		mins := yearStats.TotalMinutes % 60
		if hours > 0 {
			report += fmt.Sprintf("Total Time: %dh %dm\n", hours, mins)
		} else {
			report += fmt.Sprintf("Total Time: %dm\n", mins)
		}

		avgPerDay := float64(yearStats.SessionsCount) / 365.0
		report += fmt.Sprintf("Average: %.1f sessions per day\n", avgPerDay)
		report += fmt.Sprintf("\n")

		// Monthly breakdown for the year
		for _, monthStats := range yearStats.MonthlyStats {
			monthTime, _ := time.Parse("2006-01", monthStats.Month)
			report += fmt.Sprintf("  %s:\n", monthTime.Format("January"))
			report += fmt.Sprintf("    Sessions: %d\n", monthStats.SessionsCount)

			hours := monthStats.TotalMinutes / 60
			mins := monthStats.TotalMinutes % 60
			if hours > 0 {
				report += fmt.Sprintf("    Total Time: %dh %dm\n", hours, mins)
			} else {
				report += fmt.Sprintf("    Total Time: %dm\n", mins)
			}
		}
		report += fmt.Sprintf("\n")
	}

	// Recent Week Statistics
	_, currentWeek := now.ISOWeek()
	weekStats, err := s.GetWeekStats(now.Year(), currentWeek)
	if err == nil && weekStats.SessionsCount > 0 {
		report += fmt.Sprintf("CURRENT WEEK (Week %d, %d)\n", weekStats.Week, weekStats.Year)
		report += fmt.Sprintf("------------------------\n")
		report += fmt.Sprintf("Sessions: %d\n", weekStats.SessionsCount)

		hours := weekStats.TotalMinutes / 60
		mins := weekStats.TotalMinutes % 60
		if hours > 0 {
			report += fmt.Sprintf("Total Time: %dh %dm\n", hours, mins)
		} else {
			report += fmt.Sprintf("Total Time: %dm\n", mins)
		}

		for _, dayStats := range weekStats.DailyStats {
			date, _ := time.Parse("2006-01-02", dayStats.Date)
			hours := dayStats.TotalMinutes / 60
			mins := dayStats.TotalMinutes % 60
			var timeStr string
			if hours > 0 {
				timeStr = fmt.Sprintf("%dh %dm", hours, mins)
			} else {
				timeStr = fmt.Sprintf("%dm", mins)
			}
			report += fmt.Sprintf("  %s: %d sessions (%s)\n", date.Format("Monday"), dayStats.SessionsCount, timeStr)
		}
		report += fmt.Sprintf("\n")
	}

	// Today's Statistics
	todayStats, err := s.GetDayStats(now.Format("2006-01-02"))
	if err == nil && todayStats.SessionsCount > 0 {
		report += fmt.Sprintf("TODAY (%s)\n", now.Format("Monday, January 2, 2006"))
		report += fmt.Sprintf("-------------------------------\n")
		report += fmt.Sprintf("Sessions: %d\n", todayStats.SessionsCount)

		hours := todayStats.TotalMinutes / 60
		mins := todayStats.TotalMinutes % 60
		if hours > 0 {
			report += fmt.Sprintf("Total Time: %dh %dm\n", hours, mins)
		} else {
			report += fmt.Sprintf("Total Time: %dm\n", mins)
		}

		report += fmt.Sprintf("\nSession Details:\n")
		for i, session := range todayStats.Sessions {
			if session.Completed {
				report += fmt.Sprintf("  Session %d: %s - %s (%d min)\n",
					i+1,
					session.StartTime.Format("3:04 PM"),
					session.EndTime.Format("3:04 PM"),
					session.Duration,
				)
			}
		}
	}

	return report, nil
}
