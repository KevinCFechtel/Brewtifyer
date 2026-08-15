package localization

import "github.com/nicksnyder/go-i18n/v2/i18n"

var (
	messageTrayTooltip = &i18n.Message{
		ID:          "Tray.Tooltip",
		Description: "Tooltip for the Brewtifyer menu bar icon.",
		Other:       "Brewtifyer – Homebrew updates",
	}
	messageTrayChecking = &i18n.Message{
		ID:          "Tray.Checking",
		Description: "Status shown while Homebrew is being checked.",
		Other:       "Checking Homebrew …",
	}
	messageTrayCurrentStatusTooltip = &i18n.Message{
		ID:          "Tray.CurrentStatusTooltip",
		Description: "Tooltip for the current Homebrew status row.",
		Other:       "Current Homebrew status",
	}
	messageTrayNotChecked = &i18n.Message{
		ID:          "Tray.NotChecked",
		Description: "Status before Homebrew has been checked for the first time.",
		Other:       "Not checked yet",
	}
	messageTrayLastCheckTooltip = &i18n.Message{
		ID:          "Tray.LastCheckTooltip",
		Description: "Tooltip for the time of the last Homebrew check.",
		Other:       "Time of the last check",
	}
	messageTrayUpgradePackageMenuTooltip = &i18n.Message{
		ID:          "Tray.UpgradePackageMenuTooltip",
		Description: "Initial tooltip for package update menu entries.",
		Other:       "Update in Terminal",
	}
	messageTrayUpgradeAll = &i18n.Message{
		ID:          "Tray.UpgradeAll",
		Description: "Menu action that installs all available Homebrew updates.",
		Other:       "Install all updates …",
	}
	messageTrayUpgradeAllTooltip = &i18n.Message{
		ID:          "Tray.UpgradeAllTooltip",
		Description: "Tooltip for the action that runs brew upgrade.",
		Other:       "Run brew upgrade in Terminal",
	}
	messageTrayRefresh = &i18n.Message{
		ID:          "Tray.Refresh",
		Description: "Menu action that checks Homebrew immediately.",
		Other:       "Check now",
	}
	messageTrayRefreshTooltip = &i18n.Message{
		ID:          "Tray.RefreshTooltip",
		Description: "Tooltip for the immediate Homebrew check action.",
		Other:       "Check Homebrew for updates now",
	}
	messageTrayOpenLoginItems = &i18n.Message{
		ID:          "Tray.OpenLoginItems",
		Description: "Menu action that opens the macOS Login Items settings.",
		Other:       "Open Login Items in System Settings …",
	}
	messageTrayOpenLoginItemsTooltip = &i18n.Message{
		ID:          "Tray.OpenLoginItemsTooltip",
		Description: "Tooltip for opening Login Items settings to approve Brewtifyer.",
		Other:       "Allow Brewtifyer to launch at login in macOS",
	}
	messageTrayQuit = &i18n.Message{
		ID:          "Tray.Quit",
		Description: "Menu action that quits Brewtifyer.",
		Other:       "Quit",
	}
	messageTrayQuitTooltip = &i18n.Message{
		ID:          "Tray.QuitTooltip",
		Description: "Tooltip for the quit action.",
		Other:       "Quit Brewtifyer",
	}
	messageTrayCheckFailed = &i18n.Message{
		ID:          "Tray.CheckFailed",
		Description: "Status shown when checking Homebrew failed.",
		Other:       "Check failed",
	}
	messageTrayUpToDate = &i18n.Message{
		ID:          "Tray.UpToDate",
		Description: "Status shown when no Homebrew updates are available.",
		Other:       "Homebrew is up to date",
	}
	messageTrayLastChecked = &i18n.Message{
		ID:          "Tray.LastChecked",
		Description: "Status containing the localized time of the last successful check.",
		Other:       "Last checked: {{.Date}}",
	}
	messageTrayLastCheckedWithWarning = &i18n.Message{
		ID:          "Tray.LastCheckedWithWarning",
		Description: "Status containing the last check time when the result includes a warning.",
		Other:       "Last checked: {{.Date}} (with warning)",
	}
	messageTrayLastSuccessfulCheckTooltip = &i18n.Message{
		ID:          "Tray.LastSuccessfulCheckTooltip",
		Description: "Tooltip for a successful Homebrew check.",
		Other:       "Time of the last successful check",
	}
	messageTrayMoreUpdates = &i18n.Message{
		ID:          "Tray.MoreUpdates",
		Description: "Number of additional updates hidden because the menu shows at most ten packages.",
		One:         "… and {{.Count}} more update",
		Other:       "… and {{.Count}} more updates",
	}
	messageTrayUpgradePackageRunning = &i18n.Message{
		ID:          "Tray.UpgradePackageRunning",
		Description: "Status after an individual upgrade was opened in Terminal.",
		Other:       "Update is running in Terminal …",
	}
	messageTrayUpgradeAllRunning = &i18n.Message{
		ID:          "Tray.UpgradeAllRunning",
		Description: "Status after all upgrades were opened in Terminal.",
		Other:       "Updates are running in Terminal …",
	}
	messageTrayUpgradeLaunchFailed = &i18n.Message{
		ID:          "Tray.UpgradeLaunchFailed",
		Description: "Status shown when the update Terminal could not be opened.",
		Other:       "The update Terminal could not be opened",
	}
	messageTrayAutostartManageFailed = &i18n.Message{
		ID:          "Tray.AutostartManageFailed",
		Description: "Status shown when launch at login could not be changed.",
		Other:       "Launch at login could not be changed",
	}
	messageTrayAutostartTitle = &i18n.Message{
		ID:          "Tray.AutostartTitle",
		Description: "Menu checkbox for launching Brewtifyer when the user logs in.",
		Other:       "Launch at login",
	}
	messageTrayAutostartEnableTooltip = &i18n.Message{
		ID:          "Tray.AutostartEnableTooltip",
		Description: "Tooltip when launch at login can be enabled.",
		Other:       "Launch Brewtifyer automatically after login",
	}
	messageTrayAutostartDisableTooltip = &i18n.Message{
		ID:          "Tray.AutostartDisableTooltip",
		Description: "Tooltip when launch at login can be disabled.",
		Other:       "Disable launch at login for Brewtifyer",
	}
	messageTrayAutostartApprovalTitle = &i18n.Message{
		ID:          "Tray.AutostartApprovalTitle",
		Description: "Launch at login menu title when macOS approval is required.",
		Other:       "Launch at login (approval required)",
	}
	messageTrayAutostartApprovalTooltip = &i18n.Message{
		ID:          "Tray.AutostartApprovalTooltip",
		Description: "Tooltip explaining that launch at login must be approved in macOS.",
		Other:       "Allow Brewtifyer in macOS System Settings",
	}
	messageTrayAutostartRegisterTooltip = &i18n.Message{
		ID:          "Tray.AutostartRegisterTooltip",
		Description: "Tooltip when Brewtifyer can be registered as a login item.",
		Other:       "Register Brewtifyer as a login item",
	}
	messageTrayAutostartUnsupportedTitle = &i18n.Message{
		ID:          "Tray.AutostartUnsupportedTitle",
		Description: "Launch at login menu title on unsupported macOS versions.",
		Other:       "Launch at login (macOS 13 or later)",
	}
	messageTrayAutostartUnsupportedTooltip = &i18n.Message{
		ID:          "Tray.AutostartUnsupportedTooltip",
		Description: "Tooltip explaining the minimum macOS version for launch at login.",
		Other:       "This feature requires macOS 13 or later",
	}
	messageTrayPackageTitle = &i18n.Message{
		ID:          "Tray.PackageTitle",
		Description: "Package row containing its name, installed version, and available version.",
		Other:       "{{.Name}}: {{.Installed}} → {{.Current}}",
	}
	messageTrayPinnedPackageTitle = &i18n.Message{
		ID:          "Tray.PinnedPackageTitle",
		Description: "Package row for a pinned package.",
		Other:       "{{.Name}}: {{.Installed}} → {{.Current}} · pinned",
	}
	messageTrayPinnedPackageTooltip = &i18n.Message{
		ID:          "Tray.PinnedPackageTooltip",
		Description: "Tooltip explaining how Homebrew handles a pinned package during an upgrade.",
		Other:       "Open in Terminal; Homebrew skips pinned packages",
	}
	messageTrayPackageUpgradeTooltip = &i18n.Message{
		ID:          "Tray.PackageUpgradeTooltip",
		Description: "Tooltip for upgrading one package in Terminal.",
		Other:       "Update this package in Terminal",
	}
	messageTrayUpdatesAvailable = &i18n.Message{
		ID:          "Tray.UpdatesAvailable",
		Description: "Homebrew status containing the number of available updates.",
		One:         "{{.Count}} update available",
		Other:       "{{.Count}} updates available",
	}
	messageNotificationUpdateTitle = &i18n.Message{
		ID:          "Notification.UpdateTitle",
		Description: "Notification title when one new Homebrew update is available.",
		Other:       "Homebrew update available",
	}
	messageNotificationPackageWithoutVersion = &i18n.Message{
		ID:          "Notification.PackageWithoutVersion",
		Description: "Notification body for one package when no target version is known.",
		Other:       "A new version is available for {{.Name}}.",
	}
	messageNotificationPackageWithVersion = &i18n.Message{
		ID:          "Notification.PackageWithVersion",
		Description: "Notification body for one package with its available target version.",
		Other:       "{{.Name}} can be updated to {{.Version}}.",
	}
	messageNotificationUpdatesTitle = &i18n.Message{
		ID:          "Notification.UpdatesTitle",
		Description: "Notification title containing the number of newly available Homebrew updates.",
		One:         "{{.Count}} new Homebrew update",
		Other:       "{{.Count}} new Homebrew updates",
	}
	messageNotificationPackages = &i18n.Message{
		ID:          "Notification.Packages",
		Description: "Notification body listing all newly available package names.",
		Other:       "Newly available for: {{.Names}}.",
	}
	messageNotificationPackagesWithMore = &i18n.Message{
		ID:          "Notification.PackagesWithMore",
		Description: "Notification body listing three package names and a count of additional packages.",
		One:         "Newly available for: {{.Names}}, and {{.Count}} more update.",
		Other:       "Newly available for: {{.Names}}, and {{.Count}} more updates.",
	}
	messageUpgradePackageDescription = &i18n.Message{
		ID:          "Upgrade.PackageDescription",
		Description: "Heading in Terminal when one package is being upgraded.",
		Other:       "Homebrew update for {{.Name}}",
	}
	messageUpgradeAllDescription = &i18n.Message{
		ID:          "Upgrade.AllDescription",
		Description: "Heading in Terminal when all packages are being upgraded.",
		Other:       "All Homebrew updates",
	}
	messageUpgradeCompleted = &i18n.Message{
		ID:          "Upgrade.Completed",
		Description: "Terminal message after Homebrew finished successfully.",
		Other:       "Update completed. Check Brewtifyer again afterwards.",
	}
	messageUpgradeFailed = &i18n.Message{
		ID:          "Upgrade.Failed",
		Description: "Terminal printf format after Homebrew exited unsuccessfully; %d is replaced by the exit status.",
		Other:       "Update exited with status %d.",
	}
	messageUpgradePressAnyKey = &i18n.Message{
		ID:          "Upgrade.PressAnyKey",
		Description: "Terminal prompt shown before the update window closes.",
		Other:       "Press any key to close the window …",
	}
	messageErrorHomebrewNotFound = &i18n.Message{
		ID:          "Error.HomebrewNotFound",
		Description: "Error shown when no Homebrew executable can be found.",
		Other:       "Homebrew was not found",
	}
	messageErrorPackageNameMissing = &i18n.Message{
		ID:          "Error.PackageNameMissing",
		Description: "Error shown when an update package has no name.",
		Other:       "Package name is missing",
	}
	messageErrorPackageNameInvalid = &i18n.Message{
		ID:          "Error.PackageNameInvalid",
		Description: "Error shown when a package name contains an invalid null character.",
		Other:       "Package name contains an invalid character",
	}
	messageErrorPackageKindUnknown = &i18n.Message{
		ID:          "Error.PackageKindUnknown",
		Description: "Error shown when an update package has an unknown kind.",
		Other:       "Unknown package type: {{.Kind}}",
	}
	messageErrorCreateUpgradeCommand = &i18n.Message{
		ID:          "Error.CreateUpgradeCommand",
		Description: "Error prefix when the temporary update command cannot be created.",
		Other:       "The temporary update command could not be created",
	}
	messageErrorMakeUpgradeCommandExecutable = &i18n.Message{
		ID:          "Error.MakeUpgradeCommandExecutable",
		Description: "Error prefix when the temporary update command cannot be made executable.",
		Other:       "The update command could not be made executable",
	}
	messageErrorWriteUpgradeCommand = &i18n.Message{
		ID:          "Error.WriteUpgradeCommand",
		Description: "Error prefix when the temporary update command cannot be written.",
		Other:       "The update command could not be written",
	}
	messageErrorCloseUpgradeCommand = &i18n.Message{
		ID:          "Error.CloseUpgradeCommand",
		Description: "Error prefix when the temporary update command cannot be closed.",
		Other:       "The update command could not be closed",
	}
	messageErrorOpenTerminal = &i18n.Message{
		ID:          "Error.OpenTerminal",
		Description: "Error prefix when Terminal cannot be opened.",
		Other:       "Terminal could not be opened",
	}
)
