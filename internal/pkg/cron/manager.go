package cron

import (
	"Cornerstone/internal/job"
	log "log/slog"

	"github.com/robfig/cron/v3"
)

type Manager struct {
	engine          *cron.Cron
	userMetricJob   *job.UserMetricsJob
	postMetricJob   *job.PostMetricsJob
	userInterestJob *job.UserInterestJob
}

func NewCronManager(
	userMetricJob *job.UserMetricsJob,
	postMetricJob *job.PostMetricsJob,
	userInterestJob *job.UserInterestJob,

) *Manager {
	return &Manager{
		engine:          cron.New(cron.WithSeconds()),
		userMetricJob:   userMetricJob,
		postMetricJob:   postMetricJob,
		userInterestJob: userInterestJob,
	}
}

// RegisterJobs 注册定时任务
func (s *Manager) RegisterJobs() error {
	if _, err := s.engine.AddJob("@daily", s.userMetricJob); err != nil {
		return err
	}
	if _, err := s.engine.AddJob("@daily", s.postMetricJob); err != nil {
		return err
	}
	if _, err := s.engine.AddJob("@every 12h", s.userInterestJob); err != nil {
		return err
	}
	return nil
}

func (s *Manager) Start() {
	log.Info("Cron 定时任务引擎启动")
	s.engine.Start()
}

func (s *Manager) Stop() {
	log.Info("Cron 定时任务引擎停止")
	s.engine.Stop()
}
