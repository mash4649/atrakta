package main

import (
	"path/filepath"

	"github.com/mash4649/atrakta/v0/internal/audit"
)

const (
	runEventInitBegin             = "init.begin"
	runEventInitStep              = "init.step"
	runEventInitEnd               = "init.end"
	runEventStartBegin            = "start.begin"
	runEventStartEnd              = "start.end"
	runEventResumeBegin           = "resume.begin"
	runEventResumeEnd             = "resume.end"
	runEventApplyBegin            = "apply.begin"
	runEventApplyPerformed        = "apply.performed"
	runEventProjectionRendered    = "projection.rendered"
	runEventProjectionStatus      = "projection.status"
	runEventProjectionStatusCheck = "projection_status_check"
	runEventProjectionRepaired    = "projection_repair"
	runEventGCRun                 = "gc_run"
	runEventMigrateChecked        = "migrate.checked"
	runEventWrapInstall           = "wrap_install"
	runEventWrapUninstall         = "wrap_uninstall"
	runEventWrapRun               = "wrap_run"
	runEventHookInstall           = "hook_install"
	runEventHookUninstall         = "hook_uninstall"
	runEventHookStatusCheck       = "hook_status_check"
	runEventHookRepair            = "hook_repair"
	runEventIDEAutostartInstall   = "ide_autostart_install"
)

func appendOperationalRunEvent(projectRoot, eventType, interfaceID string, payload map[string]any) error {
	if payload == nil {
		payload = map[string]any{}
	}
	auditRoot := filepath.Join(projectRoot, ".atrakta", "audit")
	_, err := audit.AppendRunEventAndVerify(auditRoot, audit.LevelA2, eventType, payload, audit.RunEventOptions{
		Actor:     "kernel",
		Interface: interfaceID,
	})
	return err
}
