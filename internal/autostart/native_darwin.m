#import <Foundation/Foundation.h>
#import <ServiceManagement/ServiceManagement.h>
#include <string.h>

// Brewtifyer's deployment target is macOS 13, which is also the macOS floor of
// the Go toolchain pinned in go.mod. SMAppService is therefore always available
// and no @available fallback is needed. BrewtifyerAutostartStatusUnsupported
// remains part of the protocol because the non-darwin build reports it.
enum BrewtifyerAutostartStatus {
    BrewtifyerAutostartStatusError = -1,
    BrewtifyerAutostartStatusUnsupported = 0,
    BrewtifyerAutostartStatusDisabled = 1,
    BrewtifyerAutostartStatusEnabled = 2,
    BrewtifyerAutostartStatusRequiresApproval = 3,
    BrewtifyerAutostartStatusNotFound = 4,
};

static int BrewtifyerMapAutostartStatus(SMAppServiceStatus status)
{
    switch (status) {
        case SMAppServiceStatusNotRegistered:
            return BrewtifyerAutostartStatusDisabled;
        case SMAppServiceStatusEnabled:
            return BrewtifyerAutostartStatusEnabled;
        case SMAppServiceStatusRequiresApproval:
            return BrewtifyerAutostartStatusRequiresApproval;
        case SMAppServiceStatusNotFound:
            return BrewtifyerAutostartStatusNotFound;
    }
    return BrewtifyerAutostartStatusNotFound;
}

static void BrewtifyerSetErrorMessage(char **errorMessage, NSError *error)
{
    if (errorMessage == NULL) {
        return;
    }
    NSString *message = error.localizedDescription;
    if (message == nil || message.length == 0) {
        message = @"Launch at login could not be changed";
    }
    *errorMessage = strdup(message.UTF8String);
}

int BrewtifyerAutostartStatus(void)
{
    return BrewtifyerMapAutostartStatus(SMAppService.mainAppService.status);
}

int BrewtifyerSetAutostartEnabled(int enabled, char **errorMessage)
{
    if (errorMessage != NULL) {
        *errorMessage = NULL;
    }

    SMAppService *service = SMAppService.mainAppService;
    int currentStatus = BrewtifyerMapAutostartStatus(service.status);

    if ((enabled && currentStatus == BrewtifyerAutostartStatusEnabled) ||
        (!enabled && currentStatus == BrewtifyerAutostartStatusDisabled)) {
        return currentStatus;
    }
    if (enabled && currentStatus == BrewtifyerAutostartStatusRequiresApproval) {
        return currentStatus;
    }

    NSError *error = nil;
    BOOL succeeded = enabled
        ? [service registerAndReturnError:&error]
        : [service unregisterAndReturnError:&error];
    int resultingStatus = BrewtifyerMapAutostartStatus(service.status);
    if (succeeded || resultingStatus == BrewtifyerAutostartStatusRequiresApproval) {
        return resultingStatus;
    }

    BrewtifyerSetErrorMessage(errorMessage, error);
    return BrewtifyerAutostartStatusError;
}

int BrewtifyerOpenAutostartSettings(void)
{
    dispatch_async(dispatch_get_main_queue(), ^{
        [SMAppService openSystemSettingsLoginItems];
    });
    return 1;
}
