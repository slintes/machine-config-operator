package reboot

import (
	systemdDbus "github.com/coreos/go-systemd/dbus"
	"github.com/coreos/ignition/config/v2_2/types"
	"github.com/golang/glog"
)

type Filter struct {
	oldIgnConfig     types.Config
	newIgnConfig     types.Config
	actions          []PostUpdateAction
	IsDrainRequired  bool
	IsRebootRequired bool
}

func NewFilter(oldIgnConfig, newIgnConfig types.Config) *Filter {
	filter := &Filter{
		oldIgnConfig: oldIgnConfig,
		newIgnConfig: newIgnConfig,
	}

	postUpdateActions, err := getPostUpdateActions(
		getFilesChanges(oldIgnConfig.Storage.Files, newIgnConfig.Storage.Files),
		getUnitsChanges(oldIgnConfig.Systemd.Units, newIgnConfig.Systemd.Units),
		getSystemDCon(),
	)
	if err != nil {
		filter.IsRebootRequired = true
	}

	filter.actions = postUpdateActions
	filter.IsDrainRequired = filter.calcDrainRequired()

	return filter
}

func (f Filter) calcDrainRequired() bool {
	isRequired := false
	for _, action := range f.actions {
		isRequired = isRequired || action.IsDrainRequired()
	}
	return isRequired
}

// returns true in case reboot is required (some actions failed), false otherwise
func (f Filter) RunPostUpdateActions() bool {
	glog.Infof("Running %d post update action(s)...", len(f.actions))
	for _, action := range f.actions {
		if err := action.Run(); err != nil {
			glog.Errorf("Post update action failed: %s", err)
			return true
		}
	}
	glog.Info("Running post update Actions were successful")
	return false
}

func getSystemDCon() *systemdDbus.Conn {
	systemdConnection, dbusConnErr := systemdDbus.NewSystemConnection()
	if dbusConnErr == nil {
		defer systemdConnection.Close()
	} else {
		glog.Warningf("Unable to establish systemd dbus connection: %s", dbusConnErr)
		// No more actions needed here as a systemd connection is not always
		// required (only if there is systemd related post update action
		// present). If a connection should be required, getPostUpdateActions
		// function will return error if nil connection is provided and then
		// rebootRequired will be se to true
	}
	return systemdConnection
}
