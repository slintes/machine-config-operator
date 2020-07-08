package reboot

import (
	"fmt"
	"os/exec"

	systemdDbus "github.com/coreos/go-systemd/dbus"
	"github.com/golang/glog"
)

type PostUpdateAction interface {
	Run() error
	IsDrainRequired() bool
}

type DrainRequired bool

func (d DrainRequired) IsDrainRequired() bool {
	return bool(d)
}

type RunBinaryAction struct {
	binary string
	args   []string
	DrainRequired
}

func (action RunBinaryAction) Run() error {
	glog.Infof(
		"Running post update action: running command: %v %v", action.binary, action.args,
	)
	output, err := exec.Command(action.binary, action.args...).CombinedOutput()
	// TODO: Add some timeout?
	if err != nil {
		glog.Errorf("Running post update action (running command: '%s %s') failed: %s; command output: %s", action.binary, action.args, err, output)
		return err
	}
	return nil
}

type SystemdAction struct {
	unitName          string
	operation         UnitOperation
	enabled           bool
	systemdConnection *systemdDbus.Conn
	DrainRequired
}

func (action SystemdAction) Run() error {
	// TODO: add support for reload operation
	// For now only restart operation is supported
	if action.systemdConnection == nil {
		return fmt.Errorf(
			"Unable to run post update action for unit %q: systemd dbus connection not specified",
			action.unitName,
		)
	}
	var err error
	outputChannel := make(chan string)
	if action.enabled {
		glog.Infof("Restarting unit %q", action.unitName)
		_, err = action.systemdConnection.RestartUnit(action.unitName, "replace", outputChannel)
	} else {
		glog.Infof("Stopping unit %q", action.unitName)
		_, err = action.systemdConnection.StopUnit(action.unitName, "replace", outputChannel)
	}
	if err != nil {
		return fmt.Errorf("Running systemd action failed: %s", err)
	}
	output := <-outputChannel

	switch output {
	case "done":
		fallthrough
	case "skipped":
		glog.Infof("Systemd action successful: %s", output)
	default:
		return fmt.Errorf("Systemd action %s", output)
	}
	return nil
}

func getPostUpdateActions(filesChanges []*FileChange, unitsChanges []*UnitChange, systemdConnection *systemdDbus.Conn) ([]PostUpdateAction, error) {
	glog.Info("Trying to check whether changes in files and units require system reboot.")
	actions := make([]PostUpdateAction, 0, len(filesChanges)+len(unitsChanges))
	rebootRequiredMsg := ", reboot will be required"
	for _, change := range filesChanges {
		switch change.changeType {
		case changeCreated:
			fallthrough
		case changeUpdated:
			action := filterConfig.getFileAction(change.name)
			if action == nil {
				err := fmt.Errorf("No action found for file %q", change.name)
				glog.Infof("%s%s", err, rebootRequiredMsg)
				return nil, err
			}
			actions = append(actions, action)
			glog.Infof("Action found for file %q", change.name)
		default:
			err := fmt.Errorf("File %q was removed", change.name)
			glog.Infof("%s%s", err, rebootRequiredMsg)
			return nil, err
		}
	}

	for _, change := range unitsChanges {
		switch change.changeType {
		case changeCreated:
			fallthrough
		case changeUpdated:
			action := filterConfig.getUnitAction(change.newUnit, systemdConnection)
			if action == nil {
				err := fmt.Errorf("No action found for unit %q", change.name)
				glog.Infof("%s%s", err, rebootRequiredMsg)
				return nil, err
			}
			if systemdConnection == nil {
				err := fmt.Errorf(
					"Missing systemd connection for running post update action for unit %q",
					change.name,
				)
				glog.Errorf("%s%s", err, rebootRequiredMsg)
				return nil, err
			}
			actions = append(actions, action)
			glog.Infof("Action found for unit %q", change.name)
		default:
			err := fmt.Errorf("Unit %q was removed", change.name)
			glog.Infof("%s%s", err, rebootRequiredMsg)
			return nil, err
		}
	}
	return actions, nil
}
