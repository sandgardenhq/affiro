#include <windows.h>
#include <stdio.h>
#include <stdlib.h> 
#include <fcntl.h> 
#include <errno.h>
#include <string.h>
#include <fileapi.h>
#include "dllmain.h"

HHOOK keyHook;
HHOOK mouseHook;
HINSTANCE hinst;

extern LRESULT CALLBACK keyEventCallback(int code,WPARAM wParam,LPARAM lParam) {
    if (code == 0) {
        HANDLE pipe;
        // NB: keep this in sync with Go code; TODO: Go could tell us what the path to use is. 
        pipe = CreateFile(TEXT("\\\\.\\pipe\\asig-keytracker"), GENERIC_WRITE, 0, NULL, OPEN_EXISTING, 0, NULL);
        if (pipe != INVALID_HANDLE_VALUE) { // if the pipe isn't available, do nothing
            __int64 typ = 1;
            __int64 codeInt64 = (__int64)code;
            __int64 wParamInt64 = (__int64)wParam;
            __int64 lParamInt64 = (__int64)lParam;
            DWORD dwBytesWritten = 0;
            if (!WriteFile(pipe, &typ, 8, &dwBytesWritten, NULL)) {
                CloseHandle(pipe);
                return CallNextHookEx(keyHook,code,wParam,lParam);
            }
            if (!WriteFile(pipe, &codeInt64, 8, &dwBytesWritten, NULL)) {
                CloseHandle(pipe);
                return CallNextHookEx(keyHook,code,wParam,lParam);
            }
            if (!WriteFile(pipe, &wParamInt64, 8, &dwBytesWritten, NULL)) {
                CloseHandle(pipe);
                return CallNextHookEx(keyHook,code,wParam,lParam);
            }
            if (!WriteFile(pipe, &lParamInt64, 8, &dwBytesWritten, NULL)) {
                CloseHandle(pipe);
                return CallNextHookEx(keyHook,code,wParam,lParam);
            }
            CloseHandle(pipe);
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
            HANDLE pipe;
            pipe = CreateFile(TEXT("\\\\.\\pipe\\asig-keytracker"), GENERIC_WRITE, 0, NULL, OPEN_EXISTING, 0, NULL);
            if (pipe != INVALID_HANDLE_VALUE) { // if the pipe isn't available, do nothing
                __int64 typ = 2;
                __int64 codeInt64 = (__int64)code;
                __int64 wParamInt64 = (__int64)wParam;
                __int64 lParamInt64 = (__int64)lParam;
                DWORD dwBytesWritten = 0;
                if (!WriteFile(pipe, &typ, 8, &dwBytesWritten, NULL)) {
                    CloseHandle(pipe);
                    return CallNextHookEx(mouseHook,code,wParam,lParam);
                }
                if (!WriteFile(pipe, &codeInt64, 8, &dwBytesWritten, NULL)) {
                    CloseHandle(pipe);
                    return CallNextHookEx(mouseHook,code,wParam,lParam);
                }
                if (!WriteFile(pipe, &wParamInt64, 8, &dwBytesWritten, NULL)) {
                    CloseHandle(pipe);
                    return CallNextHookEx(mouseHook,code,wParam,lParam);
                }
                if (!WriteFile(pipe, &lParamInt64, 8, &dwBytesWritten, NULL)) {
                    CloseHandle(pipe);
                    return CallNextHookEx(mouseHook,code,wParam,lParam);
                }
                CloseHandle(pipe);
            }
        }
    }
    return CallNextHookEx(mouseHook,code,wParam,lParam);
}


extern void install() {
    // TODO: _LL hook variants eg WH_KEYBOARD_LL are much simpler (don't require a DLL, don't hook into every binary)
    // and would be preferred if we could resolve them quickly enough and accept their input; they are synchronous 
    // instead of async so they directly impact user perception of events     
    keyHook = SetWindowsHookEx(WH_KEYBOARD, keyEventCallback, hinst, 0);
    mouseHook = SetWindowsHookEx(WH_MOUSE, mouseEventCallback, hinst, 0);
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

