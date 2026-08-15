#import <Foundation/Foundation.h>
#import <ServiceManagement/ServiceManagement.h>
#include <string.h>

enum BrewtifyerAutostartStatus {
    BrewtifyerAutostartStatusError = -1,
    BrewtifyerAutostartStatusUnsupported = 0,
    BrewtifyerAutostartStatusDisabled = 1,
    BrewtifyerAutostartStatusEnabled = 2,
    BrewtifyerAutostartStatusRequiresApproval = 3,
    BrewtifyerAutostartStatusNotFound = 4,
};

static int BrewtifyerMapAutostartStatus(SMAppServiceStatus status)
API_AVAILABLE(macos(13.0))
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
    if (@available(macOS 13.0, *)) {
        return BrewtifyerMapAutostartStatus(SMAppService.mainAppService.status);
    }
    return BrewtifyerAutostartStatusUnsupported;
}

int BrewtifyerSetAutostartEnabled(int enabled, char **errorMessage)
{
    if (errorMessage != NULL) {
        *errorMessage = NULL;
    }
    if (@available(macOS 13.0, *)) {
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
    return BrewtifyerAutostartStatusUnsupported;
}

int BrewtifyerOpenAutostartSettings(void)
{
    if (@available(macOS 13.0, *)) {
        dispatch_async(dispatch_get_main_queue(), ^{
            [SMAppService openSystemSettingsLoginItems];
        });
        return 1;
    }
    return 0;
}
