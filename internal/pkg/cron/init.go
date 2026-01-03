package cron

import log "log/slog"

func InitCron(mgr *Manager) error {
	log.Info("Cron Jobs starting...")
	if err := mgr.RegisterJobs(); err != nil {
		return err
	}
	mgr.Start()
	return nil
}
