//go:build darwin


#import <Cocoa/Cocoa.h>

#import <AppKit/AppKit.h>
#import <Foundation/Foundation.h>

#include <stdio.h>
#include <pthread.h>

#include "keymonitor.h"

typedef struct Node Node;
typedef struct List List;
typedef struct EventData EventData;

struct Node {
    EventData data;
    Node *next;
};

struct List {
    Node *head;
    Node *tail;
};

List * events;
static pthread_mutex_t events_mutex = PTHREAD_MUTEX_INITIALIZER;

void wait_for_events() {
 [NSEvent addGlobalMonitorForEventsMatchingMask:NSEventMaskLeftMouseUp | NSEventMaskRightMouseUp | NSEventMaskKeyDown
        handler:^(NSEvent *event){
            //NSLog(@"keydown: %@", event.characters);
            EventData ed;
            ed.type = [event type];
            ed.modifierFlags = [event modifierFlags];
            if (ed.type == NSEventTypeKeyDown || ed.type == NSEventTypeKeyUp || ed.type == NSEventTypeFlagsChanged) {
                ed.keyCode = [event keyCode];
                ed.characters = [[event characters] UTF8String];
            } else {
                ed.keyCode = 0;
            }


            pthread_mutex_lock(&events_mutex);
            // i.e. linked list push
            if (events == NULL) {
                events = (List *)calloc(1, sizeof (List));
            }
            if (events->head == NULL) {
                events->head = calloc(1, sizeof (Node));
                events->head->data = ed;
                events->tail = events->head;
            } else {
                Node * next = calloc(1, sizeof (Node));
                next->data = ed;
                events->tail->next = next;
                events->tail = next;
            }
            pthread_mutex_unlock(&events_mutex);
        }];
}

@interface AppDelegate : NSObject <NSApplicationDelegate>

- (void)applicationDidFinishLaunching:(NSNotification *)aNotification;

@end

@implementation AppDelegate

// 10.9+ only, see this url for compatibility:
// http://stackoverflow.com/questions/17693408/enable-access-for-assistive-devices-programmatically-on-10-9
BOOL checkAccessibility()
{
    NSDictionary* opts = @{(__bridge id)kAXTrustedCheckOptionPrompt: @YES};
    return AXIsProcessTrustedWithOptions((__bridge CFDictionaryRef)opts);
}

- (void)applicationDidFinishLaunching:(NSNotification *)aNotification
{
    if (checkAccessibility()) {
        // TODO: mark this as a debug log
        //NSLog(@"Accessibility Enabled");
    }
    else {
        // TODO: mark this as an error log
        NSLog(@"Accessibility Disabled");
    }
    // TODO: refactor, test, etc
    wait_for_events();
}

@end

void Start_Key_Monitor() {
    pthread_mutex_lock(&events_mutex);
    if (events == NULL) {
        events = (List *)calloc(1, sizeof (List));
    }
    pthread_mutex_unlock(&events_mutex);
    if (events == NULL)
        return;

    @autoreleasepool {
        NSApplication * application;
        if (NSApp == NULL) {
            //NSLog(@"nsapp null");
            application = [NSApplication sharedApplication];
        } else {
            // An app running in this program already exists; we can start watching for events
            // without creating a second app (in fact, we have to; attempting to
            // create a second app without coordinating with the first for who gets
            // to be active, will mean we create a useless app)
            wait_for_events();
            return;
        }
        AppDelegate *delegate = [[AppDelegate alloc] init];

        //NSLog(@"setting delegate");
        [application setDelegate:delegate];
        if ([application isRunning]) {
            //NSLog(@"already running");
        } else {
            //NSLog(@"not running");
        }
        if ([application isActive]) {
            //NSLog(@"active");
        } else {
            //NSLog(@"not active");
            [application activate];
            //NSLog(@"activated");
        }
        [application run];
        // TODO: the run call above runs forever; there must be some way to release it;
        // we use a global events slice because this function can't return anything to its
        // caller given run won't terminate; you could instead create your event slice first
        // and provide it to this function to be fed into the delegate in a constructor.
        //NSLog(@"exiting run");
    }
    //NSLog(@"exiting run (2)");
}

EventData Global_Key_Monitor_Pop() {
    pthread_mutex_lock(&events_mutex);
    if (events == NULL || events->head == NULL) {
        pthread_mutex_unlock(&events_mutex);
        EventData ed;
        ed.keyCode = 32000;
        return ed;
    }
    Node * popped = events->head;
    events->head = popped->next;
    if (events->head == NULL) {
        events->tail = NULL;
    }
    EventData ed = popped->data;
    pthread_mutex_unlock(&events_mutex);
    free(popped);
    return ed;
}
