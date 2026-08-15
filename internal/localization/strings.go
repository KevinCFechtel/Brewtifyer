package localization

import (
	"time"

	"golang.org/x/text/language"
)

func (strings *Strings) TrayTooltip() string { return strings.localize(messageTrayTooltip, nil, nil) }
func (strings *Strings) Checking() string    { return strings.localize(messageTrayChecking, nil, nil) }
func (strings *Strings) CurrentStatusTooltip() string {
	return strings.localize(messageTrayCurrentStatusTooltip, nil, nil)
}
func (strings *Strings) NotChecked() string { return strings.localize(messageTrayNotChecked, nil, nil) }
func (strings *Strings) LastCheckTooltip() string {
	return strings.localize(messageTrayLastCheckTooltip, nil, nil)
}
func (strings *Strings) UpgradePackageMenuTooltip() string {
	return strings.localize(messageTrayUpgradePackageMenuTooltip, nil, nil)
}
func (strings *Strings) UpgradeAll() string { return strings.localize(messageTrayUpgradeAll, nil, nil) }
func (strings *Strings) UpgradeAllTooltip() string {
	return strings.localize(messageTrayUpgradeAllTooltip, nil, nil)
}
func (strings *Strings) Refresh() string { return strings.localize(messageTrayRefresh, nil, nil) }
func (strings *Strings) RefreshTooltip() string {
	return strings.localize(messageTrayRefreshTooltip, nil, nil)
}
func (strings *Strings) OpenLoginItems() string {
	return strings.localize(messageTrayOpenLoginItems, nil, nil)
}
func (strings *Strings) OpenLoginItemsTooltip() string {
	return strings.localize(messageTrayOpenLoginItemsTooltip, nil, nil)
}
func (strings *Strings) Quit() string { return strings.localize(messageTrayQuit, nil, nil) }
func (strings *Strings) QuitTooltip() string {
	return strings.localize(messageTrayQuitTooltip, nil, nil)
}
func (strings *Strings) CheckFailed() string {
	return strings.localize(messageTrayCheckFailed, nil, nil)
}
func (strings *Strings) UpToDate() string { return strings.localize(messageTrayUpToDate, nil, nil) }
func (strings *Strings) LastSuccessfulCheckTooltip() string {
	return strings.localize(messageTrayLastSuccessfulCheckTooltip, nil, nil)
}
func (strings *Strings) UpgradePackageRunning() string {
	return strings.localize(messageTrayUpgradePackageRunning, nil, nil)
}
func (strings *Strings) UpgradeAllRunning() string {
	return strings.localize(messageTrayUpgradeAllRunning, nil, nil)
}
func (strings *Strings) UpgradeLaunchFailed() string {
	return strings.localize(messageTrayUpgradeLaunchFailed, nil, nil)
}
func (strings *Strings) AutostartManageFailed() string {
	return strings.localize(messageTrayAutostartManageFailed, nil, nil)
}
func (strings *Strings) AutostartTitle() string {
	return strings.localize(messageTrayAutostartTitle, nil, nil)
}
func (strings *Strings) AutostartEnableTooltip() string {
	return strings.localize(messageTrayAutostartEnableTooltip, nil, nil)
}
func (strings *Strings) AutostartDisableTooltip() string {
	return strings.localize(messageTrayAutostartDisableTooltip, nil, nil)
}
func (strings *Strings) AutostartApprovalTitle() string {
	return strings.localize(messageTrayAutostartApprovalTitle, nil, nil)
}
func (strings *Strings) AutostartApprovalTooltip() string {
	return strings.localize(messageTrayAutostartApprovalTooltip, nil, nil)
}
func (strings *Strings) AutostartRegisterTooltip() string {
	return strings.localize(messageTrayAutostartRegisterTooltip, nil, nil)
}
func (strings *Strings) AutostartUnsupportedTitle() string {
	return strings.localize(messageTrayAutostartUnsupportedTitle, nil, nil)
}
func (strings *Strings) AutostartUnsupportedTooltip() string {
	return strings.localize(messageTrayAutostartUnsupportedTooltip, nil, nil)
}
func (strings *Strings) PackageUpgradeTooltip() string {
	return strings.localize(messageTrayPackageUpgradeTooltip, nil, nil)
}
func (strings *Strings) PinnedPackageTooltip() string {
	return strings.localize(messageTrayPinnedPackageTooltip, nil, nil)
}
func (strings *Strings) NotificationUpdateTitle() string {
	return strings.localize(messageNotificationUpdateTitle, nil, nil)
}
func (strings *Strings) UpgradeAllDescription() string {
	return strings.localize(messageUpgradeAllDescription, nil, nil)
}
func (strings *Strings) UpgradeCompleted() string {
	return strings.localize(messageUpgradeCompleted, nil, nil)
}
func (strings *Strings) UpgradeFailedFormat() string {
	return strings.localize(messageUpgradeFailed, nil, nil)
}
func (strings *Strings) UpgradePressAnyKey() string {
	return strings.localize(messageUpgradePressAnyKey, nil, nil)
}
func (strings *Strings) HomebrewNotFound() string {
	return strings.localize(messageErrorHomebrewNotFound, nil, nil)
}
func (strings *Strings) PackageNameMissing() string {
	return strings.localize(messageErrorPackageNameMissing, nil, nil)
}
func (strings *Strings) PackageNameInvalid() string {
	return strings.localize(messageErrorPackageNameInvalid, nil, nil)
}
func (strings *Strings) CreateUpgradeCommandError() string {
	return strings.localize(messageErrorCreateUpgradeCommand, nil, nil)
}
func (strings *Strings) MakeUpgradeCommandExecutableError() string {
	return strings.localize(messageErrorMakeUpgradeCommandExecutable, nil, nil)
}
func (strings *Strings) WriteUpgradeCommandError() string {
	return strings.localize(messageErrorWriteUpgradeCommand, nil, nil)
}
func (strings *Strings) CloseUpgradeCommandError() string {
	return strings.localize(messageErrorCloseUpgradeCommand, nil, nil)
}
func (strings *Strings) OpenTerminalError() string {
	return strings.localize(messageErrorOpenTerminal, nil, nil)
}

func (strings *Strings) LastChecked(checkedAt time.Time, warning bool) string {
	data := map[string]any{"Date": strings.FormatTime(checkedAt)}
	if warning {
		return strings.localize(messageTrayLastCheckedWithWarning, data, nil)
	}
	return strings.localize(messageTrayLastChecked, data, nil)
}

func (strings *Strings) MoreUpdates(count int) string {
	return strings.localize(messageTrayMoreUpdates, map[string]any{"Count": count}, count)
}

func (strings *Strings) UpdatesAvailable(count int) string {
	return strings.localize(messageTrayUpdatesAvailable, map[string]any{"Count": count}, count)
}

func (strings *Strings) PackageTitle(name, installed, current string, pinned bool) string {
	data := map[string]any{
		"Name":      name,
		"Installed": installed,
		"Current":   current,
	}
	if pinned {
		return strings.localize(messageTrayPinnedPackageTitle, data, nil)
	}
	return strings.localize(messageTrayPackageTitle, data, nil)
}

func (strings *Strings) NotificationPackage(name, version string) string {
	if version == "" {
		return strings.localize(messageNotificationPackageWithoutVersion, map[string]any{"Name": name}, nil)
	}
	return strings.localize(messageNotificationPackageWithVersion, map[string]any{
		"Name":    name,
		"Version": version,
	}, nil)
}

func (strings *Strings) NotificationUpdatesTitle(count int) string {
	return strings.localize(messageNotificationUpdatesTitle, map[string]any{"Count": count}, count)
}

func (strings *Strings) NotificationPackages(names string, remaining int) string {
	if remaining == 0 {
		return strings.localize(messageNotificationPackages, map[string]any{"Names": names}, nil)
	}
	return strings.localize(messageNotificationPackagesWithMore, map[string]any{
		"Names": names,
		"Count": remaining,
	}, remaining)
}

func (strings *Strings) UpgradePackageDescription(name string) string {
	return strings.localize(messageUpgradePackageDescription, map[string]any{"Name": name}, nil)
}

func (strings *Strings) UnknownPackageKind(kind string) string {
	return strings.localize(messageErrorPackageKindUnknown, map[string]any{"Kind": kind}, nil)
}

// FormatTime keeps date formatting deterministic for the currently supported
// languages without adding another formatting dependency.
func (strings *Strings) FormatTime(value time.Time) string {
	base, _ := strings.language.Base()
	german, _ := language.German.Base()
	if base == german {
		return value.Format("02.01.2006, 15:04")
	}
	return value.Format("Jan 2, 2006, 3:04 PM")
}
