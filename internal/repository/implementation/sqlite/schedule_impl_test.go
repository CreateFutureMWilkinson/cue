package sqlite_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/CreateFutureMWilkinson/cue/internal/repository"
	"github.com/CreateFutureMWilkinson/cue/internal/repository/implementation/sqlite"
	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
)

type ScheduleRepositorySuite struct {
	suite.Suite
	repo *sqlite.SQLiteScheduleRepository
}

func TestScheduleRepository(t *testing.T) {
	suite.Run(t, new(ScheduleRepositorySuite))
}

func (s *ScheduleRepositorySuite) SetupTest() {
	dbPath := filepath.Join(s.T().TempDir(), "test.db")
	repo, err := sqlite.NewSQLiteScheduleRepository(dbPath)
	s.Require().NoError(err)
	s.repo = repo
}

func (s *ScheduleRepositorySuite) sampleSchedule() *repository.Schedule {
	taskID := uuid.New()
	return &repository.Schedule{
		ID:       uuid.New(),
		Date:     time.Date(2026, time.March, 30, 0, 0, 0, 0, time.UTC),
		Strategy: "focus-maximized",
		Blocks: []repository.ScheduleBlock{
			{
				Start:    time.Date(2026, time.March, 30, 9, 0, 0, 0, time.UTC),
				End:      time.Date(2026, time.March, 30, 9, 25, 0, 0, time.UTC),
				Type:     repository.ScheduleBlockFocus,
				TaskID:   &taskID,
				TaskName: "Write tests",
			},
			{
				Start:    time.Date(2026, time.March, 30, 9, 25, 0, 0, time.UTC),
				End:      time.Date(2026, time.March, 30, 9, 30, 0, 0, time.UTC),
				Type:     repository.ScheduleBlockShortBreak,
				TaskName: "",
			},
			{
				Start:    time.Date(2026, time.March, 30, 10, 0, 0, 0, time.UTC),
				End:      time.Date(2026, time.March, 30, 10, 30, 0, 0, time.UTC),
				Type:     repository.ScheduleBlockMeeting,
				TaskName: "Standup",
			},
		},
		CreatedAt: time.Date(2026, time.March, 30, 8, 0, 0, 0, time.UTC),
	}
}

// ---------------------------------------------------------------------------
// 1. Save and LoadByDate — round-trip
// ---------------------------------------------------------------------------

func (s *ScheduleRepositorySuite) TestSaveAndLoadByDate() {
	ctx := context.Background()
	sched := s.sampleSchedule()

	err := s.repo.Save(ctx, sched)
	s.Require().NoError(err)

	loaded, err := s.repo.LoadByDate(ctx, sched.Date)
	s.Require().NoError(err)
	s.Require().NotNil(loaded)

	s.Equal(sched.ID, loaded.ID)
	s.Equal(sched.Strategy, loaded.Strategy)
	s.Equal(sched.Date.UTC(), loaded.Date.UTC())
	s.Require().Equal(len(sched.Blocks), len(loaded.Blocks))

	for i, b := range sched.Blocks {
		lb := loaded.Blocks[i]
		s.Equal(b.Start.UTC(), lb.Start.UTC())
		s.Equal(b.End.UTC(), lb.End.UTC())
		s.Equal(b.Type, lb.Type)
		s.Equal(b.TaskName, lb.TaskName)
		if b.TaskID != nil {
			s.Require().NotNil(lb.TaskID)
			s.Equal(*b.TaskID, *lb.TaskID)
		} else {
			s.Nil(lb.TaskID)
		}
	}
}

// ---------------------------------------------------------------------------
// 2. LoadByDate — not found
// ---------------------------------------------------------------------------

func (s *ScheduleRepositorySuite) TestLoadByDateNotFound() {
	ctx := context.Background()
	date := time.Date(2026, time.April, 1, 0, 0, 0, 0, time.UTC)

	loaded, err := s.repo.LoadByDate(ctx, date)
	s.Require().Error(err)
	s.Nil(loaded)
	s.ErrorIs(err, repository.ErrNotFound)
}

// ---------------------------------------------------------------------------
// 3. Delete — removes schedule
// ---------------------------------------------------------------------------

func (s *ScheduleRepositorySuite) TestDelete() {
	ctx := context.Background()
	sched := s.sampleSchedule()

	err := s.repo.Save(ctx, sched)
	s.Require().NoError(err)

	err = s.repo.Delete(ctx, sched.ID)
	s.Require().NoError(err)

	loaded, err := s.repo.LoadByDate(ctx, sched.Date)
	s.Require().Error(err)
	s.Nil(loaded)
	s.ErrorIs(err, repository.ErrNotFound)
}

// ---------------------------------------------------------------------------
// 4. Overwrite existing — Save same date replaces schedule
// ---------------------------------------------------------------------------

func (s *ScheduleRepositorySuite) TestOverwriteExisting() {
	ctx := context.Background()
	sched1 := s.sampleSchedule()

	err := s.repo.Save(ctx, sched1)
	s.Require().NoError(err)

	sched2 := &repository.Schedule{
		ID:       uuid.New(),
		Date:     sched1.Date, // Same date
		Strategy: "recovery-balanced",
		Blocks: []repository.ScheduleBlock{
			{
				Start:    time.Date(2026, time.March, 30, 9, 0, 0, 0, time.UTC),
				End:      time.Date(2026, time.March, 30, 9, 25, 0, 0, time.UTC),
				Type:     repository.ScheduleBlockFocus,
				TaskName: "New task",
			},
		},
		CreatedAt: time.Date(2026, time.March, 30, 9, 0, 0, 0, time.UTC),
	}

	err = s.repo.Save(ctx, sched2)
	s.Require().NoError(err)

	loaded, err := s.repo.LoadByDate(ctx, sched1.Date)
	s.Require().NoError(err)
	s.Equal(sched2.ID, loaded.ID, "should be the new schedule")
	s.Equal("recovery-balanced", loaded.Strategy)
	s.Equal(1, len(loaded.Blocks), "should have new schedule's blocks")
}

// ---------------------------------------------------------------------------
// 5. Save with empty blocks
// ---------------------------------------------------------------------------

func (s *ScheduleRepositorySuite) TestSaveEmptyBlocks() {
	ctx := context.Background()
	sched := &repository.Schedule{
		ID:        uuid.New(),
		Date:      time.Date(2026, time.March, 30, 0, 0, 0, 0, time.UTC),
		Strategy:  "focus-maximized",
		Blocks:    nil,
		CreatedAt: time.Date(2026, time.March, 30, 8, 0, 0, 0, time.UTC),
	}

	err := s.repo.Save(ctx, sched)
	s.Require().NoError(err)

	loaded, err := s.repo.LoadByDate(ctx, sched.Date)
	s.Require().NoError(err)
	s.Empty(loaded.Blocks)
}
