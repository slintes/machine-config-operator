package reboot

import (
	"path/filepath"

	systemdDbus "github.com/coreos/go-systemd/dbus"
	igntypes "github.com/coreos/ignition/config/v2_2/types"
)

type FileFilterEntry struct {
	glob             string
	postUpdateAction PostUpdateAction
}

type UnitFilterEntry struct {
	name          string
	drainRequired DrainRequired
}

type AvoidRebootConfig struct {
	// Files filter which do not require reboot
	Files []*FileFilterEntry
	// List of systemd unit that do not require system reboot, but rather just unit restart
	Units []*UnitFilterEntry
}

func (config AvoidRebootConfig) getFileAction(filePath string) PostUpdateAction {
	for _, entry := range config.Files {
		matched, err := filepath.Match(entry.glob, filePath)
		if err != nil {
			// TODO: log
			continue
		}
		if matched {
			return entry.postUpdateAction
		}
	}
	return nil
}

func (config AvoidRebootConfig) getUnitAction(unit igntypes.Unit, systemdConnection *systemdDbus.Conn) PostUpdateAction {
	for _, entry := range config.Units {
		if entry.name == unit.Name {
			// same logic like in writeUnits()
			enableUnit := false
			if unit.Enable {
				enableUnit = true
			} else if unit.Enabled != nil {
				enableUnit = *unit.Enabled
			}
			return SystemdAction{
				unit.Name,
				unitRestart,
				enableUnit,
				systemdConnection,
				entry.drainRequired,
			}
		}
	}
	return nil
}

// TODO: create a proper filter config as this one is just a testing one
var filterConfig = AvoidRebootConfig{
	Files: []*FileFilterEntry{
		&FileFilterEntry{
			glob: "/home/core/testfile",
			postUpdateAction: RunBinaryAction{
				binary: "/bin/bash",
				args: []string{
					"-c",
					"echo \"$(date)\" >> /home/core/testfile.out",
				},
				DrainRequired: false,
			},
		},
		&FileFilterEntry{
			glob: "/home/core/drain_required",
			postUpdateAction: RunBinaryAction{
				binary: "/bin/bash",
				args: []string{
					"-c",
					"echo \"$(date)\" >> /home/core/drain_required.out",
				},
				DrainRequired: true,
			},
		},
	},
	Units: []*UnitFilterEntry{
		&UnitFilterEntry{
			name:          "test-service-drain.service",
			drainRequired: true,
		},
		&UnitFilterEntry{
			name:          "test-service.service",
			drainRequired: false,
		},
	},
}
