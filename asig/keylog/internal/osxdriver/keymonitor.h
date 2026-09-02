//go:build darwin

typedef struct Node Node; 
typedef struct List List;
typedef struct KeyMonitor KeyMonitor;
typedef struct EventData EventData;

struct EventData {
    int type;
    int modifierFlags;
    short keyCode; 
    const char * characters;
};

void Start_Key_Monitor();
EventData Global_Key_Monitor_Pop();