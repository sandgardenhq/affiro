#include <windows.h>
#include <stdio.h>
#include <stdlib.h> 
#include <fcntl.h> 
#include "dllmain.h"

HHOOK keyHook;
HHOOK mouseHook;
HINSTANCE hinst;

extern LRESULT CALLBACK keyEventCallback(int code,WPARAM wParam,LPARAM lParam) {   
    if (code == 0) {
        int pipe;
        // NB: keep this in sync with Go code; TODO: Go could tell us what the path to use is. 
        pipe = open(TEXT("\\\\.\\pipe\\asig-keytracker"), O_WRONLY);
        if (pipe != -1) { // if the pipe isn't available, do nothing
            __int64 typ = 1;
            __int64 codeInt64 = (__int64)code;
            __int64 wParamInt64 = (__int64)wParam;
            __int64 lParamInt64 = (__int64)lParam;
            int n;
            n = write(pipe, &typ, 8);
            if (n < 8) {
                return CallNextHookEx(keyHook,code,wParam,lParam);
            }
            n = write(pipe, &codeInt64, 8);
            if (n < 8) {
                return CallNextHookEx(keyHook,code,wParam,lParam);
            }
            n = write(pipe, &wParamInt64, 8);
            if (n < 8) {
                return CallNextHookEx(keyHook,code,wParam,lParam);
            }
            n = write(pipe, &lParamInt64, 8);
            if (n < 8) {
                return CallNextHookEx(keyHook,code,wParam,lParam);
            }
            close(pipe);
        }
    }
    // Note you cannot access stderr, stdout, or make calls to Go exported functions
    // from within this callback; prints will do nothing, and Go callbacks will segfault. 
    // Also note, any changes to global variables within this callback will not be visible 
    // to successive calls to these callbacks; all of this is why we need to send the data
    // to our initializing process over a pipe. 
    return CallNextHookEx(keyHook,code,wParam,lParam);
}

extern LRESULT CALLBACK mouseEventCallback(int code,WPARAM wParam,LPARAM lParam) {   
    if (code >= 0) {
        if (wParam == WM_RBUTTONDOWN || wParam == WM_LBUTTONDOWN) {
            int pipe;
            pipe = open(TEXT("\\\\.\\pipe\\asig-keytracker"), O_WRONLY);
            if (pipe != -1) { // if the pipe isn't available, do nothing
                __int64 typ = 2;
                __int64 codeInt64 = (__int64)code;
                __int64 wParamInt64 = (__int64)wParam;
                __int64 lParamInt64 = (__int64)lParam;
                int n;
                n = write(pipe, &typ, 8);
                if (n < 8) {
                    return CallNextHookEx(mouseHook,code,wParam,lParam);
                }
                n = write(pipe, &codeInt64, 8);
                if (n < 8) {
                    return CallNextHookEx(mouseHook,code,wParam,lParam);
                }
                n = write(pipe, &wParamInt64, 8);
                if (n < 8) {
                    return CallNextHookEx(mouseHook,code,wParam,lParam);
                }
                n = write(pipe, &lParamInt64, 8);
                if (n < 8) {
                    return CallNextHookEx(mouseHook,code,wParam,lParam);
                }
                close(pipe);
            }
        }
    }
    return CallNextHookEx(mouseHook,code,wParam,lParam);
}


extern void install() {
    //fprintf(stderr, "install executed\n");
    // TODO: _LL hook variants eg WH_KEYBOARD_LL are much simpler (don't require a DLL, don't hook into every binary)
    // and would be preferred if we could resolve them quickly enough and accept their input; they are synchronous 
    // instead of async so they directly impact user perception of events     
    keyHook = SetWindowsHookEx(WH_KEYBOARD, keyEventCallback, hinst, 0);
    mouseHook = SetWindowsHookEx(WH_MOUSE, mouseEventCallback, hinst, 0);
    //fprintf(stderr, "windows hooks created\n");
}
extern void uninstall() {
    UnhookWindowsHookEx(keyHook); 
    UnhookWindowsHookEx(mouseHook); 
}

BOOL WINAPI DllMain(
    HINSTANCE _hinstDLL,  // handle to DLL module
    DWORD _fdwReason,     // reason for calling function
    LPVOID _lpReserved)   // reserved
{
    switch (_fdwReason) {
    case DLL_PROCESS_ATTACH:
        // Track HINSTANCE 
        hinst = _hinstDLL;
        break;
    case DLL_PROCESS_DETACH:
        // Perform any necessary cleanup.
        break;
    case DLL_THREAD_DETACH:
        // Do thread-specific cleanup.
        break;
    case DLL_THREAD_ATTACH:
		// Do thread-specific initialization.
        break;
    }
    return TRUE;
}

