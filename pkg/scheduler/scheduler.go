package scheduler

import (
	"fmt"
	"managed-usb-hub-wails/pkg/config"
	"sync"
	"time"
)

// Executor defines the function to execute a task (apply specific mask)
type Executor func(deviceID string, mask string) error

// Scheduler manages scheduled tasks
type Scheduler struct {
	tasks    []config.ScheduledTask
	mu       sync.Mutex
	executor Executor
	ticker   *time.Ticker
	quit     chan struct{}
}

// NewScheduler creates a new Scheduler
func NewScheduler(executor Executor) *Scheduler {
	return &Scheduler{
		tasks:    config.GetScheduledTasks(),
		executor: executor,
		quit:     make(chan struct{}),
	}
}

// ReloadTasks reloads tasks from configuration
func (s *Scheduler) ReloadTasks() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tasks = config.GetScheduledTasks()
}

// Start begins the scheduler loop
func (s *Scheduler) Start() {
	// If already running, do nothing
	if s.ticker != nil {
		return
	}

	// Create quit channel if needed
	if s.quit == nil {
		s.quit = make(chan struct{})
	}

	// Calculate time until next minute
	now := time.Now()
	nextMinute := now.Truncate(time.Minute).Add(time.Minute)
	duration := nextMinute.Sub(now)

	fmt.Printf("[Scheduler] Starting... Next check in %v\n", duration)

	// Wait until the start of the next minute
	time.AfterFunc(duration, func() {
		fmt.Println("[Scheduler] Ticker started aligned to minute")
		// Start the ticker aligned to the minute
		// Ticker will fire exactly at :00 seconds of each minute (give or take a few ms)
		s.ticker = time.NewTicker(1 * time.Minute)

		// Execute immediately for the first time (because we just hit the minute mark)
		s.checkTasks()

		go func() {
			for {
				select {
				case <-s.ticker.C:
					// No sleep needed here.
					// time.Ticker guarantees it fires AFTER the duration has elapsed.
					// So if we align it to the minute boundary, it should fire just after :00.
					// We can re-check the time inside checkTasks to be sure.
					s.checkTasks()
				case <-s.quit:
					s.ticker.Stop()
					return
				}
			}
		}()
	})
}

// Stop stops the scheduler
func (s *Scheduler) Stop() {
	close(s.quit)
}

// AddTask adds a new task
func (s *Scheduler) AddTask(task config.ScheduledTask) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check for duplicate ID
	for _, t := range s.tasks {
		if t.ID == task.ID {
			return fmt.Errorf("task with ID %s already exists", task.ID)
		}
	}

	s.tasks = append(s.tasks, task)
	return config.SaveScheduledTasks(s.tasks)
}

// RemoveTask removes a task by ID
func (s *Scheduler) RemoveTask(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	newTasks := []config.ScheduledTask{}
	found := false
	for _, t := range s.tasks {
		if t.ID == id {
			found = true
			continue
		}
		newTasks = append(newTasks, t)
	}

	if !found {
		return fmt.Errorf("task not found")
	}

	s.tasks = newTasks
	return config.SaveScheduledTasks(s.tasks)
}

// UpdateTask updates an existing task
func (s *Scheduler) UpdateTask(task config.ScheduledTask) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	found := false
	for i, t := range s.tasks {
		if t.ID == task.ID {
			s.tasks[i] = task
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("task not found")
	}

	return config.SaveScheduledTasks(s.tasks)
}

// GetTasks returns all tasks
func (s *Scheduler) GetTasks() []config.ScheduledTask {
	s.mu.Lock()
	defer s.mu.Unlock()
	// Return copy
	tasks := make([]config.ScheduledTask, len(s.tasks))
	copy(tasks, s.tasks)
	return tasks
}

// checkTasks iterates over tasks and executes them if time matches
func (s *Scheduler) checkTasks() {
	s.mu.Lock()
	tasks := make([]config.ScheduledTask, len(s.tasks))
	copy(tasks, s.tasks)
	s.mu.Unlock()

	now := time.Now()
	// Round to nearest minute to handle slight deviations (e.g. 10:09:59.999 or 10:10:00.001)
	// But since we aligned ticker, it should be close to :00 seconds.
	// However, if the system was busy and ticker fired late (e.g. 10:10:01), we still want 10:10.
	// If it fired slightly early (very rare with Ticker but possible with Sleep alignment), we might want to round.
	// Best approach: Just use Hour and Minute directly. Ticker fires *at or after* the tick.
	// So if we set it to 1 minute interval aligned to :00, it will fire at 10:10:00.00something.
	
	currentDay := int(now.Weekday()) // 0=Sunday
	currentHour := now.Hour()
	currentMin := now.Minute()

	timeStr := fmt.Sprintf("%02d:%02d", currentHour, currentMin)
	fmt.Printf("[Scheduler] Checking tasks at %s (Day: %d)\n", timeStr, currentDay)

	for _, task := range tasks {
		if !task.Enabled {
			continue
		}

		// Check Day
		dayMatch := false
		for _, d := range task.DaysOfWeek {
			if d == currentDay {
				dayMatch = true
				break
			}
		}
		if !dayMatch {
			continue
		}

		// Check Time
		// Use seconds resolution to prevent duplicate execution within the same minute if needed,
		// but here we just check equality with current minute.
		// Since ticker runs once per minute, we don't need to worry about multiple executions.
		if task.StartTime == timeStr {
			fmt.Printf("Executing Start Task for %s\n", task.DeviceID)
			mask := task.StartMask
			if mask == "" {
				mask = "FFFFFFFF" // Default All On
			}
			go s.executor(task.DeviceID, mask)
		} 
		
		if task.StopTime == timeStr {
			fmt.Printf("Executing Stop Task for %s\n", task.DeviceID)
			mask := task.StopMask
			if mask == "" {
				mask = "00000000" // Default All Off
			}
			go s.executor(task.DeviceID, mask)
		}
	}
}
