package reboot

import (
	igntypes "github.com/coreos/ignition/config/v2_2/types"
	mapset "github.com/deckarep/golang-set"
)

type UnitOperation string

const (
	unitRestart UnitOperation = "restart"
	unitReload  UnitOperation = "reload"
)

type FileChange struct {
	name       string
	file       igntypes.File
	changeType ChangeType
}

type UnitChange struct {
	name       string
	oldUnit    igntypes.Unit
	newUnit    igntypes.Unit
	changeType ChangeType
}

type ChangeType string

const (
	changeCreated ChangeType = "created"
	changeDeleted ChangeType = "deleted"
	changeUpdated ChangeType = "updated"
)

func getUnitNames(units []igntypes.Unit) []interface{} {
	names := make([]interface{}, len(units))
	for i, unit := range units {
		names[i] = unit.Name
	}
	return names
}

func unitsToMap(units []igntypes.Unit) map[string]igntypes.Unit {
	unitMap := make(map[string]igntypes.Unit, len(units))
	for _, unit := range units {
		unitMap[unit.Name] = unit
	}
	return unitMap
}

func getUnitsChanges(oldUnitsConfig, newUnitsConfig []igntypes.Unit) []*UnitChange {
	oldUnits := mapset.NewSetFromSlice(getUnitNames(oldUnitsConfig))
	oldUnitsMap := unitsToMap(oldUnitsConfig)
	newUnits := mapset.NewSetFromSlice(getUnitNames(newUnitsConfig))
	newUnitsMap := unitsToMap(newUnitsConfig)
	changes := make([]*UnitChange, 0, newUnits.Cardinality())
	for created := range newUnits.Difference(oldUnits).Iter() {
		changes = append(changes, &UnitChange{
			name:       created.(string),
			newUnit:    newUnitsMap[created.(string)],
			changeType: changeCreated,
		})
	}
	for deleted := range oldUnits.Difference(newUnits).Iter() {
		changes = append(changes, &UnitChange{
			name:       deleted.(string),
			oldUnit:    oldUnitsMap[deleted.(string)],
			changeType: changeDeleted,
		})
	}
	for changeCandidate := range newUnits.Intersect(oldUnits).Iter() {
		changedUnitName := changeCandidate.(string)
		newUnit := newUnitsMap[changedUnitName]
		oldUnit := oldUnitsMap[changedUnitName]
		// if !reflect.DeepEqual(newUnit, oldUnit) {
		// FIXME refactor this func out of daemon.go
		if !checkUnits([]igntypes.Unit{newUnit}) {
			changes = append(changes, &UnitChange{
				name:       changedUnitName,
				newUnit:    newUnit,
				oldUnit:    oldUnit,
				changeType: changeUpdated,
			})
		}
	}
	return changes
}

func getFileNames(files []igntypes.File) []interface{} {
	names := make([]interface{}, len(files))
	for i, file := range files {
		names[i] = file.Path
	}
	return names
}

func filesToMap(files []igntypes.File) map[string]igntypes.File {
	fileMap := make(map[string]igntypes.File, len(files))
	for _, file := range files {
		fileMap[file.Path] = file
	}
	return fileMap
}

func getFilesChanges(oldFilesConfig, newFilesConfig []igntypes.File) []*FileChange {
	oldFiles := mapset.NewSetFromSlice(getFileNames(oldFilesConfig))
	oldFilesMap := filesToMap(oldFilesConfig)
	newFiles := mapset.NewSetFromSlice(getFileNames(newFilesConfig))
	newFilesMap := filesToMap(newFilesConfig)
	changes := make([]*FileChange, 0, newFiles.Cardinality())
	for created := range newFiles.Difference(oldFiles).Iter() {
		changes = append(changes, &FileChange{
			name:       created.(string),
			file:       newFilesMap[created.(string)],
			changeType: changeCreated,
		})
	}
	for deleted := range oldFiles.Difference(newFiles).Iter() {
		changes = append(changes, &FileChange{
			name:       deleted.(string),
			file:       oldFilesMap[deleted.(string)],
			changeType: changeDeleted,
		})
	}
	for changeCandidate := range newFiles.Intersect(oldFiles).Iter() {
		newFile := newFilesMap[changeCandidate.(string)]
		// if !reflect.DeepEqual(newFile, oldFilesMap[changeCandidate.(string)]) {
		// FIXME refactor this func out of daemon.go
		if !checkFiles([]igntypes.File{newFile}) {
			changes = append(changes, &FileChange{
				name:       changeCandidate.(string),
				file:       newFile,
				changeType: changeUpdated,
			})
		}
	}
	return changes
}
