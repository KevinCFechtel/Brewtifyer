#import <Foundation/Foundation.h>
#import <UserNotifications/UserNotifications.h>

@interface BrewtifyerNotificationDelegate : NSObject <UNUserNotificationCenterDelegate>
@end

@implementation BrewtifyerNotificationDelegate

- (void)userNotificationCenter:(UNUserNotificationCenter *)center
       willPresentNotification:(UNNotification *)notification
         withCompletionHandler:(void (^)(UNNotificationPresentationOptions options))completionHandler
{
    completionHandler(UNNotificationPresentationOptionBanner | UNNotificationPresentationOptionSound);
}

@end

static BrewtifyerNotificationDelegate *BrewtifyerDelegate(void)
{
    static BrewtifyerNotificationDelegate *delegate;
    static dispatch_once_t onceToken;
    dispatch_once(&onceToken, ^{
        delegate = [[BrewtifyerNotificationDelegate alloc] init];
    });
    return delegate;
}

void BrewtifyerSendNotification(const char *title, const char *body)
{
    if (title == NULL || body == NULL) {
        return;
    }

    @autoreleasepool {
        NSString *notificationTitle = [NSString stringWithUTF8String:title];
        NSString *notificationBody = [NSString stringWithUTF8String:body];
        if (notificationTitle == nil || notificationBody == nil) {
            return;
        }

        UNUserNotificationCenter *center = [UNUserNotificationCenter currentNotificationCenter];
        center.delegate = BrewtifyerDelegate();
        [center requestAuthorizationWithOptions:(UNAuthorizationOptionAlert | UNAuthorizationOptionSound)
                              completionHandler:^(BOOL granted, NSError *error) {
            if (!granted || error != nil) {
                return;
            }

            @autoreleasepool {
                UNMutableNotificationContent *content = [[UNMutableNotificationContent alloc] init];
                content.title = notificationTitle;
                content.body = notificationBody;
                content.sound = [UNNotificationSound defaultSound];

                UNNotificationRequest *request = [UNNotificationRequest
                    requestWithIdentifier:[[NSUUID UUID] UUIDString]
                                  content:content
                                  trigger:nil];
                [center addNotificationRequest:request withCompletionHandler:nil];
            }
        }];
    }
}
